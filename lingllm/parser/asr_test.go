package parser

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestASRParser_StubWithoutBuildTag(t *testing.T) {
	p := &ASRParser{}
	_, err := p.Parse(context.Background(), &ParseRequest{
		FileType: FileTypeWAV,
		FileName: "clip.wav",
		Content:  []byte("RIFF...."),
	}, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "asr")
}

func TestRouter_Parse_ASRRequiresBuildTag(t *testing.T) {
	r := DefaultRouter()
	_, err := r.Parse(context.Background(), &ParseRequest{
		FileName: "clip.wav",
		Content:  []byte("not wav"),
	}, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "asr")
}

func TestASRParser_Integration_WAV(t *testing.T) {
	modelPath := os.Getenv("VOSK_MODEL")
	if modelPath == "" {
		t.Skip("VOSK_MODEL not set")
	}
	if _, err := os.Stat(modelPath); err != nil {
		t.Skipf("VOSK_MODEL path unavailable: %v", err)
	}

	samples := make([]int16, 16000) // 1s silence
	wav := buildTestWAV(t, samples, 16000)

	p := &ASRParser{ModelPath: modelPath}
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	res, err := p.Parse(ctx, &ParseRequest{
		FileType: FileTypeWAV,
		FileName: "silence.wav",
		Content:  wav,
	}, &ParseOptions{PreserveLineBreaks: true})
	require.NoError(t, err)
	assert.Equal(t, FileTypeWAV, res.FileType)
	assert.NotNil(t, res)
}
