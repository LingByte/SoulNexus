package voice

import (
	"context"

	"github.com/LingByte/SoulNexus/internal/models"
	"github.com/LingByte/SoulNexus/pkg/voice/asr"
	"github.com/gorilla/websocket"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// SessionConfig 会话配置
type SessionConfig struct {
	Conn         *websocket.Conn
	Credential   *models.UserCredential
	AssistantID  int
	Language     string
	Speaker      string
	Temperature  float64
	MaxTokens    int
	SystemPrompt string
	KnowledgeKey string
	LLMModel     string
	DB           *gorm.DB
	Logger       *zap.Logger
	Context      context.Context
	ASRPool      *asr.Pool // ASR连接池（可选）
	// VAD 配置
	EnableVAD            bool    // 是否启用VAD
	VADThreshold         float64 // VAD阈值
	VADConsecutiveFrames int     // 需要连续超过阈值的帧数
}
