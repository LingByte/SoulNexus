package intentonnx

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

const MaxEmbedIntents = 64

// EmbedPrototype is one intent centroid for similarity routing.
type EmbedPrototype struct {
	Name   string    `json:"name"`
	Vector []float32 `json:"vector"`
}

// EmbedPrototypeStore holds trained intent centroids.
type EmbedPrototypeStore struct {
	Dim     int              `json:"dim"`
	Intents []EmbedPrototype `json:"intents"`
}

// LoadEmbedPrototypes reads prototypes.json.
func LoadEmbedPrototypes(path string) (*EmbedPrototypeStore, error) {
	raw, err := os.ReadFile(strings.TrimSpace(path))
	if err != nil {
		return nil, err
	}
	var store EmbedPrototypeStore
	if err := json.Unmarshal(raw, &store); err != nil {
		return nil, err
	}
	if store.Dim <= 0 || len(store.Intents) == 0 {
		return nil, fmt.Errorf("intentonnx: invalid prototypes file")
	}
	for i, ent := range store.Intents {
		if strings.TrimSpace(ent.Name) == "" {
			return nil, fmt.Errorf("intentonnx: prototypes[%d] missing name", i)
		}
		if len(ent.Vector) != store.Dim {
			return nil, fmt.Errorf("intentonnx: prototypes[%d] dim mismatch", i)
		}
	}
	return &store, nil
}

// SaveEmbedPrototypes writes prototypes.json.
func SaveEmbedPrototypes(path string, store *EmbedPrototypeStore) error {
	if store == nil {
		return fmt.Errorf("intentonnx: nil prototype store")
	}
	raw, err := json.MarshalIndent(store, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, raw, 0o644)
}

// ValidateIntentConfigFlexible checks intents for embedding mode (any count >= 1).
func ValidateIntentConfigFlexible(cfg *IntentConfig) error {
	if cfg == nil {
		return fmt.Errorf("nil intent config")
	}
	if len(cfg.Intents) == 0 {
		return fmt.Errorf("at least one intent required")
	}
	if len(cfg.Intents) > MaxEmbedIntents {
		return fmt.Errorf("at most %d intents supported", MaxEmbedIntents)
	}
	for i := range cfg.Intents {
		if strings.TrimSpace(cfg.Intents[i].Reply) == "" {
			return fmt.Errorf("intents[%d] missing reply", i)
		}
	}
	return nil
}

// BuildEmbedPrototypes encodes intent phrases and averages them into centroids.
func BuildEmbedPrototypes(eng *EmbedEngine, cfg *IntentConfig) (*EmbedPrototypeStore, error) {
	if eng == nil {
		return nil, fmt.Errorf("intentonnx: nil embed engine")
	}
	if err := ValidateIntentConfigFlexible(cfg); err != nil {
		return nil, err
	}
	out := &EmbedPrototypeStore{Dim: eng.EmbedDim()}
	for _, ent := range cfg.Intents {
		phrases := intentPhrases(ent.Name, ent.Reply, ent.Keywords, nil)
		if len(phrases) == 0 {
			return nil, fmt.Errorf("intentonnx: intent %q has no training phrases", ent.Name)
		}
		vecs := make([][]float32, 0, len(phrases))
		for _, p := range phrases {
			v, err := eng.Embed(p)
			if err != nil {
				return nil, fmt.Errorf("embed %q: %w", p, err)
			}
			vecs = append(vecs, v)
		}
		centroid, err := averageVectors(vecs)
		if err != nil {
			return nil, err
		}
		out.Intents = append(out.Intents, EmbedPrototype{
			Name:   strings.TrimSpace(ent.Name),
			Vector: centroid,
		})
	}
	return out, nil
}

// BuildEmbedPrototypesWithSamples includes tenant sample utterances in centroid training.
func BuildEmbedPrototypesWithSamples(eng *EmbedEngine, cfg *IntentConfig, samplesByIntent [][]string) (*EmbedPrototypeStore, error) {
	if eng == nil {
		return nil, fmt.Errorf("intentonnx: nil embed engine")
	}
	if err := ValidateIntentConfigFlexible(cfg); err != nil {
		return nil, err
	}
	if len(samplesByIntent) != len(cfg.Intents) {
		return nil, fmt.Errorf("intentonnx: samples length mismatch")
	}
	out := &EmbedPrototypeStore{Dim: eng.EmbedDim()}
	for i, ent := range cfg.Intents {
		phrases := intentPhrases(ent.Name, ent.Reply, ent.Keywords, samplesByIntent[i])
		if len(phrases) == 0 {
			return nil, fmt.Errorf("intentonnx: intent %q has no training phrases", ent.Name)
		}
		vecs := make([][]float32, 0, len(phrases))
		for _, p := range phrases {
			v, err := eng.Embed(p)
			if err != nil {
				return nil, fmt.Errorf("embed %q: %w", p, err)
			}
			vecs = append(vecs, v)
		}
		centroid, err := averageVectors(vecs)
		if err != nil {
			return nil, err
		}
		out.Intents = append(out.Intents, EmbedPrototype{
			Name:   strings.TrimSpace(ent.Name),
			Vector: centroid,
		})
	}
	return out, nil
}
