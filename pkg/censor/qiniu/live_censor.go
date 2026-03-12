package qiniu

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/LingByte/SoulNexus/pkg/utils/qiniu/auth"
)

const (
	// LiveCensorEndpoint is the live stream moderation API endpoint
	LiveCensorEndpoint = "/v3/live/censor"
	// LiveCensorQueryEndpoint is the live stream moderation query results API endpoint
	LiveCensorQueryEndpoint = "/v3/live/censor/query"
	// LiveCensorJobInfoEndpoint is the live stream moderation query job API endpoint
	LiveCensorJobInfoEndpoint = "/v3/live/censor/jobinfo"
	// LiveCensorListEndpoint is the live stream moderation list jobs API endpoint
	LiveCensorListEndpoint = "/v3/live/censor/list"
	// LiveCensorCloseEndpoint is the live stream moderation close job API endpoint
	LiveCensorCloseEndpoint = "/v3/live/censor/close"
	// LiveCensorHost is the live stream moderation API host
	LiveCensorHost = "ai.qiniuapi.com"
)

// LiveCensorRequest represents the request parameters for live stream moderation
type LiveCensorRequest struct {
	Data   LiveCensorData   `json:"data"`
	Params LiveCensorParams `json:"params"`
}

// LiveCensorData represents the live stream data
type LiveCensorData struct {
	ID   string                 `json:"id,omitempty"`   // Live stream identifier, can contain lowercase letters, numbers, dashes, hyphens, max 128 characters
	URI  string                 `json:"uri"`            // Live stream address, currently supports rtmp, hls, flv, etc.
	Info map[string]interface{} `json:"info,omitempty"` // Live stream additional information, passed to client through callback
}

// LiveCensorParams represents moderation parameters
type LiveCensorParams struct {
	HookURL  string           `json:"hook_url,omitempty"`  // Callback address
	HookAuth bool             `json:"hook_auth,omitempty"` // true/false, default is false, adds authorization header to callback request
	Image    *LiveImageParams `json:"image,omitempty"`     // Image moderation parameters
	Audio    *LiveAudioParams `json:"audio,omitempty"`     // Audio moderation parameters
}

// LiveImageParams represents image moderation parameters
type LiveImageParams struct {
	IsOn          bool             `json:"is_on"`                    // true/false, default is false. Either image or audio moderation must be enabled
	Scenes        []string         `json:"scenes,omitempty"`         // Image moderation type, required if image moderation is enabled, options: pulp, terror, politician, ads
	IntervalMsecs int              `json:"interval_msecs,omitempty"` // Frame extraction frequency, required if image moderation is enabled, unit: milliseconds, range: 1000-60000
	Saver         *LiveSaverConfig `json:"saver,omitempty"`          // Save configuration
	HookRule      int              `json:"hook_rule,omitempty"`      // Image moderation result callback rule, 0/1. Default is 0, returns only violation results; set to 1 to return all results
}

// LiveAudioParams represents audio moderation parameters
type LiveAudioParams struct {
	IsOn     bool             `json:"is_on"`               // true/false, default is false. Either image or audio moderation must be enabled
	Scenes   []string         `json:"scenes,omitempty"`    // Audio moderation type, required if audio moderation is enabled, currently only option: antispam
	Saver    *LiveSaverConfig `json:"saver,omitempty"`     // Save configuration
	HookRule int              `json:"hook_rule,omitempty"` // Audio moderation result callback rule, 0/1. Default is 0, returns only violation results; set to 1 to return all results
}

// LiveSaverConfig represents save configuration
type LiveSaverConfig struct {
	Bucket string `json:"bucket"`           // Bucket for storing frame extraction files, currently only supports Qiniu Cloud East China bucket
	Prefix string `json:"prefix,omitempty"` // Prefix for saved frame extraction files
}

// LiveCensorCreateResponse represents the response for creating a live stream moderation job
type LiveCensorCreateResponse struct {
	Code    int                  `json:"code"`    // Status code of request processing result, 200 means success
	Message string               `json:"message"` // Request processing information
	Data    LiveCensorCreateData `json:"data"`
}

// LiveCensorCreateData represents the response data for creating a job
type LiveCensorCreateData struct {
	Job string `json:"job"` // Unique identifier for the live stream moderation job
}

// LiveCensorQueryRequest represents the request parameters for querying moderation results
type LiveCensorQueryRequest struct {
	Job         string   `json:"job"`                   // Unique identifier for the live stream moderation job
	Suggestions []string `json:"suggestions,omitempty"` // Query filter conditions, default is empty (query all), options: pass/review/block
	Start       int64    `json:"start"`                 // Query result start time, value is the timestamp of start time
	End         int64    `json:"end,omitempty"`         // Query result end time, default is start time plus 10min; max difference between end and start is 10min
}

// LiveCensorQueryResponse represents the response for querying moderation results
type LiveCensorQueryResponse struct {
	Code    int                 `json:"code"`
	Message string              `json:"message"`
	Data    LiveCensorQueryData `json:"data"`
}

// LiveCensorQueryData represents the query result data
type LiveCensorQueryData struct {
	Marker string               `json:"marker"` // Cursor information for next pull, returns empty string if no remaining items
	Items  LiveCensorQueryItems `json:"items"`
}

// LiveCensorQueryItems represents the query result items
type LiveCensorQueryItems struct {
	Image []LiveImageResult `json:"image,omitempty"` // Live stream image moderation results
	Audio []LiveAudioResult `json:"audio,omitempty"` // Live stream audio moderation results
}

// LiveImageResult represents the live stream image moderation result
type LiveImageResult struct {
	Code      int                 `json:"code"`      // Processing status: 200 means success
	Message   string              `json:"message"`   // Status description corresponding to code
	Job       string              `json:"job"`       // Unique identifier for the live stream moderation job
	URL       string              `json:"url"`       // Image URL
	Timestamp int64               `json:"timestamp"` // Image timestamp
	Result    *LiveResourceResult `json:"result"`    // Moderation result for this image resource
}

// LiveAudioResult represents the live stream audio moderation result
type LiveAudioResult struct {
	Code      int                 `json:"code"`       // Processing status: 200 means success
	Message   string              `json:"message"`    // Status description corresponding to code
	Job       string              `json:"job"`        // Unique identifier for the live stream moderation job
	URL       string              `json:"url"`        // Audio segment URL
	Start     int64               `json:"start"`      // Audio start timestamp
	End       int64               `json:"end"`        // Audio end timestamp
	AudioText string              `json:"audio_text"` // Language recognition text for this audio resource
	Result    *LiveResourceResult `json:"result"`     // Moderation result for this audio resource
}

// LiveResourceResult represents the moderation result for a resource
type LiveResourceResult struct {
	Suggestion string                     `json:"suggestion"` // pass (approved), review (needs manual review), block (violation)
	Scenes     map[string]LiveSceneResult `json:"scenes"`     // Moderation results for each scene
}

// LiveSceneResult represents the moderation result for a scene
type LiveSceneResult struct {
	Suggestion string            `json:"suggestion"` // Overall suggestion for this scene: pass/review/block
	Details    []LiveSceneDetail `json:"details"`    // Moderation result array for this scene
}

// LiveSceneDetail represents detailed moderation information for a scene
type LiveSceneDetail struct {
	Suggestion string  `json:"suggestion"`      // Detailed suggestion: pass/review/block
	Label      string  `json:"label"`           // Moderation label
	Group      string  `json:"group,omitempty"` // Group information
	Score      float64 `json:"score"`           // Confidence score
	Text       string  `json:"text,omitempty"`  // Audio moderation violation content result field
}

// LiveCensorJobInfoRequest represents the request parameters for querying job information
type LiveCensorJobInfoRequest struct {
	Job string `json:"job"` // Unique identifier for the live stream moderation job
}

// LiveCensorJobInfoResponse represents the response for querying job information
type LiveCensorJobInfoResponse struct {
	Code    int                   `json:"code"`
	Message string                `json:"message"`
	Data    LiveCensorJobInfoData `json:"data"`
}

// LiveCensorJobInfoData represents the job information data
type LiveCensorJobInfoData struct {
	ID        string           `json:"id"`         // Live stream moderation job identifier
	Data      LiveCensorData   `json:"data"`       // Live stream data
	Params    LiveCensorParams `json:"params"`     // Moderation parameters
	Message   string           `json:"message"`    // Job status related information
	Status    string           `json:"status"`     // Job status: waiting/doing/stop/finished
	CreatedAt int64            `json:"created_at"` // Job creation timestamp
	UpdatedAt int64            `json:"updated_at"` // Job update timestamp
}

// LiveCensorListRequest represents the request parameters for listing jobs
type LiveCensorListRequest struct {
	Start  int64  `json:"start"`            // Query time period start timestamp
	End    int64  `json:"end"`              // Query time period end timestamp
	Status string `json:"status"`           // Job status: waiting/doing/stop/finished
	Limit  int    `json:"limit,omitempty"`  // Number of moderation results to query, default is 10, max 100
	Marker string `json:"marker,omitempty"` // Marker returned from previous request, used as starting point for this request
}

// LiveCensorListResponse represents the response for listing jobs
type LiveCensorListResponse struct {
	Code    int                `json:"code"`
	Message string             `json:"message"`
	Data    LiveCensorListData `json:"data"`
}

// LiveCensorListData represents the list jobs data
type LiveCensorListData struct {
	Marker string                  `json:"marker"` // Cursor information for next pull
	Items  []LiveCensorJobInfoData `json:"items"`  // Live stream moderation job query results
}

// LiveCensorCloseRequest represents the request parameters for closing a job
type LiveCensorCloseRequest struct {
	Job string `json:"job"` // Unique identifier for the live stream moderation job
}

// LiveCensorCloseResponse represents the response for closing a job
type LiveCensorCloseResponse struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// LiveCensorClient is the client for live stream content moderation
type LiveCensorClient struct {
	AccessKey string
	SecretKey string
	Host      string
	Client    *http.Client
}

// NewLiveCensorClient creates a new live stream moderation client
func NewLiveCensorClient(accessKey, secretKey string) *LiveCensorClient {
	return &LiveCensorClient{
		AccessKey: accessKey,
		SecretKey: secretKey,
		Host:      LiveCensorHost,
		Client:    &http.Client{},
	}
}

// CreateJob creates a live stream moderation job
func (c *LiveCensorClient) CreateJob(req LiveCensorRequest) (*LiveCensorCreateResponse, error) {
	// Serialize request body
	bodyJSON, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to serialize request parameters: %w", err)
	}

	// Generate authentication token
	authReq := auth.QiniuAuthRequest{
		Method:      "POST",
		Path:        LiveCensorEndpoint,
		Host:        c.Host,
		ContentType: "application/json",
		Body:        bodyJSON,
	}

	token, err := auth.GenerateQiniuToken(c.AccessKey, c.SecretKey, authReq)
	if err != nil {
		return nil, fmt.Errorf("failed to generate authentication token: %w", err)
	}

	// Build HTTP request
	url := fmt.Sprintf("https://%s%s", c.Host, LiveCensorEndpoint)
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
	var createResp LiveCensorCreateResponse
	if err := json.Unmarshal(respBody, &createResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w, response content: %s", err, string(respBody))
	}

	// Check HTTP status code
	if resp.StatusCode != http.StatusOK {
		return &createResp, fmt.Errorf("HTTP status code error: %d, response: %s", resp.StatusCode, string(respBody))
	}

	return &createResp, nil
}

// QueryResults queries live stream moderation results
func (c *LiveCensorClient) QueryResults(req LiveCensorQueryRequest) (*LiveCensorQueryResponse, error) {
	// Serialize request body
	bodyJSON, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to serialize request parameters: %w", err)
	}

	// Generate authentication token
	authReq := auth.QiniuAuthRequest{
		Method:      "POST",
		Path:        LiveCensorQueryEndpoint,
		Host:        c.Host,
		ContentType: "application/json",
		Body:        bodyJSON,
	}

	token, err := auth.GenerateQiniuToken(c.AccessKey, c.SecretKey, authReq)
	if err != nil {
		return nil, fmt.Errorf("failed to generate authentication token: %w", err)
	}

	// Build HTTP request
	url := fmt.Sprintf("https://%s%s", c.Host, LiveCensorQueryEndpoint)
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
	var queryResp LiveCensorQueryResponse
	if err := json.Unmarshal(respBody, &queryResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w, response content: %s", err, string(respBody))
	}

	// Check HTTP status code
	if resp.StatusCode != http.StatusOK {
		return &queryResp, fmt.Errorf("HTTP status code error: %d, response: %s", resp.StatusCode, string(respBody))
	}

	return &queryResp, nil
}

// GetJobInfo retrieves live stream moderation job information
func (c *LiveCensorClient) GetJobInfo(jobID string) (*LiveCensorJobInfoResponse, error) {
	if jobID == "" {
		return nil, fmt.Errorf("jobID cannot be empty")
	}

	req := LiveCensorJobInfoRequest{
		Job: jobID,
	}

	// Serialize request body
	bodyJSON, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to serialize request parameters: %w", err)
	}

	// Generate authentication token
	authReq := auth.QiniuAuthRequest{
		Method:      "POST",
		Path:        LiveCensorJobInfoEndpoint,
		Host:        c.Host,
		ContentType: "application/json",
		Body:        bodyJSON,
	}

	token, err := auth.GenerateQiniuToken(c.AccessKey, c.SecretKey, authReq)
	if err != nil {
		return nil, fmt.Errorf("failed to generate authentication token: %w", err)
	}

	// Build HTTP request
	url := fmt.Sprintf("https://%s%s", c.Host, LiveCensorJobInfoEndpoint)
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
	var jobInfoResp LiveCensorJobInfoResponse
	if err := json.Unmarshal(respBody, &jobInfoResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w, response content: %s", err, string(respBody))
	}

	// Check HTTP status code
	if resp.StatusCode != http.StatusOK {
		return &jobInfoResp, fmt.Errorf("HTTP status code error: %d, response: %s", resp.StatusCode, string(respBody))
	}

	return &jobInfoResp, nil
}

// ListJobs lists live stream moderation jobs
func (c *LiveCensorClient) ListJobs(req LiveCensorListRequest) (*LiveCensorListResponse, error) {
	// Serialize request body
	bodyJSON, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to serialize request parameters: %w", err)
	}

	// Generate authentication token
	authReq := auth.QiniuAuthRequest{
		Method:      "POST",
		Path:        LiveCensorListEndpoint,
		Host:        c.Host,
		ContentType: "application/json",
		Body:        bodyJSON,
	}

	token, err := auth.GenerateQiniuToken(c.AccessKey, c.SecretKey, authReq)
	if err != nil {
		return nil, fmt.Errorf("failed to generate authentication token: %w", err)
	}

	// Build HTTP request
	url := fmt.Sprintf("https://%s%s", c.Host, LiveCensorListEndpoint)
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
	var listResp LiveCensorListResponse
	if err := json.Unmarshal(respBody, &listResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w, response content: %s", err, string(respBody))
	}

	// Check HTTP status code
	if resp.StatusCode != http.StatusOK {
		return &listResp, fmt.Errorf("HTTP status code error: %d, response: %s", resp.StatusCode, string(respBody))
	}

	return &listResp, nil
}

// CloseJob closes a live stream moderation job
func (c *LiveCensorClient) CloseJob(jobID string) (*LiveCensorCloseResponse, error) {
	if jobID == "" {
		return nil, fmt.Errorf("jobID cannot be empty")
	}

	req := LiveCensorCloseRequest{
		Job: jobID,
	}

	// Serialize request body
	bodyJSON, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to serialize request parameters: %w", err)
	}

	// Generate authentication token
	authReq := auth.QiniuAuthRequest{
		Method:      "POST",
		Path:        LiveCensorCloseEndpoint,
		Host:        c.Host,
		ContentType: "application/json",
		Body:        bodyJSON,
	}

	token, err := auth.GenerateQiniuToken(c.AccessKey, c.SecretKey, authReq)
	if err != nil {
		return nil, fmt.Errorf("failed to generate authentication token: %w", err)
	}

	// Build HTTP request
	url := fmt.Sprintf("https://%s%s", c.Host, LiveCensorCloseEndpoint)
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
	var closeResp LiveCensorCloseResponse
	if err := json.Unmarshal(respBody, &closeResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w, response content: %s", err, string(respBody))
	}

	// Check HTTP status code
	if resp.StatusCode != http.StatusOK {
		return &closeResp, fmt.Errorf("HTTP status code error: %d, response: %s", resp.StatusCode, string(respBody))
	}

	return &closeResp, nil
}
