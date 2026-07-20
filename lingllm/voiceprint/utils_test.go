// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

package voiceprint

import (
	"encoding/binary"
	"strings"
	"testing"
	"time"
)

func TestLoadAudioFile(t *testing.T) {
	tests := []struct {
		name    string
		path    string
		wantErr bool
	}{
		{
			name:    "invalid extension",
			path:    "test.mp3",
			wantErr: true,
		},
		{
			name:    "file not found",
			path:    "nonexistent.wav",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := LoadAudioFile(tt.path)
			if (err != nil) != tt.wantErr {
				t.Errorf("LoadAudioFile() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidateWAVFormat(t *testing.T) {
	tests := []struct {
		name    string
		data    []byte
		wantErr bool
	}{
		{
			name:    "too short",
			data:    make([]byte, 10),
			wantErr: true,
		},
		{
			name:    "invalid RIFF header",
			data:    append([]byte("XXXX"), make([]byte, 40)...),
			wantErr: true,
		},
		{
			name: "valid WAV header",
			data: func() []byte {
				data := make([]byte, 44)
				copy(data[0:4], "RIFF")
				copy(data[8:12], "WAVE")
				copy(data[12:16], "fmt ")
				return data
			}(),
			wantErr: false,
		},
		{
			name:    "invalid WAVE identifier",
			data:    append([]byte("RIFF"), append(make([]byte, 4), append([]byte("XXXX"), make([]byte, 32)...)...)...),
			wantErr: true,
		},
		{
			name:    "invalid fmt chunk",
			data:    append([]byte("RIFF"), append(make([]byte, 4), append([]byte("WAVE"), append([]byte("XXXX"), make([]byte, 28)...)...)...)...),
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateWAVFormat(tt.data)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateWAVFormat() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestGetAudioInfo(t *testing.T) {
	tests := []struct {
		name    string
		data    []byte
		want    *AudioInfo
		wantErr bool
	}{
		{
			name:    "invalid format",
			data:    make([]byte, 10),
			wantErr: true,
		},
		{
			name: "valid WAV file",
			data: func() []byte {
				data := make([]byte, 44)
				copy(data[0:4], "RIFF")
				binary.LittleEndian.PutUint32(data[4:8], 36)
				copy(data[8:12], "WAVE")
				copy(data[12:16], "fmt ")
				binary.LittleEndian.PutUint32(data[16:20], 16)

				// Audio format (1 = PCM)
				binary.LittleEndian.PutUint16(data[20:22], 1)
				// Channels (2)
				binary.LittleEndian.PutUint16(data[22:24], 2)
				// Sample rate (16000)
				binary.LittleEndian.PutUint32(data[24:28], 16000)
				// Byte rate
				binary.LittleEndian.PutUint32(data[28:32], 64000)
				// Block align
				binary.LittleEndian.PutUint16(data[32:34], 4)
				// Bits per sample (16)
				binary.LittleEndian.PutUint16(data[34:36], 16)

				return data
			}(),
			want: &AudioInfo{
				Format:        "WAV",
				SampleRate:    16000,
				Channels:      2,
				BitsPerSample: 16,
				FileSize:      44,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := GetAudioInfo(tt.data)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetAudioInfo() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && got != nil {
				if got.Format != tt.want.Format ||
					got.SampleRate != tt.want.SampleRate ||
					got.Channels != tt.want.Channels ||
					got.BitsPerSample != tt.want.BitsPerSample ||
					got.FileSize != tt.want.FileSize {
					t.Errorf("GetAudioInfo() = %+v, want %+v", got, tt.want)
				}
			}
		})
	}
}

func TestConvertToMono(t *testing.T) {
	tests := []struct {
		name    string
		data    []byte
		wantErr bool
	}{
		{
			name: "mono audio",
			data: func() []byte {
				data := make([]byte, 100)
				copy(data[0:4], "RIFF")
				binary.LittleEndian.PutUint32(data[4:8], 92)
				copy(data[8:12], "WAVE")
				copy(data[12:16], "fmt ")
				binary.LittleEndian.PutUint32(data[16:20], 16)
				binary.LittleEndian.PutUint16(data[20:22], 1) // PCM
				binary.LittleEndian.PutUint16(data[22:24], 1) // Mono
				return data
			}(),
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ConvertToMono(tt.data)
			if (err != nil) != tt.wantErr {
				t.Errorf("ConvertToMono() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestGenerateSpeakerID(t *testing.T) {
	id := GenerateSpeakerID("speaker")
	if id == "" {
		t.Error("GenerateSpeakerID() returned empty string")
	}
	if !strings.HasPrefix(id, "speaker_") {
		t.Errorf("GenerateSpeakerID() = %s, want prefix speaker_", id)
	}
}

func TestFormatDuration(t *testing.T) {
	tests := []struct {
		name string
		d    time.Duration
	}{
		{
			name: "milliseconds",
			d:    100 * time.Millisecond,
		},
		{
			name: "seconds",
			d:    5 * time.Second,
		},
		{
			name: "minutes",
			d:    2 * time.Minute,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := FormatDuration(tt.d)
			if got == "" {
				t.Error("FormatDuration() returned empty string")
			}
		})
	}
}

func TestFormatFileSize(t *testing.T) {
	tests := []struct {
		name string
		size int64
	}{
		{
			name: "bytes",
			size: 512,
		},
		{
			name: "kilobytes",
			size: 1024 * 10,
		},
		{
			name: "megabytes",
			size: 1024 * 1024 * 5,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := FormatFileSize(tt.size)
			if got == "" {
				t.Error("FormatFileSize() returned empty string")
			}
		})
	}
}

func TestIsValidSpeakerID(t *testing.T) {
	tests := []struct {
		name      string
		speakerID string
		want      bool
	}{
		{
			name:      "valid id",
			speakerID: "speaker_001",
			want:      true,
		},
		{
			name:      "valid id with dash",
			speakerID: "speaker-001",
			want:      true,
		},
		{
			name:      "empty id",
			speakerID: "",
			want:      false,
		},
		{
			name:      "invalid characters",
			speakerID: "speaker@001",
			want:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsValidSpeakerID(tt.speakerID)
			if got != tt.want {
				t.Errorf("IsValidSpeakerID() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSanitizeSpeakerID(t *testing.T) {
	tests := []struct {
		name      string
		speakerID string
	}{
		{
			name:      "valid id",
			speakerID: "speaker_001",
		},
		{
			name:      "id with invalid chars",
			speakerID: "speaker@001!",
		},
		{
			name:      "empty id",
			speakerID: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := SanitizeSpeakerID(tt.speakerID)
			if got == "" {
				t.Error("SanitizeSpeakerID() returned empty string")
			}
			if !IsValidSpeakerID(got) {
				t.Errorf("SanitizeSpeakerID() returned invalid ID: %s", got)
			}
		})
	}
}

func TestCalculateAudioHash(t *testing.T) {
	tests := []struct {
		name string
		data []byte
	}{
		{
			name: "valid WAV data",
			data: func() []byte {
				data := make([]byte, 100)
				copy(data[0:4], "RIFF")
				return data
			}(),
		},
		{
			name: "short data",
			data: make([]byte, 10),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := CalculateAudioHash(tt.data)
			// Hash can be empty for short data, that's ok
			if len(tt.data) >= 44 && got == "" {
				t.Error("CalculateAudioHash() returned empty string for valid data")
			}
		})
	}
}

func TestAudioInfo_Fields(t *testing.T) {
	info := &AudioInfo{
		Format:        "WAV",
		SampleRate:    16000,
		Channels:      2,
		BitsPerSample: 16,
		Duration:      time.Second,
		FileSize:      64000,
	}

	if info.Format != "WAV" {
		t.Errorf("AudioInfo.Format = %s, want WAV", info.Format)
	}
	if info.SampleRate != 16000 {
		t.Errorf("AudioInfo.SampleRate = %d, want 16000", info.SampleRate)
	}
	if info.Channels != 2 {
		t.Errorf("AudioInfo.Channels = %d, want 2", info.Channels)
	}
}
