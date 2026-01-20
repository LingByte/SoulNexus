package synthesizer

import (
	"context"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewBaiduTTS(t *testing.T) {
	accessToken := os.Getenv("BAIDU_ACCESS_TOKEN")
	if accessToken == "" {
		t.Skip("missing BAIDU_ACCESS_TOKEN")
	}

	opt := NewBaiduTTSOption(accessToken)
	svc := NewBaiduService(opt)

	ctx := context.Background()
	h := &testSynthesisHandler{}
	err := svc.Synthesize(ctx, h, "hello lingecho")
	assert.Nil(t, err)
}
