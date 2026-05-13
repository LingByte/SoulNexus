package recognizer

// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/LingByte/SoulNexus/pkg/voiceserver/media"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/sirupsen/logrus"
)

type FunAsrRealtime struct {
	Handler  media.MediaHandler
	opt      FunAsrRealtimeOption
	tr       TranscribeResult
	er       ProcessError
	client   *FunAsrRealtimeClient
	dialogID string
}

type FunAsrRealtimeClient struct {
	conn          *websocket.Conn
	ctx           context.Context
	cancel        context.CancelFunc
	sentence      string
	sendLastAudio bool
	sendReqTime   time.Time
	taskID        string
}

type FunAsrRealtimeOption struct {
	Url                      string `json:"url" yaml:"url" env:"FUNASR_URL" default:"wss://dashscope.aliyuncs.com/api-ws/v1/inference"`
	ApiKey                   string `json:"apiKey" yaml:"api_key" env:"DASHSCOPE_API_KEY"`
	Model                    string `json:"model" yaml:"model" default:"fun-asr-realtime"`
	SampleRate               int    `json:"sampleRate" yaml:"sample_rate" default:"16000"`
	Format                   string `json:"format" yaml:"format" default:"pcm"`
	LanguageHints            string `json:"languageHints" yaml:"language_hints" default:"zh"`
	EnableWords              bool   `json:"enableWords" yaml:"enable_words" default:"false"`
	EnableITN                bool   `json:"enableITN" yaml:"enable_itn" default:"false"`
	MaxSentenceSilence       uint   `json:"maxSentenceSilence" yaml:"max_sentence_silence" default:"1300"` // ms
	Heartbeat                bool   `json:"heartbeat" yaml:"heartbeat" default:"false"`
	DisfluencyRemovalEnabled bool   `json:"disfluencyRemovalEnabled" yaml:"disfluency_removal_enabled" default:"false"`
}

type FunHeader struct {
	Action       string                 `json:"action"`
	TaskID       string                 `json:"task_id"`
	Streaming    string                 `json:"streaming"`
	Event        string                 `json:"event"`
	ErrorCode    string                 `json:"error_code,omitempty"`
	ErrorMessage string                 `json:"error_message,omitempty"`
	Attributes   map[string]interface{} `json:"attributes"`
}

type FunPayload struct {
	TaskGroup  string `json:"task_group"`
	Task       string `json:"task"`
	Function   string `json:"function"`
	Model      string `json:"model"`
	Parameters Params `json:"parameters"`
	Input      Input  `json:"input"`
	Output     Output `json:"output,omitempty"`
	Usage      *struct {
		Duration int `json:"duration"`
	} `json:"usage,omitempty"`
}

type Event struct {
	Header  FunHeader  `json:"header"`
	Payload FunPayload `json:"payload"`
}

type Input struct {
}

type Output struct {
	Sentence struct {
		SentenceEnd bool   `json:"sentence_end"`
		BeginTime   int64  `json:"begin_time"`
		EndTime     *int64 `json:"end_time"`
		Text        string `json:"text"`
		Words       []struct {
			BeginTime   int64  `json:"begin_time"`
			EndTime     *int64 `json:"end_time"`
			Text        string `json:"text"`
			Punctuation string `json:"punctuation"`
		} `json:"words"`
	} `json:"sentence"`
}

type Params struct {
	Format                   string   `json:"format"`
	SampleRate               int      `json:"sample_rate"`
	VocabularyID             string   `json:"vocabulary_id"`
	DisfluencyRemovalEnabled bool     `json:"disfluency_removal_enabled"`
	LanguageHints            []string `json:"language_hints"`
	// fun asr special
	MaxSentenceSilence uint `json:"maxSentenceSilence"` // ms
	Heartbeat          bool `json:"heartbeat"`
}

func NewFunAsrRealtime(opt FunAsrRealtimeOption) FunAsrRealtime {
	if opt.Model == "" {
		opt.Model = "fun-asr-realtime"
	}
	if opt.SampleRate == 0 {
		opt.SampleRate = 16000
	}
	if opt.Format == "" {
		opt.Format = "pcm"
	}
	if opt.LanguageHints == "" {
		opt.LanguageHints = "zh"
	}
	if opt.MaxSentenceSilence == 0 {
		opt.MaxSentenceSilence = 1300
	}
	return FunAsrRealtime{
		opt: opt,
	}
}

func (fun *FunAsrRealtime) Init(tr TranscribeResult, er ProcessError) {
	fun.tr = tr
	fun.er = er
}

func (fun *FunAsrRealtime) Vendor() string {
	return "funasr_realtime"
}

func (fun *FunAsrRealtime) ConnAndReceive(dialogID string) error {
	var err error

	headers := http.Header{}
	headers.Add("Authorization", fmt.Sprintf("Bearer %s", fun.opt.ApiKey))
	headers.Add("X-DashScope-DataInspection", "enable")

	conn, _, err := websocket.DefaultDialer.Dial(fun.opt.Url, headers)
	if err != nil {
		logrus.WithFields(logrus.Fields{
			"url": fun.opt.Url,
		}).WithError(err).Error("funasr realtime: dial failed")
		return err
	}

	taskID := uuid.New().String()
	runTaskCmd := Event{
		Header: FunHeader{
			Action:    "run-task",
			TaskID:    taskID,
			Streaming: "duplex",
		},
		Payload: FunPayload{
			TaskGroup: "audio",
			Task:      "asr",
			Function:  "recognition",
			Model:     fun.opt.Model,
			Parameters: Params{
				Format:                   "pcm",
				SampleRate:               16000,
				DisfluencyRemovalEnabled: fun.opt.DisfluencyRemovalEnabled,
				LanguageHints:            []string{fun.opt.LanguageHints},
				MaxSentenceSilence:       fun.opt.MaxSentenceSilence,
				Heartbeat:                fun.opt.Heartbeat,
			},
			Input: Input{},
		},
	}
	runTaskCmdJSON, err := json.Marshal(runTaskCmd)
	if err != nil {
		return err
	}
	err = conn.WriteMessage(websocket.TextMessage, runTaskCmdJSON)
	if err != nil {
		return err
	}

	ctx, clientCancel := context.WithCancel(context.Background())
	client := &FunAsrRealtimeClient{
		conn:        conn,
		ctx:         ctx,
		cancel:      clientCancel,
		taskID:      taskID,
		sendReqTime: time.Now(),
	}
	fun.client = client
	fun.dialogID = dialogID

	startChan := make(chan Event, 8)
	go fun.recvFrames(client, startChan)

	select {
	case startEvent := <-startChan:
		switch startEvent.Header.Action {
		case "task-failed":
			logrus.WithFields(logrus.Fields{
				"sessionID": fun.Handler.GetSession().ID,
				"event":     startEvent,
			}).Error("funasr realtime: task-failed")
			return errors.New("funasr realtime conn failed")
		}
	}

	return nil
}

func (fun *FunAsrRealtime) Activity() bool {
	return fun.client != nil && !fun.client.sendLastAudio
}

func (fun *FunAsrRealtime) RestartClient() {
	if err := fun.StopConn(); err != nil {
		logrus.WithError(err).Error("funasr realtime: close client encounter an error")
	}
	if err := fun.ConnAndReceive(uuid.New().String()); err != nil {
		fun.er(err, true)
	}
}

func (fun *FunAsrRealtime) SendAudioBytes(data []byte) error {
	if fun.client == nil || fun.client.conn == nil {
		return fmt.Errorf("funasr realtime: client not initialized")
	}
	return fun.client.conn.WriteMessage(websocket.BinaryMessage, data)
}

func (fun *FunAsrRealtime) SendEnd() error {
	if fun.client == nil || fun.client.conn == nil {
		return fmt.Errorf("funasr realtime: client not initialized")
	}

	fun.client.sendLastAudio = true

	finishTaskCmd := Event{
		Header: FunHeader{
			Action:    "finish-task",
			TaskID:    fun.client.taskID,
			Streaming: "duplex",
		},
		Payload: FunPayload{
			Input: Input{},
		},
	}
	finishTaskCmdJSON, err := json.Marshal(finishTaskCmd)
	if err != nil {
		return err
	}
	return fun.client.conn.WriteMessage(websocket.TextMessage, finishTaskCmdJSON)
}

func (fun *FunAsrRealtime) StopConn() error {
	if fun.client != nil {
		fun.client.cancel()
		return fun.client.conn.Close()
	}
	return nil
}

func (fun *FunAsrRealtime) recvFrames(client *FunAsrRealtimeClient, startChan chan Event) {
	for {
		select {
		case <-client.ctx.Done():
			return
		default:
			messageType, message, err := client.conn.ReadMessage()
			if err != nil {
				if websocket.IsCloseError(err, websocket.CloseNormalClosure) {
					logrus.WithFields(logrus.Fields{
						"sessionID": fun.Handler.GetSession().ID,
						"client":    client.taskID,
					}).Debug("funasr realtime: recv close message, connection closed")
					fun.er(nil, false)
					return
				} else {
					logrus.WithFields(logrus.Fields{
						"sessionID":   fun.Handler.GetSession().ID,
						"message":     string(message),
						"messageType": messageType,
					}).WithError(err).Error("funasr realtime: recv error, connection closed")
				}
				if client.sentence != "" {
					fun.tr(client.sentence, true, time.Since(client.sendReqTime), "")
				}
				fun.er(err, false)
				return
			}

			var event Event
			err = json.Unmarshal(message, &event)

			switch event.Header.Event {
			case "task-started":
				startChan <- event
			case "result-generated":
				var sentence string
				if event.Payload.Output.Sentence.Text != "" {
					sentence = event.Payload.Output.Sentence.Text
				}

				if event.Payload.Output.Sentence.SentenceEnd {
					client.sentence += sentence
				} else {
					fun.tr(sentence, false, time.Since(client.sendReqTime), fun.dialogID)
				}
			case "task-finished":
				fun.tr(client.sentence, true, time.Since(client.sendReqTime), fun.dialogID)
				client.sentence = ""
				logrus.WithFields(logrus.Fields{
					"sessionID": fun.Handler.GetSession().ID,
				}).Info("funasr realtime: task finished")
				return
			case "task-failed":
				startChan <- event
				logrus.WithFields(logrus.Fields{
					"sessionID": fun.Handler.GetSession().ID,
					"event":     string(message),
				}).WithError(err).Error("funasr realtime: recv error, connection closed")
				return
			default:
			}
		}
	}
}
