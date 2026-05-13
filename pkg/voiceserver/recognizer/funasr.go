package recognizer

// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"time"

	"github.com/LingByte/SoulNexus/pkg/voiceserver/media"
	"github.com/google/uuid"

	"github.com/gorilla/websocket"
	"github.com/sirupsen/logrus"
)

var endSpeaking = []byte(`{"is_speaking":false}`)

type FunASRCallback struct {
	handler media.MediaHandler
	opt     FunASROption
	tr      TranscribeResult
	er      ProcessError
	client  *FunASRClient
}

type FunASRClient struct {
	conn          *websocket.Conn
	ctx           context.Context
	cancel        context.CancelFunc
	sentence      string
	sendLastAudio bool
	sendReqTime   *time.Time
}

type FunASROption struct {
	Url                  string `json:"url" yaml:"url" env:"FUNASR_URL"`
	Mode                 string `json:"mode" yaml:"mode"`
	ChunkSize            []int  `json:"chunkSize" yaml:"chunk_size"`
	ChunkInterval        int    `json:"chunkInterval" yaml:"chunk_interval"`
	EncoderChunkLookBack int    `json:"encoderChunkLookBack" yaml:"encoder_chunk_look_back"`
	DecoderChunkLookBack int    `json:"decoderChunkLookBack" yaml:"decoder_chunk_look_back"`
	AudioFs              int    `json:"audioFs" yaml:"audio_fs"`
	WavName              string `json:"wavName" yaml:"wav_name"`
	WavFormat            string `json:"wavFormat" yaml:"wav_format"`
	IsSpeaking           bool   `json:"isSpeaking" yaml:"is_speaking"`
	Hotwords             string `json:"hotwords" yaml:"hotwords"`
	Itn                  bool   `json:"itn" yaml:"itn"`
	ReqChanSize          int    `json:"reqChanSize" yaml:"req_chan_size" default:"128"`
}

type FunASRRequestOption struct {
	Mode                 string `json:"mode"`
	ChunkSize            []int  `json:"chunk_size"`
	ChunkInterval        int    `json:"chunk_interval"`
	EncoderChunkLookBack int    `json:"encoder_chunk_look_back"`
	DecoderChunkLookBack int    `json:"decoder_chunk_look_back"`
	AudioFs              int    `json:"audio_fs"`
	WavName              string `json:"wav_name"`
	WavFormat            string `json:"wav_format"`
	IsSpeaking           bool   `json:"is_speaking"`
	Hotwords             string `json:"hotwords"`
	Itn                  bool   `json:"itn"`
}

type FunASRMessage struct {
	Mode    string `json:"mode"`
	Text    string `json:"text"`
	IsFinal bool   `json:"is_final"`
}

func NewFunASROption(url string) FunASROption {
	return FunASROption{
		Url:                  url,
		ReqChanSize:          128,
		Mode:                 "2pass",
		ChunkSize:            []int{5, 10, 5},
		ChunkInterval:        10,
		EncoderChunkLookBack: 4,
		DecoderChunkLookBack: 0,
		AudioFs:              16000,
		WavName:              "demo",
		WavFormat:            "pcm",
		IsSpeaking:           true,
		Hotwords:             "",
		Itn:                  false,
	}
}

func NewFunASRRequestOption(opt FunASROption) FunASRRequestOption {
	return FunASRRequestOption{
		Mode:                 opt.Mode,
		ChunkSize:            opt.ChunkSize,
		ChunkInterval:        opt.ChunkInterval,
		EncoderChunkLookBack: opt.EncoderChunkLookBack,
		DecoderChunkLookBack: opt.DecoderChunkLookBack,
		AudioFs:              opt.AudioFs,
		WavName:              opt.WavName,
		WavFormat:            opt.WavFormat,
		IsSpeaking:           opt.IsSpeaking,
		Hotwords:             opt.Hotwords,
		Itn:                  opt.Itn,
	}
}

func NewFunASR(opt FunASROption) FunASRCallback {
	return FunASRCallback{
		opt: opt,
	}
}

func (fun *FunASRCallback) Init(tr TranscribeResult, er ProcessError) {
	fun.tr = tr
	fun.er = er
}
func (fun *FunASRCallback) Vendor() string {
	return "funasr"
}

func (fun *FunASRCallback) ConnAndReceive(dialogID string) error {
	var err error

	dialer := websocket.Dialer{
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: true,
		},
		HandshakeTimeout: 10 * time.Second,
		ReadBufferSize:   1024,
		WriteBufferSize:  1024,
	}
	conn, _, err := dialer.Dial(fun.opt.Url, nil)
	if err != nil {
		logrus.WithFields(logrus.Fields{
			"url": fun.opt.Url,
		}).WithError(err).Error("fun asr: dial failed")
		return err
	}
	option := NewFunASRRequestOption(fun.opt)
	jsonOption, err := json.Marshal(option)
	if err != nil {
		return err
	}
	// send initial message
	err = conn.WriteMessage(websocket.TextMessage, jsonOption)
	if err != nil {
		return err
	}
	ctx, clientCancel := context.WithCancel(context.Background())
	client := &FunASRClient{conn: conn, ctx: ctx, cancel: clientCancel}
	fun.client = client
	go fun.recvFrames(client)
	return nil
}

func (fun *FunASRCallback) Activity() bool {
	return fun.client != nil && !fun.client.sendLastAudio
}

func (fun *FunASRCallback) RestartClient() {
	if err := fun.StopConn(); err != nil {
		logrus.WithError(err).Error("funasr asr: close client encounter an error")
	}
	if err := fun.ConnAndReceive(uuid.New().String()); err != nil {
		fun.er(err, true)
	}
}

func (fun *FunASRCallback) SendAudioBytes(data []byte) error {
	return fun.client.conn.WriteMessage(websocket.BinaryMessage, data)
}

func (fun *FunASRCallback) SendEnd() error {
	fun.client.sendLastAudio = true
	return fun.client.conn.WriteMessage(websocket.TextMessage, endSpeaking)
}

func (fun *FunASRCallback) StopConn() error {
	if fun.client != nil {
		fun.client.cancel()
		return fun.client.conn.Close()
	}
	return nil
}

func (fun *FunASRCallback) recvFrames(client *FunASRClient) {
	for {
		select {
		case <-client.ctx.Done():
			return
		default:
			messageType, message, err := client.conn.ReadMessage()
			if err != nil {
				if websocket.IsCloseError(err, websocket.CloseNormalClosure) {
					logrus.WithFields(logrus.Fields{
						"sessionID": fun.handler.GetSession().ID,
						"client":    client,
					}).Debug("funasr asr: recv close message, connection closed")
					fun.er(nil, false)
					return
				} else {
					logrus.WithFields(logrus.Fields{
						"sessionID":   fun.handler.GetSession().ID,
						"message":     string(message),
						"messageType": messageType,
					}).WithError(err).Error("funasr asr: recv error, connection closed")
				}
				if client.sentence != "" {
					fun.tr(client.sentence, true, time.Since(*client.sendReqTime), "")
				}
				fun.er(err, false)
				return
			}

			var msg FunASRMessage
			err = json.Unmarshal(message, &msg)
			if err != nil {
				logrus.WithFields(logrus.Fields{
					"sessionID": fun.handler.GetSession().ID,
					"url":       fun.opt.Url,
				}).WithError(err).Error("fun asr: serialize frame failed")
				fun.er(err, false)
				return
			}

			if msg.IsFinal {
				if msg.Text != "" {
					client.sentence = msg.Text
				}
				fun.tr(client.sentence, true, time.Since(*client.sendReqTime), "")
			} else {
				client.sentence += msg.Text
				fun.tr(client.sentence, false, time.Since(*client.sendReqTime), "")
			}
		}
	}
}
