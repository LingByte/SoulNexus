package recognizer

// Copyright (c) 2026 LinByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"testing"
)

// ---------- GzipCompress / GzipDecompress round-trip ----------------------

func TestGzipRoundTrip(t *testing.T) {
	cases := [][]byte{
		[]byte("hello world"),
		[]byte(""),
		bytes.Repeat([]byte("abc"), 1000),
	}
	for i, in := range cases {
		out := GzipDecompress(GzipCompress(in))
		if !bytes.Equal(out, in) {
			t.Errorf("case %d: round-trip mismatch (in=%d out=%d)", i, len(in), len(out))
		}
	}
}

// ---------- IsWAVFile -----------------------------------------------------

func TestIsWAVFile(t *testing.T) {
	tooShort := make([]byte, 10)
	if IsWAVFile(tooShort) {
		t.Error("short buffer must not be WAV")
	}

	bogus := make([]byte, 50)
	if IsWAVFile(bogus) {
		t.Error("all-zero buffer must not be WAV")
	}

	good := make([]byte, 50)
	copy(good[0:4], []byte("RIFF"))
	copy(good[8:12], []byte("WAVE"))
	if !IsWAVFile(good) {
		t.Error("RIFF/WAVE header should pass")
	}
}

// ---------- ReadWAVInfo ---------------------------------------------------

func TestReadWAVInfo_Minimal(t *testing.T) {
	// Build a minimal 44-byte WAV header + 4 bytes PCM data.
	buf := new(bytes.Buffer)
	hdr := WAVHeader{
		ChunkSize:     36 + 4,
		AudioFormat:   1,
		NumChannels:   1,
		SampleRate:    16000,
		ByteRate:      16000 * 2,
		BlockAlign:    2,
		BitsPerSample: 16,
		Subchunk1Size: 16,
		Subchunk2Size: 4,
	}
	copy(hdr.ChunkID[:], []byte("RIFF"))
	copy(hdr.Format[:], []byte("WAVE"))
	copy(hdr.Subchunk1ID[:], []byte("fmt "))
	copy(hdr.Subchunk2ID[:], []byte("data"))
	_ = binary.Write(buf, binary.LittleEndian, hdr)
	buf.Write([]byte{0x01, 0x00, 0x02, 0x00})

	ch, sw, sr, nf, data, err := ReadWAVInfo(buf.Bytes())
	if err != nil {
		t.Fatalf("ReadWAVInfo: %v", err)
	}
	if ch != 1 || sw != 2 || sr != 16000 || nf != 2 || len(data) != 4 {
		t.Errorf("got ch=%d sw=%d sr=%d nf=%d data=%d", ch, sw, sr, nf, len(data))
	}
}

func TestReadWAVInfo_TruncatedHeader(t *testing.T) {
	if _, _, _, _, _, err := ReadWAVInfo([]byte{0x00}); err == nil {
		t.Error("truncated header should error")
	}
}

// ---------- DefaultConfig + builders --------------------------------------

func TestDefaultConfig_Defaults(t *testing.T) {
	c := DefaultConfig()
	if c == nil {
		t.Fatal("nil default config")
	}
	if c.Audio.Rate != 16000 || c.Audio.Bits != 16 || c.Audio.Channel != 1 {
		t.Errorf("audio defaults: %+v", c.Audio)
	}
	if c.Buffer.SegmentDurationMs != 200 {
		t.Errorf("segment=%d", c.Buffer.SegmentDurationMs)
	}
	if c.Request.ModelName != "bigmodel" {
		t.Errorf("model=%q", c.Request.ModelName)
	}
}

func TestConfig_Builders(t *testing.T) {
	c := DefaultConfig().
		WithURL("wss://example").
		WithAuth(AuthConfig{AccessKey: "ak", AppKey: "app"}).
		WithUser(UserConfig{UID: "u1"}).
		WithAudio(AudioConfig{Format: "pcm", Rate: 8000, Bits: 16, Channel: 1}).
		WithRequest(RequestConfig{ModelName: "custom"}).
		WithBuffer(BufferConfig{SegmentDurationMs: 500, MaxBufferSize: 0})

	if c.URL != "wss://example" || c.Auth.AccessKey != "ak" || c.User.UID != "u1" {
		t.Errorf("builders didn't apply: %+v", c)
	}
}

func TestConfig_CalculateBufferSize(t *testing.T) {
	// Explicit MaxBufferSize wins.
	c := &Config{Buffer: BufferConfig{MaxBufferSize: 1234}}
	if got := c.CalculateBufferSize(); got != 1234 {
		t.Errorf("explicit=%d", got)
	}

	// Calculated: 16000 Hz * 16/8 bytes * 1ch / 1000 = 32 bytes/ms * 200ms = 6400
	c2 := DefaultConfig()
	if got := c2.CalculateBufferSize(); got != 6400 {
		t.Errorf("calc=%d want 6400", got)
	}
}

// ---------- RequestHeader -------------------------------------------------

func TestRequestHeader_Builders(t *testing.T) {
	h := DefaultHeader().
		WithMessageType(MessageTypeClientAudioOnlyRequest).
		WithMessageTypeSpecificFlags(FlagNegWithSequence).
		WithSerializationType(SerializationNone).
		WithCompressionType(CompressionGZIP).
		WithReservedData([]byte{0xAB})

	b := h.toBytes()
	if len(b) < 4 {
		t.Fatalf("too short: %d", len(b))
	}
	// First byte: version(1)<<4 | 1 (header size in dwords) = 0x11
	if b[0] != 0x11 {
		t.Errorf("byte[0]=%#x", b[0])
	}
	// Second byte: messageType(2)<<4 | flags(3) = 0x23
	if b[1] != 0x23 {
		t.Errorf("byte[1]=%#x", b[1])
	}
	// Third byte: serialization(0)<<4 | compression(1) = 0x01
	if b[2] != 0x01 {
		t.Errorf("byte[2]=%#x", b[2])
	}
	if b[3] != 0xAB {
		t.Errorf("reserved=%#x", b[3])
	}
}

func TestNewAuthHeader(t *testing.T) {
	hdr := NewAuthHeader(AuthConfig{ResourceId: "res", AccessKey: "ak", AppKey: "app"})
	if hdr.Get("X-Api-Resource-Id") != "res" {
		t.Error("ResourceId not set")
	}
	if hdr.Get("X-Api-Access-Key") != "ak" {
		t.Error("AccessKey not set")
	}
	if hdr.Get("X-Api-App-Key") != "app" {
		t.Error("AppKey not set")
	}
	if hdr.Get("X-Api-Request-Id") == "" {
		t.Error("Request-Id should be auto-generated")
	}
}

// ---------- ParseResponse: ServerFullResponse without payload --------------

func TestParseResponse_ServerFullResponseEmpty(t *testing.T) {
	// header (4 bytes): [ver|size=1, mt=ServerFull|flags=0, ser|comp=0, reserved]
	msg := make([]byte, 8)
	msg[0] = byte(ProtocolVersionV1<<4 | 1)
	msg[1] = byte(MessageTypeServerFullResponse<<4) | 0
	msg[2] = 0x00 // serialization=none, compression=none
	msg[3] = 0x00
	// payload size = 0 → payload empty
	binary.BigEndian.PutUint32(msg[4:8], 0)

	r := ParseResponse(msg)
	if r == nil {
		t.Fatal("nil response")
	}
	if r.PayloadSize != 0 {
		t.Errorf("PayloadSize=%d", r.PayloadSize)
	}
}

func TestParseResponse_LastPackageFlag(t *testing.T) {
	// Use FlagNegWithSequence (0b0011) which triggers IsLastPackage.
	// Bit 0 is also set → 4 bytes payloadSequence parsed from payload.
	msg := make([]byte, 4+4+4) // header + seq + payload size
	msg[0] = byte(ProtocolVersionV1<<4 | 1)
	msg[1] = byte(MessageTypeServerFullResponse<<4) | byte(FlagNegWithSequence)
	msg[2] = 0x00
	msg[3] = 0x00
	binary.BigEndian.PutUint32(msg[4:8], 99)
	binary.BigEndian.PutUint32(msg[8:12], 0)

	r := ParseResponse(msg)
	if !r.IsLastPackage {
		t.Error("FlagNegWithSequence should mark last package")
	}
	if r.PayloadSequence != 99 {
		t.Errorf("seq=%d", r.PayloadSequence)
	}
}

// ---------- Factory -------------------------------------------------------

func TestFactory_SupportsAndCreates(t *testing.T) {
	f := NewTranscriberFactory()
	if f == nil {
		t.Fatal("nil factory")
	}
	vendors := f.GetSupportedVendors()
	if len(vendors) == 0 {
		t.Error("no default vendors registered")
	}

	// Must support qcloud (one of the simpler creators).
	if !f.IsVendorSupported(VendorQCloud) {
		t.Error("qcloud should be supported")
	}

	// Unknown vendor
	if f.IsVendorSupported(Vendor("nope")) {
		t.Error("unknown vendor shouldn't be supported")
	}
	if _, err := f.CreateTranscriber(&fakeUnknownOpt{}); err == nil {
		t.Error("unknown vendor should error on CreateTranscriber")
	}

	// Nil config
	if _, err := f.CreateTranscriber(nil); err == nil {
		t.Error("nil config should error")
	}
}

func TestFactory_RegisterCreator_Override(t *testing.T) {
	f := NewTranscriberFactory()
	custom := Vendor("test-custom")
	called := 0
	f.RegisterCreator(custom, func(config TranscriberConfig) (TranscribeService, error) {
		called++
		return nil, fmt.Errorf("stub")
	})
	if !f.IsVendorSupported(custom) {
		t.Error("custom vendor not registered")
	}
	_, _ = f.CreateTranscriber(&stubCustomOpt{v: custom})
	if called != 1 {
		t.Errorf("creator called=%d want 1", called)
	}
}

func TestGetGlobalFactory_Singleton(t *testing.T) {
	a := GetGlobalFactory()
	b := GetGlobalFactory()
	if a != b {
		t.Error("global factory must be singleton")
	}
}

// ---------- GetVendor accessors (recursion-free) --------------------------

func TestGetVendor_Accessors(t *testing.T) {
	pairs := []struct {
		name string
		got  Vendor
		want Vendor
	}{
		{"qcloud", (&QCloudASROption{}).GetVendor(), VendorQCloud},
		{"google", (&GoogleASROption{}).GetVendor(), VendorGoogle},
		{"qiniu", (&QiniuASROption{}).GetVendor(), VendorQiniu},
		{"funasr", (&FunASROption{}).GetVendor(), VendorFunASR},
		{"volcengine", (&VolcengineOption{}).GetVendor(), VendorVolcengine},
		{"volcengine_llm", (&VolcengineLLMOption{}).GetVendor(), VendorVolcengineLLM},
		{"gladia", (&GladiaASROption{}).GetVendor(), VendorGladia},
		{"funasr_realtime", (&FunAsrRealtimeOption{}).GetVendor(), VendorFunASRRealtime},
		{"local", (&LocalASRConfig{}).GetVendor(), VendorLocal},
		{"deepgram", (&DeepgramASROption{}).GetVendor(), VendorDeepgram},
		{"aws", (&AwsASROption{}).GetVendor(), VendorAWS},
		{"baidu", (&BaiduASROption{}).GetVendor(), VendorBaidu},
		{"whisper", (&WhisperASROption{}).GetVendor(), VendorWhisper},
		{"voiceapi", (&VoiceapiASROption{}).GetVendor(), VendorVoiceAPI},
	}
	for _, p := range pairs {
		t.Run(p.name, func(t *testing.T) {
			if p.got != p.want {
				t.Errorf("got %q want %q", p.got, p.want)
			}
		})
	}
}

// ---------- helpers -------------------------------------------------------

type fakeUnknownOpt struct{}

func (*fakeUnknownOpt) GetVendor() Vendor { return Vendor("not-registered") }

type stubCustomOpt struct{ v Vendor }

func (o *stubCustomOpt) GetVendor() Vendor { return o.v }
