//go:build !asr

package parser

import (
	"context"
	"fmt"
)

// ASRParser transcribes audio files using local Vosk (CGO). Without the asr build tag
// this stub is registered and returns a descriptive error.
type ASRParser struct {
	ModelPath string
}

func (p *ASRParser) Provider() string { return "asr" }

func (p *ASRParser) SupportedTypes() []string {
	return []string{FileTypeWAV, FileTypeMP3, FileTypeOGG, FileTypeFLAC, FileTypeM4A, FileTypeAAC}
}

func (p *ASRParser) Parse(ctx context.Context, req *ParseRequest, opts *ParseOptions) (*ParseResult, error) {
	_ = ctx
	_ = req
	_ = opts
	_ = p
	return nil, fmt.Errorf("audio ASR requires build tag 'asr', libvosk, and VOSK_MODEL: %w", ErrUnsupportedFileType)
}
