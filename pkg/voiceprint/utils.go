package voiceprint

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// AudioInfo 音频文件信息
type AudioInfo struct {
	Format        string        `json:"format"`
	SampleRate    int           `json:"sample_rate"`
	Channels      int           `json:"channels"`
	BitsPerSample int           `json:"bits_per_sample"`
	Duration      time.Duration `json:"duration"`
	FileSize      int64         `json:"file_size"`
}

// LoadAudioFile 加载音频文件
func LoadAudioFile(filePath string) ([]byte, error) {
	// 检查文件扩展名
	ext := strings.ToLower(filepath.Ext(filePath))
	if ext != ".wav" {
		return nil, ErrInvalidAudioFormat
	}

	// 读取文件
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read audio file: %w", err)
	}

	// 验证WAV格式
	if err := ValidateWAVFormat(data); err != nil {
		return nil, err
	}

	return data, nil
}

// ValidateWAVFormat 验证WAV格式
func ValidateWAVFormat(data []byte) error {
	if len(data) < 44 {
		return ErrInvalidAudioFormat
	}

	// 检查RIFF头
	if string(data[0:4]) != "RIFF" {
		return ErrInvalidAudioFormat
	}

	// 检查WAVE标识
	if string(data[8:12]) != "WAVE" {
		return ErrInvalidAudioFormat
	}

	// 检查fmt chunk
	if string(data[12:16]) != "fmt " {
		return ErrInvalidAudioFormat
	}

	return nil
}

// GetAudioInfo 获取音频文件信息
func GetAudioInfo(data []byte) (*AudioInfo, error) {
	if err := ValidateWAVFormat(data); err != nil {
		return nil, err
	}

	info := &AudioInfo{
		Format:   "WAV",
		FileSize: int64(len(data)),
	}

	// 读取fmt chunk信息
	reader := bytes.NewReader(data[20:])

	var audioFormat uint16
	var numChannels uint16
	var sampleRate uint32
	var bitsPerSample uint16

	binary.Read(reader, binary.LittleEndian, &audioFormat)
	binary.Read(reader, binary.LittleEndian, &numChannels)
	binary.Read(reader, binary.LittleEndian, &sampleRate)

	// 跳过ByteRate和BlockAlign
	reader.Seek(6, io.SeekCurrent)

	binary.Read(reader, binary.LittleEndian, &bitsPerSample)

	info.Channels = int(numChannels)
	info.SampleRate = int(sampleRate)
	info.BitsPerSample = int(bitsPerSample)

	// 计算时长
	dataSize := int64(len(data)) - 44 // 减去WAV头部大小
	bytesPerSecond := int64(sampleRate) * int64(numChannels) * int64(bitsPerSample) / 8
	if bytesPerSecond > 0 {
		info.Duration = time.Duration(dataSize*1000/bytesPerSecond) * time.Millisecond
	}

	return info, nil
}

// ValidateAudioDuration 验证音频时长
func ValidateAudioDuration(data []byte, minDuration, maxDuration time.Duration) error {
	info, err := GetAudioInfo(data)
	if err != nil {
		return err
	}

	if info.Duration < minDuration {
		return ErrAudioTooShort
	}

	if maxDuration > 0 && info.Duration > maxDuration {
		return ErrAudioTooLong
	}

	return nil
}

// ValidateAudioQuality 验证音频质量
func ValidateAudioQuality(data []byte) error {
	info, err := GetAudioInfo(data)
	if err != nil {
		return err
	}

	// 检查采样率（建议16kHz或更高）
	if info.SampleRate < 8000 {
		return ErrInvalidConfig("sample rate too low, minimum 8kHz required")
	}

	// 检查位深度（建议16位或更高）
	if info.BitsPerSample < 16 {
		return ErrInvalidConfig("bits per sample too low, minimum 16 bits required")
	}

	// 检查声道数（支持单声道和立体声）
	if info.Channels < 1 || info.Channels > 2 {
		return ErrInvalidConfig("unsupported channel count, only mono and stereo supported")
	}

	return nil
}

// ConvertToMono 转换为单声道（简单实现）
func ConvertToMono(data []byte) ([]byte, error) {
	info, err := GetAudioInfo(data)
	if err != nil {
		return nil, err
	}

	if info.Channels == 1 {
		return data, nil // 已经是单声道
	}

	if info.Channels != 2 {
		return nil, ErrInvalidConfig("only stereo to mono conversion supported")
	}

	// 简单的立体声转单声道实现
	// 这里只是示例，实际应用中可能需要更复杂的音频处理
	headerSize := 44
	audioData := data[headerSize:]
	bytesPerSample := info.BitsPerSample / 8

	monoData := make([]byte, len(audioData)/2)

	for i := 0; i < len(audioData); i += bytesPerSample * 2 {
		if i+bytesPerSample*2 <= len(audioData) {
			// 取左右声道的平均值
			if bytesPerSample == 2 { // 16位
				left := int16(binary.LittleEndian.Uint16(audioData[i : i+2]))
				right := int16(binary.LittleEndian.Uint16(audioData[i+2 : i+4]))
				mono := (left + right) / 2
				binary.LittleEndian.PutUint16(monoData[i/2:i/2+2], uint16(mono))
			}
		}
	}

	// 构建新的WAV文件
	result := make([]byte, headerSize+len(monoData))
	copy(result[:headerSize], data[:headerSize])
	copy(result[headerSize:], monoData)

	// 更新头部信息
	// 更新文件大小
	binary.LittleEndian.PutUint32(result[4:8], uint32(len(result)-8))
	// 更新声道数
	binary.LittleEndian.PutUint16(result[22:24], 1)
	// 更新数据大小
	binary.LittleEndian.PutUint32(result[40:44], uint32(len(monoData)))

	return result, nil
}

// GenerateSpeakerID 生成说话人ID
func GenerateSpeakerID(prefix string) string {
	timestamp := time.Now().Unix()
	return fmt.Sprintf("%s_%d", prefix, timestamp)
}

// FormatDuration 格式化时长
func FormatDuration(d time.Duration) string {
	if d < time.Second {
		return fmt.Sprintf("%.0fms", float64(d.Nanoseconds())/1e6)
	}
	if d < time.Minute {
		return fmt.Sprintf("%.1fs", d.Seconds())
	}
	return fmt.Sprintf("%.1fm", d.Minutes())
}

// FormatFileSize 格式化文件大小
func FormatFileSize(size int64) string {
	const unit = 1024
	if size < unit {
		return fmt.Sprintf("%d B", size)
	}
	div, exp := int64(unit), 0
	for n := size / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(size)/float64(div), "KMGTPE"[exp])
}

// IsValidSpeakerID 验证说话人ID格式
func IsValidSpeakerID(speakerID string) bool {
	if len(speakerID) == 0 || len(speakerID) > 100 {
		return false
	}

	// 只允许字母、数字、下划线和连字符
	for _, r := range speakerID {
		if !((r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') ||
			(r >= '0' && r <= '9') || r == '_' || r == '-') {
			return false
		}
	}

	return true
}

// SanitizeSpeakerID 清理说话人ID
func SanitizeSpeakerID(speakerID string) string {
	// 移除无效字符
	var result strings.Builder
	for _, r := range speakerID {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') ||
			(r >= '0' && r <= '9') || r == '_' || r == '-' {
			result.WriteRune(r)
		}
	}

	cleaned := result.String()
	if len(cleaned) == 0 {
		return "speaker_" + fmt.Sprintf("%d", time.Now().Unix())
	}

	if len(cleaned) > 100 {
		cleaned = cleaned[:100]
	}

	return cleaned
}

// CalculateAudioHash 计算音频数据的哈希值（用于去重）
func CalculateAudioHash(data []byte) string {
	// 简单的哈希实现，实际应用中可以使用更复杂的算法
	if len(data) < 44 {
		return ""
	}

	// 跳过WAV头部，只计算音频数据的哈希
	audioData := data[44:]
	hash := uint32(0)

	for i := 0; i < len(audioData); i += 4 {
		if i+4 <= len(audioData) {
			chunk := binary.LittleEndian.Uint32(audioData[i : i+4])
			hash ^= chunk
		}
	}

	return fmt.Sprintf("%08x", hash)
}
