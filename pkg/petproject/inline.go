package petproject

import (
	"encoding/json"
	"strings"
)

// StandardPaths are loaded/saved with every pet project.
var StandardPaths = []string{
	SoulpetYamlFile,
	"manifest.json",
	"pet.js",
	"style.css",
	"README.md",
}

// InlineProjectV1 matches browser-side PetProjectV1 JSON in legacy DB rows.
type InlineProjectV1 struct {
	V     int               `json:"v"`
	Entry string            `json:"entry"`
	Files map[string]string `json:"files"`
}

func ParseInlineProject(content string) (*InlineProjectV1, bool) {
	trimmed := strings.TrimSpace(content)
	if meta, _ := ParseMeta(content); meta != nil {
		return nil, false
	}
	if trimmed == "" {
		return nil, false
	}
	var p InlineProjectV1
	if err := json.Unmarshal([]byte(trimmed), &p); err != nil || p.V != 1 || len(p.Files) == 0 {
		// legacy single-file JS
		if trimmed != "" && !json.Valid([]byte(trimmed)) {
			return &InlineProjectV1{
				V:     1,
				Entry: DefaultEntry,
				Files: map[string]string{DefaultEntry: content},
			}, true
		}
		return nil, false
	}
	if p.Entry == "" {
		p.Entry = DefaultEntry
	}
	return &p, true
}

func PathsFromFiles(files map[string]string) []string {
	seen := make(map[string]struct{})
	var paths []string
	for _, p := range StandardPaths {
		if _, ok := files[p]; ok {
			paths = append(paths, p)
			seen[p] = struct{}{}
		}
	}
	for name := range files {
		if _, ok := seen[name]; !ok {
			paths = append(paths, name)
		}
	}
	return paths
}
