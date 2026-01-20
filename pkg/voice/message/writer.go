package message

import (
	"context"
	"encoding/json"
	"sync"

	"github.com/LingByte/SoulNexus/pkg/media"
	"github.com/gorilla/websocket"
	"go.uber.org/zap"
)

const (
	// WriterBufferSize 消息写入器缓冲区大小
	WriterBufferSize = 100
)

// Writer 消息写入器实现
type Writer struct {
	conn       *websocket.Conn
	logger     *zap.Logger
	mu         sync.Mutex
	msgChan    chan []byte
	binaryChan chan []byte
	ctx        context.Context
	cancel     context.CancelFunc
	wg         sync.WaitGroup
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
	}

	// 启动异步写入goroutine
	w.wg.Add(2)
	go w.writeLoop()
	go w.writeBinaryLoop()

	return w
}

// Close 关闭写入器
func (w *Writer) Close() error {
	// 先取消上下文，停止接收新消息
	w.cancel()

	// 关闭channel，让goroutine检测到并退出
	// 注意：关闭channel后，goroutine会从channel读取到零值并检测到ok=false
	close(w.msgChan)
	close(w.binaryChan)

	// 等待写入goroutine完成
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
	return w.sendJSON(map[string]interface{}{
		"type": "asr_result",
		"text": text,
	})
}

// SendTTSAudio 发送TTS音频数据（异步，非阻塞）
func (w *Writer) SendTTSAudio(data []byte) error {
	select {
	case <-w.ctx.Done():
		return w.ctx.Err()
	case w.binaryChan <- data:
		return nil
	default:
		// 缓冲区满，记录警告但不阻塞
		w.logger.Warn("TTS音频缓冲区已满，丢弃数据")
		return nil
	}
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
		"message": "连接成功",
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
	return w.sendJSON(map[string]interface{}{
		"type":       "tts_start",
		"sampleRate": format.SampleRate,
		"channels":   format.Channels,
		"bitDepth":   format.BitDepth,
	})
}

// SendTTSEnd 发送TTS结束消息
func (w *Writer) SendTTSEnd() error {
	return w.sendJSON(map[string]interface{}{
		"type": "tts_end",
	})
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
