package protocol

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/LingByte/SoulNexus/pkg/hardware/constants"
	"github.com/gorilla/websocket"
	"go.uber.org/zap"
)

const (
	// WriterBufferSize 消息写入器缓冲区大小
	// 设置为200以应对TTS流式输出的短时激增（60ms/帧 × 200 = 12秒缓冲）
	WriterBufferSize = 200
	// TTSPreBufferCount TTS预缓冲包数量（前N个包直接发送）
	TTSPreBufferCount = 5
	// TTSFrameDuration TTS帧时长（毫秒）
	TTSFrameDuration = 60
)

// HardwareWriter hardware writer
type HardwareWriter struct {
	conn             *websocket.Conn
	logger           *zap.Logger
	mu               sync.Mutex
	msgChan          chan []byte
	binaryChan       chan []byte
	ctx              context.Context
	cancel           context.CancelFunc
	wg               sync.WaitGroup
	sessionID        string
	ttsFlowControlMu sync.Mutex
	ttsFlowControl   *ttsFlowControl
}

// ttsFlowControl TTS流控状态
type ttsFlowControl struct {
	packetCount   int                // 已发送包数量
	startTime     time.Time          // 开始时间
	lastSendTime  time.Time          // 上次实际发送时间
	sendDelay     time.Duration      // 固定延迟（如果>0则使用固定延迟，否则使用时间同步）
	frameDuration time.Duration      // 帧时长
	ctx           context.Context    // 用于中断流控
	cancel        context.CancelFunc // 取消函数
}

// NewHardwareWriter create hardware writer
func NewHardwareWriter(ctx context.Context, conn *websocket.Conn, logger *zap.Logger) *HardwareWriter {
	ctx, cancel := context.WithCancel(ctx)
	hw := &HardwareWriter{
		conn:       conn,
		logger:     logger,
		msgChan:    make(chan []byte, WriterBufferSize),
		binaryChan: make(chan []byte, WriterBufferSize),
		ctx:        ctx,
		cancel:     cancel,
		sessionID:  fmt.Sprintf("%s%d", constants.HARDWARE_WRITER_PREFIX, time.Now().UnixNano()),
	}
	hw.wg.Add(2)
	go hw.writeLoop()
	go hw.writeBinaryLoop()
	return hw
}

// Close close hardware writer
func (hw *HardwareWriter) Close() error {
	hw.cancel()
	close(hw.msgChan)
	close(hw.binaryChan)
	hw.wg.Wait()
	return nil
}

func (hw *HardwareWriter) writeLoop() {
	defer hw.wg.Done()
	for {
		select {
		case <-hw.ctx.Done():
			return
		case msg, ok := <-hw.msgChan:
			if !ok {
				return
			}
			hw.mu.Lock()
			err := hw.conn.WriteMessage(websocket.TextMessage, msg)
			hw.mu.Unlock()
			if err != nil {
				if websocket.IsCloseError(err, websocket.CloseNormalClosure, websocket.CloseGoingAway, websocket.CloseNoStatusReceived) {
					hw.logger.Debug("[Websocket Writer] --- WebSocket连接已关闭，停止写入文本消息", zap.Error(err))
				} else {
					hw.logger.Error("[Websocket Writer] --- 写入WebSocket消息失败", zap.Error(err))
				}
				hw.cancel()
				return
			}
		}
	}
}

// writeBinaryLoop 二进制消息写入循环（简单发送，不做流控）
func (hw *HardwareWriter) writeBinaryLoop() {
	defer hw.wg.Done()
	for {
		select {
		case <-hw.ctx.Done():
			return
		case data, ok := <-hw.binaryChan:
			if !ok {
				return
			}
			hw.mu.Lock()
			err := hw.conn.WriteMessage(websocket.BinaryMessage, data)
			hw.mu.Unlock()
			if err != nil {
				if websocket.IsCloseError(err, websocket.CloseNormalClosure, websocket.CloseGoingAway, websocket.CloseNoStatusReceived) {
					hw.logger.Debug("[Websocket Writer] WebSocket连接已关闭", zap.Error(err))
				} else {
					hw.logger.Error("[Websocket Writer] 写入失败", zap.Error(err))
				}
				hw.cancel()
				return
			}
		}
	}
}

// sendJSON 发送JSON消息
func (hw *HardwareWriter) sendJSON(data interface{}) error {
	message, err := json.Marshal(data)
	if err != nil {
		hw.logger.Error("[Websocket Writer] --- 序列化消息失败", zap.Error(err))
		return err
	}
	select {
	case <-hw.ctx.Done():
		return hw.ctx.Err()
	case hw.msgChan <- message:
		return nil
	default:
		return nil
	}
}

// SendASRResult 发送ASR识别结果
func (hw *HardwareWriter) SendASRResult(text string) error {
	// xiaozhi协议格式：{"type": "stt", "text": "...", "session_id": "..."}
	msg := map[string]interface{}{
		"type":       "stt",
		"text":       text,
		"session_id": hw.sessionID,
	}
	hw.logger.Info(fmt.Sprintf("[Websocket Writer] --- 发送ASR识别结果：%s", text))
	return hw.sendJSON(msg)
}

// SendError 发送错误消息
func (hw *HardwareWriter) SendError(message string, fatal bool) error {
	return hw.sendJSON(map[string]interface{}{
		"type":    "error",
		"message": message,
		"fatal":   fatal,
	})
}

// SendConnected 发送连接成功消息
func (hw *HardwareWriter) SendConnected() error {
	return hw.sendJSON(map[string]interface{}{
		"type":    "connected",
		"message": "WebSocket voice connection established",
	})
}

// SendPong 发送pong响应
func (hw *HardwareWriter) SendPong() error {
	return hw.sendJSON(map[string]interface{}{
		"type":       "pong",
		"session_id": hw.sessionID,
	})
}

// SendChangeSpeakerResult 发送切换发音人结果
func (hw *HardwareWriter) SendChangeSpeakerResult(speakerID string, success bool, message string) error {
	return hw.sendJSON(map[string]interface{}{
		"type":       "speaker_changed",
		"speaker_id": speakerID,
		"success":    success,
		"message":    message,
		"session_id": hw.sessionID,
	})
}

// SendLLMResponse 发送LLM响应
func (hw *HardwareWriter) SendLLMResponse(text string) error {
	return hw.sendJSON(map[string]interface{}{
		"type": "llm_response",
		"text": text,
	})
}

// SendTTSStart 发送TTS开始消息
func (hw *HardwareWriter) SendTTSStart() error {
	// {"type": "tts", "state": "start", "session_id": "...", "audio_params": {...}}
	hw.logger.Info("[Websocket Writer] 发送 TTS 开始消息", zap.String("session_id", hw.sessionID))
	return hw.sendJSON(map[string]interface{}{
		"type":       "tts",
		"state":      "start",
		"session_id": hw.sessionID,
		"audio_params": map[string]interface{}{
			"codec":          "opus",
			"sample_rate":    16000,
			"channels":       1,
			"frame_duration": 60, // 毫秒
			"bit_depth":      16,
		},
	})
}

// SendTTSEnd 发送TTS结束消息
func (hw *HardwareWriter) SendTTSEnd() error {
	hw.logger.Info("[Websocket Writer] 发送 TTS 结束消息", zap.String("session_id", hw.sessionID))
	return hw.sendJSON(map[string]interface{}{
		"type":       "tts",
		"state":      "stop",
		"session_id": hw.sessionID,
	})
}

// SendAbortConfirmation 发送中断确认消息
func (hw *HardwareWriter) SendAbortConfirmation() error {
	hw.logger.Info("[Websocket Writer] 发送中断确认消息", zap.String("session_id", hw.sessionID))
	return hw.sendJSON(map[string]interface{}{
		"type":       "abort",
		"state":      "confirmed",
		"session_id": hw.sessionID,
	})
}

// SendWelcome 发送Welcome消息
func (hw *HardwareWriter) SendWelcome(audioFormat string, sampleRate, channels int, features map[string]interface{}) (string, error) {
	sessionID := fmt.Sprintf("hardware_%d", time.Now().UnixNano())
	audioParams := map[string]interface{}{
		"format":         audioFormat,
		"sample_rate":    sampleRate,
		"channels":       channels,
		"frame_duration": 60,
	}
	welcomeMsg := map[string]interface{}{
		"type":         "hello",
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
	if err := hw.sendJSON(welcomeMsg); err != nil {
		return "", err
	}

	// 更新sessionID
	hw.sessionID = sessionID

	return sessionID, nil
}

// SendTTSAudioWithFlowControl 发送TTS音频数据（带流控，模拟xiaozhi-esp32-server的行为）
// frameDuration: 帧时长（毫秒），默认60ms
// sendDelay: 固定延迟（毫秒），如果<=0则使用时间同步方式
func (hw *HardwareWriter) SendTTSAudioWithFlowControl(data []byte, frameDuration int, sendDelay int) error {
	if frameDuration <= 0 {
		frameDuration = TTSFrameDuration
	}

	// 初始化流控状态（每个新的TTS会话开始时重置）
	now := time.Now()
	hw.ttsFlowControlMu.Lock()
	if hw.ttsFlowControl == nil {
		ctx, cancel := context.WithCancel(hw.ctx)
		hw.ttsFlowControl = &ttsFlowControl{
			packetCount:   0,
			startTime:     now,
			lastSendTime:  now,
			sendDelay:     time.Duration(sendDelay) * time.Millisecond,
			frameDuration: time.Duration(frameDuration) * time.Millisecond,
			ctx:           ctx,
			cancel:        cancel,
		}
	}
	flowControl := hw.ttsFlowControl
	flowCtx := flowControl.ctx
	packetCount := flowControl.packetCount
	flowControl.packetCount++
	hw.ttsFlowControlMu.Unlock()

	// 流控逻辑：前N个包直接发送（预缓冲），之后根据配置延迟
	if packetCount >= TTSPreBufferCount {
		hw.ttsFlowControlMu.Lock()
		lastSendTime := flowControl.lastSendTime
		hw.ttsFlowControlMu.Unlock()

		if flowControl.sendDelay > 0 {
			// 使用固定延迟（基于上次实际发送时间，避免累积误差）
			elapsed := now.Sub(lastSendTime)
			if elapsed < flowControl.sendDelay {
				// 如果距离上次发送时间还没到帧时长，等待剩余时间
				delay := flowControl.sendDelay - elapsed
				select {
				case <-flowCtx.Done():
					// 流控被中断（新的TTS开始）
					return fmt.Errorf("TTS flow control interrupted")
				case <-time.After(delay):
					// 正常等待完成
				}
			}
		} else {
			// 使用时间同步方式（基于上次实际发送时间，避免累积误差）
			nextSendTime := lastSendTime.Add(flowControl.frameDuration)
			delay := time.Until(nextSendTime)
			if delay > 0 {
				// 等待到预期发送时间
				select {
				case <-flowCtx.Done():
					// 流控被中断（新的TTS开始）
					return fmt.Errorf("TTS flow control interrupted")
				case <-time.After(delay):
					// 正常等待完成
				}
			} else if delay < -20*time.Millisecond {
				hw.ttsFlowControlMu.Lock()
				flowControl.lastSendTime = time.Now()
				hw.ttsFlowControlMu.Unlock()
			}
		}
	}

	// 发送数据（阻塞式，提供背压保护）
	// 如果 channel 满了，会阻塞等待，而不是丢包
	select {
	case <-hw.ctx.Done():
		return hw.ctx.Err()
	case <-flowCtx.Done():
		// 流控被中断（新的TTS开始）
		return fmt.Errorf("TTS flow control interrupted")
	case hw.binaryChan <- data:
		// 更新实际发送时间（用于下次计算）
		actualSendTime := time.Now()
		hw.ttsFlowControlMu.Lock()
		flowControl.lastSendTime = actualSendTime
		hw.ttsFlowControlMu.Unlock()
		return nil
	}
}

// ResetTTSFlowControl 重置TTS流控状态（新的TTS会话开始时调用）
func (hw *HardwareWriter) ResetTTSFlowControl() {
	hw.ttsFlowControlMu.Lock()
	defer hw.ttsFlowControlMu.Unlock()

	// 取消旧的流控context，中断正在sleep的goroutine
	if hw.ttsFlowControl != nil && hw.ttsFlowControl.cancel != nil {
		hw.logger.Info("[Writer] 取消旧的TTS流控")
		hw.ttsFlowControl.cancel()
	}

	hw.ttsFlowControl = nil
}

// ClearTTSQueue 清空TTS音频队列（用于中断旧的TTS播放）
func (hw *HardwareWriter) ClearTTSQueue() {
	clearedCount := 0
	// 清空binaryChan中的所有数据
	for {
		select {
		case <-hw.binaryChan:
			// 丢弃旧的音频数据
			clearedCount++
		default:
			// channel已空
			if clearedCount > 0 {
				hw.logger.Info("[Writer] 清空TTS音频队列",
					zap.Int("cleared_packets", clearedCount))
			}
			return
		}
	}
}

// SendTTSAudio 发送TTS音频数据
func (hw *HardwareWriter) SendTTSAudio(data []byte) error {
	// 直接放入 channel
	select {
	case <-hw.ctx.Done():
		return hw.ctx.Err()
	case hw.binaryChan <- data:
		return nil
	default:
		return fmt.Errorf("TTS音频发送通道满或已关闭")
	}
}

// GetBinaryChannelLength 获取二进制通道中待发送的数据包数量
func (hw *HardwareWriter) GetBinaryChannelLength() int {
	return len(hw.binaryChan)
}
