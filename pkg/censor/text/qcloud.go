package text

import (
	"encoding/base64"
	"fmt"

	"github.com/LingByte/SoulNexus/pkg/utils"
	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common"
	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common/profile"
	tms "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/tms/v20201229"
)

const (
	// QCloud default region
	QCloudDefaultRegion = "ap-guangzhou"
	// QCloud TMS endpoint
	QCloudTMSEndpoint = "tms.tencentcloudapi.com"
	// QCloud suggestion values (from API response)
	QCloudSuggestionPass   = "Pass"
	QCloudSuggestionReview = "Review"
	QCloudSuggestionBlock  = "Block"
	// QCloud score conversion (percentage to 0-1 range)
	QCloudScoreDivisor = 100.0
	// QCloud keywords format string
	QCloudKeywordsFormat = "Keywords: %v"
)

// QCloudTextCensor is the client for Tencent Cloud text content moderation
type QCloudTextCensor struct {
	SecretID  string
	SecretKey string
	Region    string
	Client    *tms.Client
}

// NewQCloudTextCensor creates a new Tencent Cloud text moderation client
func NewQCloudTextCensor() (*QCloudTextCensor, error) {
	secretID := utils.GetEnv("QCLOUD_SECRET_ID")
	secretKey := utils.GetEnv("QCLOUD_SECRET_KEY")
	region := utils.GetEnv("QCLOUD_REGION")

	if secretID == "" || secretKey == "" {
		return nil, fmt.Errorf("not found QCLOUD_SECRET_ID or QCLOUD_SECRET_KEY")
	}

	// Default region is ap-guangzhou if not specified
	if region == "" {
		region = "ap-guangzhou"
	}

	// Create credential
	credential := common.NewCredential(secretID, secretKey)

	// Create client profile
	cpf := profile.NewClientProfile()
	cpf.HttpProfile.Endpoint = "tms.tencentcloudapi.com"

	// Create TMS client
	client, err := tms.NewClient(credential, region, cpf)
	if err != nil {
		return nil, fmt.Errorf("failed to create Tencent Cloud TMS client: %w", err)
	}

	return &QCloudTextCensor{
		SecretID:  secretID,
		SecretKey: secretKey,
		Region:    region,
		Client:    client,
	}, nil
}

// CensorText performs text content moderation
// Returns: CensorResult with suggestion, label, and score
func (c *QCloudTextCensor) CensorText(text string) (*CensorResult, error) {
	// Tencent Cloud requires base64 encoded text content
	encodedText := base64.StdEncoding.EncodeToString([]byte(text))

	// Create request
	request := tms.NewTextModerationRequest()
	request.Content = common.StringPtr(encodedText)

	// Send request
	response, err := c.Client.TextModeration(request)
	if err != nil {
		return nil, fmt.Errorf("failed to call Tencent Cloud text moderation API: %w", err)
	}

	// Check response
	if response.Response == nil {
		return nil, fmt.Errorf("empty response from Tencent Cloud")
	}

	result := &CensorResult{}

	// Map Tencent Cloud suggestion to our format
	// Tencent Cloud returns: Pass, Review, Block
	suggestion := ""
	if response.Response.Suggestion != nil {
		suggestion = *response.Response.Suggestion
	}

	// Convert to lowercase to match our interface (pass/review/block)
	switch suggestion {
	case QCloudSuggestionPass:
		result.Suggestion = SuggestionPass
	case QCloudSuggestionReview:
		result.Suggestion = SuggestionReview
	case QCloudSuggestionBlock:
		result.Suggestion = SuggestionBlock
	default:
		// If suggestion is empty or unknown, check label for safety
		if response.Response.Label != nil && *response.Response.Label != "" {
			// If there's a label, it's likely a violation
			result.Suggestion = SuggestionReview
		} else {
			// Default to pass if no clear indication
			result.Suggestion = SuggestionPass
		}
	}

	// Extract label
	if response.Response.Label != nil {
		result.Label = *response.Response.Label
	}

	// Extract score if available
	if response.Response.Score != nil {
		result.Score = float64(*response.Response.Score) / QCloudScoreDivisor // Convert percentage to 0-1 range
	}

	// Extract keywords if available (as details)
	if response.Response.Keywords != nil && len(response.Response.Keywords) > 0 {
		result.Details = fmt.Sprintf(QCloudKeywordsFormat, response.Response.Keywords)
	}

	// Build message based on label
	result.Msg = buildCensorMsg(result.Label)

	return result, nil
}
