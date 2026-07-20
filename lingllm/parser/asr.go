//go:build asr

package parser

import (
	"context"
	"os"
	"strings"
	"time"

	vosk "github.com/alphacep/vosk-api/go"
)

// ASRParser transcribes audio files using the local Vosk library (CGO).
// Build with -tags asr and install libvosk; set VOSK_MODEL to the model directory.
type ASRParser struct {
	ModelPath string
}

func (p *ASRParser) Provider() string { return "asr" }

func (p *ASRParser) SupportedTypes() []string {
	return []string{FileTypeWAV, FileTypeMP3, FileTypeOGG, FileTypeFLAC, FileTypeM4A, FileTypeAAC}
}

func (p *ASRParser) Parse(ctx context.Context, req *ParseRequest, opts *ParseOptions) (*ParseResult, error) {
	if req == nil {
		return nil, ErrEmptyInput
	}

	modelPath := ""
	if p != nil {
		modelPath = strings.TrimSpace(p.ModelPath)
	}
	if modelPath == "" {
		modelPath = strings.TrimSpace(os.Getenv("VOSK_MODEL"))
	}
	if modelPath == "" {
		return nil, ErrUnsupportedFileType
	}

	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	data, fileName, err := readRequestBytes(req)
	if err != nil {
		return nil, err
	}

	pcm, err := decodeAudioToPCM(data, fileName)
	if err != nil {
		return nil, err
	}

	vosk.SetLogLevel(-1)
	model, err := vosk.NewModel(modelPath)
	if err != nil {
		return nil, err
	}
	defer model.Free()

	recognizer, err := vosk.NewRecognizer(model, float64(pcm.SampleRate))
	if err != nil {
		return nil, err
	}
	defer recognizer.Free()

	if err := recognizer.AcceptWaveform(pcm.Samples); err != nil {
		return nil, err
	}

	text := strings.TrimSpace(recognizer.FinalResult())
	text = extractVoskText(text)
	text = normalizeText(text, opts)
	text = truncateText(text, opts)

	ft := req.FileType
	if ft == "" {
		ft = DetectFileType(req)
	}

	return &ParseResult{
		FileType: ft,
		FileName: fileName,
		Text:     text,
		Sections: []Section{{Type: SectionTypeDocument, Index: 0, Title: fileName, Text: text}},
		Metadata: req.Metadata,
		ParsedAt: time.Now(),
	}, nil
}
