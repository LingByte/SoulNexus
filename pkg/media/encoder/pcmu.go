package encoder

import (
	"github.com/LingByte/SoulNexus/pkg/media"
)

func createPCMUDecode(src, pcm media.CodecConfig) media.EncoderFunc {
	// 使用配置的采样率，如果未设置则使用 PCMU 标准采样率 8000Hz
	sourceSampleRate := src.SampleRate
	if sourceSampleRate == 0 {
		sourceSampleRate = 8000 // PCMU 标准采样率
	}
	res := media.DefaultResampler(sourceSampleRate, pcm.SampleRate)
	return func(packet media.MediaPacket) ([]media.MediaPacket, error) {
		audioPacket, ok := packet.(*media.AudioPacket)
		if !ok {
			return []media.MediaPacket{packet}, nil
		}
		data, err := pcmu2pcm(audioPacket.Payload)
		if err != nil {
			return nil, err
		}
		if _, err = res.Write(data); err != nil {
			return nil, err
		}
		data = res.Samples()
		if data == nil {
			return nil, nil
		}
		audioPacket.Payload = data
		return []media.MediaPacket{audioPacket}, nil
	}
}

func createPCMUEncode(src, pcm media.CodecConfig) media.EncoderFunc {
	// 使用配置的目标采样率，如果未设置则使用 PCMU 标准采样率 8000Hz
	targetSampleRate := src.SampleRate
	if targetSampleRate == 0 {
		targetSampleRate = 8000 // PCMU 标准采样率
	}
	res := media.DefaultResampler(pcm.SampleRate, targetSampleRate)
	return func(packet media.MediaPacket) ([]media.MediaPacket, error) {
		audioPacket, ok := packet.(*media.AudioPacket)
		if !ok {
			return []media.MediaPacket{packet}, nil
		}
		if _, err := res.Write(audioPacket.Payload); err != nil {
			return nil, err
		}
		data := res.Samples()
		if data == nil {
			return nil, nil
		}
		data, err := pcm2pcmu(data)
		if err != nil {
			return nil, err
		}
		return splitFrames(data, &src), nil
	}
}
