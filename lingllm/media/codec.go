package media

// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

// AudioCodec represents supported audio codecs
type AudioCodec string

const (
	// Uncompressed
	AudioCodecPCM  AudioCodec = "pcm"
	AudioCodecPCMU AudioCodec = "pcmu"
	AudioCodecPCMA AudioCodec = "pcma"

	// Lossless compression
	AudioCodecFLAC AudioCodec = "flac"
	AudioCodecAPE  AudioCodec = "ape"
	AudioCodecWAV  AudioCodec = "wav"

	// Lossy compression (Low bitrate)
	AudioCodecMP3    AudioCodec = "mp3"
	AudioCodecAAC    AudioCodec = "aac"
	AudioCodecOPUS   AudioCodec = "opus"
	AudioCodecVORBIS AudioCodec = "vorbis"

	// Lossy compression (High quality)
	AudioCodecFDAC AudioCodec = "fdac"
	AudioCodecALAC AudioCodec = "alac"

	// Telephony
	AudioCodecGSM  AudioCodec = "gsm"
	AudioCodecAMR  AudioCodec = "amr"
	AudioCodecSILK AudioCodec = "silk"

	// Proprietary
	AudioCodecWMA AudioCodec = "wma"
	AudioCodecAC3 AudioCodec = "ac3"
	AudioCodecDTS AudioCodec = "dts"
)

// VideoCodec represents supported video codecs
type VideoCodec string

const (
	// Uncompressed
	VideoCodecRaw VideoCodec = "raw"

	// H.26x family
	VideoCodecH261 VideoCodec = "h261"
	VideoCodecH263 VideoCodec = "h263"
	VideoCodecH264 VideoCodec = "h264"
	VideoCodecH265 VideoCodec = "h265"
	VideoCodecH266 VideoCodec = "h266"

	// VP family
	VideoCodecVP8 VideoCodec = "vp8"
	VideoCodecVP9 VideoCodec = "vp9"

	// AV family
	VideoCodecAV1 VideoCodec = "av1"

	// MPEG family
	VideoCodecMPEG1 VideoCodec = "mpeg1"
	VideoCodecMPEG2 VideoCodec = "mpeg2"
	VideoCodecMPEG4 VideoCodec = "mpeg4"

	// Proprietary
	VideoCodecWMV    VideoCodec = "wmv"
	VideoCodecRV     VideoCodec = "rv"
	VideoCodecProRes VideoCodec = "prores"
	VideoCodecDNxHD  VideoCodec = "dnxhd"

	// Older/Legacy
	VideoCodecSorenson VideoCodec = "sorenson"
	VideoCodecCinepak  VideoCodec = "cinepak"
)

// AudioProfile represents audio codec profiles
type AudioProfile string

const (
	// AAC profiles
	AudioProfileAAC_LC    AudioProfile = "aac_lc"
	AudioProfileAAC_HE    AudioProfile = "aac_he"
	AudioProfileAAC_HE_V2 AudioProfile = "aac_he_v2"
	AudioProfileAAC_LD    AudioProfile = "aac_ld"
	AudioProfileAAC_ELD   AudioProfile = "aac_eld"

	// Opus profiles
	AudioProfileOpusNarrow AudioProfile = "opus_narrow"
	AudioProfileOpusWide   AudioProfile = "opus_wide"

	// MP3 profiles
	AudioProfileMP3_MPEG1  AudioProfile = "mp3_mpeg1"
	AudioProfileMP3_MPEG2  AudioProfile = "mp3_mpeg2"
	AudioProfileMP3_MPEG25 AudioProfile = "mp3_mpeg25"
)

// VideoProfile represents video codec profiles
type VideoProfile string

const (
	// H.264 profiles
	VideoProfileH264_Baseline VideoProfile = "h264_baseline"
	VideoProfileH264_Main     VideoProfile = "h264_main"
	VideoProfileH264_High     VideoProfile = "h264_high"

	// H.265 profiles
	VideoProfileH265_Main      VideoProfile = "h265_main"
	VideoProfileH265_Main10    VideoProfile = "h265_main10"
	VideoProfileH265_MainStill VideoProfile = "h265_main_still"

	// VP9 profiles
	VideoProfileVP9_Profile0 VideoProfile = "vp9_profile0"
	VideoProfileVP9_Profile1 VideoProfile = "vp9_profile1"
	VideoProfileVP9_Profile2 VideoProfile = "vp9_profile2"
	VideoProfileVP9_Profile3 VideoProfile = "vp9_profile3"

	// AV1 profiles
	VideoProfileAV1_Main VideoProfile = "av1_main"
	VideoProfileAV1_High VideoProfile = "av1_high"
	VideoProfileAV1_Pro  VideoProfile = "av1_pro"
)

// CodecLevel represents codec level/tier
type CodecLevel string

const (
	// H.264 levels
	CodecLevelH264_1  CodecLevel = "h264_1"
	CodecLevelH264_1b CodecLevel = "h264_1b"
	CodecLevelH264_11 CodecLevel = "h264_11"
	CodecLevelH264_12 CodecLevel = "h264_12"
	CodecLevelH264_13 CodecLevel = "h264_13"
	CodecLevelH264_2  CodecLevel = "h264_2"
	CodecLevelH264_21 CodecLevel = "h264_21"
	CodecLevelH264_22 CodecLevel = "h264_22"
	CodecLevelH264_3  CodecLevel = "h264_3"
	CodecLevelH264_31 CodecLevel = "h264_31"
	CodecLevelH264_32 CodecLevel = "h264_32"
	CodecLevelH264_4  CodecLevel = "h264_4"
	CodecLevelH264_41 CodecLevel = "h264_41"
	CodecLevelH264_42 CodecLevel = "h264_42"
	CodecLevelH264_5  CodecLevel = "h264_5"
	CodecLevelH264_51 CodecLevel = "h264_51"
	CodecLevelH264_52 CodecLevel = "h264_52"

	// H.265 levels
	CodecLevelH265_1  CodecLevel = "h265_1"
	CodecLevelH265_2  CodecLevel = "h265_2"
	CodecLevelH265_21 CodecLevel = "h265_21"
	CodecLevelH265_3  CodecLevel = "h265_3"
	CodecLevelH265_31 CodecLevel = "h265_31"
	CodecLevelH265_4  CodecLevel = "h265_4"
	CodecLevelH265_41 CodecLevel = "h265_41"
	CodecLevelH265_5  CodecLevel = "h265_5"
	CodecLevelH265_51 CodecLevel = "h265_51"
	CodecLevelH265_52 CodecLevel = "h265_52"
)

// CodecInfo contains detailed information about a codec
type CodecInfo struct {
	// Codec identifier (can be audio or video codec name)
	Codec string

	// Human-readable name
	Name string

	// Description
	Description string

	// Whether the codec is lossy
	IsLossy bool

	// Supported profiles
	Profiles []string

	// Supported levels
	Levels []string

	// Typical bitrate range (kbps)
	BitrateMin int
	BitrateMax int

	// Whether hardware acceleration is available
	HardwareAcceleration bool

	// Supported container formats
	Containers []string
}

// AudioCodecInfo returns information about an audio codec
func AudioCodecInfo(codec AudioCodec) *CodecInfo {
	infoMap := map[AudioCodec]*CodecInfo{
		AudioCodecPCM: {
			Codec:       string(codec),
			Name:        "PCM",
			Description: "Pulse Code Modulation (uncompressed)",
			IsLossy:     false,
			BitrateMin:  128,
			BitrateMax:  1536,
			Containers:  []string{"wav", "raw"},
		},
		AudioCodecMP3: {
			Codec:       string(codec),
			Name:        "MP3",
			Description: "MPEG-1 Audio Layer III",
			IsLossy:     true,
			BitrateMin:  32,
			BitrateMax:  320,
			Containers:  []string{"mp3", "m4a"},
		},
		AudioCodecAAC: {
			Codec:       string(codec),
			Name:        "AAC",
			Description: "Advanced Audio Coding",
			IsLossy:     true,
			BitrateMin:  16,
			BitrateMax:  320,
			Containers:  []string{"m4a", "aac", "mp4"},
		},
		AudioCodecOPUS: {
			Codec:       string(codec),
			Name:        "Opus",
			Description: "Opus Interactive Audio Codec",
			IsLossy:     true,
			BitrateMin:  6,
			BitrateMax:  510,
			Containers:  []string{"opus", "ogg", "webm"},
		},
		AudioCodecFLAC: {
			Codec:       string(codec),
			Name:        "FLAC",
			Description: "Free Lossless Audio Codec",
			IsLossy:     false,
			BitrateMin:  128,
			BitrateMax:  1536,
			Containers:  []string{"flac"},
		},
		AudioCodecVORBIS: {
			Codec:       string(codec),
			Name:        "Vorbis",
			Description: "Ogg Vorbis",
			IsLossy:     true,
			BitrateMin:  32,
			BitrateMax:  500,
			Containers:  []string{"ogg", "ogv"},
		},
	}

	if info, exists := infoMap[codec]; exists {
		return info
	}

	return &CodecInfo{
		Codec:       string(codec),
		Name:        string(codec),
		Description: "Unknown codec",
		IsLossy:     false,
	}
}

// VideoCodecInfo returns information about a video codec
func VideoCodecInfo(codec VideoCodec) *CodecInfo {
	infoMap := map[VideoCodec]*CodecInfo{
		VideoCodecH264: {
			Codec:                string(codec),
			Name:                 "H.264",
			Description:          "MPEG-4 Part 10 Advanced Video Coding",
			IsLossy:              true,
			BitrateMin:           100,
			BitrateMax:           50000,
			HardwareAcceleration: true,
			Containers:           []string{"mp4", "mkv", "flv", "ts"},
		},
		VideoCodecH265: {
			Codec:                string(codec),
			Name:                 "H.265",
			Description:          "High Efficiency Video Coding",
			IsLossy:              true,
			BitrateMin:           50,
			BitrateMax:           50000,
			HardwareAcceleration: true,
			Containers:           []string{"mp4", "mkv", "ts"},
		},
		VideoCodecVP8: {
			Codec:                string(codec),
			Name:                 "VP8",
			Description:          "VP8 Video Codec",
			IsLossy:              true,
			BitrateMin:           100,
			BitrateMax:           50000,
			HardwareAcceleration: false,
			Containers:           []string{"webm", "mkv"},
		},
		VideoCodecVP9: {
			Codec:                string(codec),
			Name:                 "VP9",
			Description:          "VP9 Video Codec",
			IsLossy:              true,
			BitrateMin:           50,
			BitrateMax:           50000,
			HardwareAcceleration: false,
			Containers:           []string{"webm", "mkv", "mp4"},
		},
		VideoCodecAV1: {
			Codec:                string(codec),
			Name:                 "AV1",
			Description:          "AOMedia Video 1",
			IsLossy:              true,
			BitrateMin:           30,
			BitrateMax:           50000,
			HardwareAcceleration: false,
			Containers:           []string{"mp4", "mkv", "webm"},
		},
	}

	if info, exists := infoMap[codec]; exists {
		return info
	}

	return &CodecInfo{
		Codec:       string(codec),
		Name:        string(codec),
		Description: "Unknown codec",
		IsLossy:     false,
	}
}

// IsAudioCodecSupported checks if an audio codec is supported
func IsAudioCodecSupported(codec AudioCodec) bool {
	supportedCodecs := []AudioCodec{
		AudioCodecPCM, AudioCodecPCMU, AudioCodecPCMA,
		AudioCodecFLAC, AudioCodecAPE, AudioCodecWAV,
		AudioCodecMP3, AudioCodecAAC, AudioCodecOPUS, AudioCodecVORBIS,
		AudioCodecFDAC, AudioCodecALAC,
		AudioCodecGSM, AudioCodecAMR, AudioCodecSILK,
		AudioCodecWMA, AudioCodecAC3, AudioCodecDTS,
	}

	for _, supported := range supportedCodecs {
		if supported == codec {
			return true
		}
	}
	return false
}

// IsVideoCodecSupported checks if a video codec is supported
func IsVideoCodecSupported(codec VideoCodec) bool {
	supportedCodecs := []VideoCodec{
		VideoCodecRaw,
		VideoCodecH261, VideoCodecH263, VideoCodecH264, VideoCodecH265, VideoCodecH266,
		VideoCodecVP8, VideoCodecVP9,
		VideoCodecAV1,
		VideoCodecMPEG1, VideoCodecMPEG2, VideoCodecMPEG4,
		VideoCodecWMV, VideoCodecRV, VideoCodecProRes, VideoCodecDNxHD,
		VideoCodecSorenson, VideoCodecCinepak,
	}

	for _, supported := range supportedCodecs {
		if supported == codec {
			return true
		}
	}
	return false
}
