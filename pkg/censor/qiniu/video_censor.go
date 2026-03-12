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
	// VideoCensorEndpoint is the video moderation API endpoint
	VideoCensorEndpoint = "/v3/video/censor"
	// VideoJobEndpoint is the video job query API endpoint template
	VideoJobEndpoint = "/v3/jobs/video"
	// VideoCensorHost is the video moderation API host
	VideoCensorHost = "ai.qiniuapi.com"
)

// VideoCensorRequest represents the request parameters for video moderation
type VideoCensorRequest struct {
	Data   VideoCensorData   `json:"data"`
	Params VideoCensorParams `json:"params,omitempty"`
}

// VideoCensorData represents the video data to be moderated
type VideoCensorData struct {
	URI string `json:"uri"` // Video URI
	ID  string `json:"id"`  // Unique identifier for the video, optional
}

// VideoCensorParams represents moderation parameters
type VideoCensorParams struct {
	Scenes   []string       `json:"scenes,omitempty"`    // Moderation scenes: pulp (pornography), terror (violence), politician (politics), etc.
	CutParam *VideoCutParam `json:"cut_param,omitempty"` // Frame extraction parameters
}

// VideoCutParam represents frame extraction parameters
type VideoCutParam struct {
	IntervalMsecs int `json:"interval_msecs"` // Frame extraction interval (milliseconds), e.g., 5000
}

// VideoCensorSubmitResponse represents the response for submitting a video moderation task
type VideoCensorSubmitResponse struct {
	Job string `json:"job"` // Task ID
}

// VideoCensorJobResponse represents the video moderation task status response
type VideoCensorJobResponse struct {
	ID            string             `json:"id"`             // Task ID
	VID           string             `json:"vid"`            // Unique identifier for the video
	Request       json.RawMessage    `json:"request"`        // Original moderation request data
	Status        string             `json:"status"`         // Task status: WAITING/DOING/RESCHEDULED/FAILED/FINISHED
	Result        *VideoCensorResult `json:"result"`         // Moderation result
	CreatedAt     string             `json:"created_at"`     // Task creation time
	UpdatedAt     string             `json:"updated_at"`     // Task update time
	RescheduledAt string             `json:"rescheduled_at"` // Task retry time
}

// VideoCensorResult represents the video moderation result
type VideoCensorResult struct {
	Code    int                      `json:"code"`    // Processing status: 200 means success
	Message string                   `json:"message"` // Status description
	Result  *VideoCensorResultDetail `json:"result"`  // Moderation result details
}

// VideoCensorResultDetail represents detailed moderation result
type VideoCensorResultDetail struct {
	Suggestion string                            `json:"suggestion"` // pass (approved), review (needs manual review), block (violation)
	Scenes     map[string]VideoCensorSceneResult `json:"scenes"`     // Moderation results for each scene
}

// VideoCensorSceneResult represents the moderation result for a scene
type VideoCensorSceneResult struct {
	Suggestion string                 `json:"suggestion"` // Overall suggestion for this scene: pass/review/block
	Cuts       []VideoCensorCutResult `json:"cuts"`       // Frame extraction moderation result array
}

// VideoCensorCutResult represents the moderation result for a frame extraction
type VideoCensorCutResult struct {
	Offset     int                    `json:"offset"`            // Time position of the frame extraction (milliseconds)
	Suggestion string                 `json:"suggestion"`        // Suggestion for this frame: pass/review/block
	URI        string                 `json:"uri,omitempty"`     // Frame extraction URI
	Details    []VideoCensorCutDetail `json:"details,omitempty"` // Detailed information array
}

// VideoCensorCutDetail represents detailed moderation information for a frame extraction
type VideoCensorCutDetail struct {
	Suggestion string                 `json:"suggestion"`           // Detailed suggestion: pass/review/block
	Label      string                 `json:"label"`                // Moderation label
	Score      float64                `json:"score"`                // Confidence score
	Group      string                 `json:"group,omitempty"`      // Group (specific to politician type)
	Sample     *VideoPoliticianSample `json:"sample,omitempty"`     // Reference image information (specific to politician type)
	Detections []VideoCensorDetection `json:"detections,omitempty"` // Detected object information
}

// VideoPoliticianSample represents reference image information for politicians
type VideoPoliticianSample struct {
	URI string  `json:"uri"` // Reference image link
	Pts [][]int `json:"pts"` // Face bounding box coordinates of sensitive persons in the reference image
}

// VideoCensorDetection represents detected object information
type VideoCensorDetection struct {
	Pts   [][]int `json:"pts"`   // Object/face bounding box coordinates
	Score float64 `json:"score"` // Confidence score
}

// VideoCensorClient is the client for video content moderation
type VideoCensorClient struct {
	AccessKey string
	SecretKey string
	Host      string
	Client    *http.Client
}

// NewVideoCensorClient creates a new video moderation client
func NewVideoCensorClient(accessKey, secretKey string) *VideoCensorClient {
	return &VideoCensorClient{
		AccessKey: accessKey,
		SecretKey: secretKey,
		Host:      VideoCensorHost,
		Client:    &http.Client{},
	}
}

// SubmitCensor submits a video moderation task
func (c *VideoCensorClient) SubmitCensor(req VideoCensorRequest) (*VideoCensorSubmitResponse, error) {
	// Serialize request body
	bodyJSON, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to serialize request parameters: %w", err)
	}

	// Generate authentication token
	authReq := auth.QiniuAuthRequest{
		Method:      "POST",
		Path:        VideoCensorEndpoint,
		Host:        c.Host,
		ContentType: "application/json",
		Body:        bodyJSON,
	}

	token, err := auth.GenerateQiniuToken(c.AccessKey, c.SecretKey, authReq)
	if err != nil {
		return nil, fmt.Errorf("failed to generate authentication token: %w", err)
	}

	// Build HTTP request
	url := fmt.Sprintf("https://%s%s", c.Host, VideoCensorEndpoint)
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
	var submitResp VideoCensorSubmitResponse
	if err := json.Unmarshal(respBody, &submitResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w, response content: %s", err, string(respBody))
	}

	return &submitResp, nil
}

// GetCensorResult retrieves the video moderation result
func (c *VideoCensorClient) GetCensorResult(jobID string) (*VideoCensorJobResponse, error) {
	if jobID == "" {
		return nil, fmt.Errorf("jobID cannot be empty")
	}

	// Generate authentication token (GET request, no body)
	authReq := auth.QiniuAuthRequest{
		Method: "GET",
		Path:   fmt.Sprintf("%s/%s", VideoJobEndpoint, jobID),
		Host:   c.Host,
	}

	token, err := auth.GenerateQiniuToken(c.AccessKey, c.SecretKey, authReq)
	if err != nil {
		return nil, fmt.Errorf("failed to generate authentication token: %w", err)
	}

	// Build HTTP request
	url := fmt.Sprintf("https://%s%s/%s", c.Host, VideoJobEndpoint, jobID)
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
	var jobResp VideoCensorJobResponse
	if err := json.Unmarshal(respBody, &jobResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w, response content: %s", err, string(respBody))
	}

	return &jobResp, nil
}

// SubmitCensorVideo is a convenience method to submit a video moderation task
// uri: video URI
// videoID: unique identifier for the video (optional)
// scenes: moderation scene list, e.g., []string{"pulp", "terror", "politician"}
// intervalMsecs: frame extraction interval (milliseconds), e.g., 5000
func (c *VideoCensorClient) SubmitCensorVideo(uri, videoID string, scenes []string, intervalMsecs int) (*VideoCensorSubmitResponse, error) {
	req := VideoCensorRequest{
		Data: VideoCensorData{
			URI: uri,
			ID:  videoID,
		},
		Params: VideoCensorParams{
			Scenes: scenes,
		},
	}

	if intervalMsecs > 0 {
		req.Params.CutParam = &VideoCutParam{
			IntervalMsecs: intervalMsecs,
		}
	}

	return c.SubmitCensor(req)
}
