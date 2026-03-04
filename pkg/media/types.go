package media

import (
	"context"
	"errors"
	"fmt"
	"io"
	"time"
)

// MediaPacket types - represents media data packets
type MediaPacket interface {
	fmt.Stringer
	Body() []byte
}

type StreamFormat struct {
	SampleRate    int
	BitDepth      int
	Channels      int
	FrameDuration time.Duration
}

type TextPacket struct {
	PlayID         string    `json:"id,omitempty"`
	Text           string    `json:"text"`
	IsTranscribed  bool      `json:"isTranscribed"`
	IsLLMGenerated bool      `json:"isLLMGenerated"`
	IsPartial      bool      `json:"isPartial"`
	IsEnd          bool      `json:"isEnd"`
	Sequence       int       `json:"sequence"`
	StartAt        time.Time `json:"startAt"`
}

type AudioPacket struct {
	PlayID        string `json:"id,omitempty"`
	Sequence      int    `json:"sequence"`
	Payload       []byte `json:"payload"`
	IsFirstPacket bool   `json:"isFirstPacket,omitempty"`
	IsEndPacket   bool   `json:"isEndPacket,omitempty"`
	IsSynthesized bool   `json:"isSynthesized,omitempty"`
	IsSilence     bool   `json:"isSilence,omitempty"`
	SourceText    string `json:"sourceText,omitempty"`
}

type ClosePacket struct {
	Reason string `json:"reason"`
}

func (f *ClosePacket) Body() []byte {
	return nil
}

func (f *ClosePacket) String() string {
	return fmt.Sprintf("ClosePacket{Reason: %s}", f.Reason)
}

func (t *TextPacket) Body() []byte {
	return []byte(t.Text)
}

func (t *TextPacket) String() string {
	source := "user"
	if t.IsTranscribed {
		source = "Transcribed"
	}
	if t.IsLLMGenerated {
		source = "LLMGenerated"
	}
	return fmt.Sprintf("TextPacket{Text: %q, Source: %s, IsPartial: %t, IsEnd: %t, Sequence: %d}}",
		t.Text, source, t.IsPartial, t.IsEnd, t.Sequence)
}

func (d *AudioPacket) Body() []byte {
	return d.Payload
}

func (d *AudioPacket) String() string {
	return fmt.Sprintf("AudioPacket{Payload: %d bytes, IsFirstPacket: %t, IsSynthesized: %t, IsSilence: %t}",
		len(d.Payload), d.IsFirstPacket, d.IsSynthesized, d.IsSilence)
}

// State types
var (
	AllStates     = "*"
	Begin         = "begin"
	End           = "end"
	Hangup        = "hangup"
	StartSpeaking = "speaking.start"
	StartSilence  = "silence.start"
	Transcribing  = "transcribing" // params: sentence string
	Synthesizing  = "synthesizing" // params: result string
	StartPlay     = "play.start"
	StopPlay      = "play.stop"
	Completed     = "completed"
	// interrupt
	Interruption = "interruption"
)

var (
	MediaDataTypeState  = "state"
	MediaDataTypePacket = "packet"
	MediaDataTypeMetric = "metric"
)

var (
	ErrNotInputTransport  = errors.New("not input transport")
	ErrNotOutputTransport = errors.New("not output transport")
	ErrCodecNotSupported  = errors.New("codec not supported")
)

var (
	AgentRunning    = "_agent_running"
	WorkingState    = "_working_state"
	UpstreamRunning = "_upstream_running"
)

type StateChange struct {
	State  string `json:"state"`
	Params []any  `json:"params,omitempty"`
}

type MediaData struct {
	CreatedAt time.Time
	Sender    any
	Type      string
	State     StateChange
	Packet    MediaPacket
	Duration  *time.Duration
}

type CompletedData struct {
	SenderName   string        `json:"senderName"` // eg: tts.aws, asr.qcloud
	Duration     time.Duration `json:"duration"`   // total duration
	Source       MediaPacket   `json:"-"`          // last packet
	Result       any           `json:"result"`     // result
	AssistantId  uint          `json:"assistantId"`
	AssistantVid uint          `json:"assistantVid"`
	DialogID     string        `json:"dialogID"`
}

type TurnDetectionData struct {
	SenderName string `json:"senderName"`
	CostTime   int64  `json:"cost_time"`
	Status     string `json:"status"`
	Text       string `json:"text"`
	DialogID   string `json:"dialogID"`
}

type TranscribingData struct {
	SenderName string        `json:"senderName"` // eg: tts.aws, asr.qcloud
	Duration   time.Duration `json:"duration"`   // total duration
	Source     MediaPacket   `json:"-"`          // last packet
	Result     any           `json:"result"`     // result
	Direction  string        `json:"direction"`  // direction
	DialogID   string        `json:"dialogID"`
}

func (d *MediaData) String() string {
	if d.Type == MediaDataTypeState {
		return fmt.Sprintf("MediaData{Sender:%s, State:%s}", d.Sender, d.State)
	} else if d.Type == MediaDataTypePacket {
		return fmt.Sprintf("MediaData{Sender:%s, Packet:%s}", d.Sender, d.Packet)
	} else {
		return fmt.Sprintf("MediaData{Sender:%s, Type:%s}", d.Sender, d.Type)
	}
}

func (d *CompletedData) MarshalJSON() ([]byte, error) {
	return []byte(fmt.Sprintf(`{"senderName":"%s","duration":"%s","result":"%s", "dialogID": "%s"}`, d.SenderName, d.Duration.String(), d.Result, d.DialogID)), nil
}

func (d CompletedData) String() string {
	return fmt.Sprintf("CompletedData{SenderName:%s, Duration:%s, Result:%s, DialogID: %s}", d.SenderName, d.Duration, d.Result, d.DialogID)
}

func (d *TranscribingData) MarshalJSON() ([]byte, error) {
	return []byte(fmt.Sprintf(`{"senderName":"%s","duration":"%s","result":"%s", "dialogID": "%s"}`, d.SenderName, d.Duration.String(), d.Result, d.DialogID)), nil
}

func (d TranscribingData) String() string {
	return fmt.Sprintf("TranscribingData{SenderName:%s, Duration:%s, Result:%s, DialogID: %s}", d.SenderName, d.Duration, d.Result, d.DialogID)
}

func (s *StateChange) SafeGetStr(idx int) string {
	if idx < 0 || idx >= len(s.Params) {
		return ""
	}
	if str, ok := s.Params[idx].(string); ok {
		return str
	}
	return ""
}

// MediaTransport interface for media transport
type MediaTransport interface {
	io.Closer
	String() string
	Attach(s *MediaSession)
	Next(ctx context.Context) (MediaPacket, error)
	Send(ctx context.Context, packet MediaPacket) (int, error)
	Codec() CodecConfig
	Close() error
}

// CodecConfig defines codec configuration
type CodecConfig struct {
	Codec         string `json:"codec" form:"codec" default:"pcm"`
	SampleRate    int    `json:"sampleRate" form:"sample_rate" default:"16000"`
	Channels      int    `json:"channels" form:"channels" default:"1"`
	BitDepth      int    `json:"bitDepth" form:"bit_depth" default:"16"`
	FrameDuration string `json:"frameDuration" form:"frame_duration"`
	PayloadType   uint8  `json:"payloadType" form:"payload_type"`
}

func DefaultCodecConfig() CodecConfig {
	return CodecConfig{
		Codec:         "pcm",
		SampleRate:    16000,
		Channels:      1,
		BitDepth:      16,
		FrameDuration: "",
	}
}

func (c CodecConfig) String() string {
	return fmt.Sprintf("CodecConfig{Codec: %s, SampleRate: %d, Channels: %d, BitDepth: %d, FrameDuration: %s}",
		c.Codec, c.SampleRate, c.Channels, c.BitDepth, c.FrameDuration)
}

// Constants
const (
	DirectionInput  = "rx"
	DirectionOutput = "tx"
)
