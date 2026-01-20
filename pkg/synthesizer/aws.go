package synthesizer

import (
	"context"
	"fmt"

	"io/ioutil"

	"github.com/LingByte/SoulNexus/pkg/media"
	"github.com/LingByte/SoulNexus/pkg/utils"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/polly"
	"github.com/aws/aws-sdk-go-v2/service/polly/types"
	"github.com/sirupsen/logrus"
)

type AmazonTTSConfig struct {
	SampleRate    int                `json:"sampleRate" env:"sample_rate" default:"16000"`
	Region        string             `json:"region"`
	OutputFormat  types.OutputFormat `json:"outputFormat" env:"output_format" default:"pcm"`
	VoiceId       types.VoiceId      `json:"voiceId" env:"voice_id"`
	Channels      int                `json:"channels" env:"channels" default:"1"`
	BitDepth      int                `json:"bitDepth" env:"bit_depth" default:"16"`
	FrameDuration string             `json:"frameDuration" env:"frame_duration" default:"20ms"`
}

func (opt *AmazonTTSConfig) String() string {
	return fmt.Sprintf("AmazonTTSOption{SampleRate: %d, Region: %s, Channel: %d, BitDepth: %d}",
		opt.SampleRate, opt.Region, opt.Channels, opt.BitDepth)
}

func NewAmazonTTSOption(region string, outputFormat types.OutputFormat, voiceId types.VoiceId) AmazonTTSConfig {
	return AmazonTTSConfig{
		Region:       region,
		OutputFormat: outputFormat,
		VoiceId:      voiceId,
		Channels:     1,
		SampleRate:   16000,
		BitDepth:     16,
	}
}

type AmazonService struct {
	opt AmazonTTSConfig
}

func (as *AmazonService) Close() error {
	return nil
}

type amazonSpeechSynthesisListener struct {
	handler SynthesisHandler
}

func NewAmazonService(opt AmazonTTSConfig) *AmazonService {
	return &AmazonService{
		opt: opt,
	}
}
func (as *AmazonService) Provider() TTSProvider {
	return ProviderAWS
}

func (as *AmazonService) CacheKey(text string) string {
	digest := media.MediaCache().BuildKey(text)
	return fmt.Sprintf("amazon.tts-%s-%s-%s", as.opt.VoiceId, as.opt.Region, digest)
}

func (as *AmazonService) Format() media.StreamFormat {
	return media.StreamFormat{
		SampleRate:    as.opt.SampleRate,
		BitDepth:      as.opt.BitDepth,
		Channels:      as.opt.Channels,
		FrameDuration: utils.NormalizeFramePeriod(as.opt.FrameDuration),
	}
}

func (as *AmazonService) Synthesize(ctx context.Context, handler SynthesisHandler, text string) error {
	ttsReq := amazonSpeechSynthesisListener{
		handler: handler,
	}
	cfg, err := config.LoadDefaultConfig(ctx, config.WithRegion(as.opt.Region)) // Replace with your AWS region
	if err != nil {
		return err
	}
	client := polly.NewFromConfig(cfg)
	input := &polly.SynthesizeSpeechInput{
		OutputFormat: as.opt.OutputFormat,
		Text:         &text,
		VoiceId:      as.opt.VoiceId, // Replace with your preferred voice ID
	}
	resp, err := client.SynthesizeSpeech(ctx, input)
	if err != nil {
		return err
	}
	audioData, err := ioutil.ReadAll(resp.AudioStream)
	if err != nil {
		return err
	}
	ttsReq.OnMessage(audioData)
	return nil
}

func (a *amazonSpeechSynthesisListener) OnComplete() {
	logrus.WithFields(logrus.Fields{}).Info("amazon tts: complete")
}

func (a *amazonSpeechSynthesisListener) OnMessage(data []byte) {
	a.handler.OnMessage(data)
	a.OnComplete()
}
