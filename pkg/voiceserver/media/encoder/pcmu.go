package encoder

// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

import (
	"github.com/LingByte/SoulNexus/pkg/voiceserver/media"
)

func createPCMUDecode(src, pcm media.CodecConfig) media.EncoderFunc {
	// Use configured sample rate, if not set use PCMU standard sample rate 8000Hz
	sourceSampleRate := src.SampleRate
	if sourceSampleRate == 0 {
		sourceSampleRate = 8000 // PCMU standard sample rate
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
	// Use configured target sample rate, if not set use PCMU standard sample rate 8000Hz
	targetSampleRate := src.SampleRate
	if targetSampleRate == 0 {
		targetSampleRate = 8000 // PCMU standard sample rate
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
