// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

package app

import (
	"github.com/LingByte/SoulNexus/pkg/voiceserver/voice/recorder"
)

// MakeRecorderFactory returns a closure suitable for the RecorderFactory
// field on the xiaozhi and WebRTC ServerConfigs. When record=false (the
// default) it returns nil so no recorder is built and no WAV is
// written. When record=true every accepted call gets a fresh recorder
// pointing at the supplied bucket; the per-transport session pushes PCM
// into it through the call lifetime and flushes (uploads stereo WAV +
// writes call_recording row) at teardown.
func MakeRecorderFactory(record bool, bucket, transport string) func(callID, codec string, sampleRate int) *recorder.Recorder {
	if !record {
		return nil
	}
	if bucket == "" {
		bucket = "voiceserver-recordings"
	}
	return func(callID, codec string, sampleRate int) *recorder.Recorder {
		return recorder.New(recorder.Config{
			CallID:        callID,
			Bucket:        bucket,
			SampleRate:    sampleRate,
			Transport:     transport,
			Codec:         codec,
			ChunkInterval: RecordingChunkInterval(),
		})
	}
}
