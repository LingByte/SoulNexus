package sdp

import (
	"strings"
	"testing"
)

func TestParse_Basic(t *testing.T) {
	sdpBody := strings.Join([]string{
		"v=0",
		"o=- 123456 123456 IN IP4 192.168.1.100",
		"s=Session",
		"c=IN IP4 192.168.1.100",
		"t=0 0",
		"m=audio 49170 RTP/AVP 0",
		"a=rtpmap:0 PCMU/8000",
	}, "\r\n")

	info, err := Parse(sdpBody)
	if err != nil {
		t.Fatal(err)
	}
	if info.IP != "192.168.1.100" || info.Port != 49170 {
		t.Fatalf("ip/port: %+v", info)
	}
	if len(info.Codecs) != 1 || info.Codecs[0].PayloadType != 0 || info.Codecs[0].Name != "pcmu" {
		t.Fatalf("codec: %+v", info.Codecs)
	}
}

func TestParse_StaticPCMU_PlusTelephoneEvent(t *testing.T) {
	sdpBody := strings.Join([]string{
		"v=0",
		"o=- 1 1 IN IP4 10.0.0.2",
		"s=-",
		"c=IN IP4 10.0.0.2",
		"t=0 0",
		"m=audio 8000 RTP/AVP 0 101",
		"a=rtpmap:101 telephone-event/8000",
		"a=fmtp:101 0-15",
	}, "\r\n")

	info, err := Parse(sdpBody)
	if err != nil {
		t.Fatal(err)
	}
	if len(info.Codecs) != 2 {
		t.Fatalf("want 2 codecs, got %d", len(info.Codecs))
	}
	if info.Codecs[0].PayloadType != 0 || info.Codecs[0].Name != "pcmu" {
		t.Fatalf("first: %#v", info.Codecs[0])
	}
	if info.Codecs[1].Name != "telephone-event" {
		t.Fatalf("second: %#v", info.Codecs[1])
	}
}

func TestParse_SAVP_WithCrypto(t *testing.T) {
	sdpBody := strings.Join([]string{
		"v=0",
		"o=- 1 1 IN IP4 10.0.0.2",
		"s=-",
		"c=IN IP4 10.0.0.2",
		"t=0 0",
		"m=audio 8000 RTP/SAVP 0",
		"a=rtpmap:0 PCMU/8000",
		"a=crypto:1 AES_CM_128_HMAC_SHA1_80 inline:ABCDEFGHIJKLMNOPQRSTUVWX/Y",
	}, "\r\n")
	info, err := Parse(sdpBody)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(strings.ToUpper(info.Proto), "SAVP") {
		t.Fatalf("proto: %q", info.Proto)
	}
	if len(info.CryptoOffers) != 1 || info.CryptoOffers[0].Tag != 1 {
		t.Fatalf("crypto: %+v", info.CryptoOffers)
	}
}

func TestCrypto_FormatDecodeRoundTrip(t *testing.T) {
	key := make([]byte, 16)
	salt := make([]byte, 14)
	for i := range key {
		key[i] = byte(i + 1)
	}
	for i := range salt {
		salt[i] = byte(i + 10)
	}
	line, err := FormatCryptoLine(7, SuiteAESCM128HMACSHA180, key, salt)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(line, "a=crypto:7 AES_CM_128_HMAC_SHA1_80 inline:") {
		t.Fatalf("line: %s", line)
	}
	idx := strings.Index(line, "inline:")
	if idx < 0 {
		t.Fatalf("no inline in %s", line)
	}
	co := CryptoOffer{Tag: 7, Suite: SuiteAESCM128HMACSHA180, KeyParams: strings.TrimSpace(line[idx:])}
	k2, s2, err := DecodeSDESInline(co.KeyParams)
	if err != nil {
		t.Fatal(err)
	}
	if len(k2) != len(key) || len(s2) != len(salt) {
		t.Fatalf("len mismatch")
	}
	for i := range key {
		if k2[i] != key[i] {
			t.Fatalf("key byte %d", i)
		}
	}
	for i := range salt {
		if s2[i] != salt[i] {
			t.Fatalf("salt byte %d", i)
		}
	}
}

func TestGenerate_RoundTrip(t *testing.T) {
	codecs := []Codec{
		{PayloadType: 0, Name: "pcmu", ClockRate: 8000},
		{PayloadType: 8, Name: "pcma", ClockRate: 8000},
	}
	body := Generate("127.0.0.1", 5004, codecs)
	info, err := Parse(body)
	if err != nil {
		t.Fatal(err)
	}
	if info.IP != "127.0.0.1" || info.Port != 5004 || len(info.Codecs) == 0 {
		t.Fatalf("%+v", info)
	}
}

func TestGenerate_PayloadOrderPreserved(t *testing.T) {
	codecs := []Codec{
		{PayloadType: 8, Name: "pcma", ClockRate: 8000},
		{PayloadType: 111, Name: "opus", ClockRate: 48000, Channels: 1},
		{PayloadType: 0, Name: "pcmu", ClockRate: 8000},
		{PayloadType: 101, Name: "telephone-event", ClockRate: 8000},
	}
	body := Generate("127.0.0.1", 5004, codecs)
	info, err := Parse(body)
	if err != nil {
		t.Fatal(err)
	}
	if len(info.Codecs) != len(codecs) {
		t.Fatalf("count got=%d want=%d", len(info.Codecs), len(codecs))
	}
	for i := range codecs {
		if info.Codecs[i].PayloadType != codecs[i].PayloadType || info.Codecs[i].Name != codecs[i].Name {
			t.Fatalf("[%d] got=%#v want=%#v", i, info.Codecs[i], codecs[i])
		}
	}
}

func TestPickTelephoneEventFromOffer(t *testing.T) {
	offer := []Codec{
		{Name: "opus", ClockRate: 48000, PayloadType: 111},
		{Name: "telephone-event", ClockRate: 8000, PayloadType: 101},
		{Name: "telephone-event", ClockRate: 48000, PayloadType: 112},
	}
	if c, ok := PickTelephoneEventFromOffer(offer, 48000); !ok || c.PayloadType != 112 {
		t.Fatalf("got %#v ok=%v", c, ok)
	}
}

func TestNormalizeBody(t *testing.T) {
	s := NormalizeBody("  a\r\nb\r  ")
	if s != "a\nb" {
		t.Fatalf("%q", s)
	}
}

func TestParse_EmptyBody(t *testing.T) {
	if _, err := Parse(""); err == nil {
		t.Fatal("expected error")
	}
}

func TestGenerate_DefaultPortAndEmptyIP(t *testing.T) {
	body := GenerateWithProto("", 0, "", []Codec{{PayloadType: 0, Name: "pcmu", ClockRate: 8000}})
	if !strings.Contains(body, "m=audio 49172") {
		t.Fatalf("default port: %s", body)
	}
	if !strings.Contains(body, "c=IN IP4 127.0.0.1") {
		t.Fatalf("default ip: %s", body)
	}
}

func TestGenerateWithProto_AVPF(t *testing.T) {
	body := GenerateWithProto("10.0.0.1", 9000, "RTP/AVPF", []Codec{{PayloadType: 111, Name: "opus", ClockRate: 48000, Channels: 2}})
	info, err := Parse(body)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(strings.ToUpper(info.Proto), "AVPF") {
		t.Fatalf("proto %q", info.Proto)
	}
}

func TestPresets_Length(t *testing.T) {
	if len(DefaultOutboundOfferCodecs()) < 4 {
		t.Fatal("default outbound codecs")
	}
	if len(TransferAgentBridgeOfferCodecs()) < 2 {
		t.Fatal("transfer codecs")
	}
}

func TestPickTelephoneEventFromOffer_Fallback8000(t *testing.T) {
	offer := []Codec{{Name: "opus", ClockRate: 48000, PayloadType: 111}, {Name: "telephone-event", ClockRate: 8000, PayloadType: 101}}
	c, ok := PickTelephoneEventFromOffer(offer, 9999)
	if !ok || c.PayloadType != 101 {
		t.Fatalf("got %#v ok=%v", c, ok)
	}
}
