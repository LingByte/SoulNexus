package parser

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	mp3 "github.com/hajimehoshi/go-mp3"
)

const defaultASRSampleRate = 16000

// pcmAudio holds mono 16-bit PCM samples at the target sample rate for ASR engines.
type pcmAudio struct {
	SampleRate int
	Samples    []byte
}

func decodeAudioToPCM(data []byte, fileName string) (*pcmAudio, error) {
	ext := strings.ToLower(strings.TrimPrefix(filepath.Ext(fileName), "."))

	if isWAV(data) || ext == FileTypeWAV {
		return wavToPCM(data, defaultASRSampleRate)
	}
	switch ext {
	case FileTypeMP3:
		return mp3ToPCM(data, defaultASRSampleRate)
	case FileTypeOGG, FileTypeFLAC, FileTypeM4A, FileTypeAAC:
		return ffmpegBytesToPCM(data, ext, defaultASRSampleRate)
	default:
		if ext != "" {
			return ffmpegBytesToPCM(data, ext, defaultASRSampleRate)
		}
		return nil, fmt.Errorf("unsupported audio format %q", ext)
	}
}

func isWAV(data []byte) bool {
	return len(data) >= 12 && string(data[0:4]) == "RIFF" && string(data[8:12]) == "WAVE"
}

func wavToPCM(data []byte, targetRate int) (*pcmAudio, error) {
	channels, sampWidth, sampleRate, _, pcm, err := readWAVPCM(data)
	if err != nil {
		return nil, err
	}
	if sampWidth != 2 {
		return nil, fmt.Errorf("unsupported WAV sample width: %d bytes", sampWidth)
	}

	samples := bytesToInt16LE(pcm)
	if channels > 1 {
		samples = monoMixInt16(samples, channels)
	}
	if sampleRate != targetRate {
		samples = resampleInt16(samples, sampleRate, targetRate)
	}
	return &pcmAudio{
		SampleRate: targetRate,
		Samples:    int16ToBytesLE(samples),
	}, nil
}

func mp3ToPCM(data []byte, targetRate int) (*pcmAudio, error) {
	dec, err := mp3.NewDecoder(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("mp3 decode: %w", err)
	}
	raw, err := io.ReadAll(dec)
	if err != nil {
		return nil, fmt.Errorf("mp3 read: %w", err)
	}
	sampleRate := dec.SampleRate()
	samples := bytesToInt16LE(raw)
	samples = resampleInt16(samples, sampleRate, targetRate)
	return &pcmAudio{
		SampleRate: targetRate,
		Samples:    int16ToBytesLE(samples),
	}, nil
}

func ffmpegBytesToPCM(data []byte, ext string, targetRate int) (*pcmAudio, error) {
	if _, err := exec.LookPath("ffmpeg"); err != nil {
		return nil, fmt.Errorf("ffmpeg not found; install ffmpeg to decode .%s audio: %w", ext, err)
	}
	if ext == "" {
		ext = "bin"
	}

	tmp, err := os.CreateTemp("", "lingllm-audio-*."+ext)
	if err != nil {
		return nil, err
	}
	tmpPath := tmp.Name()
	defer os.Remove(tmpPath)

	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return nil, err
	}
	if err := tmp.Close(); err != nil {
		return nil, err
	}

	cmd := exec.Command("ffmpeg", "-v", "quiet", "-y", "-i", tmpPath,
		"-acodec", "pcm_s16le", "-ac", "1", "-ar", strconv.Itoa(targetRate), "-f", "wav", "-")
	var out bytes.Buffer
	cmd.Stdout = &out

	done := make(chan error, 1)
	go func() { done <- cmd.Run() }()

	select {
	case <-time.After(120 * time.Second):
		_ = cmd.Process.Kill()
		return nil, fmt.Errorf("ffmpeg timed out decoding .%s audio", ext)
	case err := <-done:
		if err != nil {
			return nil, fmt.Errorf("ffmpeg decode .%s: %w", ext, err)
		}
	}

	return wavToPCM(out.Bytes(), targetRate)
}

func readWAVPCM(data []byte) (channels, sampWidth, sampleRate, nframes int, pcm []byte, err error) {
	if !isWAV(data) {
		return 0, 0, 0, 0, nil, fmt.Errorf("not a WAV file")
	}
	reader := bytes.NewReader(data)

	var chunkID [4]byte
	var chunkSize uint32
	var format [4]byte
	if err := binary.Read(reader, binary.LittleEndian, &chunkID); err != nil {
		return 0, 0, 0, 0, nil, err
	}
	if err := binary.Read(reader, binary.LittleEndian, &chunkSize); err != nil {
		return 0, 0, 0, 0, nil, err
	}
	if err := binary.Read(reader, binary.LittleEndian, &format); err != nil {
		return 0, 0, 0, 0, nil, err
	}
	if string(chunkID[:]) != "RIFF" || string(format[:]) != "WAVE" {
		return 0, 0, 0, 0, nil, fmt.Errorf("invalid WAV header")
	}

	var audioFormat, numChannels, bitsPerSample, blockAlign uint16
	var byteRate uint32
	foundData := false

	for reader.Len() > 0 {
		var subID [4]byte
		var subSize uint32
		if err := binary.Read(reader, binary.LittleEndian, &subID); err != nil {
			if err == io.EOF {
				break
			}
			return 0, 0, 0, 0, nil, err
		}
		if err := binary.Read(reader, binary.LittleEndian, &subSize); err != nil {
			return 0, 0, 0, 0, nil, err
		}

		switch string(subID[:]) {
		case "fmt ":
			if err := binary.Read(reader, binary.LittleEndian, &audioFormat); err != nil {
				return 0, 0, 0, 0, nil, err
			}
			if err := binary.Read(reader, binary.LittleEndian, &numChannels); err != nil {
				return 0, 0, 0, 0, nil, err
			}
			var sr uint32
			if err := binary.Read(reader, binary.LittleEndian, &sr); err != nil {
				return 0, 0, 0, 0, nil, err
			}
			sampleRate = int(sr)
			if err := binary.Read(reader, binary.LittleEndian, &byteRate); err != nil {
				return 0, 0, 0, 0, nil, err
			}
			if err := binary.Read(reader, binary.LittleEndian, &blockAlign); err != nil {
				return 0, 0, 0, 0, nil, err
			}
			if err := binary.Read(reader, binary.LittleEndian, &bitsPerSample); err != nil {
				return 0, 0, 0, 0, nil, err
			}
			remaining := int(subSize) - 16
			if remaining > 0 {
				if _, err := reader.Seek(int64(remaining), io.SeekCurrent); err != nil {
					return 0, 0, 0, 0, nil, err
				}
			}
		case "data":
			pcm = make([]byte, subSize)
			if _, err := io.ReadFull(reader, pcm); err != nil {
				return 0, 0, 0, 0, nil, err
			}
			foundData = true
		default:
			if _, err := reader.Seek(int64(subSize), io.SeekCurrent); err != nil {
				return 0, 0, 0, 0, nil, err
			}
		}
	}

	if !foundData {
		return 0, 0, 0, 0, nil, fmt.Errorf("WAV data chunk not found")
	}
	if audioFormat != 1 {
		return 0, 0, 0, 0, nil, fmt.Errorf("unsupported WAV audio format: %d", audioFormat)
	}

	channels = int(numChannels)
	sampWidth = int(bitsPerSample / 8)
	if sampWidth <= 0 {
		return 0, 0, 0, 0, nil, fmt.Errorf("invalid WAV bits per sample")
	}
	nframes = len(pcm) / (channels * sampWidth)
	return channels, sampWidth, sampleRate, nframes, pcm, nil
}

func bytesToInt16LE(b []byte) []int16 {
	if len(b)%2 != 0 {
		b = b[:len(b)-1]
	}
	out := make([]int16, len(b)/2)
	for i := range out {
		out[i] = int16(binary.LittleEndian.Uint16(b[i*2 : i*2+2]))
	}
	return out
}

func int16ToBytesLE(samples []int16) []byte {
	out := make([]byte, len(samples)*2)
	for i, s := range samples {
		binary.LittleEndian.PutUint16(out[i*2:i*2+2], uint16(s))
	}
	return out
}

func monoMixInt16(samples []int16, channels int) []int16 {
	if channels <= 1 {
		return samples
	}
	frames := len(samples) / channels
	out := make([]int16, frames)
	for i := 0; i < frames; i++ {
		var sum int32
		for c := 0; c < channels; c++ {
			sum += int32(samples[i*channels+c])
		}
		out[i] = int16(sum / int32(channels))
	}
	return out
}

func resampleInt16(samples []int16, fromRate, toRate int) []int16 {
	if fromRate == toRate || len(samples) == 0 {
		return samples
	}
	ratio := float64(fromRate) / float64(toRate)
	outLen := int(float64(len(samples)) / ratio)
	if outLen <= 0 {
		return nil
	}
	out := make([]int16, outLen)
	for i := range out {
		srcIdx := int(float64(i) * ratio)
		if srcIdx >= len(samples) {
			srcIdx = len(samples) - 1
		}
		out[i] = samples[srcIdx]
	}
	return out
}

// extractVoskText pulls the "text" field from Vosk JSON output: {"text" : "hello"}.
func extractVoskText(jsonResult string) string {
	jsonResult = strings.TrimSpace(jsonResult)
	const key = `"text"`
	idx := strings.Index(jsonResult, key)
	if idx < 0 {
		return jsonResult
	}
	rest := jsonResult[idx+len(key):]
	colon := strings.Index(rest, ":")
	if colon < 0 {
		return jsonResult
	}
	rest = strings.TrimSpace(rest[colon+1:])
	if len(rest) == 0 {
		return ""
	}
	if rest[0] == '"' {
		rest = rest[1:]
		end := strings.Index(rest, `"`)
		if end >= 0 {
			return rest[:end]
		}
	}
	return strings.Trim(rest, " \t\n\r{}")
}
