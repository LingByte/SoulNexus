package recognizer

// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

import (
	"context"
	"time"

	"github.com/LingByte/SoulNexus/pkg/voiceserver/media"
	"github.com/aws/aws-sdk-go-v2/service/transcribestreaming"
	"github.com/aws/aws-sdk-go-v2/service/transcribestreaming/types"
	"github.com/sirupsen/logrus"
)

type AwsASR struct {
	handler     media.MediaHandler
	transStream *transcribestreaming.StartStreamTranscriptionEventStream
	sentence    string
	words       []byte
	ttfbDone    bool
	sendReqTime time.Time
}

type AwsASROption struct {
	AppID       string              `json:"appId" yaml:"app_id"`
	Region      string              `json:"region" yaml:"region"`
	Encoding    types.MediaEncoding `json:"encoding" yaml:"encoding"`
	SampleRate  int32               `json:"sampleRate" yaml:"sample_rate"`
	ReqChanSize int                 `json:"reqChanSize" yaml:"req_chan_size" default:"128"`
}

func NewAwsASROption(appId, region string, language string) AwsASROption {
	return AwsASROption{
		AppID:       appId,
		Region:      region,
		ReqChanSize: 128,
	}
}

func WithAwsASR(opt AwsASROption) media.MediaHandlerFunc {
	executor := media.NewAsyncTaskRunner[[]byte](opt.ReqChanSize)

	aws := &AwsASR{}

	executor.ConcurrentMode = false
	executor.RequestBuilder = func(h media.MediaHandler, packet media.MediaPacket) (*media.PacketRequest[[]byte], error) {
		audioPacket, ok := packet.(*media.AudioPacket)
		if !ok {
			h.EmitPacket(aws, packet)
			return nil, nil
		}
		if aws.handler == nil {
			aws.handler = h
		}
		decoded, _ := media.ResamplePCM(audioPacket.Payload, h.GetSession().Codec().SampleRate, 16000)
		req := media.PacketRequest[[]byte]{
			Req:       decoded,
			Interrupt: true,
		}
		return &req, nil
	}

	executor.InitCallback = func(h media.MediaHandler) error {
		client := transcribestreaming.New(transcribestreaming.Options{
			AppID:  opt.AppID,
			Region: opt.Region,
		})
		transInput := transcribestreaming.StartStreamTranscriptionInput{
			MediaEncoding:        opt.Encoding,
			MediaSampleRateHertz: &opt.SampleRate,
		}
		transOutput, err := client.StartStreamTranscription(context.Background(), &transInput, func(options *transcribestreaming.Options) {
			logrus.WithFields(logrus.Fields{
				"sessionID": aws.handler.GetSession().ID,
				"options":   options,
			}).Info("Invoke aws options.")
		})
		if err != nil {
			return err
		}
		aws.transStream = transOutput.GetStream()
		go aws.recvEvents()
		return nil
	}

	executor.TerminateCallback = func(h media.MediaHandler) error {
		return aws.transStream.Close()
	}

	executor.StateCallback = func(h media.MediaHandler, event media.StateChange) error {
		switch event.State {
		case media.Hangup:
			return aws.transStream.Close()
		}
		return nil
	}

	executor.TaskExecutor = func(ctx context.Context, h media.MediaHandler, req media.PacketRequest[[]byte]) error {
		return aws.transStream.Writer.Send(ctx, &types.AudioStreamMemberAudioEvent{Value: types.AudioEvent{AudioChunk: req.Req}})
	}

	return executor.HandleMediaData
}

func (aws *AwsASR) recvEvents() {
	eventChan := aws.transStream.Events()
	select {
	case event, ok := <-eventChan:
		if !ok {
			logrus.Error("aws stream closed.")
			return
		}
		transcriptEvent, ok := event.(*types.TranscriptResultStreamMemberTranscriptEvent)
		if !ok {
			logrus.Error("known aws stream.")
			return
		}

		if !aws.ttfbDone {
			aws.ttfbDone = true
			aws.handler.AddMetric("asr.aws.ttfb", time.Since(aws.sendReqTime))
		}

		for _, result := range transcriptEvent.Value.Transcript.Results {
			if result.IsPartial {
				logrus.Info("aws partial result:", result)
			} else {
				for _, alternative := range result.Alternatives {
					aws.words = append(aws.words, []byte(*alternative.Transcript)...)
				}
				aws.sentence = string(aws.words)
				aws.handler.EmitPacket(aws, &media.TextPacket{Text: aws.sentence, IsTranscribed: true})
				aws.handler.EmitState(aws, media.Transcribing, aws.sentence)
			}
		}
	case <-aws.handler.GetContext().Done():
		return
	}
	aws.handler.AddMetric("asr.aws", time.Since(aws.sendReqTime))
}
