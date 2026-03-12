package synthesizer

// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

import (
	"context"
	"fmt"

	texttospeech "cloud.google.com/go/texttospeech/apiv1"
	"cloud.google.com/go/texttospeech/apiv1/texttospeechpb"
	"github.com/LingByte/SoulNexus/pkg/media"
	"github.com/LingByte/SoulNexus/pkg/utils"
	"github.com/sirupsen/logrus"
)

type GoogleTTSOption struct {
	LanguageCode  string                         `json:"languageCode" yaml:"language_code"`
	SsmlGender    texttospeechpb.SsmlVoiceGender `json:"ssmlGender" yaml:"ssml_gender"`
	AudioEncoding texttospeechpb.AudioEncoding   `json:"audioEncoding" yaml:"audio_encoding" default:"LINEAR16"`
	SampleRate    int                            `json:"sampleRate" yaml:"sample_rate" default:"16000"`
	Channels      int                            `json:"channels" yaml:"channels" default:"1"`
	BitDepth      int                            `json:"bitDepth" yaml:"bit_depth" default:"16"`
	FrameDuration string                         `json:"frameDuration" yaml:"frame_duration" default:"20ms"`
}

func (opt *GoogleTTSOption) String() string {
	return fmt.Sprintf("GoogleTTSOption{LanguageCode: %s, SsmlGender: %d, AudioEncoding: %d, SampleRate: %d, Channels: %d, BitDepth: %d}",
		opt.LanguageCode, opt.SsmlGender, opt.AudioEncoding, opt.SampleRate, opt.Channels, opt.BitDepth)
}

func NewGoogleTTSOption(languageCode string) GoogleTTSOption {
	return GoogleTTSOption{
		LanguageCode:  languageCode,
		SsmlGender:    3,
		AudioEncoding: 5,
		Channels:      1,
		SampleRate:    16000,
		BitDepth:      16,
	}
}

type GoogleService struct {
	opt GoogleTTSOption
}

func (gs *GoogleService) Close() error {
	return nil
}

func NewGoogleService(opt GoogleTTSOption) *GoogleService {
	return &GoogleService{
		opt: opt,
	}
}

func (gs *GoogleService) Format() media.StreamFormat {
	return media.StreamFormat{
		Channels:      gs.opt.Channels,
		SampleRate:    gs.opt.SampleRate,
		BitDepth:      gs.opt.BitDepth,
		FrameDuration: utils.NormalizeFramePeriod(gs.opt.FrameDuration),
	}
}

func (gs *GoogleService) Provider() TTSProvider {
	return ProviderGoogle
}

type googleSpeechSynthesisListener struct {
	handler SynthesisHandler
}

func (gs *GoogleService) CacheKey(text string) string {
	digest := media.MediaCache().BuildKey(text)
	return fmt.Sprintf("google.tts-%s-%d-%s.pcm", gs.opt.LanguageCode, gs.opt.AudioEncoding, digest)
}

func (gs *GoogleService) Synthesize(ctx context.Context, handler SynthesisHandler, text string) error {
	ttsReq := googleSpeechSynthesisListener{
		handler: handler,
	}
	client, err := texttospeech.NewClient(ctx)
	if err != nil {
		return nil
	}
	defer client.Close()
	req := texttospeechpb.SynthesizeSpeechRequest{
		Input: &texttospeechpb.SynthesisInput{
			InputSource: &texttospeechpb.SynthesisInput_Text{Text: text},
		},
		Voice: &texttospeechpb.VoiceSelectionParams{
			LanguageCode: gs.opt.LanguageCode,
			SsmlGender:   gs.opt.SsmlGender,
		},
		AudioConfig: &texttospeechpb.AudioConfig{
			AudioEncoding: gs.opt.AudioEncoding,
		},
	}
	resp, err := client.SynthesizeSpeech(ctx, &req)
	if err != nil {
		return err
	}
	ttsReq.OnMessage(resp.AudioContent)
	return nil
}

func (g *googleSpeechSynthesisListener) OnComplete() {
	logrus.WithFields(logrus.Fields{}).Info("google tts: complete")
}

func (g *googleSpeechSynthesisListener) OnMessage(data []byte) {
	g.handler.OnMessage(data)
	g.OnComplete()
}
