package recognizer

// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

import (
	"context"
	"encoding/json"
	"time"

	"github.com/LingByte/SoulNexus/pkg/voiceserver/media"
	"github.com/gorilla/websocket"
	"github.com/sirupsen/logrus"
)

type WhisperASR struct {
	handler     media.MediaHandler
	conn        *websocket.Conn
	words       []byte
	Sentence    string
	EndTime     uint32
	sendReqTime *time.Time
}

type WhisperASROption struct {
	Url         string `json:"url" yaml:"url"`
	Model       string `json:"model" yaml:"model"`
	ReqChanSize int    `json:"reqChanSize" yaml:"req_chan_size" default:"128"`
}

type WhisperResult struct {
	IsFinal bool   `json:"is_final"`
	Text    string `json:"text"`
}

func NewWhisperASROption(url, model string) WhisperASROption {
	return WhisperASROption{
		Url:         url,
		Model:       model,
		ReqChanSize: 128,
	}
}

func WithWhisperASR(opt WhisperASROption) media.MediaHandlerFunc {
	executor := media.NewAsyncTaskRunner[[]byte](opt.ReqChanSize)

	wp := &WhisperASR{}

	executor.ConcurrentMode = true
	executor.RequestBuilder = func(h media.MediaHandler, packet media.MediaPacket) (*media.PacketRequest[[]byte], error) {
		audioPacket, ok := packet.(*media.AudioPacket)
		if !ok {
			h.EmitPacket(wp, packet)
			return nil, nil
		}
		if wp.handler == nil {
			wp.handler = h
		}
		decoded, _ := media.ResamplePCM(audioPacket.Payload, h.GetSession().Codec().SampleRate, 16000)
		req := media.PacketRequest[[]byte]{
			Req:       decoded,
			Interrupt: true,
		}
		return &req, nil
	}

	executor.InitCallback = func(h media.MediaHandler) error {
		var err error
		wp.conn, _, err = websocket.DefaultDialer.Dial(opt.Url, nil)
		if err != nil {
			logrus.WithFields(logrus.Fields{
				"sessionID": h.GetSession().ID,
				"url":       opt.Url,
			}).WithError(err).Error("whisper asr: dial failed")
			return err
		}
		go wp.recvFrames()
		return nil
	}

	executor.TerminateCallback = func(h media.MediaHandler) error {
		err := wp.conn.Close()
		wp.conn = nil
		return err
	}

	executor.StateCallback = func(h media.MediaHandler, event media.StateChange) error {
		switch event.State {
		case media.Hangup:
			return wp.conn.Close()
		}
		return nil
	}

	executor.TaskExecutor = func(ctx context.Context, h media.MediaHandler, req media.PacketRequest[[]byte]) error {
		if wp.sendReqTime == nil {
			n := time.Now()
			wp.sendReqTime = &n
			logrus.Info("whisper asr start send request")
		}
		return wp.conn.WriteMessage(websocket.BinaryMessage, req.Req)
	}
	return executor.HandleMediaData
}

func (wp *WhisperASR) recvFrames() {
	ttfbDone := false
	for {
		messageType, message, err := wp.conn.ReadMessage()
		if err != nil {
			if websocket.IsCloseError(err, websocket.CloseNormalClosure) {
				logrus.Info("whisper asr: recv close message, connection closed")
			} else {
				logrus.WithFields(logrus.Fields{
					"sessionID":   wp.handler.GetSession().ID,
					"err":         err,
					"message":     string(message),
					"messageType": messageType,
				}).WithError(err).Error("whisper asr: recv error, connection closed")
				wp.handler.CauseError(wp, err)
			}
			if wp.Sentence != "" {
				wp.handler.EmitPacket(wp, &media.TextPacket{Text: wp.Sentence, IsTranscribed: true})
				wp.handler.EmitState(wp, media.Completed, wp.Sentence)
				if wp.sendReqTime != nil {
					wp.handler.AddMetric("asr.whisper", time.Since(*wp.sendReqTime))
				}
			}
			break
		}

		if string(message) == "ping" {
			continue
		}

		var result WhisperResult

		err = json.Unmarshal(message, &result)
		if err != nil {
			logrus.WithFields(logrus.Fields{
				"sessionID": wp.handler.GetSession().ID,
			}).WithError(err).Error("whisper asr: unmarshal result failed")
			break
		}

		if !ttfbDone {
			ttfbDone = true
			if wp.sendReqTime != nil {
				wp.handler.AddMetric("asr.whisper.ttfb", time.Since(*wp.sendReqTime))
			}
		}

		wp.Sentence = result.Text

		if result.IsFinal {
			packet := &media.TextPacket{
				Text:          wp.Sentence,
				IsTranscribed: true,
			}
			wp.handler.EmitPacket(wp, packet)
			wp.handler.EmitState(wp, media.Completed, wp.Sentence)
		}

		logrus.WithFields(logrus.Fields{
			"sessionID": wp.handler.GetSession().ID,
			"word":      wp.Sentence,
		}).Debug("whisper asr: recv frame")

		wp.handler.EmitState(wp, media.Transcribing, wp.Sentence)
	}
}
