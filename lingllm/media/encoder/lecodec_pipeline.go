// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

//go:build lingcodec

package encoder

import (
	"fmt"
	"strings"

	"github.com/LingByte/lingllm/media"
)

func init() {
	RegisterCodec(CodecG729, createG729Encode, createG729Decode)
}

func downmixStereo(interleaved []int16) []int16 {
	if len(interleaved) < 2 {
		return interleaved
	}
	n := len(interleaved) / 2
	out := make([]int16, n)
	for i := 0; i < n; i++ {
		out[i] = int16((int32(interleaved[2*i]) + int32(interleaved[2*i+1])) / 2)
	}
	return out
}

func leDecodePipeline(codecID uint8, defaultRate int, src, pcm media.CodecConfig) media.EncoderFunc {
	rate := src.SampleRate
	if rate == 0 {
		rate = defaultRate
	}
	res := media.DefaultResampler(rate, pcm.SampleRate)
	dec, err := openLEDec(codecID)
	if err != nil {
		panic(err)
	}
	return func(packet media.MediaPacket) ([]media.MediaPacket, error) {
		ap, ok := packet.(*media.AudioPacket)
		if !ok {
			return []media.MediaPacket{packet}, nil
		}
		samples, err := dec.Decode(ap.Payload)
		if err != nil {
			return nil, err
		}
		data := samplesToLE(samples)
		if _, err = res.Write(data); err != nil {
			return nil, err
		}
		data = res.Samples()
		if data == nil {
			return nil, nil
		}
		ap.Payload = data
		return []media.MediaPacket{ap}, nil
	}
}

func leEncodePipeline(codecID uint8, defaultRate int, src, pcm media.CodecConfig, useSplit bool) media.EncoderFunc {
	rate := src.SampleRate
	if rate == 0 {
		rate = defaultRate
	}
	res := media.DefaultResampler(pcm.SampleRate, rate)
	enc, err := openLEEnc(codecID)
	if err != nil {
		panic(err)
	}
	return func(packet media.MediaPacket) ([]media.MediaPacket, error) {
		ap, ok := packet.(*media.AudioPacket)
		if !ok {
			return []media.MediaPacket{packet}, nil
		}
		if _, err := res.Write(ap.Payload); err != nil {
			return nil, err
		}
		data := res.Samples()
		if data == nil {
			return nil, nil
		}
		encoded, err := enc.Encode(leToSamples(data))
		if err != nil {
			return nil, err
		}
		if useSplit {
			return splitFrames(encoded, &src), nil
		}
		ap.Payload = encoded
		return []media.MediaPacket{ap}, nil
	}
}

func createPCMUDecode(src, pcm media.CodecConfig) media.EncoderFunc {
	return leDecodePipeline(lePCMU, 8000, src, pcm)
}
func createPCMUEncode(src, pcm media.CodecConfig) media.EncoderFunc {
	return leEncodePipeline(lePCMU, 8000, src, pcm, true)
}
func createPCMADecode(src, pcm media.CodecConfig) media.EncoderFunc {
	return leDecodePipeline(lePCMA, 8000, src, pcm)
}
func createPCMAEncode(src, pcm media.CodecConfig) media.EncoderFunc {
	return leEncodePipeline(lePCMA, 8000, src, pcm, true)
}
func createG722Decode(src, pcm media.CodecConfig) media.EncoderFunc {
	return leDecodePipeline(leG722, 16000, src, pcm)
}
func createG722Encode(src, pcm media.CodecConfig) media.EncoderFunc {
	return leEncodePipeline(leG722, 16000, src, pcm, false)
}
func createG729Decode(src, pcm media.CodecConfig) media.EncoderFunc {
	return leDecodePipeline(leG729, 8000, src, pcm)
}
func createG729Encode(src, pcm media.CodecConfig) media.EncoderFunc {
	return leEncodePipeline(leG729, 8000, src, pcm, true)
}

func createOPUSDecode(src, pcm media.CodecConfig) media.EncoderFunc {
	rate := src.SampleRate
	if rate == 0 {
		rate = 48000
	}
	pcmOutCh := pcm.Channels
	if pcmOutCh <= 0 {
		pcmOutCh = 1
	}
	decodeCh := 1
	if pcmOutCh >= 2 {
		decodeCh = src.Channels
		if decodeCh == 0 {
			decodeCh = 1
		}
		if src.OpusDecodeChannels >= 1 && src.OpusDecodeChannels <= 2 {
			decodeCh = src.OpusDecodeChannels
		}
	} else if strings.EqualFold(strings.TrimSpace(src.Codec), "opus") && src.OpusDecodeChannels == 2 && src.OpusPCMBridgeDecodeStereo {
		decodeCh = 2
	}
	dec, err := openOpusDec(uint32(rate), uint16(decodeCh))
	if err != nil {
		panic(fmt.Errorf("opus decoder: %w", err))
	}
	res := media.DefaultResampler(rate, pcm.SampleRate)
	return func(packet media.MediaPacket) ([]media.MediaPacket, error) {
		ap, ok := packet.(*media.AudioPacket)
		if !ok {
			return []media.MediaPacket{packet}, nil
		}
		samples, err := dec.Decode(ap.Payload)
		if err != nil {
			return nil, err
		}
		if decodeCh == 2 && pcmOutCh < 2 {
			samples = downmixStereo(samples)
		}
		data := samplesToLE(samples)
		if _, err = res.Write(data); err != nil {
			return nil, err
		}
		data = res.Samples()
		if data == nil {
			return nil, nil
		}
		ap.Payload = data
		return []media.MediaPacket{ap}, nil
	}
}

func createOPUSEncode(src, pcm media.CodecConfig) media.EncoderFunc {
	rate := src.SampleRate
	if rate == 0 {
		rate = 48000
	}
	ch := src.Channels
	if ch <= 0 {
		ch = 1
	}
	res := media.DefaultResampler(pcm.SampleRate, rate)
	enc, err := openOpusEnc(uint32(rate), uint16(ch), opusAppVoIP)
	if err != nil {
		panic(fmt.Errorf("opus encoder: %w", err))
	}
	frameMs := 20
	if src.FrameDuration != "" {
		var ms int
		if _, err := fmt.Sscanf(src.FrameDuration, "%dms", &ms); err == nil && ms > 0 {
			frameMs = ms
		}
	}
	frameSize := rate * frameMs / 1000
	return func(packet media.MediaPacket) ([]media.MediaPacket, error) {
		ap, ok := packet.(*media.AudioPacket)
		if !ok {
			return []media.MediaPacket{packet}, nil
		}
		var data []byte
		if pcm.SampleRate == rate {
			data = append([]byte(nil), ap.Payload...)
		} else {
			if _, err := res.Write(ap.Payload); err != nil {
				return nil, err
			}
			data = res.Samples()
		}
		if len(data) == 0 {
			return nil, nil
		}
		if len(data)%2 != 0 {
			data = data[:len(data)-1]
		}
		samples := leToSamples(data)
		if ch == 2 && pcm.Channels < 2 {
			stereo := make([]int16, len(samples)*2)
			for i, s := range samples {
				stereo[2*i] = s
				stereo[2*i+1] = s
			}
			samples = stereo
		}
		var packets []media.MediaPacket
		spf := frameSize * ch
		for off := 0; off+spf <= len(samples); off += spf {
			encoded, err := enc.Encode(samples[off : off+spf])
			if err != nil {
				return nil, err
			}
			packets = append(packets, &media.AudioPacket{Payload: encoded, RTPSamples: uint32(frameSize)})
		}
		if rem := len(samples) % spf; rem > 0 && len(packets) == 0 {
			encoded, err := enc.Encode(samples[len(samples)-rem:])
			if err != nil {
				return nil, err
			}
			packets = append(packets, &media.AudioPacket{Payload: encoded})
		}
		return packets, nil
	}
}
