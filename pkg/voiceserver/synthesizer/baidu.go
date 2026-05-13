package synthesizer

// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

import (
	"context"
	"fmt"
	"net/url"
	"strings"

	"github.com/LingByte/SoulNexus/pkg/voiceserver/media"
	"github.com/carlmjohnson/requests"
	"github.com/sirupsen/logrus"
)

type BaiduTTSConfig struct {
	Tok           string `json:"tok" env:"tok" env:"BAIDU_ACCESS_TOKEN"`
	Cuid          string `json:"cuid" env:"cuid"`
	Ctp           string `json:"ctp" env:"ctp" default:"1"`
	Lan           string `json:"lan" env:"lan" default:"zh"`
	Spd           string `json:"spd" env:"spd" default:"5"`
	Pit           string `json:"pit" env:"pit" default:"5"`
	Vol           string `json:"vol" env:"vol" default:"5"`
	Aue           string `json:"aue" env:"aue" default:"3"`
	Channels      int    `json:"channels" env:"channels" default:"1"`
	SampleRate    int    `json:"sampleRate" env:"sample_rate" default:"16000"`
	BitDepth      int    `json:"bitDepth" env:"bit_depth" default:"16"`
	FrameDuration string `json:"frameDuration" env:"frame_duration" default:"20ms"`
}

func (opt *BaiduTTSConfig) String() string {
	return fmt.Sprintf("BaiduTTSOption{Cuid: %s, Ctp: %s, Lan: %s, Spd: %s, Pit: %s, Vol: %s, Aue: %s, Channel: %d, SampleRate: %d, BitDepth: %d}",
		opt.Cuid, opt.Ctp, opt.Lan, opt.Spd, opt.Pit, opt.Vol, opt.Aue, opt.Channels, opt.SampleRate, opt.BitDepth)
}

func NewBaiduTTSOption(token string) BaiduTTSConfig {
	return BaiduTTSConfig{
		Tok:        token,
		Ctp:        "1",
		Lan:        "zh",
		Spd:        "5",
		Pit:        "5",
		Vol:        "5",
		Aue:        "3",
		Channels:   1,
		SampleRate: 16000,
		BitDepth:   16,
	}
}

type BaiduTTSService struct {
	opt BaiduTTSConfig
}

func (bs *BaiduTTSService) Close() error {
	return nil
}

func NewBaiduService(opt BaiduTTSConfig) *BaiduTTSService {
	return &BaiduTTSService{
		opt: opt,
	}
}

func (bs *BaiduTTSService) Provider() TTSProvider {
	return ProviderBaidu
}

func (bs *BaiduTTSService) Format() media.StreamFormat {
	return media.StreamFormat{
		FrameDuration: media.NormalizeFramePeriod(bs.opt.FrameDuration),
		Channels:      bs.opt.Channels,
		SampleRate:    bs.opt.SampleRate,
		BitDepth:      bs.opt.BitDepth,
	}
}
func (bs *BaiduTTSService) CacheKey(text string) string {
	digest := media.MediaCache().BuildKey(text)
	return fmt.Sprintf("baidu.tts-%s-%s-%s.pcm", bs.opt.Lan, bs.opt.Ctp, digest)
}

type baiduSpeechSynthesisListener struct {
	handler SynthesisHandler
}

func (bs *BaiduTTSService) Synthesize(ctx context.Context, handler SynthesisHandler, text string) error {
	ttsReq := baiduSpeechSynthesisListener{
		handler: handler,
	}
	dataBytes, err := bs.sendRequest(ctx, text)
	if err != nil {
		return err
	}
	ttsReq.OnMessage(dataBytes)
	return nil
}
func (bs *BaiduTTSService) sendRequest(ctx context.Context, text string) ([]byte, error) {
	var data string
	reUrl := "https://tsn.baidu.com/text2audio"
	values := url.Values{
		"tex":  []string{bs.DoubleURLEncode(text)},
		"tok":  []string{bs.opt.Tok},
		"cuid": []string{"cuid"},
		"ctp":  []string{bs.opt.Ctp},
		"lan":  []string{bs.opt.Lan},
		"aue":  []string{bs.opt.Aue},
		"spd":  []string{bs.opt.Spd},
		"pit":  []string{bs.opt.Pit},
		"vol":  []string{bs.opt.Vol},
	}
	err := requests.
		URL(reUrl).
		BodyForm(values).
		Header("Content-Type", "application/x-www-form-urlencoded").
		Header("Accept", "*/*").
		ToString(&data).
		Fetch(ctx)
	if err != nil {
		return nil, err
	}
	if strings.Contains(data, "err_no") {
		return nil, fmt.Errorf("baidu tts: %s", data)
	}
	return []byte(data), nil
}

func (bs *BaiduTTSService) DoubleURLEncode(text string) string {
	encoded1 := url.QueryEscape(text)
	encoded2 := url.QueryEscape(encoded1)
	return encoded2
}

func (b *baiduSpeechSynthesisListener) OnComplete() {
	logrus.WithFields(logrus.Fields{}).Info("baidu tts: complete")
}

func (b *baiduSpeechSynthesisListener) OnMessage(data []byte) {
	b.handler.OnMessage(data)
	b.OnComplete()
}
