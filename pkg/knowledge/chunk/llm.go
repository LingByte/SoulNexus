package chunk

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/LingByte/SoulNexus/pkg/llm"
)

type LLMChunker struct {
	LLM   llm.LLMHandler
	Model string
}

func (c *LLMChunker) Provider() string { return "llm" }

func (c *LLMChunker) Chunk(ctx context.Context, text string, opts *ChunkOptions) ([]Chunk, error) {
	_ = ctx
	text = strings.TrimSpace(text)
	if text == "" {
		return nil, ErrEmptyText
	}
	if c == nil || c.LLM == nil {
		return nil, errors.New("LLM is required")
	}

	model := strings.TrimSpace(c.Model)
	if model == "" {
		model = ""
	}

	maxChars := 1200
	overlap := 120
	minChars := 50
	docTitle := ""
	if opts != nil {
		if opts.MaxChars > 0 {
			maxChars = opts.MaxChars
		}
		if opts.OverlapChars >= 0 {
			overlap = opts.OverlapChars
		}
		if opts.MinChars > 0 {
			minChars = opts.MinChars
		}
		docTitle = strings.TrimSpace(opts.DocumentTitle)
	}

	prompt := buildChunkPrompt(text, docTitle, maxChars, overlap, minChars)
	resp, err := c.LLM.Query(prompt, model)
	if err != nil {
		return nil, err
	}

	chunks, err := parseLLMChunks(resp)
	if err != nil {
		return nil, err
	}
	for i := range chunks {
		chunks[i].Index = i
		chunks[i].Text = strings.TrimSpace(chunks[i].Text)
	}
	return chunks, nil
}

func buildChunkPrompt(text string, title string, maxChars int, overlap int, minChars int) string {
	titleLine := ""
	if title != "" {
		titleLine = fmt.Sprintf("DocumentTitle: %s\n", title)
	}
	return fmt.Sprintf(`You are a text chunking engine.
%sSplit the input into semantically coherent chunks for retrieval.

STRICT OUTPUT RULES:
- Output MUST be a single JSON array and nothing else.
- Do NOT output markdown.
- Do NOT wrap the JSON in code fences.
- Do NOT add explanations.

JSON SCHEMA:
- Output: [{"title": string, "text": string, "metadata"?: object}]

CHUNKING RULES:
- Each chunk "text" length should be <= %d characters.
- Overlap between consecutive chunks should be about %d characters when helpful.
- Avoid chunks shorter than %d characters unless necessary.

INPUT:
%s
`, titleLine, maxChars, overlap, minChars, text)
}

func parseLLMChunks(s string) ([]Chunk, error) {
	s = strings.TrimSpace(s)
	s = extractJSONArrayString(s)

	var raw []struct {
		Title string         `json:"title"`
		Text  string         `json:"text"`
		Meta  map[string]any `json:"metadata"`
	}
	if err := json.Unmarshal([]byte(s), &raw); err != nil {
		return nil, err
	}
	out := make([]Chunk, 0, len(raw))
	for _, r := range raw {
		out = append(out, Chunk{Title: r.Title, Text: r.Text, Metadata: r.Meta})
	}
	if len(out) == 0 {
		return nil, errors.New("no chunks returned")
	}
	return out, nil
}

func extractJSONArrayString(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return s
	}
	// Handle fenced blocks.
	if strings.Contains(s, "```") {
		// Prefer ```json ... ```
		lower := strings.ToLower(s)
		idx := strings.Index(lower, "```json")
		if idx < 0 {
			idx = strings.Index(lower, "```")
		}
		if idx >= 0 {
			// Find end fence
			endFence := strings.Index(lower[idx+3:], "```")
			if endFence >= 0 {
				inner := s[idx+3 : idx+3+endFence]
				inner = strings.TrimSpace(inner)
				// Strip a leading 'json' identifier if present
				innerLower := strings.ToLower(inner)
				if strings.HasPrefix(innerLower, "json") {
					inner = strings.TrimSpace(inner[4:])
				}
				s = inner
			}
		}
	}
	// Locate first JSON array.
	start := strings.Index(s, "[")
	end := strings.LastIndex(s, "]")
	if start >= 0 && end > start {
		return strings.TrimSpace(s[start : end+1])
	}
	return s
}
