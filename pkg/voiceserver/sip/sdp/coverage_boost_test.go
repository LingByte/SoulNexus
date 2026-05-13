package sdp

import (
	"strings"
	"testing"
)

// ---------- crypto helpers --------------------------------------------------

func TestDecodeSDESInline_Errors(t *testing.T) {
	if _, _, err := DecodeSDESInline(""); err == nil {
		t.Error("empty key-params must error")
	}
	if _, _, err := DecodeSDESInline("   "); err == nil {
		t.Error("whitespace key-params must error")
	}
	if _, _, err := DecodeSDESInline("notinline:abc"); err == nil {
		t.Error("missing inline: prefix must error")
	}
	if _, _, err := DecodeSDESInline("inline:!!!not-base64!!!"); err == nil {
		t.Error("bad base64 must error")
	}
	// too-short material (valid base64 but few bytes)
	if _, _, err := DecodeSDESInline("inline:YWJj"); err == nil { // "abc"
		t.Error("short material must error")
	}
}

func TestDecodeSDESInline_LifetimeSuffixIgnored(t *testing.T) {
	// Build a valid 30-byte buffer via FormatCryptoLine, then stick |2^31 onto the inline value.
	masterKey := make([]byte, 16)
	masterSalt := make([]byte, 14)
	for i := range masterKey {
		masterKey[i] = byte(i)
	}
	for i := range masterSalt {
		masterSalt[i] = byte(0xA0 + i)
	}
	line, err := FormatCryptoLine(1, SuiteAESCM128HMACSHA180, masterKey, masterSalt)
	if err != nil {
		t.Fatalf("FormatCryptoLine: %v", err)
	}
	// extract the "inline:..." token from the formatted line and append |2^31
	idx := strings.Index(line, "inline:")
	if idx < 0 {
		t.Fatalf("no inline token in %q", line)
	}
	inline := line[idx:] + "|2^31"
	gotKey, gotSalt, err := DecodeSDESInline(inline)
	if err != nil {
		t.Fatalf("DecodeSDESInline lifetime: %v", err)
	}
	if string(gotKey) != string(masterKey) || string(gotSalt) != string(masterSalt) {
		t.Errorf("round-trip mismatch")
	}
}

func TestFormatCryptoLine_Errors(t *testing.T) {
	// empty suite
	if _, err := FormatCryptoLine(1, "", make([]byte, 16), make([]byte, 14)); err == nil {
		t.Error("empty suite must error")
	}
	// whitespace-only suite
	if _, err := FormatCryptoLine(1, "   ", make([]byte, 16), make([]byte, 14)); err == nil {
		t.Error("whitespace suite must error")
	}
	// wrong key length
	if _, err := FormatCryptoLine(1, SuiteAESCM128HMACSHA180, make([]byte, 4), make([]byte, 14)); err == nil {
		t.Error("short key must error")
	}
	// wrong salt length
	if _, err := FormatCryptoLine(1, SuiteAESCM128HMACSHA180, make([]byte, 16), make([]byte, 4)); err == nil {
		t.Error("short salt must error")
	}
}

func TestPickAESCM128Offer(t *testing.T) {
	offers := []CryptoOffer{
		{Tag: 1, Suite: "OTHER_SUITE", KeyParams: "inline:..."},
		{Tag: 2, Suite: "aes_cm_128_hmac_sha1_80", KeyParams: "inline:abc"}, // case-insensitive match
		{Tag: 3, Suite: "AES_CM_128_HMAC_SHA1_80"},
	}
	got, ok := PickAESCM128Offer(offers)
	if !ok || got.Tag != 2 {
		t.Errorf("picked %+v ok=%v, want tag 2", got, ok)
	}
	// no match
	if _, ok := PickAESCM128Offer([]CryptoOffer{{Suite: "ZZZ"}}); ok {
		t.Error("unexpected match for unknown suite")
	}
	// empty
	if _, ok := PickAESCM128Offer(nil); ok {
		t.Error("unexpected match for nil offers")
	}
}

// ---------- staticPayloadCodec / normalizeCodecName -----------------------

func TestStaticPayloadCodec_AllBranches(t *testing.T) {
	cases := []struct {
		pt   uint8
		want string
		ok   bool
	}{
		{0, "pcmu", true},
		{8, "pcma", true},
		{9, "g722", true},
		{18, "", false},  // dynamic / unknown
		{101, "", false},
	}
	for _, c := range cases {
		got, ok := staticPayloadCodec(c.pt)
		if ok != c.ok {
			t.Errorf("staticPayloadCodec(%d): ok=%v want %v", c.pt, ok, c.ok)
		}
		if ok && got.Name != c.want {
			t.Errorf("staticPayloadCodec(%d): name=%q want %q", c.pt, got.Name, c.want)
		}
	}
}

func TestNormalizeCodecName_AllBranches(t *testing.T) {
	for _, in := range []string{"PCMU", " pcma ", "G722", "OPUS", "PCM", "telephone-event", "weirdcodec"} {
		got := normalizeCodecName(in)
		if got != strings.ToLower(strings.TrimSpace(in)) {
			t.Errorf("normalizeCodecName(%q) = %q", in, got)
		}
	}
}

// ---------- Parse error branches ----------------------------------------

func TestParse_EmptyBody_Errors(t *testing.T) {
	if _, err := Parse(""); err == nil {
		t.Error("empty body must error")
	}
	if _, err := Parse("   \r\n  "); err == nil {
		t.Error("blank body must error")
	}
}

func TestParse_InvalidAudioPort(t *testing.T) {
	body := strings.Join([]string{
		"v=0",
		"o=- 0 0 IN IP4 127.0.0.1",
		"s=-",
		"c=IN IP4 127.0.0.1",
		"t=0 0",
		"m=audio NOTANUMBER RTP/AVP 0",
	}, "\r\n")
	if _, err := Parse(body); err == nil {
		t.Error("non-numeric m=audio port must error")
	}
}

func TestParse_NonAudioMSection_DoesNotPickRtpmap(t *testing.T) {
	// confirms the inAudioSection flag flips off for video sections
	body := strings.Join([]string{
		"v=0",
		"o=- 0 0 IN IP4 127.0.0.1",
		"s=-",
		"c=IN IP4 127.0.0.1",
		"t=0 0",
		"m=audio 12000 RTP/AVP 0",
		"a=rtpmap:0 PCMU/8000",
		"m=video 12002 RTP/AVP 96",
		"a=rtpmap:96 H264/90000", // must not be added to audio codec list
	}, "\r\n")
	info, err := Parse(body)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	for _, c := range info.Codecs {
		if c.Name == "H264" || c.Name == "h264" {
			t.Fatalf("H264 leaked into audio codec list: %+v", info.Codecs)
		}
	}
}

// ---------- GenerateWithProtoExtras --------------------------------------

func TestGenerateWithProtoExtras_AppendsExtraLines(t *testing.T) {
	codecs := []Codec{{Name: "pcmu", PayloadType: 0, ClockRate: 8000, Channels: 1}}
	out := GenerateWithProtoExtras("127.0.0.1", 5004, "RTP/AVP", codecs, []string{"a=sendrecv", "a=ptime:20"})
	if !strings.Contains(out, "a=sendrecv") || !strings.Contains(out, "a=ptime:20") {
		t.Errorf("extras not embedded:\n%s", out)
	}
	if !strings.Contains(out, "m=audio 5004 RTP/AVP 0") {
		t.Errorf("m-line missing:\n%s", out)
	}
}
