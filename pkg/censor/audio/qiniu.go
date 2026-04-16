package audio

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/LingByte/SoulNexus/pkg/utils/qiniu/auth"
)

const (
	// AudioCensorEndpoint is the audio moderation API endpoint
	AudioCensorEndpoint = "/v3/audio/censor"
	// AudioJobEndpoint is the audio job query API endpoint template
	AudioJobEndpoint = "/v3/jobs/audio"
	// AudioCensorHost is the audio moderation API host
	AudioCensorHost = "ai.qiniuapi.com"
)

// AudioCensorRequest represents the request parameters for audio moderation
type AudioCensorRequest struct {
	Data   AudioCensorData   `json:"data"`
	Params AudioCensorParams `json:"params"`
}

// AudioCensorData represents the audio data to be moderated
type AudioCensorData struct {
	URI string `json:"uri"` // Audio URL address
	ID  string `json:"id"`  // Unique identifier set by the caller, optional
}

// AudioCensorParams represents moderation parameters
type AudioCensorParams struct {
	Scenes   []string `json:"scenes"`              // Moderation type, required field, currently only option: antispam
	HookURL  string   `json:"hook_url,omitempty"`  // Callback address after audio detection completes, optional
	HookAuth bool     `json:"hook_auth,omitempty"` // true/false, default is false, adds authorization header to callback request
}

// AudioCensorSubmitResponse represents the response for submitting an audio moderation task
type AudioCensorSubmitResponse struct {
	ID string `json:"id"` // Server-returned unique identifier for the audio moderation task
}

// AudioCensorJobResponse represents the audio moderation task status response
type AudioCensorJobResponse struct {
	ID        string             `json:"id"`         // Task ID
	Status    string             `json:"status"`     // Task status: WAITING/DOING/FINISHED/FAILED
	Request   json.RawMessage    `json:"request"`    // Request body from audio moderation request
	Response  *AudioCensorResult `json:"response"`   // Audio content moderation result
	Error     string             `json:"error"`      // Task error information
	CreatedAt string             `json:"created_at"` // Task creation time
	UpdatedAt string             `json:"updated_at"` // Task update time
}

// AudioCensorResult represents the audio moderation result
type AudioCensorResult struct {
	Code      int                      `json:"code"`       // Processing status: 200 means success
	Message   string                   `json:"message"`    // Status description
	AudioText string                   `json:"audio_text"` // Audio text
	Result    *AudioCensorResultDetail `json:"result"`     // Moderation result details
}

// AudioCensorResultDetail represents detailed moderation result
type AudioCensorResultDetail struct {
	Suggestion string                      `json:"suggestion"` // pass (approved), review (needs manual review), block (violation)
	Scenes     map[string]AudioSceneResult `json:"scenes"`     // Moderation results for each scene
}

// AudioSceneResult represents the moderation result for a scene
type AudioSceneResult struct {
	Suggestion string           `json:"suggestion"` // Overall suggestion for this scene: pass/review/block
	Cuts       []AudioCutResult `json:"cuts"`       // Audio segment moderation result array
}

// AudioCutResult represents the moderation result for an audio segment
type AudioCutResult struct {
	Suggestion string           `json:"suggestion"` // Suggestion for this audio segment: pass/review/block
	StartTime  int              `json:"start_time"` // Start offset of this audio segment in the original audio, unit: seconds
	EndTime    int              `json:"end_time"`   // End offset of this audio segment in the original audio, unit: seconds
	AudioText  string           `json:"audio_text"` // Text of this audio segment
	Details    []AudioCutDetail `json:"details"`    // Detailed information array
}

// AudioCutDetail represents detailed moderation information for an audio segment
type AudioCutDetail struct {
	Suggestion string  `json:"suggestion"` // Detailed suggestion: pass/review/block
	Label      string  `json:"label"`      // Moderation label: normal, politics, porn, abuse, contraband
	Score      float64 `json:"score"`      // Confidence score
}

// QiniuAudioCensor is the client for audio content moderation
type QiniuAudioCensor struct {
	AccessKey string
	SecretKey string
	Host      string
	Client    *http.Client
}

// NewQiniuAudioCensor creates a new audio moderation client
func NewQiniuAudioCensor(accessKey, secretKey string) *QiniuAudioCensor {
	return &QiniuAudioCensor{
		AccessKey: accessKey,
		SecretKey: secretKey,
		Host:      AudioCensorHost,
		Client:    &http.Client{},
	}
}

// SubmitCensor submits an audio moderation task
func (c *QiniuAudioCensor) SubmitCensor(req AudioCensorRequest) (*AudioCensorSubmitResponse, error) {
	// Serialize request body
	bodyJSON, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to serialize request parameters: %w", err)
	}

	// Generate authentication token
	authReq := auth.QiniuAuthRequest{
		Method:      "POST",
		Path:        AudioCensorEndpoint,
		Host:        c.Host,
		ContentType: "application/json",
		Body:        bodyJSON,
	}

	token, err := auth.GenerateQiniuToken(c.AccessKey, c.SecretKey, authReq)
	if err != nil {
		return nil, fmt.Errorf("failed to generate authentication token: %w", err)
	}

	// Build HTTP request
	url := fmt.Sprintf("https://%s%s", c.Host, AudioCensorEndpoint)
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

	// Check HTTP status code
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP status code error: %d, response: %s", resp.StatusCode, string(respBody))
	}

	// Parse response
	var submitResp AudioCensorSubmitResponse
	if err := json.Unmarshal(respBody, &submitResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w, response content: %s", err, string(respBody))
	}

	return &submitResp, nil
}

// GetCensorResult retrieves the audio moderation result
// This method implements the AudioCensor interface
func (c *QiniuAudioCensor) GetCensorResult(taskID string) (interface{}, error) {
	if taskID == "" {
		return nil, fmt.Errorf("taskID cannot be empty")
	}

	resp, err := c.GetCensorResultFull(taskID)
	if err != nil {
		return nil, err
	}

	return resp, nil
}

// GetCensorResultFull retrieves the audio moderation result with full details
func (c *QiniuAudioCensor) GetCensorResultFull(taskID string) (*AudioCensorJobResponse, error) {
	if taskID == "" {
		return nil, fmt.Errorf("taskID cannot be empty")
	}

	// Generate authentication token (GET request, no body)
	authReq := auth.QiniuAuthRequest{
		Method: "GET",
		Path:   fmt.Sprintf("%s/%s", AudioJobEndpoint, taskID),
		Host:   c.Host,
	}

	token, err := auth.GenerateQiniuToken(c.AccessKey, c.SecretKey, authReq)
	if err != nil {
		return nil, fmt.Errorf("failed to generate authentication token: %w", err)
	}

	// Build HTTP request
	url := fmt.Sprintf("https://%s%s/%s", c.Host, AudioJobEndpoint, taskID)
	httpReq, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP request: %w", err)
	}

	// Set request headers
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

	// Check HTTP status code
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP status code error: %d, response: %s", resp.StatusCode, string(respBody))
	}

	// Parse response
	var jobResp AudioCensorJobResponse
	if err := json.Unmarshal(respBody, &jobResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w, response content: %s", err, string(respBody))
	}

	return &jobResp, nil
}

// SubmitCensorAudio submits an audio moderation task with just the audio URL
// This method implements the AudioCensor interface
func (c *QiniuAudioCensor) SubmitCensorAudio(audioURL string) (string, error) {
	if audioURL == "" {
		return "", fmt.Errorf("audioURL cannot be empty")
	}

	req := AudioCensorRequest{
		Data: AudioCensorData{
			URI: audioURL,
		},
		Params: AudioCensorParams{
			Scenes: []string{"antispam"},
		},
	}

	resp, err := c.SubmitCensor(req)
	if err != nil {
		return "", err
	}

	return resp.ID, nil
}

// SubmitCensorAudioFull is a full-featured method to submit an audio moderation task
// uri: audio URI
// audioID: unique identifier for the audio (optional)
// scenes: moderation scene list, e.g., []string{"antispam"}
// hookURL: callback address (optional)
func (c *QiniuAudioCensor) SubmitCensorAudioFull(uri, audioID string, scenes []string, hookURL string) (*AudioCensorSubmitResponse, error) {
	req := AudioCensorRequest{
		Data: AudioCensorData{
			URI: uri,
			ID:  audioID,
		},
		Params: AudioCensorParams{
			Scenes:  scenes,
			HookURL: hookURL,
		},
	}

	return c.SubmitCensor(req)
}
