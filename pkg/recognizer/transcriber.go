package recognizer

import (
	"context"
	"sync"
	"sync/atomic"
	"time"

	"github.com/LingByte/SoulNexus/pkg/media"
	"github.com/LingByte/SoulNexus/pkg/utils"
	"github.com/gorilla/websocket"
	gonanoid "github.com/matoous/go-nanoid"
	"github.com/sirupsen/logrus"
)

type TranscribeService interface {
	Init(tr TranscribeResult, er ProcessError)
	Vendor() string
	ConnAndReceive(dialogId string) error
	Activity() bool
	RestartClient()
	SendAudioBytes(data []byte) error
	SendEnd() error
	StopConn() error
}

type TranscribeResult func(text string, isLast bool, duration time.Duration, uuid string)

type ProcessError func(err error, isFatal bool)

type HotWord struct {
	Word   string `json:"word"`
	Weight int    `json:"weight"`
}

type TranscribeOneShot struct {
	handler media.MediaHandler
	buffer  chan []byte
	client  *websocket.Conn
	mode    int
	Padding int
}

func channelSize() int {
	return 700 * utils.ComputeSampleByteCount(16000, 16, 1)
}

func AudioIntercept() media.MediaHandlerFunc {
	return func(h media.MediaHandler, data media.MediaData) {
		if data.Type == media.MediaDataTypePacket {
			if _, ok := data.Packet.(*media.AudioPacket); ok {
				return
			}
			h.EmitPacket(h, data.Packet)
		}
	}
}

type TranscribeOption struct {
	Direction    string             `json:"direction,omitempty"`
	AsrOptions   map[string]any     `json:"asrOptions,omitempty"`
	FuzzyOptions AsrCorrectorOption `json:"fuzzyOptions,omitempty"`
}

func WithTranscribeFilter(asr TranscribeService, h media.MediaHandler, opt TranscribeOption) media.PacketFilter {
	senderName := "asr." + asr.Vendor()

	err := asr.ConnAndReceive("")
	if err != nil {
		h.CauseError(senderName, err)
	}

	var localDialogID string
	var startTranscribingAt time.Time
	asr.Init(func(text string, isLast bool, duration time.Duration, dialogID string) {
		if text == "" {
			return
		}
		if startTranscribingAt.IsZero() {
			startTranscribingAt = time.Now()
		}
		if isLast {
			h.EmitState(senderName, media.Completed, &media.CompletedData{
				SenderName: ASRFilterSenderName(senderName, opt.Direction, h),
				Result:     text,
				Duration:   duration,
				DialogID:   localDialogID,
			})

			// Metrics functionality removed
			localDialogID = ""
			startTranscribingAt = time.Time{}
		} else {
			if localDialogID == "" {
				localDialogID, _ = gonanoid.Nanoid()
				startTranscribingAt = time.Now()
			}
			h.EmitState(senderName, media.Transcribing, &media.TranscribingData{
				SenderName: senderName,
				Result:     text,
				Duration:   duration,
				Direction:  opt.Direction,
				DialogID:   localDialogID,
			})
		}
	}, func(err error, isFatal bool) {
		if isFatal {
			h.CauseError(senderName, err)
		} else {
			asr.RestartClient()
		}
	})

	packetChan := make(chan media.MediaPacket, 1024)
	go func() {
		for {
			select {
			case <-h.GetSession().GetContext().Done():
				return
			case packet := <-packetChan:
				switch packet := packet.(type) {
				case *media.AudioPacket:
					originalPayload := packet.Payload
					copiedPayload := make([]byte, len(originalPayload))
					copy(copiedPayload, originalPayload)
					// ignore system voice data
					if opt.Direction == media.DirectionOutput {
						if val, ok := h.GetSession().Get(media.UpstreamRunning); !ok || !val.(bool) {
							_ = asr.SendAudioBytes(make([]byte, 0))
							continue
						}
					}
					err := asr.SendAudioBytes(copiedPayload)
					if err != nil {
						asr.RestartClient()
					}
				case *media.ClosePacket:
					_ = asr.StopConn()
				}
			}
		}
	}()

	return func(packet media.MediaPacket) (bool, error) {
		packetChan <- packet
		return false, nil
	}
}

func WithTranscribeFilterState(asr TranscribeService, h media.MediaHandler, opt TranscribeOption) media.PacketFilter {
	senderName := "asr." + asr.Vendor()

	bytePerMillSecond := utils.ComputeSampleByteCount(16000, 16, 1)
	frameLen := 20 * bytePerMillSecond
	bufferLen := 700 * bytePerMillSecond

	type state struct {
		dataChan chan []byte
		ctx      context.Context
		cancel   context.CancelFunc
		dialogID string
	}

	var (
		currentState atomic.Value
		lock         sync.Mutex
		audioBuffer  = make([]byte, 0, bufferLen)
	)

	var currentDialogID string
	var startTranscribingAt time.Time
	corrector := NewAsrCorrector(opt.FuzzyOptions)
	asr.Init(func(text string, isLast bool, duration time.Duration, textDialogID string) {
		if text == "" {
			return
		}
		if startTranscribingAt.IsZero() {
			startTranscribingAt = time.Now()
		}
		if isLast {
			correctText := corrector.Correct(text)
			logrus.WithFields(logrus.Fields{
				"asrResult":       text,
				"correctedResult": correctText,
				"sessionID":       h.GetSession(),
				"dialogID":        textDialogID,
			}).Infof("asr result correction ")

			h.EmitState(senderName, media.Completed, &media.CompletedData{
				SenderName: ASRFilterSenderName(senderName, opt.Direction, h),
				Result:     correctText,
				Duration:   duration,
				DialogID:   textDialogID,
			})
		} else {
			if textDialogID != currentDialogID {
				startTranscribingAt = time.Now()
			}
			h.EmitState(senderName, media.Transcribing, &media.TranscribingData{
				SenderName: senderName,
				Result:     text,
				Duration:   duration,
				Direction:  opt.Direction,
				DialogID:   textDialogID,
			})
		}
	}, func(err error, isFatal bool) {
		if isFatal {
			h.CauseError(senderName, err)
		} else {
			asr.RestartClient()
		}
	})

	h.GetSession().On(media.StartSpeaking, func(event media.StateChange) {
		if h.GetSession().GetString(media.WorkingState) != media.AgentRunning {
			return
		}
		ctx, cancel := context.WithCancel(h.GetContext())
		go func() {

			defer cancel()

			newState := &state{
				dataChan: make(chan []byte, 1024),
				ctx:      ctx,
				cancel:   cancel,
			}

			lock.Lock()
			if len(audioBuffer) > 0 {
				for i := 0; i < len(audioBuffer); i += frameLen {
					end := i + frameLen
					if end > len(audioBuffer) {
						end = len(audioBuffer)
					}

					select {
					case newState.dataChan <- audioBuffer[i:end]:
					default:
						break
					}
				}
				audioBuffer = audioBuffer[:0]
			}
			lock.Unlock()

			currentState.Store(newState)

			if len(event.Params) > 0 {
				newState.dialogID = event.Params[0].(string)
			} else {
				newState.dialogID, _ = gonanoid.Nanoid()
			}

			if err := asr.ConnAndReceive(newState.dialogID); err != nil {
				h.CauseError(senderName, err)
				return
			}

			for {
				select {
				case bytes, ok := <-newState.dataChan:
					if !ok {
						_ = asr.SendEnd()
						return
					}
					if err := asr.SendAudioBytes(bytes); err != nil {
						logrus.WithError(err).Warn("asr: send audio bytes failed")
					}
				}
			}
		}()
	})

	h.GetSession().On(media.StartSilence, func(event media.StateChange) {
		if s, ok := currentState.Load().(*state); ok && s != nil {
			s.cancel()
			close(s.dataChan)
			currentState.Store((*state)(nil))
		}
	})

	return func(packet media.MediaPacket) (bool, error) {
		if _, ok := packet.(*media.ClosePacket); ok {
			if s, ok := currentState.Load().(*state); ok && s != nil {
				s.cancel()
			}
			return false, asr.StopConn()
		}

		originalPayload := packet.Body()
		if s, ok := currentState.Load().(*state); ok && s != nil {
			select {
			case s.dataChan <- originalPayload:
			case <-s.ctx.Done():
				break
			default:
				break
			}
		}

		lock.Lock()
		if len(audioBuffer)+len(originalPayload) > bufferLen {
			audioBuffer = append(audioBuffer[:0], audioBuffer[frameLen:]...)
		}
		audioBuffer = append(audioBuffer, originalPayload...)
		lock.Unlock()

		return false, nil
	}
}

func ASRFilterSenderName(senderName, direction string, h media.MediaHandler) string {
	var name string
	if direction == media.DirectionInput {
		name = senderName + ".customer"
	} else if direction == media.DirectionOutput {
		val, ok := h.GetSession().Get(media.UpstreamRunning)
		if ok && val.(bool) {
			name = senderName + ".agent"
		}
	}
	return name
}

// 识别结果处理函数
func handleAsrResult(senderName string, opt TranscribeOption, h media.MediaHandler, corrector *AsrCorrector, currentDialogID *string, startTranscribingAt *time.Time) func(text string, isLast bool, duration time.Duration, textDialogID string) {
	return func(text string, isLast bool, duration time.Duration, textDialogID string) {
		if text == "" {
			return
		}
		if isLast {
			corrected := corrector.Correct(text)
			logrus.WithFields(logrus.Fields{
				"asrResult": text,
				"corrected": corrected,
				"sessionID": h.GetSession().ID,
				"dialogID":  textDialogID,
			}).Infof("asr result correction ")

			h.EmitState(senderName, media.Completed, &media.CompletedData{
				SenderName: ASRFilterSenderName(senderName, opt.Direction, h),
				Result:     corrected,
				Duration:   duration,
				DialogID:   textDialogID,
			})
		} else {
			if textDialogID != *currentDialogID {
				*startTranscribingAt = time.Now()
			}
			h.EmitState(senderName, media.Transcribing, &media.TranscribingData{
				SenderName: senderName,
				Result:     text,
				Duration:   duration,
				Direction:  opt.Direction,
				DialogID:   textDialogID,
			})
		}
	}
}

// 错误处理函数
func handleAsrError(senderName string, h media.MediaHandler, asr TranscribeService) func(err error, isFatal bool) {
	return func(err error, isFatal bool) {
		if isFatal {
			h.CauseError(senderName, err)
		} else {
			//asr.RestartClient()
			//_ = asr.StopConn()
		}
	}
}

func WithTranscribeFilterStateV2(asr TranscribeService, h media.MediaHandler, opt TranscribeOption) media.PacketFilter {
	senderName := "asr." + asr.Vendor()
	bytePerMillSecond := utils.ComputeSampleByteCount(16000, 16, 1)
	vadSpeaking := false
	ringbuffer := NewRingBuffer(500 * bytePerMillSecond)

	var currentDialogID string
	var startTranscribingAt time.Time
	corrector := NewAsrCorrector(opt.FuzzyOptions)
	asr.Init(
		handleAsrResult(senderName, opt, h, corrector, &currentDialogID, &startTranscribingAt),
		handleAsrError(senderName, h, asr),
	)

	h.GetSession().On(media.StartSpeaking, func(event media.StateChange) {
		if h.GetSession().GetString(media.WorkingState) != media.AgentRunning {
			return
		}

		var dialogID string
		if len(event.Params) > 0 {
			dialogID = event.Params[0].(string)
		} else {
			dialogID, _ = gonanoid.Nanoid()
		}
		if err := asr.ConnAndReceive(dialogID); err != nil {
			h.CauseError(senderName, err)
			return
		}

		padding := ringbuffer.Read(ringbuffer.size)
		if len(padding) > 0 {
			err := asr.SendAudioBytes(padding)
			if err != nil {
				h.CauseError(senderName, err)
			}
		}
		vadSpeaking = true
	})

	h.GetSession().On(media.StartSilence, func(event media.StateChange) {
		vadSpeaking = false
		_ = asr.SendEnd()
	})

	return func(packet media.MediaPacket) (bool, error) {
		if _, ok := packet.(*media.ClosePacket); ok {
			return false, asr.StopConn()
		}

		originalPayload := packet.Body()
		ringbuffer.Write(originalPayload)

		if vadSpeaking && asr.Activity() {
			if err := asr.SendAudioBytes(originalPayload); err != nil {
				h.CauseError(senderName, err)
			}
		}

		return false, nil
	}
}

type RingBuffer struct {
	buf        []byte
	size       int
	writeIndex int
	readIndex  int
	full       bool
	mu         sync.Mutex
}

func NewRingBuffer(size int) *RingBuffer {
	return &RingBuffer{
		buf:  make([]byte, size),
		size: size,
	}
}

func (r *RingBuffer) Write(data []byte) int {
	r.mu.Lock()
	defer r.mu.Unlock()

	n := len(data)
	for i := 0; i < n; i++ {
		r.buf[r.writeIndex] = data[i]
		r.writeIndex = (r.writeIndex + 1) % r.size
		if r.full {
			r.readIndex = (r.readIndex + 1) % r.size
		}
		r.full = r.writeIndex == r.readIndex
	}
	return n
}

func (r *RingBuffer) Read(n int) []byte {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.readIndex == r.writeIndex && !r.full {
		return nil
	}

	var result []byte
	for i := 0; i < n && (r.readIndex != r.writeIndex || r.full); i++ {
		result = append(result, r.buf[r.readIndex])
		r.readIndex = (r.readIndex + 1) % r.size
		r.full = false
	}
	return result
}
