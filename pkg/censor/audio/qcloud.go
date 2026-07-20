package audio

import (
	"fmt"
	"os"
	"strings"

	ams "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/ams/v20201229"
	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common"
	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common/profile"
)

const (
	QCloudAMSDefaultRegion = "ap-guangzhou"
)

// QCloudAudioCensor uses Tencent AMS CreateAudioModerationTask / DescribeTaskDetail.
type QCloudAudioCensor struct {
	SecretID  string
	SecretKey string
	Region    string
	BizType   string
	Client    *ams.Client
}

// NewQCloudAudioCensor creates an AMS audio moderation client.
func NewQCloudAudioCensor(secretID, secretKey, region string) (*QCloudAudioCensor, error) {
	if secretID == "" || secretKey == "" {
		return nil, fmt.Errorf("secretID and secretKey cannot be empty")
	}
	if region == "" {
		region = strings.TrimSpace(os.Getenv("QCLOUD_CENSOR_REGION"))
	}
	if region == "" {
		region = QCloudAMSDefaultRegion
	}
	biz := strings.TrimSpace(os.Getenv("QCLOUD_CENSOR_AUDIO_BIZ_TYPE"))
	if biz == "" {
		biz = "default"
	}
	credential := common.NewCredential(secretID, secretKey)
	cpf := profile.NewClientProfile()
	cpf.HttpProfile.Endpoint = "ams.tencentcloudapi.com"
	client, err := ams.NewClient(credential, region, cpf)
	if err != nil {
		return nil, fmt.Errorf("failed to create Tencent AMS client: %w", err)
	}
	return &QCloudAudioCensor{
		SecretID:  secretID,
		SecretKey: secretKey,
		Region:    region,
		BizType:   biz,
		Client:    client,
	}, nil
}

// SubmitCensorAudio creates a point-of-presence AUDIO moderation task.
func (c *QCloudAudioCensor) SubmitCensorAudio(audioURL string) (string, error) {
	if audioURL == "" {
		return "", fmt.Errorf("audioURL cannot be empty")
	}
	req := ams.NewCreateAudioModerationTaskRequest()
	req.BizType = common.StringPtr(c.BizType)
	req.Type = common.StringPtr("AUDIO")
	req.Tasks = []*ams.TaskInput{{
		Input: &ams.StorageInfo{
			Type: common.StringPtr("URL"),
			Url:  common.StringPtr(audioURL),
		},
	}}
	resp, err := c.Client.CreateAudioModerationTask(req)
	if err != nil {
		return "", fmt.Errorf("qcloud CreateAudioModerationTask: %w", err)
	}
	if resp == nil || resp.Response == nil || len(resp.Response.Results) == 0 {
		return "", fmt.Errorf("qcloud CreateAudioModerationTask: empty results")
	}
	r0 := resp.Response.Results[0]
	if r0 == nil {
		return "", fmt.Errorf("qcloud CreateAudioModerationTask: nil result")
	}
	if r0.Code != nil && strings.ToUpper(strings.TrimSpace(*r0.Code)) != "OK" {
		msg := ""
		if r0.Message != nil {
			msg = *r0.Message
		}
		return "", fmt.Errorf("qcloud CreateAudioModerationTask: code=%s message=%s", *r0.Code, msg)
	}
	if r0.TaskId == nil || strings.TrimSpace(*r0.TaskId) == "" {
		return "", fmt.Errorf("qcloud CreateAudioModerationTask: missing TaskId")
	}
	return strings.TrimSpace(*r0.TaskId), nil
}

// GetCensorResult returns a JobSnapshot.
func (c *QCloudAudioCensor) GetCensorResult(taskID string) (interface{}, error) {
	return c.PollCensorAudio(taskID)
}

// PollCensorAudio queries DescribeTaskDetail.
func (c *QCloudAudioCensor) PollCensorAudio(taskID string) (*JobSnapshot, error) {
	if taskID == "" {
		return nil, fmt.Errorf("taskID cannot be empty")
	}
	req := ams.NewDescribeTaskDetailRequest()
	req.TaskId = common.StringPtr(taskID)
	req.ShowAllSegments = common.BoolPtr(false)
	resp, err := c.Client.DescribeTaskDetail(req)
	if err != nil {
		return nil, fmt.Errorf("qcloud DescribeTaskDetail: %w", err)
	}
	if resp == nil || resp.Response == nil {
		return nil, fmt.Errorf("qcloud DescribeTaskDetail: empty response")
	}
	body := resp.Response
	snap := &JobSnapshot{Raw: body}
	status := ""
	if body.Status != nil {
		status = strings.ToUpper(strings.TrimSpace(*body.Status))
	}
	switch status {
	case "PENDING":
		snap.Status = JobWaiting
	case "RUNNING":
		snap.Status = JobDoing
	case "FINISH":
		snap.Status = JobFinished
	case "ERROR", "CANCELLED":
		snap.Status = JobFailed
		snap.Error = status
	default:
		if status == "" {
			snap.Status = JobDoing
		} else {
			snap.Status = JobDoing
			snap.Msg = status
		}
	}
	if snap.Status != JobFinished {
		return snap, nil
	}
	sug := ""
	if body.Suggestion != nil {
		sug = strings.TrimSpace(*body.Suggestion)
	}
	switch strings.ToLower(sug) {
	case "pass":
		snap.Suggestion = SuggestionPass
	case "review":
		snap.Suggestion = SuggestionReview
	case "block":
		snap.Suggestion = SuggestionBlock
	default:
		snap.Suggestion = SuggestionPass
	}
	if len(body.Labels) > 0 && body.Labels[0] != nil {
		if body.Labels[0].Label != nil {
			snap.Label = strings.ToLower(strings.TrimSpace(*body.Labels[0].Label))
		}
		if body.Labels[0].Score != nil {
			snap.Score = float64(*body.Labels[0].Score) / 100.0
		}
	}
	if snap.Label == "" {
		snap.Label = "normal"
	}
	if body.AudioText != nil {
		snap.Msg = strings.TrimSpace(*body.AudioText)
		if len(snap.Msg) > 200 {
			snap.Msg = snap.Msg[:200]
		}
	}
	return snap, nil
}
