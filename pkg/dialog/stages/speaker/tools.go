package speaker

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/LingByte/SoulNexus/pkg/dialog/callbinding"
	"github.com/LingByte/SoulNexus/pkg/utils/audutil"
	"github.com/LingByte/lingllm/voiceprint"
	"go.uber.org/zap"
)

const (
	GetContextToolName = "get_speaker_context"
	IdentifyToolName   = "identify_speaker"
)

const GetContextToolDescription = "查询本通已绑定的说话人上下文（姓名、角色属性、是否已声纹确认）。不含任何 token/密钥。身份不明时先调用此工具或 identify_speaker。"

const IdentifyToolDescription = "用用户语音做声纹 1:N 识别并绑定本通说话人上下文。会话中无需传 audioBase64（系统自动使用上一句用户音频）；仅在调试/HTTP 场景才传 WAV base64。识别成功后后续工具可按说话人凭证调用业务 API。"

var GetContextToolParams = json.RawMessage(`{"type":"object","properties":{},"additionalProperties":false}`)

var IdentifyToolParams = json.RawMessage(`{
	"type":"object",
	"properties":{
		"audioBase64":{"type":"string","description":"optional WAV base64; omit on live calls"},
		"threshold":{"type":"number","description":"optional similarity threshold"}
	},
	"required":[],
	"additionalProperties":false
}`)

// ExecuteGetContextTool returns public speaker JSON for the call.
func ExecuteGetContextTool(callID string) string {
	roster := callbinding.GetSpeakerRoster(callID)
	ctx, ok := callbinding.GetSpeakerContext(callID)
	if !ok {
		if len(roster) == 0 {
			return `{"bound":false,"identified":false,"message":"本助手未绑定声纹，或本通尚未识别说话人"}`
		}
		b, _ := json.Marshal(map[string]any{
			"bound":      true,
			"identified": false,
			"roster":     roster,
			"message":    "助手已绑定声纹候选人，但本通尚未完成声纹识别；请调用 identify_speaker",
		})
		return string(b)
	}
	var speaker any
	_ = json.Unmarshal([]byte(PublicJSON(ctx)), &speaker)
	b, _ := json.Marshal(map[string]any{
		"bound":      true,
		"identified": true,
		"roster":     roster,
		"speaker":    speaker,
	})
	return string(b)
}

// ExecuteIdentifyTool runs identify from tool args and binds context.
func ExecuteIdentifyTool(
	ctx context.Context,
	callID string,
	tenantID uint,
	assistantID uint,
	args map[string]interface{},
	lg *zap.Logger,
) string {
	if strings.TrimSpace(callID) == "" || tenantID == 0 {
		return `{"ok":false,"error":"missing call/tenant"}`
	}
	audio, src, err := resolveIdentifyAudio(callID, args)
	if err != nil || len(audio) == 0 {
		msg := "no user audio for identify; wait for speech then retry"
		if err != nil {
			msg = err.Error()
		}
		return `{"ok":false,"error":` + jsonString(msg) + `}`
	}
	if lg != nil {
		lg.Info("speaker: identify audio resolved",
			zap.String("call_id", callID),
			zap.String("source", src),
			zap.Int("wav_bytes", len(audio)),
		)
	}
	threshold := 0.0
	switch v := args["threshold"].(type) {
	case float64:
		threshold = v
	}
	var aid *uint
	if assistantID > 0 {
		aid = &assistantID
	}
	bound, out, err := IdentifyAndBind(ctx, callID, tenantID, aid, audio, nil, threshold, lg)
	if err != nil {
		return `{"ok":false,"error":` + jsonString(err.Error()) + `}`
	}
	match := false
	score := 0.0
	if out != nil {
		match = out.IsMatch
		score = out.Score
	}
	return `{"ok":true,"isMatch":` + boolStr(match) + `,"score":` + floatStr(score) + `,"speaker":` + PublicJSON(bound) + `}`
}

// resolveIdentifyAudio prefers valid tool WAV; otherwise uses call-buffered PCM.
func resolveIdentifyAudio(callID string, args map[string]interface{}) ([]byte, string, error) {
	if args != nil {
		if raw, _ := args["audioBase64"].(string); strings.TrimSpace(raw) != "" {
			audio, err := decodeAudioBase64(raw)
			if err == nil && len(audio) > 0 && voiceprint.ValidateWAVFormat(audio) == nil {
				return audio, "audioBase64", nil
			}
		}
	}
	pcm, sr, ok := callbinding.GetUserUtterancePCM(callID)
	if !ok {
		return nil, "", fmt.Errorf("no buffered user audio on call")
	}
	wav, err := audutil.MonoPCM16ToWAV(pcm, sr)
	if err != nil {
		return nil, "", err
	}
	return wav, "call_utterance", nil
}

func decodeAudioBase64(b64 string) ([]byte, error) {
	b64 = strings.TrimSpace(b64)
	if b64 == "" {
		return nil, nil
	}
	if i := strings.Index(b64, ","); i >= 0 && strings.Contains(strings.ToLower(b64[:i]), "base64") {
		b64 = b64[i+1:]
	}
	return base64.StdEncoding.DecodeString(b64)
}

func jsonString(s string) string {
	b, err := json.Marshal(s)
	if err != nil {
		return `""`
	}
	return string(b)
}

func boolStr(v bool) string {
	if v {
		return "true"
	}
	return "false"
}

func floatStr(v float64) string {
	b, _ := json.Marshal(v)
	return string(b)
}
