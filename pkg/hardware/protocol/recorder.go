package protocol

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"go.uber.org/zap"
)

// AudioRecorder 音频录音器（本地文件存储）
type AudioRecorder struct {
	mu            sync.RWMutex
	file          *os.File
	filePath      string
	isRecording   bool
	startTime     time.Time
	totalBytes    int64
	sampleRate    int
	channels      int
	bitsPerSample int
	logger        *zap.Logger
}

// NewAudioRecorder 创建音频录音器
func NewAudioRecorder(filePath string, sampleRate, channels, bitsPerSample int, logger *zap.Logger) (*AudioRecorder, error) {
	if logger == nil {
		logger = zap.L()
	}

	// 确保目录存在
	dir := filepath.Dir(filePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("创建录音目录失败: %w", err)
	}

	// 创建文件
	file, err := os.Create(filePath)
	if err != nil {
		return nil, fmt.Errorf("创建录音文件失败: %w", err)
	}

	recorder := &AudioRecorder{
		file:          file,
		filePath:      filePath,
		isRecording:   true,
		startTime:     time.Now(),
		sampleRate:    sampleRate,
		channels:      channels,
		bitsPerSample: bitsPerSample,
		logger:        logger,
	}

	// 写入WAV文件头（先写占位符，最后更新）
	if err := recorder.writeWAVHeader(); err != nil {
		file.Close()
		return nil, err
	}

	logger.Info("[Recorder] 音频录音器已创建",
		zap.String("filePath", filePath),
		zap.Int("sampleRate", sampleRate),
		zap.Int("channels", channels))

	return recorder, nil
}

// WriteAudio 写入音频数据
func (r *AudioRecorder) WriteAudio(data []byte) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if !r.isRecording || r.file == nil {
		return fmt.Errorf("录音器未激活")
	}

	n, err := r.file.Write(data)
	if err != nil {
		return fmt.Errorf("写入音频数据失败: %w", err)
	}

	r.totalBytes += int64(n)
	return nil
}

// Stop 停止录音并关闭文件
func (r *AudioRecorder) Stop() error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if !r.isRecording || r.file == nil {
		return nil
	}

	r.isRecording = false

	// 更新WAV文件头
	if err := r.updateWAVHeader(); err != nil {
		r.logger.Error("[Recorder] 更新WAV文件头失败", zap.Error(err))
	}

	// 关闭文件
	if err := r.file.Close(); err != nil {
		return fmt.Errorf("关闭录音文件失败: %w", err)
	}

	duration := time.Since(r.startTime)
	r.logger.Info("[Recorder] 音频录音已停止",
		zap.String("filePath", r.filePath),
		zap.Int64("totalBytes", r.totalBytes),
		zap.Duration("duration", duration))

	return nil
}

// GetFilePath 获取录音文件路径
func (r *AudioRecorder) GetFilePath() string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.filePath
}

// GetTotalBytes 获取录音总字节数
func (r *AudioRecorder) GetTotalBytes() int64 {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.totalBytes
}

// GetDuration 获取录音时长（秒）
func (r *AudioRecorder) GetDuration() int {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if r.sampleRate == 0 {
		return 0
	}

	// PCM数据字节数 / (采样率 * 声道数 * 字节数/样本)
	bytesPerSample := r.bitsPerSample / 8
	totalSamples := r.totalBytes / int64(r.channels*bytesPerSample)
	return int(totalSamples / int64(r.sampleRate))
}

// writeWAVHeader 写入WAV文件头
func (r *AudioRecorder) writeWAVHeader() error {
	// WAV文件头（44字节）
	// 这里先写占位符，最后在Stop时更新
	header := make([]byte, 44)

	// RIFF标识
	copy(header[0:4], []byte("RIFF"))
	// 文件大小（先设为0，后续更新）
	// copy(header[4:8], []byte{0, 0, 0, 0})
	// WAVE标识
	copy(header[8:12], []byte("WAVE"))

	// fmt子块
	copy(header[12:16], []byte("fmt "))
	// fmt子块大小（16字节）
	header[16] = 16
	header[17] = 0
	header[18] = 0
	header[19] = 0

	// 音频格式（1=PCM）
	header[20] = 1
	header[21] = 0

	// 声道数
	header[22] = byte(r.channels)
	header[23] = 0

	// 采样率
	sampleRate := uint32(r.sampleRate)
	header[24] = byte(sampleRate)
	header[25] = byte(sampleRate >> 8)
	header[26] = byte(sampleRate >> 16)
	header[27] = byte(sampleRate >> 24)

	// 字节率（采样率 * 声道数 * 字节数/样本）
	byteRate := uint32(r.sampleRate * r.channels * r.bitsPerSample / 8)
	header[28] = byte(byteRate)
	header[29] = byte(byteRate >> 8)
	header[30] = byte(byteRate >> 16)
	header[31] = byte(byteRate >> 24)

	// 块对齐（声道数 * 字节数/样本）
	blockAlign := uint16(r.channels * r.bitsPerSample / 8)
	header[32] = byte(blockAlign)
	header[33] = byte(blockAlign >> 8)

	// 位深
	header[34] = byte(r.bitsPerSample)
	header[35] = 0

	// data子块
	copy(header[36:40], []byte("data"))
	// data子块大小（先设为0，后续更新）
	// copy(header[40:44], []byte{0, 0, 0, 0})

	_, err := r.file.Write(header)
	return err
}

// updateWAVHeader 更新WAV文件头中的文件大小信息
func (r *AudioRecorder) updateWAVHeader() error {
	if r.file == nil {
		return fmt.Errorf("文件未打开")
	}
	fileSize := r.totalBytes + 36 // 36 = 44(header) - 8(RIFF标识和大小字段)
	dataSize := r.totalBytes
	if _, err := r.file.Seek(4, 0); err != nil {
		return err
	}
	fileSizeBytes := make([]byte, 4)
	fileSizeBytes[0] = byte(fileSize)
	fileSizeBytes[1] = byte(fileSize >> 8)
	fileSizeBytes[2] = byte(fileSize >> 16)
	fileSizeBytes[3] = byte(fileSize >> 24)
	if _, err := r.file.Write(fileSizeBytes); err != nil {
		return err
	}

	// 更新data子块大小（位置40-44）
	if _, err := r.file.Seek(40, 0); err != nil {
		return err
	}
	dataSizeBytes := make([]byte, 4)
	dataSizeBytes[0] = byte(dataSize)
	dataSizeBytes[1] = byte(dataSize >> 8)
	dataSizeBytes[2] = byte(dataSize >> 16)
	dataSizeBytes[3] = byte(dataSize >> 24)
	if _, err := r.file.Write(dataSizeBytes); err != nil {
		return err
	}

	return nil
}
