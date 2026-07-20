package intentonnx

import (
	"fmt"
	"os"
	"strings"
	"sync"

	smtok "github.com/sugarme/tokenizer"
	ort "github.com/yalue/onnxruntime_go"
)

// Options configures a single ONNX intent engine (one model + one tokenizer).
type Options struct {
	// SharedLibraryPath is required (e.g. libonnxruntime.dylib). See onnxruntime_go docs.
	SharedLibraryPath string
	ModelPath         string
	TokenizerPath     string
	SeqLen            int
	UseCoreML         bool
}

// Engine loads the model once and runs thread-safe inference with [Engine.Route].
type Engine struct {
	mu sync.Mutex

	seqLen int

	tokenizer *smtok.Tokenizer

	sess *ort.DynamicAdvancedSession

	inputs     []ort.InputOutputInfo
	outputs    []ort.InputOutputInfo
	inNames    []string
	outNames   []string
	numClasses int
}

// NewEngine creates an engine. Initializes ONNX Runtime once per process (first call wins for library path).
func NewEngine(opt Options) (*Engine, error) {
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
	nClass := logitsClassCount(outputs)
	if nClass <= 0 {
		return nil, fmt.Errorf("intentonnx: could not infer logits class count")
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

	return &Engine{
		seqLen:     seq,
		tokenizer:  tk,
		sess:       sess,
		inputs:     inputs,
		outputs:    outputs,
		inNames:    inNames,
		outNames:   outNames,
		numClasses: nClass,
	}, nil
}

// Close releases the ONNX session. Does not shut down the global ORT environment.
func (e *Engine) Close() error {
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

// NumClasses returns the logits trailing dimension (number of intent labels).
func (e *Engine) NumClasses() int {
	if e == nil {
		return 0
	}
	return e.numClasses
}

// SeqLen returns the configured sequence length.
func (e *Engine) SeqLen() int {
	if e == nil {
		return 0
	}
	return e.seqLen
}

// Route runs one forward pass and returns a single-channel routing decision.
func (e *Engine) Route(text string, cfg *IntentConfig, opts RouteOptions) (*RouteOutput, error) {
	if e == nil {
		return nil, fmt.Errorf("intentonnx: nil engine")
	}
	text = NormalizeTranscript(text)
	if text == "" {
		return nil, fmt.Errorf("intentonnx: empty text")
	}
	if err := ValidateIntentConfig(cfg, e.numClasses); err != nil {
		return nil, err
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

	inVals, err := e.buildInputTensors(enc)
	if err != nil {
		return nil, err
	}
	defer destroyOrtValues(inVals)

	outVals := make([]ort.Value, len(e.outputs))
	defer destroyOrtValues(outVals)

	if err := e.sess.Run(inVals, outVals); err != nil {
		return nil, err
	}

	logits, err := extractLogits(outVals, e.outputs)
	if err != nil {
		return nil, err
	}

	pred := buildPrediction(text, logits, cfg, opts)
	reply, ch := finalizeRoute(&pred, cfg, opts)
	out := &RouteOutput{
		Channel:    ch,
		Reply:      reply,
		Prediction: pred,
	}
	if err := ValidateSingleChannel(out); err != nil {
		return nil, err
	}
	return out, nil
}

func (e *Engine) buildInputTensors(enc *smtok.Encoding) ([]ort.Value, error) {
	var vals []ort.Value
	for _, meta := range e.inputs {
		if meta.OrtValueType != ort.ONNXTypeTensor {
			destroyOrtValues(vals)
			return nil, fmt.Errorf("intentonnx: unsupported input type for %q", meta.Name)
		}
		shape, err := concreteShape(meta.Dimensions, e.seqLen, 1)
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
		case ort.TensorElementDataTypeFloat:
			data := make([]float32, n)
			t, err := ort.NewTensor(shape, data)
			if err != nil {
				destroyOrtValues(vals)
				return nil, err
			}
			vals = append(vals, t)
		case ort.TensorElementDataTypeDouble:
			data := make([]float64, n)
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

func destroyOrtValues(vs []ort.Value) {
	for _, v := range vs {
		if v == nil {
			continue
		}
		_ = v.Destroy()
	}
}

func extractLogits(outVals []ort.Value, meta []ort.InputOutputInfo) ([]float32, error) {
	for i, v := range outVals {
		if v == nil {
			continue
		}
		if meta[i].Name != "logits" {
			continue
		}
		t, ok := v.(*ort.Tensor[float32])
		if !ok {
			return nil, fmt.Errorf("intentonnx: logits output is not float32 tensor")
		}
		data := t.GetData()
		return append([]float32(nil), data...), nil
	}
	for i, v := range outVals {
		if v == nil {
			continue
		}
		if t, ok := v.(*ort.Tensor[float32]); ok {
			return append([]float32(nil), t.GetData()...), nil
		}
		_ = i
	}
	return nil, fmt.Errorf("intentonnx: no float32 logits tensor found")
}
