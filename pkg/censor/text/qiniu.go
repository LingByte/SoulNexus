package text

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/LingByte/SoulNexus/pkg/utils"
	"github.com/LingByte/SoulNexus/pkg/utils/qiniu/auth"
)

const (
	// TextCensorEndpoint is the text moderation API endpoint
	TextCensorEndpoint = "/v3/text/censor"
	// TextCensorHost is the text moderation API host
	TextCensorHost   = "ai.qiniuapi.com"
	TextCensorScenes = "antispam"
)

// TextCensorRequest represents the request parameters for text moderation
type TextCensorRequest struct {
	Data   TextCensorData   `json:"data"`
	Params TextCensorParams `json:"params"`
}

// TextCensorData represents the text data to be moderated
type TextCensorData struct {
	Text string `json:"text"` // Text content
}

// TextCensorParams represents moderation parameters
type TextCensorParams struct {
	Scenes []string `json:"scenes"` // Moderation type, required field, options: antispam
}

// TextCensorResponse represents the text moderation response
type TextCensorResponse struct {
	Code    int               `json:"code"`    // Processing status: 200 means success
	Message string            `json:"message"` // Status description corresponding to code
	Result  *TextCensorResult `json:"result"`  // Moderation result
}

// TextCensorResult represents the text moderation result
type TextCensorResult struct {
	Suggestion string                     `json:"suggestion"` // pass (approved), review (needs manual review), block (violation)
	Scenes     map[string]TextSceneResult `json:"scenes"`     // Moderation results for each scene
}

// TextSceneResult represents the moderation result for a scene
type TextSceneResult struct {
	Suggestion string            `json:"suggestion"` // Overall suggestion for this scene: pass/review/block
	Details    []TextSceneDetail `json:"details"`    // Detailed information array
}

// TextSceneDetail represents detailed moderation information for a scene
type TextSceneDetail struct {
	Label       string  `json:"label"`       // Moderation label: normal, spam, ad, politics, terrorism, abuse, porn, flood, contraband, meaningless
	Score       float64 `json:"score"`       // Confidence score
	Description string  `json:"description"` // Description of violation content
}

// QiniuTextCensor is the client for text content moderation
type QiniuTextCensor struct {
	AccessKey string
	SecretKey string
	Host      string
	Client    *http.Client
}

// NewQiniuTextCensor creates a new text moderation client
func NewQiniuTextCensor() (*QiniuTextCensor, error) {
	accessKey := utils.GetEnv("QINIU_ACCESS_KEY")
	secretKey := utils.GetEnv("QINIU_SECRET_KEY")
	if accessKey == "" || secretKey == "" {
		return nil, fmt.Errorf("not found QINIU_ACCESS_KEY or QINIU_SECRET_KEY")
	}
	return &QiniuTextCensor{
		AccessKey: accessKey,
		SecretKey: secretKey,
		Host:      TextCensorHost,
		Client:    &http.Client{},
	}, nil
}

// Censor performs text content moderation
func (c *QiniuTextCensor) Censor(req TextCensorRequest) (*TextCensorResponse, error) {
	// Serialize request body
	bodyJSON, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to serialize request parameters: %w", err)
	}

	// Generate authentication token
	authReq := auth.QiniuAuthRequest{
		Method:      "POST",
		Path:        TextCensorEndpoint,
		Host:        c.Host,
		ContentType: "application/json",
		Body:        bodyJSON,
	}

	token, err := auth.GenerateQiniuToken(c.AccessKey, c.SecretKey, authReq)
	if err != nil {
		return nil, fmt.Errorf("failed to generate authentication token: %w", err)
	}

	// Build HTTP request
	url := fmt.Sprintf("https://%s%s", c.Host, TextCensorEndpoint)
	httpReq, err := http.NewRequest("POST", url, bytes.NewReader(bodyJSON))
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP request: %w", err)
	}

	// Set request headers
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", token)

	// Send request
	resp, err := c.Client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to send HTTP request: %w", err)
	}
	defer resp.Body.Close()

	// Read response
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	// Parse response
	var censorResp TextCensorResponse
	if err := json.Unmarshal(respBody, &censorResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w, response content: %s", err, string(respBody))
	}

	// Check HTTP status code
	if resp.StatusCode != http.StatusOK {
		return &censorResp, fmt.Errorf("HTTP status code error: %d, response: %s", resp.StatusCode, string(respBody))
	}

	return &censorResp, nil
}

// CensorText is a convenience method to moderate text content
// text: text content to be moderated
func (c *QiniuTextCensor) CensorText(text string) (*CensorResult, error) {
	req := TextCensorRequest{
		Data: TextCensorData{
			Text: text,
		},
		Params: TextCensorParams{
			Scenes: []string{TextCensorScenes},
		},
	}
	censor, err := c.Censor(req)
	if err != nil {
		return nil, err
	}

	result := &CensorResult{
		Suggestion: censor.Result.Suggestion,
	}

	// Extract label and score from scenes
	if censor.Result.Scenes != nil {
		if antispam, ok := censor.Result.Scenes[TextCensorScenes]; ok {
			if len(antispam.Details) > 0 {
				detail := antispam.Details[0]
				result.Label = detail.Label
				result.Score = detail.Score
				if detail.Description != "" {
					result.Details = detail.Description
				}
			}
		}
	}

	// Build message based on label
	result.Msg = buildCensorMsg(result.Label)

	return result, nil
}
