package intentonnx

import (
	"fmt"
	"os"
	"strings"
	"sync"

	smtok "github.com/sugarme/tokenizer"
	ort "github.com/yalue/onnxruntime_go"
)

// EmbedEngine runs a feature-extraction ONNX model and returns L2-normalized sentence vectors.
type EmbedEngine struct {
	mu sync.Mutex

	seqLen    int
	embedDim  int
	tokenizer *smtok.Tokenizer
	sess      *ort.DynamicAdvancedSession
	inputs    []ort.InputOutputInfo
	outputs   []ort.InputOutputInfo
	inNames   []string
	outNames  []string
}

// NewEmbedEngine loads a sentence-embedding ONNX model (e.g. bge-small-zh-v1.5).
func NewEmbedEngine(opt Options) (*EmbedEngine, error) {
	lib := strings.TrimSpace(opt.SharedLibraryPath)
	if lib == "" {
		lib = strings.TrimSpace(os.Getenv("ONNXRUNTIME_SHARED_LIBRARY_PATH"))
	}
	if lib == "" {
		return nil, fmt.Errorf("intentonnx: set Options.SharedLibraryPath or ONNXRUNTIME_SHARED_LIBRARY_PATH")
	}
	model := strings.TrimSpace(opt.ModelPath)
	if model == "" {
		return nil, fmt.Errorf("intentonnx: ModelPath required")
	}
	tokPath := strings.TrimSpace(opt.TokenizerPath)
	if tokPath == "" {
		return nil, fmt.Errorf("intentonnx: TokenizerPath required")
	}
	seq := opt.SeqLen
	if seq <= 0 {
		seq = 128
	}

	if err := requireRuntime(); err != nil {
		return nil, err
	}

	var sessOpts *ort.SessionOptions
	if opt.UseCoreML {
		o, err := ort.NewSessionOptions()
		if err != nil {
			return nil, err
		}
		if err := o.AppendExecutionProviderCoreMLV2(nil); err != nil {
			_ = o.Destroy()
			return nil, err
		}
		sessOpts = o
	}

	tk, err := newPreparedTokenizer(tokPath, seq)
	if err != nil {
		return nil, err
	}

	inputs, outputs, err := ort.GetInputOutputInfoWithOptions(model, sessOpts)
	if err != nil {
		return nil, err
	}
	dim := hiddenDim(outputs)
	if dim <= 0 {
		return nil, fmt.Errorf("intentonnx: could not infer embedding hidden dim (need last_hidden_state or 3D float tensor)")
	}

	inNames := make([]string, len(inputs))
	for i := range inputs {
		inNames[i] = inputs[i].Name
	}
	outNames := make([]string, len(outputs))
	for i := range outputs {
		outNames[i] = outputs[i].Name
	}

	sess, err := ort.NewDynamicAdvancedSession(model, inNames, outNames, sessOpts)
	if sessOpts != nil {
		_ = sessOpts.Destroy()
	}
	if err != nil {
		return nil, err
	}

	return &EmbedEngine{
		seqLen:    seq,
		embedDim:  dim,
		tokenizer: tk,
		sess:      sess,
		inputs:    inputs,
		outputs:   outputs,
		inNames:   inNames,
		outNames:  outNames,
	}, nil
}

func hiddenDim(outputs []ort.InputOutputInfo) int {
	for _, o := range outputs {
		if o.OrtValueType != ort.ONNXTypeTensor || o.DataType != ort.TensorElementDataTypeFloat {
			continue
		}
		if len(o.Dimensions) == 3 {
			last := o.Dimensions[2]
			if last > 0 {
				return int(last)
			}
		}
	}
	for _, o := range outputs {
		if o.OrtValueType != ort.ONNXTypeTensor || o.DataType != ort.TensorElementDataTypeFloat {
			continue
		}
		if len(o.Dimensions) >= 1 {
			last := o.Dimensions[len(o.Dimensions)-1]
			if last > 0 && last != 128 && last != 256 && last != 512 && last != 768 && last != 1024 {
				continue
			}
			if last > 0 {
				return int(last)
			}
		}
	}
	return 0
}

// Close releases the ONNX session.
func (e *EmbedEngine) Close() error {
	if e == nil {
		return nil
	}
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.sess != nil {
		err := e.sess.Destroy()
		e.sess = nil
		return err
	}
	return nil
}

// EmbedDim returns the sentence vector size.
func (e *EmbedEngine) EmbedDim() int {
	if e == nil {
		return 0
	}
	return e.embedDim
}

// Embed encodes one utterance into a unit-length vector.
func (e *EmbedEngine) Embed(text string) ([]float32, error) {
	if e == nil {
		return nil, fmt.Errorf("intentonnx: nil embed engine")
	}
	text = NormalizeTranscript(text)
	if text == "" {
		return nil, fmt.Errorf("intentonnx: empty text")
	}

	enc, err := e.tokenizer.EncodeSingle(text, true)
	if err != nil {
		return nil, err
	}
	if len(enc.Ids) != e.seqLen {
		return nil, fmt.Errorf("intentonnx: encoded len %d != seq %d", len(enc.Ids), e.seqLen)
	}

	e.mu.Lock()
	defer e.mu.Unlock()

	inVals, err := buildInputTensorsFromMeta(e.inputs, e.seqLen, enc)
	if err != nil {
		return nil, err
	}
	defer destroyOrtValues(inVals)

	outVals := make([]ort.Value, len(e.outputs))
	defer destroyOrtValues(outVals)

	if err := e.sess.Run(inVals, outVals); err != nil {
		return nil, err
	}

	hidden, err := extractHiddenStates(outVals, e.outputs)
	if err != nil {
		return nil, err
	}
	return meanPoolNormalize(hidden, e.seqLen, e.embedDim, enc.AttentionMask)
}

func buildInputTensorsFromMeta(inputs []ort.InputOutputInfo, seqLen int, enc *smtok.Encoding) ([]ort.Value, error) {
	var vals []ort.Value
	for _, meta := range inputs {
		if meta.OrtValueType != ort.ONNXTypeTensor {
			destroyOrtValues(vals)
			return nil, fmt.Errorf("intentonnx: unsupported input type for %q", meta.Name)
		}
		shape, err := concreteShape(meta.Dimensions, seqLen, 1)
		if err != nil {
			destroyOrtValues(vals)
			return nil, err
		}
		n := int(shape.FlattenedSize())
		if n <= 0 {
			destroyOrtValues(vals)
			return nil, fmt.Errorf("intentonnx: empty shape for %q", meta.Name)
		}
		switch meta.DataType {
		case ort.TensorElementDataTypeInt64:
			data := make([]int64, n)
			switch meta.Name {
			case "input_ids":
				intsToInt64Row(data, enc.Ids)
			case "attention_mask":
				intsToInt64Row(data, enc.AttentionMask)
			case "token_type_ids":
				intsToInt64Row(data, enc.TypeIds)
			default:
				destroyOrtValues(vals)
				return nil, fmt.Errorf("intentonnx: unknown int64 input %q", meta.Name)
			}
			t, err := ort.NewTensor(shape, data)
			if err != nil {
				destroyOrtValues(vals)
				return nil, err
			}
			vals = append(vals, t)
		case ort.TensorElementDataTypeInt32:
			data := make([]int32, n)
			switch meta.Name {
			case "input_ids":
				intsToInt32Row(data, enc.Ids)
			case "attention_mask":
				intsToInt32Row(data, enc.AttentionMask)
			case "token_type_ids":
				intsToInt32Row(data, enc.TypeIds)
			default:
				destroyOrtValues(vals)
				return nil, fmt.Errorf("intentonnx: unknown int32 input %q", meta.Name)
			}
			t, err := ort.NewTensor(shape, data)
			if err != nil {
				destroyOrtValues(vals)
				return nil, err
			}
			vals = append(vals, t)
		default:
			destroyOrtValues(vals)
			return nil, fmt.Errorf("intentonnx: unsupported element type for %q", meta.Name)
		}
	}
	return vals, nil
}

func extractHiddenStates(outVals []ort.Value, meta []ort.InputOutputInfo) ([]float32, error) {
	for i, v := range outVals {
		if v == nil {
			continue
		}
		if meta[i].Name == "logits" {
			continue
		}
		if t, ok := v.(*ort.Tensor[float32]); ok {
			dims := meta[i].Dimensions
			if len(dims) >= 3 || meta[i].Name == "last_hidden_state" {
				return append([]float32(nil), t.GetData()...), nil
			}
		}
	}
	for _, v := range outVals {
		if v == nil {
			continue
		}
		if t, ok := v.(*ort.Tensor[float32]); ok {
			return append([]float32(nil), t.GetData()...), nil
		}
	}
	return nil, fmt.Errorf("intentonnx: no float32 hidden-state tensor found")
}
