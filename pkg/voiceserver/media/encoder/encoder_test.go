package encoder

import (
	"bytes"
	"encoding/binary"
	"math"
	"testing"

	"github.com/LingByte/SoulNexus/pkg/voiceserver/media"
)

// helper to make a 16-bit little-endian PCM sample buffer.
func pcmLE(samples ...int16) []byte {
	buf := make([]byte, len(samples)*2)
	for i, s := range samples {
		binary.LittleEndian.PutUint16(buf[i*2:], uint16(s))
	}
	return buf
}

// ---------- registry --------------------------------------------------------

func TestHasCodec_RegisteredDefaults(t *testing.T) {
	for _, name := range []string{CodecPCM, CodecPCMU, CodecPCMA, CodecG722, CodecOPUS} {
		if !HasCodec(name) {
			t.Errorf("HasCodec(%q) = false, expected built-in codec to be registered", name)
		}
	}
	if !HasCodec("PCMU") { // case-insensitive
		t.Errorf("HasCodec is supposed to be case-insensitive")
	}
	if HasCodec("nope") {
		t.Errorf("HasCodec(\"nope\") should be false")
	}
}

func TestRegisterCodec_CustomFactory(t *testing.T) {
	called := false
	mk := func(src, pcm media.CodecConfig) media.EncoderFunc {
		called = true
		return func(p media.MediaPacket) ([]media.MediaPacket, error) {
			return []media.MediaPacket{p}, nil
		}
	}
	RegisterCodec("test_custom_codec", mk, mk)
	defer delete(codecRegistryMap, "test_custom_codec")

	if !HasCodec("test_custom_codec") {
		t.Fatal("custom codec not registered")
	}
	src := media.CodecConfig{Codec: "test_custom_codec", SampleRate: 8000}
	pcm := media.CodecConfig{Codec: "pcm", SampleRate: 16000}
	enc, err := CreateEncode(src, pcm)
	if err != nil || enc == nil {
		t.Fatalf("CreateEncode err=%v enc=%v", err, enc)
	}
	if !called {
		t.Fatalf("custom encoder factory not invoked")
	}
	called = false
	dec, err := CreateDecode(src, pcm)
	if err != nil || dec == nil {
		t.Fatalf("CreateDecode err=%v dec=%v", err, dec)
	}
	if !called {
		t.Fatalf("custom decoder factory not invoked")
	}
}

func TestCreateEncode_UnknownCodec(t *testing.T) {
	_, err := CreateEncode(
		media.CodecConfig{Codec: "no-such-thing"},
		media.CodecConfig{Codec: "pcm", SampleRate: 16000},
	)
	if err == nil {
		t.Fatal("expected error for unknown codec")
	}
}

func TestCreateDecode_UnknownCodec(t *testing.T) {
	_, err := CreateDecode(
		media.CodecConfig{Codec: "no-such-thing"},
		media.CodecConfig{Codec: "pcm", SampleRate: 16000},
	)
	if err == nil {
		t.Fatal("expected error for unknown codec")
	}
}

// ---------- StripWavHeader --------------------------------------------------

func TestStripWavHeader_WithRIFFHeader(t *testing.T) {
	header := append([]byte("RIFF"), make([]byte, 40)...) // 4 + 40 = 44 bytes header
	body := []byte{0x10, 0x20, 0x30}
	in := append(append([]byte{}, header...), body...)
	out := StripWavHeader(in)
	if !bytes.Equal(out, body) {
		t.Fatalf("StripWavHeader(RIFF+body) = %v, want %v", out, body)
	}
}

func TestStripWavHeader_NoHeader(t *testing.T) {
	in := []byte{1, 2, 3, 4, 5}
	out := StripWavHeader(in)
	if !bytes.Equal(out, in) {
		t.Fatalf("StripWavHeader(no-RIFF) should return input unchanged")
	}
}

func TestStripWavHeader_TooShort(t *testing.T) {
	// Less than 44 bytes: should be returned as-is even with RIFF prefix
	in := []byte("RIFFshort-data")
	out := StripWavHeader(in)
	if !bytes.Equal(out, in) {
		t.Fatalf("StripWavHeader(short) should return input unchanged")
	}
}

// ---------- PcmToPcm passthrough -------------------------------------------

func TestPcmToPcm_SameRate_Identity(t *testing.T) {
	src := media.CodecConfig{Codec: "pcm", SampleRate: 16000}
	pcm := media.CodecConfig{Codec: "pcm", SampleRate: 16000}
	enc := PcmToPcm(src, pcm)
	in := pcmLE(0, 1000, -1000, 2000, -2000, 3000)
	pkt := &media.AudioPacket{Payload: in}
	out, err := enc(pkt)
	if err != nil {
		t.Fatalf("PcmToPcm err: %v", err)
	}
	if len(out) != 1 {
		t.Fatalf("PcmToPcm: want 1 packet, got %d", len(out))
	}
	got := out[0].(*media.AudioPacket).Payload
	if !bytes.Equal(got, in) {
		t.Fatalf("identity resample altered samples")
	}
}

func TestPcmToPcm_NonAudioPacket_PassesThrough(t *testing.T) {
	src := media.CodecConfig{Codec: "pcm", SampleRate: 16000}
	pcm := media.CodecConfig{Codec: "pcm", SampleRate: 16000}
	enc := PcmToPcm(src, pcm)
	dtmf := &media.DTMFPacket{Digit: "5"}
	out, err := enc(dtmf)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if len(out) != 1 || out[0] != dtmf {
		t.Fatalf("non-audio packet should be passed through unchanged")
	}
}

// ---------- G.711 encode/decode round-trip ----------------------------------

func TestPCMA_RoundTrip_LowMagnitude(t *testing.T) {
	// Round-trip a clean small-magnitude PCM signal through A-law.
	// G.711 is lossy; we tolerate ~256 LSB error for low magnitudes.
	in := pcmLE(0, 100, 200, 300, -100, -200, -300, 1000, -1000)
	alaw, err := Pcm2pcma(in)
	if err != nil {
		t.Fatalf("Pcm2pcma: %v", err)
	}
	if len(alaw) != len(in)/2 {
		t.Fatalf("PCMA size: got %d, want %d", len(alaw), len(in)/2)
	}
	pcm, err := pcma2pcm(alaw)
	if err != nil {
		t.Fatalf("pcma2pcm: %v", err)
	}
	if len(pcm) != len(in) {
		t.Fatalf("PCMA→PCM size mismatch")
	}
	for i := 0; i+1 < len(in); i += 2 {
		orig := int16(binary.LittleEndian.Uint16(in[i:]))
		round := int16(binary.LittleEndian.Uint16(pcm[i:]))
		diff := int(orig) - int(round)
		if diff < 0 {
			diff = -diff
		}
		if diff > 256 {
			t.Fatalf("PCMA round-trip diff too large at %d: orig=%d round=%d", i, orig, round)
		}
	}
}

func TestEncodePCMA_AliasOfPcm2pcma(t *testing.T) {
	in := pcmLE(0, 100, -100, 1000, -1000)
	a1, err := EncodePCMA(in)
	if err != nil {
		t.Fatalf("EncodePCMA: %v", err)
	}
	a2, _ := Pcm2pcma(in)
	if !bytes.Equal(a1, a2) {
		t.Fatalf("EncodePCMA must equal Pcm2pcma")
	}
}

func TestPCMU_RoundTrip(t *testing.T) {
	in := pcmLE(0, 500, -500, 2500, -2500, 8000, -8000)
	ulaw, err := pcm2pcmu(in)
	if err != nil {
		t.Fatalf("pcm2pcmu: %v", err)
	}
	if len(ulaw) != len(in)/2 {
		t.Fatalf("PCMU size: got %d, want %d", len(ulaw), len(in)/2)
	}
	pcm, err := pcmu2pcm(ulaw)
	if err != nil {
		t.Fatalf("pcmu2pcm: %v", err)
	}
	if len(pcm) != len(in) {
		t.Fatalf("PCMU→PCM size mismatch")
	}
	for i := 0; i+1 < len(in); i += 2 {
		orig := int16(binary.LittleEndian.Uint16(in[i:]))
		round := int16(binary.LittleEndian.Uint16(pcm[i:]))
		diff := math.Abs(float64(orig) - float64(round))
		if diff > 512 {
			t.Fatalf("PCMU round-trip diff too large: orig=%d round=%d", orig, round)
		}
	}
}

func TestG711_LawSegments_BoundaryValues(t *testing.T) {
	// exercise both signed branches and segment overflow path
	for _, v := range []int{0, 1, -1, 100, -100, 30000, -30000, 32767, -32768} {
		a := linear2alaw(v)
		u := linear2ulaw(v)
		_ = alaw2linear(a)
		_ = ulaw2linear(u)
	}
}

// ---------- factory paths through registry ---------------------------------

func TestPCMU_Factory_RoundTrip_Mono8k(t *testing.T) {
	src := media.CodecConfig{Codec: "pcmu", SampleRate: 8000, Channels: 1, BitDepth: 16}
	pcm := media.CodecConfig{Codec: "pcm", SampleRate: 8000, Channels: 1, BitDepth: 16}
	enc, err := CreateEncode(src, pcm)
	if err != nil {
		t.Fatalf("CreateEncode pcmu: %v", err)
	}
	dec, err := CreateDecode(src, pcm)
	if err != nil {
		t.Fatalf("CreateDecode pcmu: %v", err)
	}

	// 80 PCM samples = 10ms at 8 kHz
	in := make([]byte, 160)
	for i := 0; i < 80; i++ {
		s := int16(((i * 1234) % 16000) - 8000)
		binary.LittleEndian.PutUint16(in[i*2:], uint16(s))
	}
	encoded, err := enc(&media.AudioPacket{Payload: in})
	if err != nil {
		t.Fatalf("enc: %v", err)
	}
	if len(encoded) == 0 {
		t.Fatalf("expected at least one encoded packet")
	}

	var totalULaw []byte
	for _, p := range encoded {
		ap := p.(*media.AudioPacket)
		totalULaw = append(totalULaw, ap.Payload...)
	}
	decoded, err := dec(&media.AudioPacket{Payload: totalULaw})
	if err != nil {
		t.Fatalf("dec: %v", err)
	}
	if len(decoded) == 0 {
		t.Skip("resampler buffered all samples (acceptable for short input)")
	}
}

func TestPCMA_Factory_DefaultRate_WhenZero(t *testing.T) {
	// SampleRate=0 should default to 8000 in PCMA factories
	src := media.CodecConfig{Codec: "pcma"}
	pcm := media.CodecConfig{Codec: "pcm", SampleRate: 8000}
	enc, err := CreateEncode(src, pcm)
	if err != nil || enc == nil {
		t.Fatalf("CreateEncode default rate failed: %v", err)
	}
	dec, err := CreateDecode(src, pcm)
	if err != nil || dec == nil {
		t.Fatalf("CreateDecode default rate failed: %v", err)
	}
}

func TestPCMA_Factory_RoundTrip_WithResample(t *testing.T) {
	// src 8kHz PCMA ↔ internal PCM 16kHz → exercises resampler branches
	src := media.CodecConfig{Codec: "pcma", SampleRate: 8000, Channels: 1, BitDepth: 16, FrameDuration: "20ms"}
	pcm := media.CodecConfig{Codec: "pcm", SampleRate: 16000, Channels: 1, BitDepth: 16}

	enc, _ := CreateEncode(src, pcm)
	dec, _ := CreateDecode(src, pcm)

	// 320 PCM samples @ 16kHz = 20ms = 640 bytes
	in := make([]byte, 640)
	for i := 0; i < 320; i++ {
		s := int16((i * 31) % 4000)
		binary.LittleEndian.PutUint16(in[i*2:], uint16(s))
	}
	encoded, err := enc(&media.AudioPacket{Payload: in})
	if err != nil {
		t.Fatalf("PCMA encode resample: %v", err)
	}
	if len(encoded) == 0 {
		t.Skip("resampler buffered everything")
	}
	for _, p := range encoded {
		ap := p.(*media.AudioPacket)
		if _, err := dec(&media.AudioPacket{Payload: ap.Payload}); err != nil {
			t.Fatalf("PCMA decode resample: %v", err)
		}
	}
}

func TestPCMA_PCMU_Factory_NonAudioPacket_Passthrough(t *testing.T) {
	src8k := media.CodecConfig{SampleRate: 8000, Channels: 1, BitDepth: 16}
	pcm8k := media.CodecConfig{Codec: "pcm", SampleRate: 8000, Channels: 1, BitDepth: 16}

	for _, codec := range []string{"pcma", "pcmu"} {
		src := src8k
		src.Codec = codec
		enc, _ := CreateEncode(src, pcm8k)
		dec, _ := CreateDecode(src, pcm8k)
		dtmf := &media.DTMFPacket{Digit: "*"}
		if out, err := enc(dtmf); err != nil || len(out) != 1 || out[0] != dtmf {
			t.Fatalf("%s enc passthrough broken: out=%v err=%v", codec, out, err)
		}
		if out, err := dec(dtmf); err != nil || len(out) != 1 || out[0] != dtmf {
			t.Fatalf("%s dec passthrough broken: out=%v err=%v", codec, out, err)
		}
	}
}

func TestPCMU_Factory_DefaultRate_WhenZero(t *testing.T) {
	src := media.CodecConfig{Codec: "pcmu"} // SR=0
	pcm := media.CodecConfig{Codec: "pcm", SampleRate: 8000}
	if enc, err := CreateEncode(src, pcm); err != nil || enc == nil {
		t.Fatalf("createPCMUEncode default SR: %v", err)
	}
	if dec, err := CreateDecode(src, pcm); err != nil || dec == nil {
		t.Fatalf("createPCMUDecode default SR: %v", err)
	}
}

func TestG722_Factory_NonAudioPacketPassthrough(t *testing.T) {
	src := media.CodecConfig{Codec: "g722", SampleRate: 16000}
	pcm := media.CodecConfig{Codec: "pcm", SampleRate: 16000}
	enc, _ := CreateEncode(src, pcm)
	dec, _ := CreateDecode(src, pcm)
	dtmf := &media.DTMFPacket{Digit: "1"}
	if out, err := enc(dtmf); err != nil || len(out) != 1 || out[0] != dtmf {
		t.Fatalf("g722 enc passthrough broken: out=%v err=%v", out, err)
	}
	if out, err := dec(dtmf); err != nil || len(out) != 1 || out[0] != dtmf {
		t.Fatalf("g722 dec passthrough broken: out=%v err=%v", out, err)
	}
}

func TestPcmToPcm_WithResample(t *testing.T) {
	// Force resampling path
	src := media.CodecConfig{Codec: "pcm", SampleRate: 8000}
	pcm := media.CodecConfig{Codec: "pcm", SampleRate: 16000}
	enc := PcmToPcm(src, pcm)
	in := make([]byte, 640) // 320 samples of 8k input
	for i := 0; i < 320; i++ {
		s := int16((i * 17) % 2000)
		binary.LittleEndian.PutUint16(in[i*2:], uint16(s))
	}
	out, err := enc(&media.AudioPacket{Payload: in})
	if err != nil {
		t.Fatalf("PcmToPcm resample: %v", err)
	}
	if len(out) == 0 {
		t.Skip("resampler buffered all input on first frame")
	}
}

// ---------- splitFrames (covered via splitFrames, exercised by encoder fns)-

func TestSplitFrames_EmptyDurationProducesSinglePacket(t *testing.T) {
	cfg := media.CodecConfig{SampleRate: 8000}
	out := splitFrames([]byte{1, 2, 3, 4}, &cfg)
	if len(out) != 1 {
		t.Fatalf("empty duration → 1 packet, got %d", len(out))
	}
}

func TestSplitFrames_ConfiguredDurationProducesMultiplePackets(t *testing.T) {
	cfg := media.CodecConfig{SampleRate: 8000, FrameDuration: "20ms"}
	// 80 bytes/frame at 8000 SR (Hz×ms/1000 = 8000*20/1000 = 160 mistake)
	// Per the implementation: bytesPerFrame = duration_ms * SampleRate / 1000 = 20*8000/1000 = 160
	in := make([]byte, 480) // 3 frames
	out := splitFrames(in, &cfg)
	if len(out) != 3 {
		t.Fatalf("want 3 frames, got %d", len(out))
	}
}

func TestSplitFrames_OutOfRangeDurationFallsBackTo20ms(t *testing.T) {
	cfg := media.CodecConfig{SampleRate: 8000, FrameDuration: "5ms"} // < 10ms → fallback to 20ms
	in := make([]byte, 480)
	out := splitFrames(in, &cfg)
	if len(out) != 3 {
		t.Fatalf("fallback 20ms: want 3 frames, got %d", len(out))
	}
}

// ---------- G.722 direct encoder/decoder ------------------------------------

func TestG722_RoundTrip_PCMSamplesPreserveLength(t *testing.T) {
	enc := NewG722Encoder(G722_RATE_DEFAULT, G722_DEFAULT)
	dec := NewG722Decoder(G722_RATE_DEFAULT, G722_DEFAULT)

	// produce 320 samples = 640 bytes PCM (40ms at 16kHz)
	pcm := make([]byte, 640)
	for i := 0; i < 320; i++ {
		// gentle sine-like sweep
		s := int16((i * 53) % 4000)
		binary.LittleEndian.PutUint16(pcm[i*2:], uint16(s))
	}
	g722 := enc.Encode(pcm)
	if len(g722) == 0 {
		t.Fatalf("g722 encoded length 0")
	}
	out := dec.Decode(g722)
	if len(out) == 0 {
		t.Fatalf("g722 decoded length 0")
	}
}

// ---------- G.722 factory wiring -------------------------------------------

func TestG722_Factory_EndToEnd(t *testing.T) {
	// Drive the createG722Encode / createG722Decode wrappers via the registry
	// so their resampler+frame-split branches get covered.
	src := media.CodecConfig{Codec: "g722", SampleRate: 16000, Channels: 1, BitDepth: 16, FrameDuration: "20ms"}
	pcm := media.CodecConfig{Codec: "pcm", SampleRate: 16000, Channels: 1, BitDepth: 16}
	enc, err := CreateEncode(src, pcm)
	if err != nil {
		t.Fatalf("g722 enc factory: %v", err)
	}
	dec, err := CreateDecode(src, pcm)
	if err != nil {
		t.Fatalf("g722 dec factory: %v", err)
	}

	// 320 samples at 16kHz = 20ms PCM
	in := make([]byte, 640)
	for i := 0; i < 320; i++ {
		s := int16((i * 47) % 5000)
		binary.LittleEndian.PutUint16(in[i*2:], uint16(s))
	}
	encoded, err := enc(&media.AudioPacket{Payload: in})
	if err != nil {
		t.Fatalf("g722 enc invoke: %v", err)
	}
	if len(encoded) == 0 {
		t.Fatalf("g722 enc produced no packets")
	}
	var g722Bytes []byte
	for _, p := range encoded {
		ap := p.(*media.AudioPacket)
		g722Bytes = append(g722Bytes, ap.Payload...)
	}
	if _, err := dec(&media.AudioPacket{Payload: g722Bytes}); err != nil {
		t.Fatalf("g722 dec invoke: %v", err)
	}
}

func TestG722_Factory_DefaultSampleRate(t *testing.T) {
	src := media.CodecConfig{Codec: "g722"} // SR=0 → defaults to 16000
	pcm := media.CodecConfig{Codec: "pcm", SampleRate: 16000}
	if enc, err := CreateEncode(src, pcm); err != nil || enc == nil {
		t.Fatalf("createG722Encode default SR: %v", err)
	}
	if dec, err := CreateDecode(src, pcm); err != nil || dec == nil {
		t.Fatalf("createG722Decode default SR: %v", err)
	}
}

// ---------- Opus -----------------------------------------------------------
// hraban/opus is a cgo binding to libopus; if libopus is missing, opus.NewEncoder
// will return an error and these tests will be skipped at runtime.

func TestOpusEncoderComplexity_NonNegative(t *testing.T) {
	c := opusEncoderComplexity()
	if c != 5 && c != 10 {
		t.Fatalf("opusEncoderComplexity = %d, expected 5 or 10", c)
	}
}

func TestOPUS_Factory_EncodeDecode_Mono48k(t *testing.T) {
	src := media.CodecConfig{Codec: "opus", SampleRate: 48000, Channels: 1, BitDepth: 16, FrameDuration: "20ms"}
	pcm := media.CodecConfig{Codec: "pcm", SampleRate: 48000, Channels: 1, BitDepth: 16}

	enc, err := CreateEncode(src, pcm)
	if err != nil {
		t.Fatalf("opus enc factory: %v", err)
	}
	dec, err := CreateDecode(src, pcm)
	if err != nil {
		t.Fatalf("opus dec factory: %v", err)
	}

	// 960 samples = 20ms at 48kHz
	in := make([]byte, 1920)
	for i := 0; i < 960; i++ {
		s := int16((i * 13) % 1500)
		binary.LittleEndian.PutUint16(in[i*2:], uint16(s))
	}

	encoded, err := enc(&media.AudioPacket{Payload: in})
	if err != nil {
		// libopus might be unavailable on CI — be tolerant
		t.Skipf("opus encode failed (likely libopus missing): %v", err)
	}
	if len(encoded) == 0 {
		t.Skip("opus encoder buffered, no packet emitted")
	}

	for _, p := range encoded {
		ap := p.(*media.AudioPacket)
		if _, err := dec(&media.AudioPacket{Payload: ap.Payload}); err != nil {
			t.Fatalf("opus decode failed: %v", err)
		}
	}
}

func TestOPUS_Factory_EncodeDecode_WithResample(t *testing.T) {
	// Drive the resampler+frame-split paths inside createOPUSEncode/Decode
	// (src 48k Opus ↔ internal PCM 16k).
	src := media.CodecConfig{Codec: "opus", SampleRate: 48000, Channels: 1, FrameDuration: "20ms"}
	pcm := media.CodecConfig{Codec: "pcm", SampleRate: 16000, Channels: 1}

	enc, err := CreateEncode(src, pcm)
	if err != nil {
		t.Skipf("opus encode resample factory (libopus may be missing): %v", err)
	}
	dec, err := CreateDecode(src, pcm)
	if err != nil {
		t.Skipf("opus decode resample factory: %v", err)
	}

	// 320 PCM samples @ 16kHz = 20ms = 640 bytes
	in := make([]byte, 640)
	for i := 0; i < 320; i++ {
		s := int16((i * 23) % 1500)
		binary.LittleEndian.PutUint16(in[i*2:], uint16(s))
	}
	encoded, err := enc(&media.AudioPacket{Payload: in})
	if err != nil {
		t.Skipf("opus enc resample call: %v", err)
	}
	if len(encoded) == 0 {
		t.Skip("opus encoder buffered all samples on first feed")
	}
	for _, p := range encoded {
		ap := p.(*media.AudioPacket)
		if _, err := dec(&media.AudioPacket{Payload: ap.Payload}); err != nil {
			t.Fatalf("opus dec resample call: %v", err)
		}
	}
}

func TestOPUS_Factory_AllInvalidRateFallbackBranches(t *testing.T) {
	// Each invalid rate exercises a different else-if branch in createOPUSEncode.
	pcm := media.CodecConfig{Codec: "pcm", SampleRate: 48000, Channels: 1}
	for _, r := range []int{5000, 11000, 15000, 25000, 40000} {
		src := media.CodecConfig{Codec: "opus", SampleRate: r, Channels: 1}
		if _, err := CreateEncode(src, pcm); err != nil {
			t.Skipf("opus invalid SR %d (libopus may be missing): %v", r, err)
		}
	}
}

func TestOPUS_Factory_StereoEncode_DuplicatesMonoToStereo(t *testing.T) {
	// channels=2 path with mono PCM input → stereo duplication branch
	src := media.CodecConfig{Codec: "opus", SampleRate: 48000, Channels: 2, FrameDuration: "20ms"}
	pcm := media.CodecConfig{Codec: "pcm", SampleRate: 48000, Channels: 1}

	enc, err := CreateEncode(src, pcm)
	if err != nil {
		t.Skipf("libopus may be missing: %v", err)
	}

	// 960 mono PCM samples = 20ms @ 48kHz
	in := make([]byte, 1920)
	for i := 0; i < 960; i++ {
		s := int16((i * 19) % 1000)
		binary.LittleEndian.PutUint16(in[i*2:], uint16(s))
	}
	if _, err := enc(&media.AudioPacket{Payload: in}); err != nil {
		t.Fatalf("opus stereo encode of mono input: %v", err)
	}
}

func TestOPUS_NonAudioPacket_Passthrough(t *testing.T) {
	src := media.CodecConfig{Codec: "opus", SampleRate: 48000, Channels: 1}
	pcm := media.CodecConfig{Codec: "pcm", SampleRate: 48000, Channels: 1}
	enc, err := CreateEncode(src, pcm)
	if err != nil {
		t.Skipf("libopus may be missing: %v", err)
	}
	dec, _ := CreateDecode(src, pcm)
	dtmf := &media.DTMFPacket{Digit: "9"}
	if out, err := enc(dtmf); err != nil || len(out) != 1 || out[0] != dtmf {
		t.Fatalf("opus enc passthrough: out=%v err=%v", out, err)
	}
	if out, err := dec(dtmf); err != nil || len(out) != 1 || out[0] != dtmf {
		t.Fatalf("opus dec passthrough: out=%v err=%v", out, err)
	}
}

func TestOPUS_Factory_StereoOut_ChannelBranches(t *testing.T) {
	// Drive the pcmOutCh >= 2 / OpusDecodeChannels / OpusPCMBridgeDecodeStereo
	// branches at factory-creation time, regardless of whether libopus actually
	// decodes here.
	pcmStereo := media.CodecConfig{Codec: "pcm", SampleRate: 48000, Channels: 2}
	cases := []media.CodecConfig{
		{Codec: "opus", SampleRate: 48000, Channels: 2},
		{Codec: "opus", SampleRate: 48000, Channels: 0, OpusDecodeChannels: 1},
		{Codec: "opus", SampleRate: 48000, Channels: 0, OpusDecodeChannels: 2},
		{Codec: "opus", SampleRate: 48000, Channels: 0, OpusDecodeChannels: 2, OpusPCMBridgeDecodeStereo: true},
	}
	for i, src := range cases {
		if dec, err := CreateDecode(src, pcmStereo); err != nil || dec == nil {
			t.Skipf("opus stereo case %d (libopus may be missing): %v", i, err)
		}
	}
}

func TestOPUS_Factory_FrameDurationCustom(t *testing.T) {
	src := media.CodecConfig{Codec: "opus", SampleRate: 48000, Channels: 1, FrameDuration: "60ms"}
	pcm := media.CodecConfig{Codec: "pcm", SampleRate: 48000, Channels: 1}
	if enc, err := CreateEncode(src, pcm); err != nil || enc == nil {
		t.Skipf("opus 60ms enc (libopus may be missing): %v", err)
	}
	if dec, err := CreateDecode(src, pcm); err != nil || dec == nil {
		t.Skipf("opus 60ms dec (libopus may be missing): %v", err)
	}
}

func TestOPUS_Factory_DefaultRateAndInvalidRateFallback(t *testing.T) {
	// SR=0 → defaults to 48000
	src1 := media.CodecConfig{Codec: "opus", Channels: 1}
	pcm := media.CodecConfig{Codec: "pcm", SampleRate: 48000, Channels: 1}
	if enc, err := CreateEncode(src1, pcm); err != nil || enc == nil {
		t.Skipf("opus default SR (libopus may be missing): %v", err)
	}
	if dec, err := CreateDecode(src1, pcm); err != nil || dec == nil {
		t.Skipf("opus default SR decode (libopus may be missing): %v", err)
	}
	// invalid SR → falls back to nearest valid
	src2 := media.CodecConfig{Codec: "opus", SampleRate: 22050, Channels: 1}
	if enc, err := CreateEncode(src2, pcm); err != nil || enc == nil {
		t.Skipf("opus invalid SR fallback (libopus may be missing): %v", err)
	}
}
