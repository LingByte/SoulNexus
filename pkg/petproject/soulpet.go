package petproject

import (
	"encoding/json"
	"fmt"
	"strings"

	"gopkg.in/yaml.v3"
)

const (
	SoulpetSpecVersion = 1
	SoulpetYamlFile    = "soulpet.yaml"

	KindSprite  = "sprite"
	KindLive2D  = "live2d"
	KindCustom  = "custom"
	ManifestFile = "manifest.json"
)

// SoulpetMeta is package-level metadata (soulpet.yaml).
type SoulpetMeta struct {
	SpecVersion int    `yaml:"specVersion" json:"specVersion"`
	Name        string `yaml:"name" json:"name"`
	Description string `yaml:"description" json:"description,omitempty"`
	Author      string `yaml:"author" json:"author,omitempty"`
	License     string `yaml:"license" json:"license,omitempty"`
	Kind        string `yaml:"kind" json:"kind"`
	Version     string `yaml:"version" json:"version,omitempty"`
	Voice       *SoulpetVoiceMeta `yaml:"voice" json:"voice,omitempty"`
	Market      *SoulpetMarketMeta `yaml:"market" json:"market,omitempty"`
}

type SoulpetVoiceMeta struct {
	AgentID      string `yaml:"agentId" json:"agentId,omitempty"`
	CmdVoiceBase string `yaml:"cmdVoiceBase" json:"cmdVoiceBase,omitempty"`
}

type SoulpetMarketMeta struct {
	Tags         []string `yaml:"tags" json:"tags,omitempty"`
	PreviewEmoji string   `yaml:"previewEmoji" json:"previewEmoji,omitempty"`
	Visibility   string   `yaml:"visibility" json:"visibility,omitempty"`
}

// PetManifest is the runtime manifest.json (sprite / live2d / custom).
type PetManifest struct {
	Version int    `json:"version"`
	Type    string `json:"type"`
	Name    string `json:"name"`
	Assets  struct {
		Sprite *SpriteAssets `json:"sprite,omitempty"`
		Live2D *Live2DAssets `json:"live2d,omitempty"`
	} `json:"assets"`
	Layout     map[string]interface{} `json:"layout,omitempty"`
	EmotionMap map[string]string      `json:"emotionMap,omitempty"`
	Behaviors  map[string]interface{} `json:"behaviors,omitempty"`
}

type SpriteAssets struct {
	BaseURL          string                            `json:"baseUrl"`
	DefaultAnimation string                            `json:"defaultAnimation,omitempty"`
	Animations       map[string]map[string]interface{} `json:"animations"`
}

type Live2DAssets struct {
	BaseURL     string            `json:"baseUrl"`
	Model       string            `json:"model"`
	Textures    []string          `json:"textures,omitempty"`
	Motions     map[string]string `json:"motions,omitempty"`
	Expressions map[string]string `json:"expressions,omitempty"`
	Physics     string            `json:"physics,omitempty"`
	Pose        string            `json:"pose,omitempty"`
}

// ValidationIssue describes a package validation problem.
type ValidationIssue struct {
	Field   string `json:"field"`
	Message string `json:"message"`
	Level   string `json:"level"` // error | warn
}

func ParseSoulpetYAML(raw string) (*SoulpetMeta, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, fmt.Errorf("empty soulpet.yaml")
	}
	var meta SoulpetMeta
	if err := yaml.Unmarshal([]byte(raw), &meta); err != nil {
		return nil, err
	}
	if meta.SpecVersion == 0 {
		meta.SpecVersion = SoulpetSpecVersion
	}
	return &meta, nil
}

func ParseManifestJSON(raw string) (*PetManifest, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, fmt.Errorf("empty manifest.json")
	}
	var m PetManifest
	if err := json.Unmarshal([]byte(raw), &m); err != nil {
		return nil, err
	}
	return &m, nil
}

func InferKind(files map[string]string) string {
	if raw, ok := files[SoulpetYamlFile]; ok && strings.TrimSpace(raw) != "" {
		if meta, err := ParseSoulpetYAML(raw); err == nil && meta.Kind != "" {
			return meta.Kind
		}
	}
	if raw, ok := files[ManifestFile]; ok {
		if m, err := ParseManifestJSON(raw); err == nil && m.Type != "" {
			return m.Type
		}
	}
	return KindCustom
}

func EnsureSoulpetYAML(files map[string]string, fallbackName string) string {
	if raw, ok := files[SoulpetYamlFile]; ok && strings.TrimSpace(raw) != "" {
		return raw
	}
	kind := InferKind(files)
	name := strings.TrimSpace(fallbackName)
	if name == "" {
		if m, err := ParseManifestJSON(files[ManifestFile]); err == nil && m.Name != "" {
			name = m.Name
		}
	}
	if name == "" {
		name = "Soul Pet"
	}
	meta := SoulpetMeta{
		SpecVersion: SoulpetSpecVersion,
		Name:        name,
		Kind:        kind,
		Version:     "1.0.0",
	}
	b, err := yaml.Marshal(&meta)
	if err != nil {
		return ""
	}
	return string(b)
}

func ValidateSoulpetPackage(files map[string]string, entryScriptValidator func(entry string, files map[string]string) (string, []string)) (kind string, issues []ValidationIssue) {
	if len(files) == 0 {
		return "", []ValidationIssue{{Field: "files", Message: "包为空", Level: "error"}}
	}
	if err := ValidateFiles(files); err != nil {
		issues = append(issues, ValidationIssue{Field: "files", Message: err.Error(), Level: "error"})
	}

	manifestRaw, hasManifest := files[ManifestFile]
	if !hasManifest || strings.TrimSpace(manifestRaw) == "" {
		issues = append(issues, ValidationIssue{Field: ManifestFile, Message: "缺少 manifest.json", Level: "error"})
		return InferKind(files), issues
	}

	manifest, err := ParseManifestJSON(manifestRaw)
	if err != nil {
		issues = append(issues, ValidationIssue{Field: ManifestFile, Message: "manifest.json 无效: " + err.Error(), Level: "error"})
		return InferKind(files), issues
	}

	if manifest.Version != 1 {
		issues = append(issues, ValidationIssue{Field: "manifest.version", Message: "仅支持 version: 1", Level: "error"})
	}

	kind = manifest.Type
	if kind == "" {
		kind = InferKind(files)
	}
	switch kind {
	case KindSprite:
		issues = append(issues, validateSpriteManifest(manifest)...)
	case KindLive2D:
		issues = append(issues, validateLive2DManifest(manifest)...)
	case KindCustom:
		// custom: manifest + pet.js
	default:
		issues = append(issues, ValidationIssue{Field: "manifest.type", Message: "type 必须是 sprite | live2d | custom", Level: "error"})
	}

	if raw, ok := files[SoulpetYamlFile]; ok && strings.TrimSpace(raw) != "" {
		meta, err := ParseSoulpetYAML(raw)
		if err != nil {
			issues = append(issues, ValidationIssue{Field: SoulpetYamlFile, Message: err.Error(), Level: "error"})
		} else if meta.Kind != "" && manifest.Type != "" && meta.Kind != manifest.Type {
			issues = append(issues, ValidationIssue{
				Field:   SoulpetYamlFile,
				Message: fmt.Sprintf("soulpet.yaml kind=%q 与 manifest.type=%q 不一致", meta.Kind, manifest.Type),
				Level:   "error",
			})
		}
	}

	entry := DefaultEntry
	if _, ok := files[DefaultEntry]; !ok {
		issues = append(issues, ValidationIssue{Field: DefaultEntry, Message: "缺少 pet.js 入口", Level: "error"})
	} else if entryScriptValidator != nil {
		entry, violations := entryScriptValidator(entry, files)
		_ = entry
		for _, v := range violations {
			issues = append(issues, ValidationIssue{Field: DefaultEntry, Message: v, Level: "error"})
		}
	}

	for path := range files {
		if strings.HasPrefix(path, "assets/") {
			continue
		}
		switch path {
		case SoulpetYamlFile, ManifestFile, DefaultEntry, "style.css", "README.md":
			continue
		default:
			if !strings.Contains(path, "/") && !IsBinaryPath(path) {
				issues = append(issues, ValidationIssue{Field: path, Message: "非常规根目录文件（建议放入 assets/）", Level: "warn"})
			}
		}
	}

	return kind, issues
}

func validateSpriteManifest(m *PetManifest) []ValidationIssue {
	var issues []ValidationIssue
	if m.Assets.Sprite == nil {
		return []ValidationIssue{{Field: "assets.sprite", Message: "sprite 类型需要 assets.sprite", Level: "error"}}
	}
	s := m.Assets.Sprite
	if strings.TrimSpace(s.BaseURL) == "" {
		issues = append(issues, ValidationIssue{Field: "assets.sprite.baseUrl", Message: "baseUrl 必填", Level: "error"})
	}
	if len(s.Animations) == 0 {
		issues = append(issues, ValidationIssue{Field: "assets.sprite.animations", Message: "至少一个动画", Level: "error"})
	}
	return issues
}

func validateLive2DManifest(m *PetManifest) []ValidationIssue {
	var issues []ValidationIssue
	if m.Assets.Live2D == nil {
		return []ValidationIssue{{Field: "assets.live2d", Message: "live2d 类型需要 assets.live2d", Level: "error"}}
	}
	l := m.Assets.Live2D
	if strings.TrimSpace(l.BaseURL) == "" {
		issues = append(issues, ValidationIssue{Field: "assets.live2d.baseUrl", Message: "baseUrl 必填", Level: "error"})
	}
	if strings.TrimSpace(l.Model) == "" {
		issues = append(issues, ValidationIssue{Field: "assets.live2d.model", Message: "model 必填（如 model.model3.json）", Level: "error"})
	}
	if len(l.Motions) == 0 && len(l.Expressions) == 0 {
		issues = append(issues, ValidationIssue{
			Field:   "assets.live2d",
			Message: "建议至少配置 motions 或 expressions",
			Level:   "warn",
		})
	}
	return issues
}

func HasValidationErrors(issues []ValidationIssue) bool {
	for _, i := range issues {
		if i.Level == "error" {
			return true
		}
	}
	return false
}
