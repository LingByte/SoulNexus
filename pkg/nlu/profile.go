package nlu

import (
	"fmt"
	"sync"

	"github.com/LingByte/SoulNexus/pkg/intentonnx"
)

type profileKey string

func profileCacheKey(modelPath, tokenizerPath, intentsPath, prototypesPath string) profileKey {
	return profileKey(modelPath + "\x00" + tokenizerPath + "\x00" + intentsPath + "\x00" + prototypesPath)
}

type classifierProfile struct {
	eng *intentonnx.Engine
	cfg *intentonnx.IntentConfig
	err error
}

type embeddingProfile struct {
	eng   *intentonnx.EmbedEngine
	cfg   *intentonnx.IntentConfig
	store *intentonnx.EmbedPrototypeStore
	err   error
}

var (
	profileMu      sync.Mutex
	classProfiles  = map[profileKey]classifierProfile{}
	embedProfiles  = map[profileKey]embeddingProfile{}
)

// ProfilePaths locates ONNX assets for one tenant NLU model.
type ProfilePaths struct {
	ModelPath      string
	TokenizerPath  string
	IntentsPath    string
	PrototypesPath string
	MinConfidence  float64
}

// InvalidateProfile drops a cached engine (after train or config change).
func InvalidateProfile(modelPath, tokenizerPath, intentsPath string) {
	InvalidateProfileFull(modelPath, tokenizerPath, intentsPath, "")
}

// InvalidateProfileFull drops cached engines including embedding prototypes.
func InvalidateProfileFull(modelPath, tokenizerPath, intentsPath, prototypesPath string) {
	profileMu.Lock()
	defer profileMu.Unlock()
	k := profileCacheKey(modelPath, tokenizerPath, intentsPath, prototypesPath)
	if e, ok := classProfiles[k]; ok && e.eng != nil {
		_ = e.eng.Close()
	}
	delete(classProfiles, k)
	if e, ok := embedProfiles[k]; ok && e.eng != nil {
		_ = e.eng.Close()
	}
	delete(embedProfiles, k)
}

func getClassifierProfile(paths ProfilePaths) (*intentonnx.Engine, *intentonnx.IntentConfig, EngineState, error) {
	global := Get()
	st := EngineState{SeqLen: global.SeqLen, Mode: ModeClassifier}
	if paths.ModelPath == "" || paths.TokenizerPath == "" || paths.IntentsPath == "" {
		return nil, nil, st, fmt.Errorf("nlu: incomplete profile paths")
	}
	if global.ORTLibPath == "" {
		return nil, nil, st, fmt.Errorf("nlu: ONNX Runtime library not configured")
	}

	k := profileCacheKey(paths.ModelPath, paths.TokenizerPath, paths.IntentsPath, "")
	profileMu.Lock()
	if e, ok := classProfiles[k]; ok {
		if e.err != nil {
			st.LoadError = e.err.Error()
			profileMu.Unlock()
			return nil, nil, st, e.err
		}
		st.Ready = true
		st.NumClasses = e.eng.NumClasses()
		st.Intents = intentNames(e.cfg)
		profileMu.Unlock()
		return e.eng, e.cfg, st, nil
	}
	profileMu.Unlock()

	if err := intentonnx.InitRuntime(global.ORTLibPath); err != nil {
		st.LoadError = err.Error()
		return nil, nil, st, err
	}
	eng, err := intentonnx.NewEngine(intentonnx.Options{
		SharedLibraryPath: global.ORTLibPath,
		ModelPath:         paths.ModelPath,
		TokenizerPath:     paths.TokenizerPath,
		SeqLen:            global.SeqLen,
		UseCoreML:         global.UseCoreML,
	})
	if err != nil {
		st.LoadError = err.Error()
		profileMu.Lock()
		classProfiles[k] = classifierProfile{err: err}
		profileMu.Unlock()
		return nil, nil, st, err
	}
	icfg, err := intentonnx.LoadIntentConfig(paths.IntentsPath)
	if err != nil {
		_ = eng.Close()
		st.LoadError = err.Error()
		profileMu.Lock()
		classProfiles[k] = classifierProfile{err: err}
		profileMu.Unlock()
		return nil, nil, st, err
	}
	if err := intentonnx.ValidateIntentConfig(icfg, eng.NumClasses()); err != nil {
		_ = eng.Close()
		st.LoadError = err.Error()
		profileMu.Lock()
		classProfiles[k] = classifierProfile{err: err}
		profileMu.Unlock()
		return nil, nil, st, err
	}

	profileMu.Lock()
	classProfiles[k] = classifierProfile{eng: eng, cfg: icfg}
	st.Ready = true
	st.NumClasses = eng.NumClasses()
	st.Intents = intentNames(icfg)
	profileMu.Unlock()
	return eng, icfg, st, nil
}

func getEmbeddingProfile(paths ProfilePaths) (*intentonnx.EmbedEngine, *intentonnx.IntentConfig, *intentonnx.EmbedPrototypeStore, EngineState, error) {
	global := Get()
	st := EngineState{SeqLen: global.SeqLen, Mode: ModeEmbedding}
	if paths.ModelPath == "" || paths.TokenizerPath == "" || paths.IntentsPath == "" || paths.PrototypesPath == "" {
		return nil, nil, nil, st, fmt.Errorf("nlu: incomplete embedding profile paths")
	}
	if global.ORTLibPath == "" {
		return nil, nil, nil, st, fmt.Errorf("nlu: ONNX Runtime library not configured")
	}

	k := profileCacheKey(paths.ModelPath, paths.TokenizerPath, paths.IntentsPath, paths.PrototypesPath)
	profileMu.Lock()
	if e, ok := embedProfiles[k]; ok {
		if e.err != nil {
			st.LoadError = e.err.Error()
			profileMu.Unlock()
			return nil, nil, nil, st, e.err
		}
		st.Ready = true
		st.NumClasses = len(e.cfg.Intents)
		st.Intents = intentNames(e.cfg)
		profileMu.Unlock()
		return e.eng, e.cfg, e.store, st, nil
	}
	profileMu.Unlock()

	if err := intentonnx.InitRuntime(global.ORTLibPath); err != nil {
		st.LoadError = err.Error()
		return nil, nil, nil, st, err
	}
	eng, err := intentonnx.NewEmbedEngine(intentonnx.Options{
		SharedLibraryPath: global.ORTLibPath,
		ModelPath:         paths.ModelPath,
		TokenizerPath:     paths.TokenizerPath,
		SeqLen:            global.SeqLen,
		UseCoreML:         global.UseCoreML,
	})
	if err != nil {
		st.LoadError = err.Error()
		profileMu.Lock()
		embedProfiles[k] = embeddingProfile{err: err}
		profileMu.Unlock()
		return nil, nil, nil, st, err
	}
	icfg, err := intentonnx.LoadIntentConfig(paths.IntentsPath)
	if err != nil {
		_ = eng.Close()
		st.LoadError = err.Error()
		profileMu.Lock()
		embedProfiles[k] = embeddingProfile{err: err}
		profileMu.Unlock()
		return nil, nil, nil, st, err
	}
	if err := intentonnx.ValidateIntentConfigFlexible(icfg); err != nil {
		_ = eng.Close()
		st.LoadError = err.Error()
		profileMu.Lock()
		embedProfiles[k] = embeddingProfile{err: err}
		profileMu.Unlock()
		return nil, nil, nil, st, err
	}
	store, err := intentonnx.LoadEmbedPrototypes(paths.PrototypesPath)
	if err != nil {
		_ = eng.Close()
		st.LoadError = err.Error()
		profileMu.Lock()
		embedProfiles[k] = embeddingProfile{err: err}
		profileMu.Unlock()
		return nil, nil, nil, st, err
	}

	profileMu.Lock()
	embedProfiles[k] = embeddingProfile{eng: eng, cfg: icfg, store: store}
	st.Ready = true
	st.NumClasses = len(icfg.Intents)
	st.Intents = intentNames(icfg)
	profileMu.Unlock()
	return eng, icfg, store, st, nil
}

// ProfileStatus checks whether a profile can load without running inference.
func ProfileStatus(paths ProfilePaths) (EngineState, error) {
	global := Get()
	if global.IsEmbeddingMode() {
		_, _, _, st, err := getEmbeddingProfile(paths)
		return st, err
	}
	_, _, st, err := getClassifierProfile(paths)
	return st, err
}

// ParseProfile runs intent routing for a tenant NLU profile.
func ParseProfile(paths ProfilePaths, text string) (*intentonnx.RouteOutput, error) {
	global := Get()
	minConf := paths.MinConfidence
	if minConf <= 0 {
		minConf = global.MinConfidence
	}
	opts := intentonnx.RouteOptions{
		VoiceASRHints:     true,
		UncertainMeansLLM: true,
	}

	if global.IsEmbeddingMode() {
		eng, icfg, store, _, err := getEmbeddingProfile(paths)
		if err != nil {
			return nil, err
		}
		out, err := eng.RouteByEmbedding(text, icfg, store, opts)
		if err != nil {
			return nil, err
		}
		if out != nil && out.Prediction.Confidence < minConf {
			out.Channel = intentonnx.AnswerChannelLLM
			out.Reply = ""
		}
		return out, nil
	}

	eng, icfg, _, err := getClassifierProfile(paths)
	if err != nil {
		return nil, err
	}
	out, err := eng.Route(text, icfg, opts)
	if err != nil {
		return nil, err
	}
	if out != nil && out.Prediction.Confidence < minConf {
		out.Channel = intentonnx.AnswerChannelLLM
		out.Reply = ""
	}
	return out, nil
}

// ProbeNumClasses loads the base ONNX model once to read output class count (classifier mode only).
func ProbeNumClasses(modelPath, tokenizerPath string) (int, error) {
	global := Get()
	if global.IsEmbeddingMode() {
		return 0, nil
	}
	if global.ORTLibPath == "" {
		return 0, fmt.Errorf("nlu: ORT library not configured")
	}
	if err := intentonnx.InitRuntime(global.ORTLibPath); err != nil {
		return 0, err
	}
	eng, err := intentonnx.NewEngine(intentonnx.Options{
		SharedLibraryPath: global.ORTLibPath,
		ModelPath:         modelPath,
		TokenizerPath:     tokenizerPath,
		SeqLen:            global.SeqLen,
		UseCoreML:         global.UseCoreML,
	})
	if err != nil {
		return 0, err
	}
	defer func() { _ = eng.Close() }()
	return eng.NumClasses(), nil
}

// BuildEmbeddingPrototypes trains centroid vectors for a tenant spec (embedding mode).
func BuildEmbeddingPrototypes(prototypesPath string, icfg *intentonnx.IntentConfig, samplesByIntent [][]string) error {
	global := Get()
	if !global.Ready() {
		return fmt.Errorf("nlu: platform base model not configured")
	}
	if err := intentonnx.InitRuntime(global.ORTLibPath); err != nil {
		return err
	}
	eng, err := intentonnx.NewEmbedEngine(intentonnx.Options{
		SharedLibraryPath: global.ORTLibPath,
		ModelPath:         global.ModelPath,
		TokenizerPath:     global.TokenizerPath,
		SeqLen:            global.SeqLen,
		UseCoreML:         global.UseCoreML,
	})
	if err != nil {
		return err
	}
	defer func() { _ = eng.Close() }()

	var store *intentonnx.EmbedPrototypeStore
	if len(samplesByIntent) > 0 {
		store, err = intentonnx.BuildEmbedPrototypesWithSamples(eng, icfg, samplesByIntent)
	} else {
		store, err = intentonnx.BuildEmbedPrototypes(eng, icfg)
	}
	if err != nil {
		return err
	}
	return intentonnx.SaveEmbedPrototypes(prototypesPath, store)
}
