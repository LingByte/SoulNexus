package synthesizer

import (
	"context"
	"log"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/joho/godotenv"
	"github.com/sirupsen/logrus"
)

func TestMain(m *testing.M) {
	logrus.SetReportCaller(true)
	if err := godotenv.Load("../../.env.development"); err != nil {
		log.Println("Error loading .env.development file")
	}

	code := m.Run()
	os.Exit(code)
}

func TestNewCoquiTTS(t *testing.T) {
	url := os.Getenv("COQUI_URL")
	if url == "" {
		t.Skip("missing parameters")
	}

	opt := NewCoquiTTSOption(url)
	server := NewCoquiService(opt)
	ctx := context.Background()
	h := &testSynthesisHandler{}

	err := server.Synthesize(ctx, h, "hello world")
	assert.Nil(t, err)
}
