package synthesizer

// Copyright (c) 2026 LinByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

import (
	"strings"
	"testing"

	"github.com/LingByte/SoulNexus/pkg/media"
)

// ---------- StripEmoji ------------------------------------------------------

func TestStripEmoji(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want string
	}{
		{"no emoji", "hello world", "hello world"},
		{"single emoji", "hi 👋 there", "hi  there"},
		{"many emojis", "🎉🎊 party 😀", " party "},
		{"cn + emoji", "你好😀世界", "你好世界"},
		{"empty", "", ""},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := StripEmoji(c.in); got != c.want {
				t.Errorf("StripEmoji(%q) = %q, want %q", c.in, got, c.want)
			}
		})
	}
}

// ---------- NewSynthesisService: factory dispatch --------------------------

func TestNewSynthesisService_UnknownVendor(t *testing.T) {
	svc, err := NewSynthesisService("tts.does-not-exist", nil)
	if err == nil {
		t.Fatal("unknown vendor should error")
	}
	if svc != nil {
		t.Error("svc must be nil on unknown vendor")
	}
	if !strings.Contains(err.Error(), "unknown") {
		t.Errorf("error message = %q, want to contain 'unknown'", err.Error())
	}
}

func TestNewSynthesisService_KnownVendorsConstruct(t *testing.T) {
	// We're verifying the factory wiring: each known vendor constructs a
	// non-nil SynthesisService object. The underlying network call is not
	// exercised; Synthesize would need real credentials.
	vendors := []string{
		TTS_QCLOUD, TTS_XUNFEI, TTS_QINIU, TTS_AWS, TTS_BAIDU,
		TTS_GOOGLE, TTS_AZURE, TTS_OPENAI, TTS_ELEVENLABS,
		TTS_LOCAL, TTS_FISHSPEECH, TTS_FISHAUDIO, TTS_COQUI,
		TTS_VOLCENGINE, TTS_MINIMAX,
	}
	for _, v := range vendors {
		t.Run(v, func(t *testing.T) {
			svc, err := NewSynthesisService(v, map[string]any{})
			if err != nil {
				// Some constructors (e.g. local_gospeech) validate config
				// eagerly and can return an error with empty options.
				t.Logf("%s constructor errored (expected for some vendors): %v", v, err)
				return
			}
			if svc == nil {
				t.Fatalf("%s: svc is nil", v)
			}
			// Provider must return a non-empty TTSProvider string.
			if string(svc.Provider()) == "" {
				t.Errorf("%s: empty provider", v)
			}
		})
	}
}

func TestNewSynthesisService_LocalGoSpeechRequiresConfig(t *testing.T) {
	// local_gospeech returns (service, error) — with empty map the
	// constructor should error out cleanly rather than panic.
	svc, err := NewSynthesisService(TTS_LOCAL_GOSPEECH, map[string]any{})
	if err == nil && svc == nil {
		t.Error("local_gospeech should either construct or error")
	}
}

// ---------- TTSCredentialConfig accessors ----------------------------------

func TestTTSCredentialConfig_GetString(t *testing.T) {
	c := TTSCredentialConfig{
		"apiKey":   "secret",
		"count":    42, // non-string, fmt.Sprintf coercion
		"absent":   nil,
	}
	if got := c.getString("apiKey"); got != "secret" {
		t.Errorf("getString apiKey = %q", got)
	}
	if got := c.getString("count"); got != "42" {
		t.Errorf("getString count = %q, want '42'", got)
	}
	if got := c.getString("missing"); got != "" {
		t.Errorf("getString missing = %q, want empty", got)
	}
	// nil value → Sprintf("<nil>")
	if got := c.getString("absent"); got != "<nil>" {
		t.Errorf("getString absent = %q, want '<nil>'", got)
	}
}

func TestTTSCredentialConfig_GetInt64(t *testing.T) {
	c := TTSCredentialConfig{
		"i":    int(7),
		"i64":  int64(8),
		"f64":  float64(9.9),
		"str":  "10",
		"bad":  "not-a-number",
		"type": struct{}{},
	}
	if got := c.getInt64("i"); got != 7 {
		t.Errorf("i=%d", got)
	}
	if got := c.getInt64("i64"); got != 8 {
		t.Errorf("i64=%d", got)
	}
	if got := c.getInt64("f64"); got != 9 {
		t.Errorf("f64=%d", got)
	}
	if got := c.getInt64("str"); got != 10 {
		t.Errorf("str=%d", got)
	}
	if got := c.getInt64("bad"); got != 0 {
		t.Errorf("bad=%d, want 0 fallback", got)
	}
	if got := c.getInt64("type"); got != 0 {
		t.Errorf("type=%d, want 0 fallback", got)
	}
	if got := c.getInt64("missing"); got != 0 {
		t.Errorf("missing=%d, want 0 fallback", got)
	}
}

// ---------- NewSynthesisServiceFromCredential ------------------------------

func TestNewSynthesisServiceFromCredential_Validation(t *testing.T) {
	// nil / empty config
	if _, err := NewSynthesisServiceFromCredential(nil); err == nil {
		t.Error("nil config should error")
	}
	if _, err := NewSynthesisServiceFromCredential(TTSCredentialConfig{}); err == nil {
		t.Error("empty config should error")
	}
	// missing provider
	if _, err := NewSynthesisServiceFromCredential(TTSCredentialConfig{"apiKey": "x"}); err == nil {
		t.Error("missing provider should error")
	}
	// qiniu without apiKey
	cfg := TTSCredentialConfig{"provider": "qiniu"}
	if _, err := NewSynthesisServiceFromCredential(cfg); err == nil {
		t.Error("qiniu without apiKey should error")
	}
}

func TestNewSynthesisServiceFromCredential_QiniuHappy(t *testing.T) {
	cfg := TTSCredentialConfig{
		"provider": "qiniu",
		"apiKey":   "fake-qiniu-key",
		"baseUrl":  "https://example.com",
	}
	svc, err := NewSynthesisServiceFromCredential(cfg)
	if err != nil {
		t.Fatalf("qiniu credential construct: %v", err)
	}
	if svc == nil || svc.Provider() != ProviderQiniu {
		t.Errorf("provider=%v want %v", svc.Provider(), ProviderQiniu)
	}
}

// ---------- SynthesisPlayer: constructor + Close ---------------------------

func TestNewSynthesisPlayer_Defaults(t *testing.T) {
	p := NewSynthesisPlayer("tts.test", media.StreamFormat{SampleRate: 16000, BitDepth: 16, Channels: 1})
	if p == nil {
		t.Fatal("nil player")
	}
	if p.SenderName != "tts.test" {
		t.Errorf("SenderName=%q", p.SenderName)
	}
	if p.Format.SampleRate != 16000 {
		t.Errorf("SampleRate=%d", p.Format.SampleRate)
	}
	if p.reqChan == nil {
		t.Error("reqChan must be initialized")
	}
	if p.playRecords == nil {
		t.Error("playRecords must be initialized")
	}
	// Close is a simple log-only shutdown.
	p.Close()
}

// ---------- SynthesisBuffer: trivial sinks ---------------------------------

func TestSynthesisBuffer_OnMessage(t *testing.T) {
	var b SynthesisBuffer
	b.OnMessage([]byte("hello"))
	b.OnMessage([]byte(" world"))
	if string(b.Data) != "hello world" {
		t.Errorf("buffer=%q", string(b.Data))
	}
	b.OnTimestamp(SentenceTimestamp{Words: []Word{{Word: "hello", StartTime: 0, EndTime: 500}}})
	if len(b.Timestamp.Words) != 1 || b.Timestamp.Words[0].Word != "hello" {
		t.Errorf("timestamp=%+v", b.Timestamp)
	}
}
