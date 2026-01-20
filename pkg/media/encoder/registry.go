package encoder

import (
	"strings"
	"time"

	"github.com/LingByte/SoulNexus/pkg/media"
)

const (
	CodecPCM  = "pcm"
	CodecPCMU = "pcmu"
	CodecPCMA = "pcma"
	CodecG722 = "g722"
	CodecOPUS = "opus"
)

func init() {
	RegisterCodec(CodecPCMU, createPCMUEncode, createPCMUDecode)
	RegisterCodec(CodecPCMA, createPCMAEncode, createPCMADecode)
	RegisterCodec(CodecPCM, PcmToPcm, PcmToPcm)
	//RegisterCodec(CodecOPUS, createOPUSEncode, createOPUSDecode)
	RegisterCodec(CodecG722, createG722Encode, createG722Decode)
}

// CodecFactory defines function type for creating codec encoders/decoders
type CodecFactory func(src, pcm media.CodecConfig) media.EncoderFunc

// codecRegistry stores encoder/decoder factory pairs
type codecRegistry struct {
	encoderFactory CodecFactory
	decoderFactory CodecFactory
}

var codecRegistryMap = make(map[string]codecRegistry)

// RegisterCodec registers a codec with encoder and decoder factories
func RegisterCodec(name string, encoderFactory, decoderFactory CodecFactory) {
	codecRegistryMap[strings.ToLower(name)] = codecRegistry{
		encoderFactory: encoderFactory,
		decoderFactory: decoderFactory,
	}
}

// BuildEncoder creates an encoder function for the specified codec
func CreateEncode(src, pcm media.CodecConfig) (encode media.EncoderFunc, err error) {
	registry, exists := codecRegistryMap[strings.ToLower(src.Codec)]
	if !exists {
		err = media.ErrCodecNotSupported
		return
	}
	encode = registry.encoderFactory(src, pcm)
	return
}

// BuildDecoder creates a decoder function for the specified codec
func CreateDecode(src, pcm media.CodecConfig) (decode media.EncoderFunc, err error) {
	registry, exists := codecRegistryMap[strings.ToLower(src.Codec)]
	if !exists {
		err = media.ErrCodecNotSupported
		return
	}
	decode = registry.decoderFactory(src, pcm)
	return
}

// IsCodecSupported checks if a codec is registered
func HasCodec(name string) bool {
	_, exists := codecRegistryMap[strings.ToLower(name)]
	return exists
}

// RemoveWavHeader removes WAV file header if present
func StripWavHeader(data []byte) []byte {
	const wavHeaderSize = 44
	const riffSignature = "RIFF"
	if len(data) > wavHeaderSize &&
		data[0] == riffSignature[0] &&
		data[1] == riffSignature[1] &&
		data[2] == riffSignature[2] &&
		data[3] == riffSignature[3] {
		return data[wavHeaderSize:]
	}
	return data
}

// splitFrames splits audio data into packets based on duration
func splitFrames(data []byte, src *media.CodecConfig) []media.MediaPacket {
	if src.FrameDuration == "" {
		return []media.MediaPacket{&media.AudioPacket{Payload: data}}
	}
	duration, _ := time.ParseDuration(src.FrameDuration)
	if duration < 10*time.Millisecond || duration > 300*time.Millisecond {
		duration = 20 * time.Millisecond
	}
	bytesPerFrame := int(duration.Milliseconds()) * src.SampleRate / 1000
	packets := make([]media.MediaPacket, 0)

	for offset := 0; offset < len(data); offset += bytesPerFrame {
		frameEnd := offset + bytesPerFrame
		if frameEnd > len(data) {
			frameEnd = len(data)
		}
		packets = append(packets, &media.AudioPacket{Payload: data[offset:frameEnd]})
	}
	return packets
}
