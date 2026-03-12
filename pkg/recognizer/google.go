package recognizer

import (
	"context"
	"io"
	"time"

	speech "cloud.google.com/go/speech/apiv1"
	"cloud.google.com/go/speech/apiv1/speechpb"
	"github.com/LingByte/SoulNexus/pkg/media"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

type GoogleASR struct {
	handler     media.MediaHandler
	stream      speechpb.Speech_StreamingRecognizeClient
	opt         GoogleASROption
	ttfbDone    bool
	sendReqTime *time.Time
	Sentence    string
	words       []byte

	tr TranscribeResult
	er ProcessError
}

type GoogleASROption struct {
	Encoding        speechpb.RecognitionConfig_AudioEncoding `json:"encoding" yaml:"encoding"`
	SampleRateHertz int32                                    `json:"sampleRateHertz" yaml:"sample_rate_hertz"`
	LanguageCode    string                                   `json:"languageCode" yaml:"language_code"`
	ReqChainSize    int                                      `json:"reqChainSize" yaml:"req_chain_size" default:"128"`
}

func NewGoogleASR(opt GoogleASROption) GoogleASR {
	return GoogleASR{
		opt: opt,
	}
}

func NewGoogleASROption(encoding speechpb.RecognitionConfig_AudioEncoding, sampleRateHertz int32, languageCode string) GoogleASROption {
	return GoogleASROption{
		Encoding:        encoding,
		SampleRateHertz: sampleRateHertz,
		LanguageCode:    languageCode,
		ReqChainSize:    128,
	}
}

func (google *GoogleASR) receiveFrames() {
	for {
		resp, err := google.stream.Recv()
		if err == io.EOF {
			if google.sendReqTime != nil {
				google.handler.AddMetric("asr_google", time.Since(*google.sendReqTime))
			}
			google.handler.EmitState(google, media.Completed)
			break
		}
		if err != nil {
			google.handler.CauseError(google, err)
			logrus.WithFields(logrus.Fields{
				"sessionID": google.handler.GetSession().ID,
				"error":     err,
			}).WithError(err).Error("Cannot stream results.")
			break
		}
		if err := resp.Error; err != nil {
			if err.Code == 3 || err.Code == 11 {
				logrus.Warning("WARNING: Speech recognition request exceeded limit of 60 seconds.")
			}
			logrus.WithFields(logrus.Fields{
				"sessionID": google.handler.GetSession().ID,
				"error":     err,
			}).Error("Cannot stream results.")
			break
		}

		if !google.ttfbDone {
			google.ttfbDone = true
			if google.sendReqTime != nil {
				google.handler.AddMetric("asr_google_ttfb", time.Since(*google.sendReqTime))
			}
		}

		for _, result := range resp.Results {
			google.words = append(google.words, result.Alternatives[0].Transcript...)
		}
		google.Sentence = string(google.words)
		google.words = nil

		google.handler.EmitState(google, media.Transcribing, google.Sentence)
		google.handler.EmitPacket(google, &media.AudioPacket{
			Payload:       []byte(google.Sentence),
			IsSynthesized: true,
		})
	}
}

func (google *GoogleASR) Init(tr TranscribeResult, er ProcessError) {
	google.tr = tr
	google.er = er
}
func (google *GoogleASR) Vendor() string {
	return "google"
}
func (google *GoogleASR) ConnAndReceive(dialogID string) error {
	ctx := context.Background()

	client, err := speech.NewClient(ctx)
	if err != nil {
		return err
	}
	stream, err := client.StreamingRecognize(ctx)
	if err != nil {
		return err
	}
	err = stream.Send(&speechpb.StreamingRecognizeRequest{
		StreamingRequest: &speechpb.StreamingRecognizeRequest_StreamingConfig{
			StreamingConfig: &speechpb.StreamingRecognitionConfig{
				Config: &speechpb.RecognitionConfig{
					Encoding:        google.opt.Encoding,
					SampleRateHertz: google.opt.SampleRateHertz,
					LanguageCode:    google.opt.LanguageCode,
				},
			},
		},
	})
	if err != nil {
		return err
	}
	go google.receiveFrames()
	return nil
}
func (google *GoogleASR) Activity() bool {
	return google.stream != nil
}
func (google *GoogleASR) RestartClient() {
	google.stream = nil
	if err := google.ConnAndReceive(uuid.New().String()); err != nil {
		google.er(err, true)
	}
}
func (google *GoogleASR) SendAudioBytes(data []byte) error {
	return google.stream.Send(&speechpb.StreamingRecognizeRequest{
		StreamingRequest: &speechpb.StreamingRecognizeRequest_AudioContent{
			AudioContent: data,
		},
	})
}
func (google *GoogleASR) SendEnd() error {
	err := google.stream.CloseSend()
	return err
}
func (google *GoogleASR) StopConn() error {
	err := google.stream.CloseSend()
	google.stream = nil
	return err
}
