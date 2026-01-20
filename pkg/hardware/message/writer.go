package message

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/LingByte/SoulNexus/pkg/media"
	"github.com/gorilla/websocket"
	"go.uber.org/zap"
)

const (
	// WriterBufferSize 消息写入器缓冲区大小
	WriterBufferSize = 100
	// TTSPreBufferCount TTS预缓冲包数量（前N个包直接发送）
	TTSPreBufferCount = 5
	// TTSFrameDuration TTS帧时长（毫秒）
	TTSFrameDuration = 60
)

// Writer 消息写入器实现（异步写入，减少延迟）
type Writer struct {
	conn       *websocket.Conn
	logger     *zap.Logger
	mu         sync.Mutex
	msgChan    chan []byte
	binaryChan chan []byte
	ctx        context.Context
	cancel     context.CancelFunc
	wg         sync.WaitGroup
	isXiaozhi  bool   // 是否使用xiaozhi协议格式
	sessionID  string // 会话ID（用于xiaozhi协议）

	// TTS流控相关
	ttsFlowControlMu sync.Mutex
	ttsFlowControl   *ttsFlowControl
}

// ttsFlowControl TTS流控状态
type ttsFlowControl struct {
	packetCount   int           // 已发送包数量
	startTime     time.Time     // 开始时间
	lastSendTime  time.Time     // 上次实际发送时间
	sendDelay     time.Duration // 固定延迟（如果>0则使用固定延迟，否则使用时间同步）
	frameDuration time.Duration // 帧时长
}

// NewWriter 创建消息写入器
func NewWriter(conn *websocket.Conn, logger *zap.Logger) *Writer {
	ctx, cancel := context.WithCancel(context.Background())
	w := &Writer{
		conn:       conn,
		logger:     logger,
		msgChan:    make(chan []byte, WriterBufferSize),
		binaryChan: make(chan []byte, WriterBufferSize),
		ctx:        ctx,
		cancel:     cancel,
		sessionID:  fmt.Sprintf("%d", time.Now().UnixNano()),
	}

	// 启动异步写入goroutine
	w.wg.Add(2)
	go w.writeLoop()
	go w.writeBinaryLoop()

	return w
}

// SetXiaozhiMode 设置为xiaozhi协议模式
func (w *Writer) SetXiaozhiMode(sessionID string) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.isXiaozhi = true
	if sessionID != "" {
		w.sessionID = sessionID
	}
}

// Close 关闭写入器
func (w *Writer) Close() error {
	w.cancel()
	close(w.msgChan)
	close(w.binaryChan)
	w.wg.Wait()
	return nil
}

// writeLoop 文本消息写入循环
func (w *Writer) writeLoop() {
	defer w.wg.Done()
	for {
		select {
		case <-w.ctx.Done():
			return
		case msg, ok := <-w.msgChan:
			if !ok {
				return
			}
			w.mu.Lock()
			err := w.conn.WriteMessage(websocket.TextMessage, msg)
			w.mu.Unlock()

			if err != nil {
				// 检查是否是正常的关闭错误，不记录为错误
				if websocket.IsCloseError(err, websocket.CloseNormalClosure, websocket.CloseGoingAway, websocket.CloseNoStatusReceived) {
					// 连接已关闭，取消context，停止继续写入
					w.logger.Debug("WebSocket连接已关闭，停止写入文本消息", zap.Error(err))
				} else {
					w.logger.Error("写入WebSocket消息失败", zap.Error(err))
				}
				// 取消context，停止继续写入（无论什么错误）
				w.cancel()
				return
			}
		}
	}
}

// writeBinaryLoop 二进制消息写入循环
func (w *Writer) writeBinaryLoop() {
	defer w.wg.Done()
	for {
		select {
		case <-w.ctx.Done():
			return
		case data, ok := <-w.binaryChan:
			if !ok {
				return
			}
			w.mu.Lock()
			err := w.conn.WriteMessage(websocket.BinaryMessage, data)
			w.mu.Unlock()

			if err != nil {
				// 检查是否是正常的关闭错误，不记录为错误
				if websocket.IsCloseError(err, websocket.CloseNormalClosure, websocket.CloseGoingAway, websocket.CloseNoStatusReceived) {
					// 连接已关闭，取消context，停止继续写入
					w.logger.Debug("WebSocket连接已关闭，停止写入二进制消息", zap.Error(err))
				} else {
					w.logger.Error("写入WebSocket二进制消息失败", zap.Error(err))
				}
				// 取消context，停止继续写入（无论什么错误）
				w.cancel()
				return
			}
		}
	}
}

// SendASRResult 发送ASR识别结果
func (w *Writer) SendASRResult(text string) error {
	if w.isXiaozhi {
		// xiaozhi协议格式：{"type": "stt", "text": "...", "session_id": "..."}
		msg := map[string]interface{}{
			"type":       "stt",
			"text":       text,
			"session_id": w.sessionID,
		}
		return w.sendJSON(msg)
	}
	// 通用格式
	return w.sendJSON(map[string]interface{}{
		"type": "asr_result",
		"text": text,
	})
}

// SendTTSAudioWithFlowControl 发送TTS音频数据（带流控，模拟xiaozhi-esp32-server的行为）
// frameDuration: 帧时长（毫秒），默认60ms
// sendDelay: 固定延迟（毫秒），如果<=0则使用时间同步方式
func (w *Writer) SendTTSAudioWithFlowControl(data []byte, frameDuration int, sendDelay int) error {
	if frameDuration <= 0 {
		frameDuration = TTSFrameDuration
	}

	// 初始化流控状态（每个新的TTS会话开始时重置）
	now := time.Now()
	w.ttsFlowControlMu.Lock()
	if w.ttsFlowControl == nil {
		w.ttsFlowControl = &ttsFlowControl{
			packetCount:   0,
			startTime:     now,
			lastSendTime:  now,
			sendDelay:     time.Duration(sendDelay) * time.Millisecond,
			frameDuration: time.Duration(frameDuration) * time.Millisecond,
		}
	}
	flowControl := w.ttsFlowControl
	packetCount := flowControl.packetCount
	flowControl.packetCount++
	w.ttsFlowControlMu.Unlock()

	// 流控逻辑：前N个包直接发送（预缓冲），之后根据配置延迟
	if packetCount >= TTSPreBufferCount {
		w.ttsFlowControlMu.Lock()
		lastSendTime := flowControl.lastSendTime
		w.ttsFlowControlMu.Unlock()

		if flowControl.sendDelay > 0 {
			// 使用固定延迟（基于上次实际发送时间，避免累积误差）
			elapsed := now.Sub(lastSendTime)
			if elapsed < flowControl.sendDelay {
				// 如果距离上次发送时间还没到帧时长，等待剩余时间
				time.Sleep(flowControl.sendDelay - elapsed)
			}
			// 如果已经超过帧时长，立即发送（不等待）
		} else {
			// 使用时间同步方式（基于上次实际发送时间，避免累积误差）
			nextSendTime := lastSendTime.Add(flowControl.frameDuration)
			delay := time.Until(nextSendTime)
			if delay > 0 {
				// 等待到预期发送时间
				time.Sleep(delay)
			} else if delay < -20*time.Millisecond {
				// 如果延迟超过20ms（说明发送太慢了），不等待，立即发送
				// 但更新lastSendTime为当前时间，避免后续帧追赶过快
				w.ttsFlowControlMu.Lock()
				flowControl.lastSendTime = time.Now()
				w.ttsFlowControlMu.Unlock()
			}
		}
	}

	// 发送数据（阻塞等待，不丢弃数据）
	select {
	case <-w.ctx.Done():
		return w.ctx.Err()
	case w.binaryChan <- data:
		// 更新实际发送时间（用于下次计算）
		actualSendTime := time.Now()
		w.ttsFlowControlMu.Lock()
		flowControl.lastSendTime = actualSendTime
		w.ttsFlowControlMu.Unlock()
		return nil
	}
}

// SendTTSAudio 发送TTS音频数据（兼容接口，使用默认流控参数）
func (w *Writer) SendTTSAudio(data []byte) error {
	return w.SendTTSAudioWithFlowControl(data, 60, 0)
}

// ResetTTSFlowControl 重置TTS流控状态（新的TTS会话开始时调用）
func (w *Writer) ResetTTSFlowControl() {
	w.ttsFlowControlMu.Lock()
	defer w.ttsFlowControlMu.Unlock()
	w.ttsFlowControl = nil
}

// SendError 发送错误消息
func (w *Writer) SendError(message string, fatal bool) error {
	return w.sendJSON(map[string]interface{}{
		"type":    "error",
		"message": message,
		"fatal":   fatal,
	})
}

// SendConnected 发送连接成功消息
func (w *Writer) SendConnected() error {
	return w.sendJSON(map[string]interface{}{
		"type":    "connected",
		"message": "WebSocket voice connection established",
	})
}

// SendLLMResponse 发送LLM响应
func (w *Writer) SendLLMResponse(text string) error {
	return w.sendJSON(map[string]interface{}{
		"type": "llm_response",
		"text": text,
	})
}

// SendTTSStart 发送TTS开始消息
func (w *Writer) SendTTSStart(format media.StreamFormat) error {
	if w.isXiaozhi {
		// xiaozhi协议格式：{"type": "tts", "state": "start", "session_id": "..."}
		return w.sendJSON(map[string]interface{}{
			"type":       "tts",
			"state":      "start",
			"session_id": w.sessionID,
		})
	}
	// 通用格式
	return w.sendJSON(map[string]interface{}{
		"type":       "tts_start",
		"sampleRate": format.SampleRate,
		"channels":   format.Channels,
		"bitDepth":   format.BitDepth,
	})
}

// SendTTSEnd 发送TTS结束消息
func (w *Writer) SendTTSEnd() error {
	if w.isXiaozhi {
		// xiaozhi协议格式：{"type": "tts", "state": "stop", "session_id": "..."}
		return w.sendJSON(map[string]interface{}{
			"type":       "tts",
			"state":      "stop",
			"session_id": w.sessionID,
		})
	}
	// 通用格式
	return w.sendJSON(map[string]interface{}{
		"type": "tts_end",
	})
}

// SendWelcome 发送Welcome消息（xiaozhi协议）
func (w *Writer) SendWelcome(audioFormat string, sampleRate, channels int, features map[string]interface{}) (string, error) {
	// 生成会话ID（使用时间戳）
	sessionID := fmt.Sprintf("%d", time.Now().UnixNano())

	// 构建audio_params
	audioParams := map[string]interface{}{
		"format":         audioFormat,
		"sample_rate":    sampleRate,
		"channels":       channels,
		"frame_duration": 60, // 默认帧时长（毫秒）
	}

	// 构建响应消息（格式与原版xiaozhi-esp32一致）
	welcomeMsg := map[string]interface{}{
		"type":         "hello", // 注意：是 "hello" 不是 "server"
		"version":      1,
		"transport":    "websocket",
		"session_id":   sessionID,
		"audio_params": audioParams,
	}

	// 如果有features，添加到响应中
	if features != nil && len(features) > 0 {
		welcomeMsg["features"] = features
	}

	// 发送消息
	if err := w.sendJSON(welcomeMsg); err != nil {
		return "", err
	}

	// 更新sessionID
	w.SetXiaozhiMode(sessionID)

	return sessionID, nil
}

// sendJSON 发送JSON消息（异步，非阻塞）
func (w *Writer) sendJSON(data interface{}) error {
	message, err := json.Marshal(data)
	if err != nil {
		w.logger.Error("序列化消息失败", zap.Error(err))
		return err
	}

	select {
	case <-w.ctx.Done():
		return w.ctx.Err()
	case w.msgChan <- message:
		return nil
	default:
		// 缓冲区满，记录警告但不阻塞
		w.logger.Warn("消息缓冲区已满，丢弃消息", zap.String("type", getMessageType(data)))
		return nil
	}
}

// getMessageType 获取消息类型（用于日志）
func getMessageType(data interface{}) string {
	if m, ok := data.(map[string]interface{}); ok {
		if t, ok := m["type"].(string); ok {
			return t
		}
	}
	return "unknown"
}
