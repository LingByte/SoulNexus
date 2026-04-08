package chunk

import (
	"errors"
	"strings"

	"github.com/LingByte/SoulNexus/pkg/llm"
)

const (
	ChunkerTypeRule = "rule"
	ChunkerTypeLLM  = "llm"
)

var ErrUnsupportedChunkerType = errors.New("unsupported chunker type")

type FactoryOptions struct {
	LLM   llm.LLMProvider
	Model string
}

func New(kind string, opts *FactoryOptions) (Chunker, error) {
	k := strings.ToLower(strings.TrimSpace(kind))
	if k == "" {
		k = ChunkerTypeRule
	}

	switch k {
	case ChunkerTypeRule:
		return &RuleChunker{}, nil
	case ChunkerTypeLLM:
		if opts == nil || opts.LLM == nil {
			return nil, errors.New("LLM is required")
		}
		return &LLMChunker{LLM: opts.LLM, Model: opts.Model}, nil
	default:
		return nil, ErrUnsupportedChunkerType
	}
}
