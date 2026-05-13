package synthesizer

// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

import (
	"context"
	"encoding/base64"
	"fmt"
	"net/url"

	"github.com/LingByte/SoulNexus/pkg/voiceserver/media"
	"github.com/carlmjohnson/requests"
	"github.com/sirupsen/logrus"
)

type CoquiTTSOption struct {
	Url           string `json:"url" yaml:"url" env:"COQUI_URL"`
	Language      string `json:"language" yaml:"language" default:"en_US"`
	Speaker       string `json:"speaker" yaml:"speaker" default:"p226"`
	SampleRate    int    `json:"sampleRate" yaml:"sample_rate" default:"16000"`
	Channels      int    `json:"channels" yaml:"channels" default:"1"`
	BitDepth      int    `json:"bitDepth" yaml:"bit_depth" default:"16"`
	FrameDuration string `json:"frameDuration" yaml:"frame_duration" default:"20ms"`
}

type CoquiResponse struct {
	Audio string `json:"audio"`
}

func (opt *CoquiTTSOption) String() string {
	return fmt.Sprintf("CoquiTTSOption{Url: %s, Language: %s, Channels: %d, SampleRate: %d, Speaker: %s, BitDepth: %d}",
		opt.Url, opt.Language, opt.Channels, opt.SampleRate, opt.Speaker, opt.BitDepth)
}

func NewCoquiTTSOption(url string) CoquiTTSOption {
	return CoquiTTSOption{
		Url:        url,
		Language:   "en-US",
		Speaker:    "p226",
		SampleRate: 16000,
		Channels:   1,
		BitDepth:   16,
	}
}

type CoquiService struct {
	opt CoquiTTSOption
}

func (c *CoquiService) Close() error {
	return nil
}

func NewCoquiService(opt CoquiTTSOption) *CoquiService {
	return &CoquiService{
		opt: opt,
	}
}

func (c *CoquiService) Provider() TTSProvider {
	return ProviderCoqui
}

func (c *CoquiService) Format() media.StreamFormat {
	return media.StreamFormat{
		SampleRate:    c.opt.SampleRate,
		BitDepth:      c.opt.BitDepth,
		Channels:      c.opt.Channels,
		FrameDuration: media.NormalizeFramePeriod(c.opt.FrameDuration),
	}
}

func (c *CoquiService) CacheKey(text string) string {
	digest := media.MediaCache().BuildKey(text)
	return fmt.Sprintf("qcloud.tts-%s-%s-%s.pcm", c.opt.Language, c.opt.Speaker, digest)
}

type coquiSpeechSynthesisListener struct {
	handler SynthesisHandler
}

func (c *CoquiService) Synthesize(ctx context.Context, handler SynthesisHandler, text string) error {
	ttsReq := coquiSpeechSynthesisListener{
		handler: handler,
	}
	dataBytes, err := ttsReq.sendRequest(ctx, text, c.opt)
	if err != nil {
		return err
	}
	ttsReq.OnMessage(dataBytes)
	return nil
}

func (c *coquiSpeechSynthesisListener) sendRequest(ctx context.Context, text string, opt CoquiTTSOption) ([]byte, error) {
	var resp CoquiResponse
	if err := requests.URL(opt.Url).BodyForm(url.Values{
		"text":        []string{text},
		"language_id": []string{opt.Language},
		"speaker_id":  []string{opt.Speaker},
	}).ToJSON(&resp).Fetch(ctx); err != nil {
		logrus.WithFields(logrus.Fields{
			"handler": c.handler,
			"text":    text,
		}).WithError(err).Info("coqui tts: send request failed")
		return nil, err
	}
	dataBytes, err := base64.StdEncoding.DecodeString(resp.Audio)
	if err != nil {
		logrus.WithFields(logrus.Fields{
			"handler": c.handler,
		}).WithError(err).Info("coqui tts: decode string failed")
		return nil, err
	}
	return dataBytes, nil
}

func (c *coquiSpeechSynthesisListener) OnComplete() {
	logrus.WithFields(logrus.Fields{}).Info("coqui tts: complete")
}

func (c *coquiSpeechSynthesisListener) OnMessage(data []byte) {
	c.handler.OnMessage(data)
	c.OnComplete()
}
