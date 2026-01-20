package protocol

import (
	"encoding/json"
	"fmt"
	"log"
)

// MessageHandler 消息处理器接口
type MessageHandler interface {
	HandleInit(msg *Message) error
	HandleOffer(msg *Message) error
	HandleAnswer(msg *Message) error
	HandleICECandidate(msg *Message) error
	HandleConnected(msg *Message) error
	HandleTextMessage(msg *Message) error
	HandleTextResponse(msg *Message) error
	HandleASRStart(msg *Message) error
	HandleASRResult(msg *Message) error
	HandleASRInterim(msg *Message) error
	HandleASRStop(msg *Message) error
	HandleTTSRequest(msg *Message) error
	HandleTTSStart(msg *Message) error
	HandleTTSComplete(msg *Message) error
	HandleReady(msg *Message) error
	HandlePing(msg *Message) error
	HandlePong(msg *Message) error
	HandleError(msg *Message) error
	HandleDisconnect(msg *Message) error
}

// RouteMessage 路由消息到对应的处理器
func RouteMessage(handler MessageHandler, rawData []byte) error {
	var msg Message
	if err := json.Unmarshal(rawData, &msg); err != nil {
		return fmt.Errorf("failed to unmarshal message: %w", err)
	}

	switch msg.Type {
	case TypeInit:
		return handler.HandleInit(&msg)
	case TypeOffer:
		return handler.HandleOffer(&msg)
	case TypeAnswer:
		return handler.HandleAnswer(&msg)
	case TypeICECandidate:
		return handler.HandleICECandidate(&msg)
	case TypeConnected:
		return handler.HandleConnected(&msg)
	case TypeTextMessage:
		return handler.HandleTextMessage(&msg)
	case TypeTextResponse:
		return handler.HandleTextResponse(&msg)
	case TypeASRStart:
		return handler.HandleASRStart(&msg)
	case TypeASRResult:
		return handler.HandleASRResult(&msg)
	case TypeASRInterim:
		return handler.HandleASRInterim(&msg)
	case TypeASRStop:
		return handler.HandleASRStop(&msg)
	case TypeTTSRequest:
		return handler.HandleTTSRequest(&msg)
	case TypeTTSStart:
		return handler.HandleTTSStart(&msg)
	case TypeTTSComplete:
		return handler.HandleTTSComplete(&msg)
	case TypeReady:
		return handler.HandleReady(&msg)
	case TypePing:
		return handler.HandlePing(&msg)
	case TypePong:
		return handler.HandlePong(&msg)
	case TypeError:
		return handler.HandleError(&msg)
	case TypeDisconnect:
		return handler.HandleDisconnect(&msg)
	default:
		log.Printf("Unknown message type: %s", msg.Type)
		return fmt.Errorf("unknown message type: %s", msg.Type)
	}
}

// ValidateMessage 验证消息格式
func ValidateMessage(msg *Message) error {
	if msg.Type == "" {
		return fmt.Errorf("message type is required")
	}
	if msg.SessionID == "" && msg.Type != TypeInit {
		return fmt.Errorf("session_id is required for message type: %s", msg.Type)
	}
	return nil
}

// ExtractData 从消息中提取数据到指定类型
func ExtractData[T any](msg *Message) (*T, error) {
	if msg.Data == nil {
		return nil, fmt.Errorf("message data is nil")
	}

	// 如果Data已经是目标类型，直接返回
	if data, ok := msg.Data.(*T); ok {
		return data, nil
	}

	// 尝试通过JSON序列化/反序列化转换
	dataJSON, err := json.Marshal(msg.Data)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal data: %w", err)
	}

	var data T
	if err := json.Unmarshal(dataJSON, &data); err != nil {
		return nil, fmt.Errorf("failed to unmarshal data: %w", err)
	}

	return &data, nil
}
