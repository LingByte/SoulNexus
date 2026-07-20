package speaker

import (
	"fmt"
	"strings"

	"github.com/LingByte/SoulNexus/internal/models"
	"github.com/LingByte/SoulNexus/pkg/dialog/callbinding"
	"go.uber.org/zap"
)

// PrepareCallSpeakerBinding ensures tenant/assistant ids are on the call and
// appends a system-prompt hint when the assistant has bound voiceprints.
func PrepareCallSpeakerBinding(callID string, tenantID, assistantID uint, lg *zap.Logger) string {
	callID = strings.TrimSpace(callID)
	if callID == "" {
		return ""
	}
	if tenantID > 0 {
		callbinding.SetTenantID(callID, tenantID)
	}
	if assistantID > 0 {
		callbinding.SetAssistantID(callID, assistantID)
	}

	names, hint := assistantBoundNamesAndHint(tenantID, assistantID)
	callbinding.SetSpeakerRoster(callID, names)

	if lg != nil {
		lg.Info("speaker: prepare call binding",
			zap.String("call_id", callID),
			zap.Uint("tenant_id", tenantID),
			zap.Uint("assistant_id", assistantID),
			zap.Int("bound_voiceprints", len(names)),
			zap.Strings("speakers", names),
		)
	}
	return hint
}

// AssistantBoundPromptHint returns an LLM instruction block when voiceprints are linked to the assistant.
func AssistantBoundPromptHint(tenantID, assistantID uint) string {
	_, hint := assistantBoundNamesAndHint(tenantID, assistantID)
	return hint
}

func assistantBoundNamesAndHint(tenantID, assistantID uint) ([]string, string) {
	if tenantID == 0 || assistantID == 0 {
		return nil, ""
	}
	handle := getDB()
	if handle == nil {
		return nil, ""
	}
	rows, err := models.ListVoiceprintProfilesByAssistantID(handle, tenantID, assistantID)
	if err != nil || len(rows) == 0 {
		return nil, ""
	}
	names := make([]string, 0, len(rows))
	for _, r := range rows {
		if r.Status != models.VoiceprintStatusActive {
			continue
		}
		n := strings.TrimSpace(r.Name)
		if n == "" {
			n = r.FeatureID
		}
		names = append(names, n)
	}
	if len(names) == 0 {
		return nil, ""
	}
	max := 8
	shown := names
	extra := ""
	if len(shown) > max {
		shown = shown[:max]
		extra = "…"
	}
	hint := fmt.Sprintf(
		"【声纹身份】本助手已绑定 %d 个说话人声纹：%s%s。"+
			"会话中应主动通过声纹确认说话人身份：用户开口后立刻调用 identify_speaker（无需传 audioBase64，系统自动取上一句语音）；"+
			"也可用 get_speaker_context 查看本通识别结果与已绑定名单。"+
			"识别成功后按该说话人凭证调用业务工具，禁止编造身份。",
		len(names),
		strings.Join(shown, "、"),
		extra,
	)
	return names, hint
}
