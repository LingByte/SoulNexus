package nlu

import (
	"fmt"
	"sync"

	"github.com/LingByte/SoulNexus/pkg/intentonnx"
)

var (
	engineMu     sync.Mutex
	engineInst   *intentonnx.Engine
	intentCfg    *intentonnx.IntentConfig
	engineReady  bool
	engineGaveUp bool
	engineErr    error
)

// ResetEngine drops the cached ONNX engine so the next call reloads model/config.
func ResetEngine() {
	engineMu.Lock()
	defer engineMu.Unlock()
	if engineInst != nil {
		_ = engineInst.Close()
	}
	engineInst = nil
	intentCfg = nil
	engineReady = false
	engineGaveUp = false
	engineErr = nil
}

// EngineState describes runtime ONNX NLU availability.
type EngineState struct {
	Ready      bool
	LoadError  string
	Mode       string
	NumClasses int
	SeqLen     int
	Intents    []string
}

// GetEngine returns a loaded engine and intent config, or an error when unavailable.
func GetEngine() (*intentonnx.Engine, *intentonnx.IntentConfig, EngineState, error) {
	cfg := Get()
	st := EngineState{SeqLen: cfg.SeqLen}
	if !cfg.Ready() {
		return nil, nil, st, fmt.Errorf("nlu: model/tokenizer/ORT library not configured")
	}

	engineMu.Lock()
	defer engineMu.Unlock()

	if engineGaveUp {
		st.LoadError = engineErrString()
		return nil, nil, st, fmt.Errorf("nlu: %s", st.LoadError)
	}
	if engineReady && engineInst != nil {
		st.Ready = true
		st.NumClasses = engineInst.NumClasses()
		st.Intents = intentNames(intentCfg)
		return engineInst, intentCfg, st, nil
	}

	if err := intentonnx.InitRuntime(cfg.ORTLibPath); err != nil {
		engineGaveUp = true
		engineErr = err
		st.LoadError = err.Error()
		return nil, nil, st, err
	}
	eng, err := intentonnx.NewEngine(intentonnx.Options{
		SharedLibraryPath: cfg.ORTLibPath,
		ModelPath:         cfg.ModelPath,
		TokenizerPath:     cfg.TokenizerPath,
		SeqLen:            cfg.SeqLen,
		UseCoreML:         cfg.UseCoreML,
	})
	if err != nil {
		engineGaveUp = true
		engineErr = err
		st.LoadError = err.Error()
		return nil, nil, st, err
	}
	icfg, err := intentonnx.LoadIntentConfig(cfg.IntentsPath)
	if err != nil {
		_ = eng.Close()
		engineGaveUp = true
		engineErr = err
		st.LoadError = err.Error()
		return nil, nil, st, err
	}
	if err := intentonnx.ValidateIntentConfig(icfg, eng.NumClasses()); err != nil {
		_ = eng.Close()
		engineGaveUp = true
		engineErr = err
		st.LoadError = err.Error()
		return nil, nil, st, err
	}

	engineInst = eng
	intentCfg = icfg
	engineReady = true
	st.Ready = true
	st.NumClasses = eng.NumClasses()
	st.Intents = intentNames(icfg)
	return engineInst, intentCfg, st, nil
}

// Parse runs intent classification for lab / debugging.
func Parse(text string) (*intentonnx.RouteOutput, error) {
	eng, icfg, _, err := GetEngine()
	if err != nil {
		return nil, err
	}
	cfg := Get()
	out, err := eng.Route(text, icfg, intentonnx.RouteOptions{
		VoiceASRHints:     true,
		UncertainMeansLLM: true,
	})
	if err != nil {
		return nil, err
	}
	if out != nil && out.Prediction.Confidence < cfg.MinConfidence {
		out.Channel = intentonnx.AnswerChannelLLM
		out.Reply = ""
	}
	return out, nil
}

func intentNames(cfg *intentonnx.IntentConfig) []string {
	if cfg == nil {
		return nil
	}
	out := make([]string, 0, len(cfg.Intents))
	for _, ent := range cfg.Intents {
		out = append(out, ent.Name)
	}
	return out
}

func engineErrString() string {
	if engineErr != nil {
		return engineErr.Error()
	}
	return "engine unavailable"
}
