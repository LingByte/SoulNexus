package session

import (
	"context"
	"sync"
	"time"

	"github.com/LingByte/SoulNexus/pkg/dialog/cascaded"
	"github.com/LingByte/SoulNexus/pkg/dialog/engine"
	"github.com/LingByte/SoulNexus/pkg/dialog/pipeline"
	"github.com/LingByte/SoulNexus/pkg/dialog/tenantcfg"
)

// TransportKind identifies how a session receives user audio.
type TransportKind string

const (
	TransportWebSocket TransportKind = "websocket"
	TransportWebRTC    TransportKind = "webrtc"
	TransportText      TransportKind = "text"
)

// CreateParams holds metadata for a new voice session.
type CreateParams struct {
	TenantID    uint
	Transport   TransportKind
	AssistantID uint
	SampleRate  int
	// CallID scopes assistant / knowledge binding (session binding key).
	CallID string
	// UserID is the human tenant user when created via JWT (0 for API key).
	UserID uint
	// CredentialID is the API key row when created via machine credential.
	CredentialID uint
	// SkipCallPersist disables turn/recording persistence (assistant debug).
	SkipCallPersist bool
	// DialogMode selects engine attach (default) or voicedialog gateway simulation.
	DialogMode string
	// RealtimeTemperature overrides tenant realtime_config.temperature when > 0 (debug UI).
	RealtimeTemperature float64
}

type Info struct {
	SessionID        string        `json:"sessionId"`
	TenantID         uint          `json:"tenantId"`
	AssistantID      uint          `json:"assistantId,omitempty"`
	Transport        TransportKind `json:"transport"`
	SampleRate       int           `json:"sampleRateHz,omitempty"`
	WebSocketPath    string        `json:"webSocketPath,omitempty"`
	WebRTCOfferPath  string        `json:"webRtcOfferPath,omitempty"`
	DialogMode       string        `json:"dialogMode,omitempty"`
	VoiceDialogWsURL string        `json:"voiceDialogWsUrl,omitempty"`
	CreatedAt        time.Time     `json:"createdAt"`
}

// Session is one in-flight voice dialog.
type Session struct {
	ID              string
	TenantID        uint
	AssistantID     uint
	Transport       TransportKind
	SampleRate      int
	CallID          string
	CreatedAt       time.Time
	skipCallPersist bool
	dialogMode      string
	realtimeTemp    float64

	mu            sync.Mutex
	port          engine.MediaPort
	detach        engine.Detach
	cancel        context.CancelFunc
	attached      bool
	ended         bool
	onTurn        func(cascaded.TurnRecord)
	wireWrite     func([]byte) error
	liveTurnID    string
	textLLM       cascaded.LLMService
	textWelcome   string
	textLLMCancel context.CancelFunc
	dialogConvID  uint // persisted dialog/chat conversation when TextChatBridge is set
	turnSeq       uint64
}

// AttachParams wires an engine to an open MediaPort.
type AttachParams struct {
	Port            engine.MediaPort
	Env             tenantcfg.VoiceEnv
	Mode            engine.Mode
	Logger          engine.Logger
	TTSRecorderTap  func([]byte) // optional optional stereo recorder hook
	OnTurn          func(cascaded.TurnRecord)
	OnLiveTranscript func(pipeline.Frame)
	SkipCallPersist bool
	// UplinkOnPort means the MediaPort already runs denoise/NS once before
	// fanout (StreamingPort). Skip WrapASRPreprocessor to avoid double NS.
	UplinkOnPort bool
}
