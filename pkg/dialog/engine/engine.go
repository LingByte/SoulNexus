package engine

import "context"

// Engine is the central abstraction of the dialog layer. ONE Engine
// instance serves ONE call from ACK (or earlier in some flows) to
// hangup. Implementations:
//
//   - cascaded — ASR + LLM + TTS pipeline (pkg/dialog/engine/cascaded)
//   - realtime — full-duplex multimodal (pkg/dialog/engine/realtime)
//
// Engine never sees voice messages, transactions, dialogs or RTP. It
// receives PCM through MediaPort and emits PCM the same way. Transport-side
// concerns (codec negotiation, jitter, RTCP) stay in the media layer; the
// media layer constructs an Engine via Factory and calls Attach once per call.
//
// Concurrency: Attach is called once. Detach is safe to call from any
// goroutine. Implementations MUST be safe against the MediaPort
// returning errors / closed channels mid-flight (network drop).
type Engine interface {
	// Attach binds the engine to a single call. It SHOULD return
	// quickly (no blocking on first audio); long-running work runs
	// in goroutines owned by the engine. The returned Detach handle
	// is the ONLY supported teardown path.
	//
	// ctx is the call context — cancellation triggers the same
	// teardown as Detach (idempotent). lg is request-scoped logger
	// with at least call_id / tenant_id pre-set.
	Attach(ctx context.Context, port MediaPort, lg Logger) (Detach, error)

	// Mode returns the static mode of this engine (matches Factory
	// registration). Used for metrics labelling and observability.
	Mode() Mode
}

// MediaPort is the minimal media interface an Engine needs. It is a
// pure abstraction over a duplex audio channel + control signals;
// any transport (media RTP / WebRTC / file / fake) can implement it.
//
// Lifecycle:
//
//	Attach receives an open MediaPort.
//	Engine reads from InputPCM until the channel is closed by the
//	transport (call ended, network broke, transfer occurred).
//	Engine writes via SendOutputPCM until Detach is called.
//	OnBargeIn registers a callback fired by the transport when the
//	user starts speaking while AI is mid-sentence (used by engines
//	to interrupt their own TTS / realtime output).
type MediaPort interface {
	// InputPCM returns a receive-only channel of incoming audio
	// frames from the user. Closed by the transport on hangup.
	InputPCM() <-chan PCMFrame

	// SendOutputPCM queues a frame for playback to the user. Returns
	// an error if the transport is no longer accepting frames
	// (e.g. call ended). MUST be safe to call from any goroutine.
	SendOutputPCM(PCMFrame) error

	// OnBargeIn registers a single callback fired by the transport
	// when user-side voice activity is detected mid-AI-output. Only
	// the most-recently-registered callback fires (later registrations
	// replace earlier ones). nil clears the callback.
	OnBargeIn(func())

	// Codec describes the negotiated voice codec for this call (e.g.
	// PCMU / PCMA / G722). Engines normally don't care, but a few
	// (recorder, raw passthrough optimisations) do.
	Codec() CodecSpec

	// SampleRate is the bridged-side sample rate (8000 for narrowband
	// G.711, 16000 for G.722). All PCMFrame Data on this port arrives
	// at this rate; engines that need different rates resample.
	SampleRate() int

	// CallID is the unique ID for this call. Used for log correlation,
	// recording filenames, persistence keys.
	CallID() string

	// TenantID is the tenant that owns this call. Used for billing
	// and configuration scoping.
	TenantID() string
}

// CodecSpec is a small metadata bundle describing the transport
// codec. We don't use SDP codec types directly here to keep this
// package telephony-free; concrete adapters fill it in.
type CodecSpec struct {
	Name       string // "PCMU", "PCMA", "G722", "OPUS", ...
	SampleRate int    // wire sample rate (typically 8000 / 16000)
	Channels   int    // 1 for telephony
}

// Logger is the minimal logging surface an Engine needs. Concrete
// adapters wrap go.uber.org/zap or another framework. Decoupling
// here lets us avoid pulling zap into pure-logic test fixtures.
type Logger interface {
	Debug(msg string, fields ...Field)
	Info(msg string, fields ...Field)
	Warn(msg string, fields ...Field)
	Error(msg string, fields ...Field)
	With(fields ...Field) Logger
}

// Field is a typed key/value pair. We model it as a small struct
// rather than depend on zap.Field directly.
type Field struct {
	Key   string
	Value any
}

// F is the constructor used everywhere in this package.
func F(key string, value any) Field { return Field{Key: key, Value: value} }

// NopLogger is a Logger that drops every message. Useful as a safe
// default in tests and for engines that haven't been wired with a
// real logger yet. Returns itself from With so chained calls are
// also no-op.
type NopLogger struct{}

func (NopLogger) Debug(string, ...Field) {}
func (NopLogger) Info(string, ...Field)  {}
func (NopLogger) Warn(string, ...Field)  {}
func (NopLogger) Error(string, ...Field) {}
func (NopLogger) With(...Field) Logger   { return NopLogger{} }

// Compile-time assertion that NopLogger implements Logger.
var _ Logger = NopLogger{}
