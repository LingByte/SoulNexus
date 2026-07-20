package handlers

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/LingByte/SoulNexus/internal/models"
	"github.com/LingByte/SoulNexus/pkg/intentonnx"
	"github.com/LingByte/SoulNexus/pkg/nlu"
)

func copyNluFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}
	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()
	if _, err := io.Copy(out, in); err != nil {
		return err
	}
	return out.Close()
}

// trainTenantNluModelMaterialize writes tenant intents and trains routing assets.
func trainTenantNluModelMaterialize(row *models.TenantNluModel, spec models.TenantNluSpec) error {
	if row == nil {
		return fmt.Errorf("nlu: nil model")
	}
	global := nlu.Get()
	if !global.Ready() {
		return fmt.Errorf("nlu: platform base model not configured (NLU_MODEL / NLU_TOKENIZER / NLU_ORT_LIB)")
	}
	if len(spec.Intents) == 0 {
		return fmt.Errorf("nlu: at least one intent required")
	}
	if len(spec.Intents) > intentonnx.MaxEmbedIntents {
		return fmt.Errorf("nlu: at most %d intents supported", intentonnx.MaxEmbedIntents)
	}
	for i, ent := range spec.Intents {
		if strings.TrimSpace(ent.Name) == "" || strings.TrimSpace(ent.Reply) == "" {
			return fmt.Errorf("nlu: intents[%d] requires name and reply", i)
		}
	}

	dir := models.TenantNluDir(row.TenantID, row.ID)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}

	icfg := spec.ToIntentConfig()
	if global.IsEmbeddingMode() {
		if spec.MinSoftmaxProb <= 0 {
			icfg.MinSoftmaxProb = 0.55
		}
	} else {
		if spec.MinSoftmaxProb <= 0 {
			icfg.MinSoftmaxProb = 0.22
		}
	}
	if spec.KeywordLogitBonus <= 0 {
		icfg.KeywordLogitBonus = 3.5
	}
	if strings.TrimSpace(icfg.DefaultReply) == "" {
		icfg.DefaultReply = "抱歉，没能完全确定您的需求。"
	}

	intentsPath := filepath.Join(dir, "intents.json")
	raw, err := json.MarshalIndent(icfg, "", "  ")
	if err != nil {
		return err
	}
	if err := os.WriteFile(intentsPath, raw, 0o644); err != nil {
		return err
	}

	modelPath := global.ModelPath
	tokenizerPath := global.TokenizerPath
	prototypesPath := filepath.Join(dir, "prototypes.json")

	if global.IsEmbeddingMode() {
		samples := make([][]string, len(spec.Intents))
		for i, ent := range spec.Intents {
			samples[i] = append([]string(nil), ent.Samples...)
		}
		if err := nlu.BuildEmbeddingPrototypes(prototypesPath, icfg, samples); err != nil {
			return err
		}
		nlu.InvalidateProfileFull(modelPath, tokenizerPath, intentsPath, prototypesPath)
		paths := nlu.ProfilePaths{
			ModelPath:      modelPath,
			TokenizerPath:  tokenizerPath,
			IntentsPath:    intentsPath,
			PrototypesPath: prototypesPath,
			MinConfidence:  row.MinConfidence,
		}
		st, err := nlu.ProfileStatus(paths)
		if err != nil {
			return err
		}
		if !st.Ready {
			return fmt.Errorf("nlu: engine not ready after train")
		}
		row.NumClasses = len(spec.Intents)
		row.StorageDir = dir
		row.Status = models.TenantNluStatusReady
		row.TrainError = ""
		return nil
	}

	numClasses := row.NumClasses
	if numClasses <= 0 {
		n, err := nlu.ProbeNumClasses(global.ModelPath, global.TokenizerPath)
		if err != nil {
			return err
		}
		numClasses = n
	}
	if len(spec.Intents) != numClasses {
		return fmt.Errorf("nlu: intent count %d must match base model classes %d (classifier mode; set NLU_MODE=embedding to add intents freely)", len(spec.Intents), numClasses)
	}

	modelDst := filepath.Join(dir, "model.onnx")
	tokDst := filepath.Join(dir, "tokenizer.json")
	if err := copyNluFile(global.ModelPath, modelDst); err != nil {
		return fmt.Errorf("copy base model: %w", err)
	}
	if err := copyNluFile(global.TokenizerPath, tokDst); err != nil {
		return fmt.Errorf("copy tokenizer: %w", err)
	}
	modelPath = modelDst
	tokenizerPath = tokDst

	nlu.InvalidateProfile(modelPath, tokenizerPath, intentsPath)
	paths := nlu.ProfilePaths{
		ModelPath:     modelPath,
		TokenizerPath: tokenizerPath,
		IntentsPath:   intentsPath,
		MinConfidence: row.MinConfidence,
	}
	st, err := nlu.ProfileStatus(paths)
	if err != nil {
		return err
	}
	if !st.Ready {
		return fmt.Errorf("nlu: engine not ready after train")
	}

	row.NumClasses = numClasses
	row.StorageDir = dir
	row.Status = models.TenantNluStatusReady
	row.TrainError = ""
	return nil
}
