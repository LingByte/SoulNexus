package text

import (
	"encoding/json"
	"fmt"

	"github.com/LingByte/SoulNexus/pkg/utils"
	"github.com/alibabacloud-go/darabonba-openapi/v2/client"
	green "github.com/alibabacloud-go/green-20220302/v2/client"
	"github.com/alibabacloud-go/tea/tea"
)

const (
	// Aliyun default endpoint
	AliyunDefaultEndpoint = "green-cip.cn-shanghai.aliyuncs.com"
	// Aliyun service type for text moderation
	AliyunServiceChatDetection = "chat_detection"
	// Aliyun success code
	AliyunSuccessCode = 200
	// Aliyun high-risk labels that result in block
	AliyunHighRiskLabelTerrorism  = "terrorism"
	AliyunHighRiskLabelPorn       = "porn"
	AliyunHighRiskLabelContraband = "contraband"
)

// AliyunTextCensor is the client for Alibaba Cloud text content moderation
type AliyunTextCensor struct {
	AccessKeyID     string
	AccessKeySecret string
	Endpoint        string
	Client          *green.Client
}

// ServiceParameters represents the service parameters for text moderation
type ServiceParameters struct {
	Content string `json:"content"`
}

// NewAliyunTextCensor creates a new Alibaba Cloud text moderation client
func NewAliyunTextCensor() (*AliyunTextCensor, error) {
	accessKeyID := utils.GetEnv("ALIYUN_CENSOR_ACCESS_KEY_ID")
	accessKeySecret := utils.GetEnv("ALIYUN_CENSOR_ACCESS_KEY_SECRET")
	endpoint := utils.GetEnv("ALIYUN_CENSOR_GREEN_ENDPOINT")

	if accessKeyID == "" || accessKeySecret == "" {
		return nil, fmt.Errorf("not found ALIYUN_ACCESS_KEY_ID or ALIYUN_ACCESS_KEY_SECRET")
	}

	// Default endpoint is Shanghai if not specified
	if endpoint == "" {
		endpoint = "green-cip.cn-shanghai.aliyuncs.com"
	}

	// Create OpenAPI config
	config := &client.Config{
		AccessKeyId:     tea.String(accessKeyID),
		AccessKeySecret: tea.String(accessKeySecret),
		Endpoint:        tea.String(endpoint),
	}

	// Create Green client
	greenClient, err := green.NewClient(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create Alibaba Cloud Green client: %w", err)
	}

	return &AliyunTextCensor{
		AccessKeyID:     accessKeyID,
		AccessKeySecret: accessKeySecret,
		Endpoint:        endpoint,
		Client:          greenClient,
	}, nil
}

// CensorText performs text content moderation
// Returns: CensorResult with suggestion, label, and score
func (c *AliyunTextCensor) CensorText(text string) (*CensorResult, error) {
	// Build service parameters
	serviceParams := ServiceParameters{
		Content: text,
	}
	serviceParamsJSON, err := json.Marshal(serviceParams)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal service parameters: %w", err)
	}

	// Create request
	request := &green.TextModerationRequest{
		Service:           tea.String(AliyunServiceChatDetection), // Text moderation service
		ServiceParameters: tea.String(string(serviceParamsJSON)),
	}

	// Send request
	response, err := c.Client.TextModeration(request)
	if err != nil {
		return nil, fmt.Errorf("failed to call Alibaba Cloud text moderation API: %w", err)
	}

	// Check response
	if response.Body == nil {
		return nil, fmt.Errorf("empty response from Alibaba Cloud")
	}

	body := response.Body

	// Check response code (200 means success)
	code := tea.Int32Value(body.Code)
	if code != 200 {
		message := tea.StringValue(body.Message)
		return nil, fmt.Errorf("Alibaba Cloud API error: code=%d, message=%s", code, message)
	}

	result := &CensorResult{}

	// Extract suggestion from response
	// Alibaba Cloud returns labels in Data.Labels, we need to determine suggestion from labels
	// Common labels: "normal", "spam", "ad", "politics", "terrorism", "abuse", "porn", "flood", "contraband"
	data := body.Data
	if data == nil {
		// No data means pass
		result.Suggestion = SuggestionPass
		result.Label = LabelNormal
		result.Msg = buildCensorMsg(LabelNormal)
		return result, nil
	}

	labels := tea.StringValue(data.Labels)
	result.Label = labels

	// Map labels to suggestion
	// If labels is empty or "normal", return "pass"
	if labels == "" || labels == LabelNormal {
		result.Suggestion = SuggestionPass
		if labels == "" {
			result.Label = LabelNormal
		}
		result.Msg = buildCensorMsg(result.Label)
		return result, nil
	}

	// For non-normal labels, determine if it's review or block
	// High-risk labels like "terrorism", "porn" typically result in "block"
	// Others like "spam", "ad" might be "review"
	highRiskLabels := []string{AliyunHighRiskLabelTerrorism, AliyunHighRiskLabelPorn, AliyunHighRiskLabelContraband}
	for _, highRisk := range highRiskLabels {
		if labels == highRisk {
			result.Suggestion = SuggestionBlock
			result.Msg = buildCensorMsg(result.Label)
			return result, nil
		}
	}

	// For other labels, return "review" for manual review
	result.Suggestion = SuggestionReview

	// Extract reason if available (as details)
	reason := tea.StringValue(data.Reason)
	if reason != "" {
		result.Details = reason
	}

	// Build message based on label
	result.Msg = buildCensorMsg(result.Label)

	return result, nil
}
