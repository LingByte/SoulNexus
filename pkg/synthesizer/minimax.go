package synthesizer

// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

import (
	"context"
	"crypto/tls"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/LingByte/SoulNexus/pkg/media"
	"github.com/LingByte/SoulNexus/pkg/utils"
	"github.com/gorilla/websocket"
	"github.com/sirupsen/logrus"
)

const (
	MinimaxWebSocketURL         = "wss://api.minimaxi.com/ws/v1/t2a_v2"
	MinimaxSpeech25TurboPreview = "speech-2.5-turbo-preview"
)

type MinimaxConnectionResponse struct {
	SessionID string `json:"session_id"`
	Event     string `json:"event"`
	TraceID   string `json:"trace_id"`
	BaseResp  struct {
		StatusCode int    `json:"status_code"`
		StatusMsg  string `json:"status_msg"`
	} `json:"base_resp"`
}

type MinimaxTaskContinueResponse struct {
	Data struct {
		Audio string `json:"audio"`
	} `json:"data"`
	SessionID string `json:"session_id"`
	Event     string `json:"event"`
	IsFinal   bool   `json:"is_final"`
	TraceID   string `json:"trace_id"`
	BaseResp  struct {
		StatusCode int    `json:"status_code"`
		StatusMsg  string `json:"status_msg"`
	} `json:"base_resp"`
}

type MinimaxTaskStartResponse struct {
	Event    string `json:"event"`
	BaseResp struct {
		StatusCode int    `json:"status_code"`
		StatusMsg  string `json:"status_msg"`
	} `json:"base_resp"`
}

type MinimaxTaskStartRequest struct {
	Event             string                    `json:"event"`
	Model             string                    `json:"model"`
	VoiceSetting      MinimaxVoiceSetting       `json:"voice_setting"`
	AudioSetting      MinimaxAudioSetting       `json:"audio_setting"`
	PronunciationDict *MinimaxPronunciationDict `json:"pronunciation_dict,omitempty"`
	LanguageBoost     string                    `json:"language_boost,omitempty"`
}

type MinimaxVoiceSetting struct {
	VoiceID       string                `json:"voice_id,omitempty"`
	Weight        int                   `json:"weight,omitempty"`
	TimbreWeights []MinimaxTimbreWeight `json:"timbre_weights,omitempty"`
	Speed         float64               `json:"speed"`
	Volume        float64               `json:"vol"`
	Pitch         float64               `json:"pitch"`
	Emotion       string                `json:"emotion"`
	ToneList      []string              `json:"tonelist,omitempty"`
}

type MinimaxTimbreWeight struct {
	VoiceID string `json:"voice_id"`
	Weight  int    `json:"weight"`
}

type MinimaxAudioSetting struct {
	SampleRate int    `json:"sample_rate"`
	Bitrate    *int   `json:"bitrate,omitempty"`
	Format     string `json:"format"`
	Channel    int    `json:"channel"`
}

type MinimaxPronunciationDict struct {
}

type MinimaxOption struct {
	Model         string  `json:"model" yaml:"model" default:"speech-2.5-turbo-preview"`
	APIKey        string  `json:"apiKey" yaml:"api_key" env:"MINIMAX_API_KEY"`
	VoiceID       string  `json:"voiceId" yaml:"voice_id" default:"male-qn-qingse"`
	SpeedRatio    float64 `json:"speedRatio" yaml:"speed_ratio" default:"1.0"`
	Volume        float64 `json:"volume" yaml:"volume" default:"1.0"`
	Pitch         float64 `json:"pitch" yaml:"pitch" default:"0.0"`
	Emotion       string  `json:"emotion" yaml:"emotion" default:"neutral"`
	LanguageBoost string  `json:"languageBoost" yaml:"language_boost" default:"auto"`
	TrainingTimes int     `json:"trainingTimes" yaml:"training_times" default:"1"`

	SampleRate    int    `json:"sampleRate" yaml:"sample_rate" default:"8000"`
	Bitrate       int    `json:"bitrate" yaml:"bitrate" default:"16"`
	Format        string `json:"format" yaml:"format" default:"pcm"`
	Channels      int    `json:"channels" yaml:"channels" default:"1"`
	FrameDuration string `json:"frameDuration" yaml:"frame_duration" default:"20ms"`
}

func (opt *MinimaxOption) String() string {
	return fmt.Sprintf("MinimaxOption{Model: %s, APIKey: %s, VoiceID: %s, Speed: %.2f, Volume: %.2f, Pitch: %.2f, Emotion: %s, SampleRate: %d, Bitrate: %d, Format: %s, Channels: %d}",
		opt.Model, opt.APIKey, opt.VoiceID, opt.SpeedRatio, opt.Volume, opt.Pitch, opt.Emotion, opt.SampleRate, opt.Bitrate, opt.Format, opt.Channels)
}

func NewMinimaxOption(apiKey string) MinimaxOption {
	return MinimaxOption{
		Model:         MinimaxSpeech25TurboPreview,
		APIKey:        apiKey,
		VoiceID:       "male-qn-qingse",
		SpeedRatio:    1.0,
		Volume:        1.0,
		Pitch:         0.0,
		Emotion:       "neutral",
		SampleRate:    8000,
		Bitrate:       16,
		Format:        "pcm",
		Channels:      1,
		FrameDuration: "20ms",
	}
}

type MinimaxService struct {
	opt           MinimaxOption
	ConnSessionID string
	TraceID       string
	startAt       time.Time
	conn          *websocket.Conn
}

func NewMinimaxService(opt MinimaxOption) *MinimaxService {
	return &MinimaxService{
		opt: opt,
	}
}

func (ms *MinimaxService) Close() error {
	if ms.conn != nil {
		return ms.conn.Close()
	}
	return nil
}

func (ms *MinimaxService) Provider() TTSProvider {
	return ProviderMinimax
}

func (ms *MinimaxService) Format() media.StreamFormat {
	return media.StreamFormat{
		SampleRate:    ms.opt.SampleRate,
		BitDepth:      16,
		Channels:      ms.opt.Channels,
		FrameDuration: utils.NormalizeFramePeriod(ms.opt.FrameDuration),
	}
}

func (ms *MinimaxService) CacheKey(text string) string {
	digest := media.MediaCache().BuildKey(text)
	speedRatio := int(ms.opt.SpeedRatio * 100)
	return fmt.Sprintf("minimax.tts-%s-%s-%d-%s-%d-%d.%s", ms.opt.VoiceID, ms.opt.Emotion, ms.opt.SampleRate, digest, ms.opt.TrainingTimes, speedRatio, ms.opt.Format)
}

func (ms *MinimaxService) GetConnSessionID() string {
	return ms.ConnSessionID
}

func (ms *MinimaxService) GetTraceID() string {
	return ms.TraceID
}

func (ms *MinimaxService) Synthesize(ctx context.Context, handler SynthesisHandler, text string) error {
	ms.startAt = time.Now()

	ws, err := ms.establishConnection()
	if err != nil {
		return fmt.Errorf("minimax: failed to establish connection: %w", err)
	}
	ms.conn = ws

	logrus.WithFields(logrus.Fields{
		"connSessionID": ms.GetConnSessionID(),
		"traceID":       ms.GetTraceID(),
	}).Info("minimax: using established connection")

	if err := ms.startTask(ws); err != nil {
		return fmt.Errorf("minimax: failed to start task: %w", err)
	}

	logrus.WithFields(logrus.Fields{
		"connSessionID": ms.GetConnSessionID(),
		"traceID":       ms.GetTraceID(),
		"latency":       time.Since(ms.startAt).Milliseconds(),
	}).Info("minimax: start task cost time")

	err = ms.continueTask(ws, text, handler)
	if err != nil {
		return fmt.Errorf("minimax: failed to continue task: %w", err)
	}

	if err := ms.closeConnection(ws); err != nil {
		logrus.WithError(err).Warn("minimax: failed to close connection gracefully")
	}

	return nil
}

func (ms *MinimaxService) establishConnection() (*websocket.Conn, error) {
	dialer := websocket.Dialer{
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: true,
		},
	}

	headers := http.Header{}
	headers.Set("Authorization", fmt.Sprintf("Bearer %s", ms.opt.APIKey))

	ws, _, err := dialer.Dial(MinimaxWebSocketURL, headers)
	if err != nil {
		return nil, fmt.Errorf("failed to dial websocket: %w", err)
	}

	_, message, err := ws.ReadMessage()
	if err != nil {
		return nil, fmt.Errorf("failed to read connection message: %w", err)
	}

	var connResponse MinimaxConnectionResponse
	if err := json.Unmarshal(message, &connResponse); err != nil {
		return nil, fmt.Errorf("failed to unmarshal connection response: %w", err)
	}

	if connResponse.Event != "connected_success" {
		return nil, fmt.Errorf("connection failed: %v", connResponse)
	}

	ms.ConnSessionID = connResponse.SessionID
	ms.TraceID = connResponse.TraceID

	logrus.WithFields(logrus.Fields{
		"connSessionID": connResponse.SessionID,
		"traceID":       connResponse.TraceID,
		"statusCode":    connResponse.BaseResp.StatusCode,
		"statusMsg":     connResponse.BaseResp.StatusMsg,
	}).Info("minimax: websocket connection established successfully")

	return ws, nil
}

func (ms *MinimaxService) startTask(ws *websocket.Conn) error {
	voiceSetting := MinimaxVoiceSetting{
		VoiceID: ms.opt.VoiceID,
		Speed:   ms.opt.SpeedRatio,
		Volume:  ms.opt.Volume,
		Pitch:   ms.opt.Pitch,
		Emotion: ms.opt.Emotion,
	}

	audioSetting := MinimaxAudioSetting{
		SampleRate: ms.opt.SampleRate,
		Format:     ms.opt.Format,
		Channel:    ms.opt.Channels,
	}

	if ms.opt.Format == "mp3" {
		bitrate := ms.opt.Bitrate
		audioSetting.Bitrate = &bitrate
	}

	startMsg := MinimaxTaskStartRequest{
		Event:         "task_start",
		Model:         ms.opt.Model,
		VoiceSetting:  voiceSetting,
		AudioSetting:  audioSetting,
		LanguageBoost: ms.opt.LanguageBoost,
	}

	startData, err := json.Marshal(startMsg)
	if err != nil {
		return fmt.Errorf("failed to marshal start message: %w", err)
	}

	if err := ws.WriteMessage(websocket.TextMessage, startData); err != nil {
		return fmt.Errorf("failed to send start message: %w", err)
	}

	_, message, err := ws.ReadMessage()
	if err != nil {
		return fmt.Errorf("failed to read start response: %w", err)
	}

	var response MinimaxTaskStartResponse
	if err := json.Unmarshal(message, &response); err != nil {
		return fmt.Errorf("failed to unmarshal start response: %w", err)
	}

	// Check base_resp status
	if response.BaseResp.StatusCode != 0 {
		return fmt.Errorf("minimax: task start failed with status code %d: %s", response.BaseResp.StatusCode, response.BaseResp.StatusMsg)
	}

	if response.Event != "task_started" {
		return fmt.Errorf("minimax: task start failed, unexpected event: %s", response.Event)
	}

	logrus.Info("minimax: task started successfully")
	return nil
}

func (ms *MinimaxService) continueTask(ws *websocket.Conn, text string, handler SynthesisHandler) error {
	continueMsg := map[string]interface{}{
		"event": "task_continue",
		"text":  text,
	}

	continueData, err := json.Marshal(continueMsg)
	if err != nil {
		return fmt.Errorf("failed to marshal continue message: %w", err)
	}

	if err = ws.WriteMessage(websocket.TextMessage, continueData); err != nil {
		return fmt.Errorf("failed to send continue message: %w", err)
	}

	ttfbDone := false
	for {
		_, message, err := ws.ReadMessage()
		if err != nil {
			if err == io.EOF {
				break
			}
			return fmt.Errorf("failed to read message: %w", err)
		}

		var response MinimaxTaskContinueResponse
		if err := json.Unmarshal(message, &response); err != nil {
			return fmt.Errorf("failed to unmarshal response: %w", err)
		}

		// Check base_resp status
		if response.BaseResp.StatusCode != 0 {
			return fmt.Errorf("minimax: task continue failed with status code %d: %s", response.BaseResp.StatusCode, response.BaseResp.StatusMsg)
		}

		if response.Data.Audio != "" {
			audioBytes, err := hex.DecodeString(response.Data.Audio)
			if err != nil {
				logrus.WithError(err).Warn("minimax: failed to decode hex audio chunk, skipping")
				continue
			}

			if len(audioBytes) > 0 {
				if !ttfbDone {
					ttfbDone = true
					logrus.WithFields(logrus.Fields{
						"connSessionID": ms.GetConnSessionID(),
						"traceID":       ms.GetTraceID(),
						"ttfb":          time.Since(ms.startAt).Milliseconds(),
					}).Info("minimax: ttfb")
				}

				handler.OnMessage(audioBytes)
			}
		}

		if response.IsFinal {
			break
		}
	}

	logrus.WithFields(logrus.Fields{
		"connSessionID": ms.GetConnSessionID(),
		"traceID":       ms.GetTraceID(),
	}).Info("minimax: streaming synthesis completed")

	return nil
}

func (ms *MinimaxService) closeConnection(ws *websocket.Conn) error {
	finishMsg := map[string]interface{}{
		"event": "task_finish",
	}

	finishData, err := json.Marshal(finishMsg)
	if err != nil {
		return fmt.Errorf("failed to marshal finish message: %w", err)
	}

	if err := ws.WriteMessage(websocket.TextMessage, finishData); err != nil {
		return fmt.Errorf("failed to send finish message: %w", err)
	}

	if err := ws.Close(); err != nil {
		return fmt.Errorf("failed to close websocket: %w", err)
	}

	logrus.Info("minimax: connection closed successfully")
	return nil
}
