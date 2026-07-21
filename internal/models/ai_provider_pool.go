package models

import (
	"encoding/json"
	"strings"

	"github.com/LingByte/SoulNexus/internal/constants"
	constants2 "github.com/LingByte/SoulNexus/pkg/constants"
	"github.com/LingByte/SoulNexus/pkg/utils/common"
	"gorm.io/datatypes"
)

// AIProviderPool is a platform-managed vendor config bucket (号池) for ASR/TTS/LLM/Realtime.
type AIProviderPool struct {
	common.BaseModel
	Name        string         `json:"name" gorm:"size:128;not null"`
	Modality    string         `json:"modality" gorm:"size:32;index;not null"`
	Provider    string         `json:"provider" gorm:"size:64;index;not null"`
	Config      datatypes.JSON `json:"config" gorm:"type:json;not null"`
	VoiceIDs    datatypes.JSON `json:"voiceIds" gorm:"column:voice_ids;type:json"`
	Priority    int            `json:"priority" gorm:"not null;default:0"`
	Enabled     bool           `json:"enabled" gorm:"not null;default:true;index"`
	QuotaLimit  int64          `json:"quotaLimit" gorm:"column:quota_limit;not null;default:0"`
	QuotaUsed   int64          `json:"quotaUsed" gorm:"column:quota_used;not null;default:0"`
	Description string         `json:"description,omitempty" gorm:"size:512"`
}

func (AIProviderPool) TableName() string {
	return constants2.AI_PROVIDER_POOL_TABLE_NAME
}

func NormalizeAIPoolModality(m string) string {
	switch strings.TrimSpace(strings.ToLower(m)) {
	case constants.AIPoolModalityASR:
		return constants.AIPoolModalityASR
	case constants.AIPoolModalityTTS:
		return constants.AIPoolModalityTTS
	case constants.AIPoolModalityLLM:
		return constants.AIPoolModalityLLM
	case constants.AIPoolModalityRealtime:
		return constants.AIPoolModalityRealtime
	default:
		return ""
	}
}

func parsePoolVoiceIDs(raw datatypes.JSON) []string {
	s := strings.TrimSpace(string(raw))
	if s == "" || s == "null" {
		return nil
	}
	var ids []string
	if err := json.Unmarshal([]byte(s), &ids); err != nil {
		return nil
	}
	out := make([]string, 0, len(ids))
	for _, id := range ids {
		id = strings.TrimSpace(id)
		if id != "" {
			out = append(out, id)
		}
	}
	return out
}

func poolHasQuota(p AIProviderPool) bool {
	if p.QuotaLimit <= 0 {
		return true
	}
	return p.QuotaUsed < p.QuotaLimit
}

func containsStar(ids []string) bool {
	for _, id := range ids {
		if normalizeVoiceKey(id) == "*" {
			return true
		}
	}
	return false
}
