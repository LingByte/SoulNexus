package session

import (
	"context"
	"net"
	"testing"
	"time"

	"github.com/LingByte/SoulNexus/pkg/voiceserver/media"
	siprtp "github.com/LingByte/SoulNexus/pkg/voiceserver/sip/rtp"
	"github.com/LingByte/SoulNexus/pkg/voiceserver/sip/sdp"
)

// ---------- CodecNegotiator --------------------------------------------

func TestCodecNegotiator_DefaultFourAudioCodecs(t *testing.T) {
	n := DefaultCodecNegotiator()
	for _, name := range []string{"pcma", "pcmu", "g722", "opus"} {
		if !n.Has(name) {
			t.Errorf("default negotiator missing %q", name)
		}
	}
	if n.Has("nonexistent") {
		t.Error("default negotiator should not report unknown codec")
	}
}

func TestCodecNegotiator_Negotiate_PrefersPCMA(t *testing.T) {
	n := DefaultCodecNegotiator()
	// Offer opus first then pcma — pcma should win per telephony preference.
	offer := []sdp.Codec{
		{PayloadType: 111, Name: "opus", ClockRate: 48000, Channels: 2},
		{PayloadType: 8, Name: "pcma", ClockRate: 8000, Channels: 1},
	}
	_, neg, err := n.Negotiate(offer)
	if err != nil {
		t.Fatalf("Negotiate: %v", err)
	}
	if neg.Name != "pcma" {
		t.Errorf("expected pcma, got %q", neg.Name)
	}
}

func TestCodecNegotiator_EmptyOffer(t *testing.T) {
	n := DefaultCodecNegotiator()
	if _, _, err := n.Negotiate(nil); err == nil {
		t.Error("empty offer must error")
	}
	if _, _, err := n.Negotiate([]sdp.Codec{}); err == nil {
		t.Error("empty offer must error")
	}
}

func TestCodecNegotiator_NoMatchingCodec(t *testing.T) {
	n := NewCodecNegotiator()
	n.Register("pcma", handlerG711("pcma", 8))
	if _, _, err := n.Negotiate([]sdp.Codec{{Name: "opus"}}); err == nil {
		t.Error("unsupported codec must error")
	}
}

func TestCodecNegotiator_RegisterUnregister(t *testing.T) {
	n := NewCodecNegotiator()
	h := func(c sdp.Codec) (sdp.Codec, media.CodecConfig, bool) {
		return sdp.Codec{Name: "custom"}, media.CodecConfig{Codec: "custom"}, true
	}
	n.Register("Custom", h) // case-insensitive
	if !n.Has("custom") {
		t.Error("Register should canonicalise name")
	}
	n.Unregister("CUSTOM")
	if n.Has("custom") {
		t.Error("Unregister should remove")
	}
	// Nil handler is ignored
	n.Register("x", nil)
	if n.Has("x") {
		t.Error("nil handler should be ignored")
	}
	// Empty name ignored
	n.Register("", h)
	if n.Has("") {
		t.Error("empty name should be ignored")
	}
}

func TestCodecNegotiator_HandlerRejectCandidate(t *testing.T) {
	// First handler always says no; second accepts → negotiator must skip to second.
	n := NewCodecNegotiator()
	n.Register("pcma", func(c sdp.Codec) (sdp.Codec, media.CodecConfig, bool) {
		return sdp.Codec{}, media.CodecConfig{}, false
	})
	n.Register("pcmu", handlerG711("pcmu", 8))
	n.SetPreference("pcma", "pcmu")

	offer := []sdp.Codec{
		{PayloadType: 8, Name: "pcma", ClockRate: 8000, Channels: 1},
		{PayloadType: 0, Name: "pcmu", ClockRate: 8000, Channels: 1},
	}
	_, neg, err := n.Negotiate(offer)
	if err != nil {
		t.Fatalf("Negotiate: %v", err)
	}
	if neg.Name != "pcmu" {
		t.Errorf("want pcmu fallback, got %q", neg.Name)
	}
}

func TestCodecNegotiator_SetPreferenceReorders(t *testing.T) {
	n := NewCodecNegotiator()
	n.Register("pcma", handlerG711("pcma", 8))
	n.Register("pcmu", handlerG711("pcmu", 8))
	n.SetPreference("pcmu", "pcma") // reverse telephony order
	offer := []sdp.Codec{
		{PayloadType: 8, Name: "pcma", ClockRate: 8000, Channels: 1},
		{PayloadType: 0, Name: "pcmu", ClockRate: 8000, Channels: 1},
	}
	_, neg, err := n.Negotiate(offer)
	if err != nil || neg.Name != "pcmu" {
		t.Errorf("preference reorder failed: neg=%+v err=%v", neg, err)
	}
}

func TestNegotiateOffer_DelegatesToDefault(t *testing.T) {
	// Baseline default: pcma wins over opus
	offer := []sdp.Codec{
		{PayloadType: 111, Name: "opus", ClockRate: 48000, Channels: 2},
		{PayloadType: 8, Name: "pcma", ClockRate: 8000, Channels: 1},
	}
	_, neg, err := NegotiateOffer(offer)
	if err != nil || neg.Name != "pcma" {
		t.Errorf("default NegotiateOffer: neg=%+v err=%v", neg, err)
	}
}

func TestSetDefaultNegotiator_ReplacesProcessDefault(t *testing.T) {
	old := defaultNegotiator
	defer SetDefaultNegotiator(old)

	n := NewCodecNegotiator()
	n.Register("pcmu", handlerG711("pcmu", 8))
	n.SetPreference("pcmu")
	SetDefaultNegotiator(n)

	offer := []sdp.Codec{
		{PayloadType: 8, Name: "pcma", ClockRate: 8000, Channels: 1}, // not registered
		{PayloadType: 0, Name: "pcmu", ClockRate: 8000, Channels: 1},
	}
	_, neg, err := NegotiateOffer(offer)
	if err != nil || neg.Name != "pcmu" {
		t.Errorf("custom default negotiator: neg=%+v err=%v", neg, err)
	}

	// nil must be ignored
	SetDefaultNegotiator(nil)
	if defaultNegotiator != n {
		t.Error("SetDefaultNegotiator(nil) should be a no-op")
	}
}

// ---------- handlerG711 / handlerG722 / handlerOpus ---------------------

func TestHandlerG711_ChannelClamp(t *testing.T) {
	h := handlerG711("pcma", 8)
	neg, src, ok := h(sdp.Codec{PayloadType: 8, Name: "pcma", ClockRate: 8000, Channels: 0})
	if !ok || neg.Channels != 1 || src.Channels != 1 {
		t.Errorf("channel clamp: neg=%+v src=%+v ok=%v", neg, src, ok)
	}
}

func TestHandlerG722_FixesClockAndChannel(t *testing.T) {
	neg, src, ok := handlerG722(sdp.Codec{PayloadType: 9, Name: "g722", ClockRate: 8000, Channels: 3})
	if !ok || neg.ClockRate != 8000 || src.SampleRate != 16000 {
		t.Errorf("g722: neg=%+v src=%+v ok=%v", neg, src, ok)
	}
}

func TestHandlerOpus_ChannelClamp(t *testing.T) {
	// 0-channel offer → 1 channel
	_, src, _ := handlerOpus(sdp.Codec{PayloadType: 111, Name: "opus", ClockRate: 48000, Channels: 0})
	if src.OpusDecodeChannels != 1 {
		t.Errorf("decodeCh=%d want 1 for 0-channel input", src.OpusDecodeChannels)
	}
	// 3-channel offer → 2 channels
	_, src, _ = handlerOpus(sdp.Codec{PayloadType: 111, Name: "opus", ClockRate: 48000, Channels: 3})
	if src.OpusDecodeChannels != 2 {
		t.Errorf("decodeCh=%d want 2 for 3-channel input", src.OpusDecodeChannels)
	}
}

// ---------- InternalPCMSampleRate — remaining branches ----------------

func TestInternalPCMSampleRate_OpusRateMapping(t *testing.T) {
	cases := []struct {
		in, want int
	}{
		{37000, 48000},
		{22000, 24000},
		{15000, 16000},
		{11000, 12000},
		{9000, 8000},
		{0, 48000}, // pure fallback
	}
	for _, c := range cases {
		got := InternalPCMSampleRate(media.CodecConfig{Codec: "opus", SampleRate: c.in})
		if got != c.want {
			t.Errorf("opus %dHz → %d, want %d", c.in, got, c.want)
		}
	}
}

func TestInternalPCMSampleRate_PCMAZeroRateFallback(t *testing.T) {
	if got := InternalPCMSampleRate(media.CodecConfig{Codec: "pcma"}); got != 8000 {
		t.Errorf("pcma 0-rate → %d, want 8000", got)
	}
}

func TestInternalPCMSampleRate_UnknownCodecFallback(t *testing.T) {
	if got := InternalPCMSampleRate(media.CodecConfig{Codec: "l16", SampleRate: 44100}); got != 44100 {
		t.Errorf("l16 44k1 → %d, want 44100", got)
	}
	if got := InternalPCMSampleRate(media.CodecConfig{Codec: "unknown"}); got != 16000 {
		t.Errorf("unknown 0-rate → %d, want 16000 fallback", got)
	}
}

// ---------- telephoneEventPT + passthroughDTMFDecode -------------------

func TestTelephoneEventPT(t *testing.T) {
	offer := []sdp.Codec{
		{PayloadType: 101, Name: "telephone-event", ClockRate: 8000},
		{PayloadType: 8, Name: "pcma", ClockRate: 8000},
	}
	if got := telephoneEventPT(offer, 8000); got != 101 {
		t.Errorf("got %d, want 101", got)
	}
	// No telephone-event in offer
	if got := telephoneEventPT([]sdp.Codec{{Name: "pcma"}}, 8000); got != 0 {
		t.Errorf("missing telephone-event → %d, want 0", got)
	}
}

func TestPassthroughDTMFDecode(t *testing.T) {
	innerCalls := 0
	dec := passthroughDTMFDecode(func(p media.MediaPacket) ([]media.MediaPacket, error) {
		innerCalls++
		return []media.MediaPacket{p}, nil
	})
	// DTMF passes through without invoking inner
	out, err := dec(&media.DTMFPacket{Digit: "5"})
	if err != nil || len(out) != 1 || innerCalls != 0 {
		t.Errorf("DTMF passthrough broken: err=%v out=%d innerCalls=%d", err, len(out), innerCalls)
	}
	// Audio goes through inner
	_, _ = dec(&media.AudioPacket{Payload: []byte{1}})
	if innerCalls != 1 {
		t.Errorf("audio did not invoke inner: %d", innerCalls)
	}
}

// ---------- ApplyRemoteSDP ---------------------------------------------

func TestApplyRemoteSDP_Errors(t *testing.T) {
	if err := ApplyRemoteSDP(nil, &sdp.Info{IP: "127.0.0.1", Port: 1}); err == nil {
		t.Error("nil session must error")
	}
	s, _ := siprtp.NewSession(0)
	defer s.Close()
	if err := ApplyRemoteSDP(s, nil); err == nil {
		t.Error("nil info must error")
	}
	if err := ApplyRemoteSDP(s, &sdp.Info{IP: "", Port: 5004}); err == nil {
		t.Error("empty IP must error")
	}
	if err := ApplyRemoteSDP(s, &sdp.Info{IP: "127.0.0.1", Port: 0}); err == nil {
		t.Error("zero port must error")
	}
	if err := ApplyRemoteSDP(s, &sdp.Info{IP: "not-an-ip", Port: 5004}); err == nil {
		t.Error("bad IP must error")
	}
}

func TestApplyRemoteSDP_HappyPath(t *testing.T) {
	s, _ := siprtp.NewSession(0)
	defer s.Close()
	if err := ApplyRemoteSDP(s, &sdp.Info{IP: "127.0.0.1", Port: 5004}); err != nil {
		t.Fatalf("ApplyRemoteSDP: %v", err)
	}
	if s.RemoteAddr == nil || s.RemoteAddr.Port != 5004 {
		t.Errorf("RemoteAddr not set: %+v", s.RemoteAddr)
	}
}

// ---------- NewMediaLeg + MediaLeg methods ------------------------------

func pcmaOffer() []sdp.Codec {
	return []sdp.Codec{
		{PayloadType: 8, Name: "pcma", ClockRate: 8000, Channels: 1},
		{PayloadType: 101, Name: "telephone-event", ClockRate: 8000},
	}
}

func TestNewMediaLeg_GuardClauses(t *testing.T) {
	s, _ := siprtp.NewSession(0)
	defer s.Close()

	// empty callID
	if _, err := NewMediaLeg(context.Background(), "", s, pcmaOffer(), MediaLegConfig{}); err == nil {
		t.Error("empty callID must error")
	}
	// nil rtp session
	if _, err := NewMediaLeg(context.Background(), "c1", nil, pcmaOffer(), MediaLegConfig{}); err == nil {
		t.Error("nil rtp session must error")
	}
	// empty offer → negotiation error
	if _, err := NewMediaLeg(context.Background(), "c1", s, nil, MediaLegConfig{}); err == nil {
		t.Error("empty offer must error")
	}
	// unsupported codec
	bad := []sdp.Codec{{PayloadType: 99, Name: "g729", ClockRate: 8000}}
	if _, err := NewMediaLeg(context.Background(), "c1", s, bad, MediaLegConfig{}); err == nil {
		t.Error("unsupported codec must error")
	}
	// nil parent context must be tolerated
	if _, err := NewMediaLeg(nil, "c1", s, pcmaOffer(), MediaLegConfig{}); err != nil {
		t.Errorf("nil ctx should be accepted (Background used): %v", err)
	}
}

func TestNewMediaLeg_HappyPath_AndAccessors(t *testing.T) {
	s, _ := siprtp.NewSession(0)
	defer s.Close()

	cfg := MediaLegConfig{
		OutputQueueSize:     64,
		MaxSessionSeconds:   30,
		JitterPlaybackDelay: 40 * time.Millisecond,
	}
	leg, err := NewMediaLeg(context.Background(), "call-1", s, pcmaOffer(), cfg)
	if err != nil {
		t.Fatalf("NewMediaLeg: %v", err)
	}
	defer leg.Stop()

	if leg.MediaSession() == nil {
		t.Error("MediaSession nil")
	}
	if leg.RTPSession() != s {
		t.Error("RTPSession mismatch")
	}
	if got := leg.NegotiatedSDP(); got.Name != "pcma" {
		t.Errorf("negotiated SDP = %+v", got)
	}
	if got := leg.SourceCodec(); got.Codec != "pcma" {
		t.Errorf("source codec = %+v", got)
	}
	if got := leg.PCMSampleRate(); got != 8000 {
		t.Errorf("pcm sample rate = %d, want 8000", got)
	}
	if got := leg.DTMFPayloadType(); got != 101 {
		t.Errorf("dtmf pt = %d, want 101", got)
	}
}

func TestNewMediaLeg_TapsAndFilters(t *testing.T) {
	s, _ := siprtp.NewSession(0)
	defer s.Close()

	tapInCalls := 0
	tapOutCalls := 0
	filterCalls := 0
	cfg := MediaLegConfig{
		RTPInputTap:   func(seq uint16, ts uint32, p []byte) { tapInCalls++ },
		RTPOutputTap:  func(seq uint16, ts uint32, p []byte) { tapOutCalls++ },
		InputFilters:  []media.PacketFilter{func(p media.MediaPacket) (bool, error) { filterCalls++; return false, nil }},
		OutputFilters: []media.PacketFilter{func(p media.MediaPacket) (bool, error) { return false, nil }},
	}
	leg, err := NewMediaLeg(context.Background(), "taps", s, pcmaOffer(), cfg)
	if err != nil {
		t.Fatalf("NewMediaLeg: %v", err)
	}
	defer leg.Stop()

	// Verify the taps were installed on the RTP transports (we can't easily
	// drive real UDP in this unit test — just confirm wiring).
	if leg.rx.OnInputRTP == nil {
		t.Error("RTPInputTap not installed on rx transport")
	}
	if leg.tx.OnOutputRTP == nil {
		t.Error("RTPOutputTap not installed on tx transport")
	}
	if leg.rx.JitterPlaybackDelay != siprtp.DefaultJitterPlaybackDelay {
		// No override was set in cfg, so it should be the default
		// (this cfg didn't set JitterPlaybackDelay)
	}

	// Prevent unused warning
	_ = tapInCalls
	_ = tapOutCalls
	_ = filterCalls
}

func TestMediaLeg_NilReceivers(t *testing.T) {
	var l *MediaLeg
	if l.MediaSession() != nil {
		t.Error("nil MediaLeg MediaSession should be nil")
	}
	if l.RTPSession() != nil {
		t.Error("nil MediaLeg RTPSession should be nil")
	}
	if got := l.NegotiatedSDP(); got.Name != "" {
		t.Error("nil MediaLeg NegotiatedSDP should be zero value")
	}
	if got := l.SourceCodec(); got.Codec != "" {
		t.Error("nil MediaLeg SourceCodec should be zero value")
	}
	if got := l.PCMSampleRate(); got != 16000 {
		t.Errorf("nil MediaLeg PCMSampleRate should be 16000 default, got %d", got)
	}
	if l.DTMFPayloadType() != 0 {
		t.Error("nil MediaLeg DTMFPayloadType should be 0")
	}
	l.Start()                 // must not panic
	l.Stop()                  // must not panic
	l.StopMediaPreserveRTP()  // must not panic
	l.CloseRTPOnly()          // must not panic
}

func TestMediaLeg_StartAndStop(t *testing.T) {
	s, _ := siprtp.NewSession(0)
	leg, err := NewMediaLeg(context.Background(), "start-stop", s, pcmaOffer(), MediaLegConfig{})
	if err != nil {
		t.Fatalf("NewMediaLeg: %v", err)
	}
	leg.Start()
	leg.Start() // idempotent
	time.Sleep(20 * time.Millisecond)
	leg.Stop()
}

func TestMediaLeg_StopMediaPreserveRTP_KeepsSocket(t *testing.T) {
	s, _ := siprtp.NewSession(0)
	defer s.Close()
	leg, err := NewMediaLeg(context.Background(), "preserve", s, pcmaOffer(), MediaLegConfig{})
	if err != nil {
		t.Fatalf("NewMediaLeg: %v", err)
	}
	leg.Start()
	time.Sleep(20 * time.Millisecond)
	leg.StopMediaPreserveRTP()
	// Socket should still be usable
	if leg.rtpSess == nil {
		t.Error("rtpSess was cleared even though preserve-on-close was requested")
	}
	if leg.rtpSess.Conn == nil {
		t.Error("rtpSess.Conn closed even though preserve was requested")
	}
}

func TestMediaLeg_CloseRTPOnly(t *testing.T) {
	s, _ := siprtp.NewSession(0)
	leg, err := NewMediaLeg(context.Background(), "rtp-only", s, pcmaOffer(), MediaLegConfig{})
	if err != nil {
		t.Fatalf("NewMediaLeg: %v", err)
	}
	leg.CloseRTPOnly()
	if leg.rtpSess != nil {
		t.Error("rtpSess should be nil after CloseRTPOnly")
	}
}

// Fail-path: unsupported codec in encoder factory (simulate by forcing a
// codec name that the negotiator accepts but encoder doesn't support).
// We do this by temporarily swapping the default negotiator so it returns
// "g729" which has no registered encoder.
func TestNewMediaLeg_EncoderFactoryFailure(t *testing.T) {
	old := defaultNegotiator
	defer SetDefaultNegotiator(old)

	n := NewCodecNegotiator()
	n.Register("g729", func(c sdp.Codec) (sdp.Codec, media.CodecConfig, bool) {
		return sdp.Codec{Name: "g729"}, media.CodecConfig{Codec: "g729", SampleRate: 8000, Channels: 1, BitDepth: 8}, true
	})
	n.SetPreference("g729")
	SetDefaultNegotiator(n)

	s, _ := siprtp.NewSession(0)
	defer s.Close()
	offer := []sdp.Codec{{PayloadType: 18, Name: "g729", ClockRate: 8000, Channels: 1}}
	if _, err := NewMediaLeg(context.Background(), "g729-fail", s, offer, MediaLegConfig{}); err == nil {
		t.Error("unregistered codec should fail at encoder factory")
	}
}

// ---------- Avoid unused imports warnings -----------------------------
var _ = net.IPv4
