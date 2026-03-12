package recognizer

// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/LingByte/SoulNexus/pkg/media"
	interfacesv1 "github.com/deepgram/deepgram-go-sdk/pkg/api/listen/v1/websocket/interfaces"
	"github.com/deepgram/deepgram-go-sdk/pkg/client/interfaces"
	client "github.com/deepgram/deepgram-go-sdk/pkg/client/listen"
	websocketv1 "github.com/deepgram/deepgram-go-sdk/pkg/client/listen/v1/websocket"
	"github.com/sirupsen/logrus"
)

type DeepgramASR struct {
	handler     media.MediaHandler
	opt         DeepgramASROption
	client      *websocketv1.Client
	Sentence    string
	EndTime     uint32
	sendReqTime *time.Time
	ttfbDone    bool
	closeChan   chan struct{}
}

type DeepgramASROption struct {
	ApiKey            string `json:"apiKey" yaml:"api_key" env:"DEEPGRAM_API_KEY"`
	Model             string `json:"model" yaml:"model" default:"nova-2"`
	Language          string `json:"language" yaml:"language" default:"en-US"`
	SampleRate        int    `json:"sampleRate" yaml:"sample_rate" default:"16000"`
	Channels          int    `json:"channels" yaml:"channels" default:"1"`
	Encoding          string `json:"encoding" yaml:"encoding" default:"linear16"`
	ReqChanSize       int    `json:"reqChanSize" yaml:"req_chan_size" default:"128"`
	KeepAliveDuration string `json:"keepAlineDuration" yaml:"keep_aline_duration" default:"3s"`
}

func NewDeepgramASROption(apiKey string, model string, language string) DeepgramASROption {
	return DeepgramASROption{
		ApiKey:            apiKey,
		Model:             model,
		Language:          language,
		SampleRate:        16000,
		Channels:          1,
		Encoding:          "linear16",
		ReqChanSize:       128,
		KeepAliveDuration: "3s",
	}
}

func WithDeepgramASR(opt DeepgramASROption) media.MediaHandlerFunc {
	executor := media.NewAsyncTaskRunner[[]byte](opt.ReqChanSize)

	dg := &DeepgramASR{opt: opt, closeChan: make(chan struct{})}

	executor.ConcurrentMode = false
	executor.RequestBuilder = func(h media.MediaHandler, packet media.MediaPacket) (*media.PacketRequest[[]byte], error) {
		audioPacket, ok := packet.(*media.AudioPacket)
		if !ok {
			h.EmitPacket(dg, packet)
			return nil, nil
		}
		if dg.handler == nil {
			dg.handler = h
		}
		decoded, _ := media.ResamplePCM(audioPacket.Payload, h.GetSession().Codec().SampleRate, 16000)
		req := media.PacketRequest[[]byte]{
			Req:       decoded,
			Interrupt: true,
		}
		return &req, nil
	}

	executor.InitCallback = func(h media.MediaHandler) error {
		client.InitWithDefault()
		ctx := h.GetContext()

		transcriptOptions := interfaces.LiveTranscriptionOptions{
			Model:          opt.Model,
			Language:       opt.Language,
			SampleRate:     opt.SampleRate,
			Channels:       opt.Channels,
			Encoding:       opt.Encoding,
			SmartFormat:    true,
			Punctuate:      true,
			VadEvents:      true,
			InterimResults: true,
			UtteranceEndMs: "1000",
		}

		clientOptions := interfaces.ClientOptions{}

		var err error
		dg.client, err = client.NewWebSocketUsingCallback(ctx, opt.ApiKey, &clientOptions, &transcriptOptions, dg)
		if err != nil {
			logrus.WithFields(logrus.Fields{
				"sessionID": h.GetSession().ID,
				"error":     err,
			}).Error("deepgram asr: error on creating deepgram client")
			return err
		}

		bConnected := dg.client.Connect()
		if !bConnected {
			logrus.WithFields(logrus.Fields{
				"sessionID": h.GetSession().ID,
				"error":     err,
			}).Error("deepgram asr: error on connecting to deepgram")
			return err
		}
		go dg.keepAlive()
		return nil
	}
	executor.TerminateCallback = func(h media.MediaHandler) error {
		dg.closeChan <- struct{}{}
		dg.client.Stop()
		return nil
	}

	executor.StateCallback = func(h media.MediaHandler, event media.StateChange) error {
		switch event.State {
		case media.Hangup:
			return dg.client.Finalize()
		}
		return nil
	}

	executor.TaskExecutor = func(ctx context.Context, h media.MediaHandler, req media.PacketRequest[[]byte]) error {
		if dg.sendReqTime == nil {
			n := time.Now()
			dg.sendReqTime = &n
			logrus.Info("deepgram asr start send request")
		}
		_, err := dg.client.Write(req.Req)
		return err
	}

	return executor.HandleMediaData
}

func (dg *DeepgramASR) keepAlive() {
	keepAliveDuration, _ := time.ParseDuration(dg.opt.KeepAliveDuration)
	if keepAliveDuration <= 0 {
		keepAliveDuration = 3 * time.Second
	}

	ticker := time.NewTicker(keepAliveDuration)
	defer ticker.Stop()

	for {
		select {
		case <-dg.closeChan:
			return
		case <-ticker.C:
			if err := dg.client.KeepAlive(); err != nil {
				logrus.WithFields(logrus.Fields{
					"sessionID": dg.handler.GetSession().ID,
				}).WithError(err).Error("deepgram asr: keep alive error")
				dg.handler.CauseError(dg, err)
				return
			}
		}
	}
}

func (dg *DeepgramASR) Open(or *interfacesv1.OpenResponse) error {
	logrus.WithFields(logrus.Fields{
		"sessionID":    dg.handler.GetSession().ID,
		"openResponse": or,
	}).Info("deepgram asr: opening deepgram asr")
	return nil
}

func (dg *DeepgramASR) Message(mr *interfacesv1.MessageResponse) error {
	sentence := strings.TrimSpace(mr.Channel.Alternatives[0].Transcript)
	if len(mr.Channel.Alternatives) == 0 || len(sentence) == 0 {
		return nil
	}

	logrus.WithFields(logrus.Fields{
		"sessionID": dg.handler.GetSession().ID,
		"Sentence":  sentence,
	}).Info("deepgram asr: received message")
	if !dg.ttfbDone {
		dg.ttfbDone = true
		if dg.sendReqTime != nil {
			dg.handler.AddMetric("asr.deepgram.ttfb", time.Since(*dg.sendReqTime))
		}
	}

	if mr.IsFinal {
		dg.Sentence = sentence

		dg.handler.EmitPacket(dg, &media.TextPacket{
			IsTranscribed: true,
			Text:          dg.Sentence,
		})

		dg.handler.EmitState(dg, media.Completed, dg.Sentence)
	}
	return nil
}

func (dg *DeepgramASR) Metadata(md *interfacesv1.MetadataResponse) error {
	logrus.WithFields(logrus.Fields{
		"sessionID": dg.handler.GetSession().ID,
		"metadata":  md,
	}).Info("deepgram asr: metadata received")
	return nil
}

func (dg *DeepgramASR) SpeechStarted(ssr *interfacesv1.SpeechStartedResponse) error {
	logrus.WithFields(logrus.Fields{
		"sessionID":             dg.handler.GetSession().ID,
		"speechStartedResponse": ssr,
	}).Info("deepgram asr: speech started")
	return nil
}

func (dg *DeepgramASR) UtteranceEnd(ur *interfacesv1.UtteranceEndResponse) error {
	logrus.WithFields(logrus.Fields{
		"sessionID":            dg.handler.GetSession().ID,
		"utteranceEndResponse": ur,
	}).Info("deepgram asr: utterance ended")
	dg.handler.EmitState(dg, media.Completed, dg.Sentence)
	if dg.sendReqTime != nil {
		dg.handler.AddMetric("asr.deepgram", time.Since(*dg.sendReqTime))
	}
	return nil
}

func (dg *DeepgramASR) Close(cr *interfacesv1.CloseResponse) error {
	logrus.WithFields(logrus.Fields{
		"sessionID":     dg.handler.GetSession().ID,
		"closeResponse": cr,
	}).Info("deepgram asr: closing deepgram asr")
	return nil
}

func (dg *DeepgramASR) Error(er *interfacesv1.ErrorResponse) error {
	errMsgFmt := fmt.Sprintf(
		"deepgram asr: error.type: %s, error.errcode: %s, error.description: %s",
		er.ErrCode,
		er.ErrMsg,
		er.Description,
	)
	logrus.WithFields(logrus.Fields{
		"sessionID": dg.handler.GetSession().ID,
	}).Error(errMsgFmt)
	dg.handler.CauseError(dg, errors.New(errMsgFmt))
	return nil
}

func (dg *DeepgramASR) UnhandledEvent(byData []byte) error {
	logrus.WithFields(logrus.Fields{
		"sessionID": dg.handler.GetSession().ID,
		"data":      string(byData),
	}).Warning("deepgram asr: unhandled event")
	return nil
}
