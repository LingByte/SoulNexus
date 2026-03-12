package recognizer

// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"time"

	"github.com/LingByte/SoulNexus/pkg/media"
	"github.com/gorilla/websocket"
	"github.com/sirupsen/logrus"
)

var closeMsg = []byte(`{"event": "terminate"}`)

type GladiaASR struct {
	handler     media.MediaHandler
	conn        *websocket.Conn
	Sentence    string
	sendReqTime *time.Time
	ttfbDone    bool
	opt         GladiaASROption
	tr          TranscribeResult
	er          ProcessError
}

type GladiaASROption struct {
	Url         string `json:"url" yaml:"url" default:"wss://api.gladia.io/audio/text/audio-transcription"`
	ApiKey      string `json:"apiKey" yaml:"api_key"`
	Encoding    string `json:"encoding" yaml:"encoding" default:"WAV/PCM"`
	ReqChanSize int    `json:"reqChanSize" yaml:"req_chan_size" default:"128"`
}

type GladiaUtterance struct {
	Transcription string  `json:"transcription"`
	TimeBegin     float64 `json:"time_begin"`
	TimeEnd       float64 `json:"time_end"`
	Language      string  `json:"language"`
	Confidence    float64 `json:"confidence"`
	Stable        bool    `json:"stable"`
	ID            int     `json:"id"`
}

type Transcript struct {
	Type          string            `json:"type"`
	Transcription string            `json:"transcription"`
	TimeBegin     float64           `json:"time_begin"`
	TimeEnd       float64           `json:"time_end"`
	Confidence    float64           `json:"confidence"`
	Language      string            `json:"language"`
	Utterances    []GladiaUtterance `json:"utterances"`
	RequestID     string            `json:"request_id"`
	InferenceTime float64           `json:"inference_time"`
	Duration      float64           `json:"duration"`
	Event         string            `json:"event"`
	Code          string            `json:"code"`
	Message       string            `json:"message"`
}

func NewGladiaASROption(apiKey string, encoding string) GladiaASROption {
	return GladiaASROption{
		Url:         "wss://api.gladia.io/audio/text/audio-transcription",
		ApiKey:      apiKey,
		Encoding:    encoding,
		ReqChanSize: 256,
	}
}

func NewGladiaASR(opt GladiaASROption) GladiaASR {
	return GladiaASR{
		opt: opt,
	}
}

func (gla *GladiaASR) recvFrames() {
	for {
		messageType, message, err := gla.conn.ReadMessage()
		if err != nil {
			if websocket.IsCloseError(err, websocket.CloseNormalClosure) {
				logrus.Info("gladia asr: recv close message, connection closed")
			} else {
				logrus.WithFields(logrus.Fields{
					"sessionID":   gla.handler.GetSession().ID,
					"err":         err,
					"message":     string(message),
					"messageType": messageType,
				}).WithError(err).Error("gladia asr: recv error, connection closed")
				gla.handler.CauseError(gla, err)
			}
			if gla.Sentence != "" {
				gla.handler.EmitPacket(gla, &media.TextPacket{Text: gla.Sentence, IsTranscribed: true})
				gla.handler.EmitState(gla, media.Completed, gla.Sentence)
				if gla.sendReqTime != nil {
					gla.handler.AddMetric("asr.gladia", time.Since(*gla.sendReqTime))
				}
			}
			break
		}

		var transcript Transcript
		err = json.Unmarshal(message, &transcript)
		if err != nil {
			logrus.WithFields(logrus.Fields{
				"sessionID": gla.handler.GetSession().ID,
			}).WithError(err).Error("gladia asr: failed to unmarshal message")
			break
		}

		if transcript.Transcription == "connected" {
			logrus.Info(fmt.Sprintf("gladia asr: connection id: %s", transcript.RequestID))
		}

		if transcript.Transcription == "error" {
			logrus.WithFields(logrus.Fields{
				"sessionID": gla.handler.GetSession().ID,
			}).WithError(err).Error(fmt.Sprintf("gladia asr: error: [%s], %s", transcript.Code, transcript.Message))
			gla.handler.CauseError(gla, fmt.Errorf("%s: %s", transcript.Code, transcript.Message))
			break
		}

		if transcript.Event == "transcript" {
			if !gla.ttfbDone {
				gla.ttfbDone = true
				if gla.sendReqTime != nil {
					gla.handler.AddMetric("asr.gladia.ttfb", time.Since(*gla.sendReqTime))
				}
			}

			gla.Sentence = transcript.Transcription
			gla.handler.EmitPacket(gla, &media.TextPacket{
				Text:          gla.Sentence,
				IsTranscribed: true,
			})
			gla.handler.EmitState(gla, media.Transcribing, gla.Sentence)
			if transcript.Type == "final" {
				gla.handler.EmitState(gla, media.Completed, gla.Sentence)
				if gla.sendReqTime != nil {
					gla.handler.AddMetric("asr.gladia", time.Since(*gla.sendReqTime))
				}
			}
		}
	}
}

func (gla *GladiaASR) Init(tr TranscribeResult, er ProcessError) {
	gla.tr = tr
	gla.er = er
}
func (gla *GladiaASR) Vendor() string {
	return "gladia"
}
func (gla *GladiaASR) ConnAndReceive(dialogID string) error {
	var err error
	gla.conn, _, err = websocket.DefaultDialer.Dial(gla.opt.Url, nil)
	if err != nil {
		logrus.WithFields(logrus.Fields{
			"sessionID": gla.handler.GetSession().ID,
			"error":     err,
		}).Error("gladia asr: failed to dial websocket")
	}

	initMsg := map[string]string{
		"x_gladia_key": gla.opt.ApiKey,
		"encoding":     gla.opt.Encoding,
	}

	err = gla.conn.WriteJSON(initMsg)
	if err != nil {
		logrus.WithFields(logrus.Fields{
			"sessionID": gla.handler.GetSession().ID,
			"error":     err,
		}).Error("gladia asr: failed to init gladia connection")
	}
	go gla.recvFrames()
	return nil
}
func (gla *GladiaASR) Activity() bool {
	return true
}
func (gla *GladiaASR) RestartClient() {

}
func (gla *GladiaASR) SendAudioBytes(data []byte) error {
	audioMsg := map[string]string{
		"frames": base64.StdEncoding.EncodeToString(data),
	}
	return gla.conn.WriteJSON(audioMsg)
}
func (gla *GladiaASR) SendEnd() error {
	return gla.conn.WriteMessage(websocket.TextMessage, closeMsg)
}
func (gla *GladiaASR) StopConn() error {
	return gla.conn.Close()
}
