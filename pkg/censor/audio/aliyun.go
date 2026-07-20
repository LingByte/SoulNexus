package audio

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/alibabacloud-go/darabonba-openapi/v2/client"
	green "github.com/alibabacloud-go/green-20220302/v2/client"
	"github.com/alibabacloud-go/tea/tea"
)

const (
	AliyunGreenDefaultEndpoint = "green-cip.cn-shanghai.aliyuncs.com"
	AliyunAudioService         = "audio_media_detection"
	aliyunCodeOK               = 200
	aliyunCodeProcessing       = 280
)

// AliyunAudioCensor uses Green VoiceModeration / VoiceModerationResult.
type AliyunAudioCensor struct {
	AccessKeyID     string
	AccessKeySecret string
	Endpoint        string
	Service         string
	Client          *green.Client
}

// NewAliyunAudioCensor creates a Green voice moderation client.
func NewAliyunAudioCensor(accessKeyID, accessKeySecret, endpoint string) (*AliyunAudioCensor, error) {
	if accessKeyID == "" || accessKeySecret == "" {
		return nil, fmt.Errorf("accessKeyID and accessKeySecret cannot be empty")
	}
	if endpoint == "" {
		endpoint = strings.TrimSpace(os.Getenv("ALIYUN_CENSOR_GREEN_ENDPOINT"))
	}
	if endpoint == "" {
		endpoint = AliyunGreenDefaultEndpoint
	}
	service := strings.TrimSpace(os.Getenv("ALIYUN_CENSOR_AUDIO_SERVICE"))
	if service == "" {
		service = AliyunAudioService
	}
	cfg := &client.Config{
		AccessKeyId:     tea.String(accessKeyID),
		AccessKeySecret: tea.String(accessKeySecret),
		Endpoint:        tea.String(endpoint),
	}
	greenClient, err := green.NewClient(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create Alibaba Cloud Green client: %w", err)
	}
	return &AliyunAudioCensor{
		AccessKeyID:     accessKeyID,
		AccessKeySecret: accessKeySecret,
		Endpoint:        endpoint,
		Service:         service,
		Client:          greenClient,
	}, nil
}

// SubmitCensorAudio submits an async voice moderation task and returns taskId.
func (c *AliyunAudioCensor) SubmitCensorAudio(audioURL string) (string, error) {
	if audioURL == "" {
		return "", fmt.Errorf("audioURL cannot be empty")
	}
	params, err := json.Marshal(map[string]string{"url": audioURL})
	if err != nil {
		return "", err
	}
	req := &green.VoiceModerationRequest{
		Service:           tea.String(c.Service),
		ServiceParameters: tea.String(string(params)),
	}
	resp, err := c.Client.VoiceModeration(req)
	if err != nil {
		return "", fmt.Errorf("aliyun VoiceModeration: %w", err)
	}
	if resp == nil || resp.Body == nil {
		return "", fmt.Errorf("aliyun VoiceModeration: empty response")
	}
	code := tea.Int32Value(resp.Body.Code)
	if code != aliyunCodeOK {
		return "", fmt.Errorf("aliyun VoiceModeration: code=%d message=%s", code, tea.StringValue(resp.Body.Message))
	}
	if resp.Body.Data == nil || tea.StringValue(resp.Body.Data.TaskId) == "" {
		return "", fmt.Errorf("aliyun VoiceModeration: missing taskId")
	}
	return tea.StringValue(resp.Body.Data.TaskId), nil
}

// GetCensorResult returns a JobSnapshot (also used by PollCensorAudio).
func (c *AliyunAudioCensor) GetCensorResult(taskID string) (interface{}, error) {
	return c.PollCensorAudio(taskID)
}

// PollCensorAudio queries VoiceModerationResult.
// Code 280 = still processing; 200 = finished (may be none/high risk).
func (c *AliyunAudioCensor) PollCensorAudio(taskID string) (*JobSnapshot, error) {
	if taskID == "" {
		return nil, fmt.Errorf("taskID cannot be empty")
	}
	params, err := json.Marshal(map[string]string{"taskId": taskID})
	if err != nil {
		return nil, err
	}
	req := &green.VoiceModerationResultRequest{
		Service:           tea.String(c.Service),
		ServiceParameters: tea.String(string(params)),
	}
	resp, err := c.Client.VoiceModerationResult(req)
	if err != nil {
		return nil, fmt.Errorf("aliyun VoiceModerationResult: %w", err)
	}
	if resp == nil || resp.Body == nil {
		return nil, fmt.Errorf("aliyun VoiceModerationResult: empty response")
	}
	code := tea.Int32Value(resp.Body.Code)
	msg := tea.StringValue(resp.Body.Message)
	snap := &JobSnapshot{Raw: resp.Body, Msg: msg}
	switch code {
	case aliyunCodeProcessing:
		snap.Status = JobDoing
		return snap, nil
	case aliyunCodeOK:
		snap.Status = JobFinished
	default:
		snap.Status = JobFailed
		snap.Error = fmt.Sprintf("code=%d message=%s", code, msg)
		return snap, nil
	}
	data := resp.Body.Data
	if data == nil {
		snap.Suggestion = SuggestionPass
		snap.Label = "normal"
		return snap, nil
	}
	risk := strings.ToLower(strings.TrimSpace(tea.StringValue(data.RiskLevel)))
	switch risk {
	case "high":
		snap.Suggestion = SuggestionBlock
	case "medium":
		snap.Suggestion = SuggestionReview
	case "low", "none", "":
		snap.Suggestion = SuggestionPass
	default:
		snap.Suggestion = SuggestionReview
	}
	// Prefer first risky slice label/score.
	for _, slice := range data.SliceDetails {
		if slice == nil {
			continue
		}
		if lbl := strings.TrimSpace(tea.StringValue(slice.Labels)); lbl != "" {
			snap.Label = strings.Split(lbl, ",")[0]
		}
		if slice.Score != nil {
			snap.Score = float64(*slice.Score)
			if snap.Score > 1 {
				snap.Score = snap.Score / 100.0
			}
		}
		if snap.Label != "" {
			break
		}
	}
	if snap.Label == "" {
		if risk == "" || risk == "none" {
			snap.Label = "normal"
		} else {
			snap.Label = risk
		}
	}
	return snap, nil
}
