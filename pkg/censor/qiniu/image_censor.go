package qiniu

// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/LingByte/SoulNexus/pkg/utils/qiniu/auth"
)

const (
	// ImageCensorEndpoint is the image moderation API endpoint
	ImageCensorEndpoint = "/v3/image/censor"
	// ImageCensorHost is the image moderation API host
	ImageCensorHost = "ai.qiniuapi.com"
)

// ImageCensorRequest represents the request parameters for image moderation
type ImageCensorRequest struct {
	Data   ImageCensorData   `json:"data"`
	Params ImageCensorParams `json:"params,omitempty"`
}

// ImageCensorData represents the image data to be moderated
type ImageCensorData struct {
	URI string `json:"uri"` // Image URI
}

// ImageCensorParams represents moderation parameters
type ImageCensorParams struct {
	Scenes []string `json:"scenes"` // Moderation scenes: pulp (pornography), terror (violence), politician (politics), ad (advertisement), etc.
}

// ImageCensorResponse represents the image moderation response
type ImageCensorResponse struct {
	Code    int                    `json:"code"`
	Message string                 `json:"message,omitempty"`
	Result  *ImageCensorResult     `json:"result,omitempty"`
	Data    map[string]interface{} `json:"data,omitempty"`
}

// ImageCensorResult represents the moderation result
type ImageCensorResult struct {
	Suggestion string                          `json:"suggestion"` // pass (approved), review (needs manual review), block (violation)
	Scenes     map[string]ImageCensorSceneInfo `json:"scenes"`
}

// ImageCensorSceneInfo represents scene moderation information
type ImageCensorSceneInfo struct {
	Suggestion string  `json:"suggestion"` // pass, review, block
	Label      string  `json:"label"`      // Moderation label
	Score      float64 `json:"score"`      // Confidence score
}

// ImageCensorClient is the client for image content moderation
type ImageCensorClient struct {
	AccessKey string
	SecretKey string
	Host      string
	Client    *http.Client
}

// NewImageCensorClient creates a new image moderation client
func NewImageCensorClient(accessKey, secretKey string) *ImageCensorClient {
	return &ImageCensorClient{
		AccessKey: accessKey,
		SecretKey: secretKey,
		Host:      ImageCensorHost,
		Client:    &http.Client{},
	}
}

// Censor performs image content moderation
func (c *ImageCensorClient) Censor(req ImageCensorRequest) (*ImageCensorResponse, error) {
	// Serialize request body
	bodyJSON, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to serialize request parameters: %w", err)
	}

	// Generate authentication token
	authReq := auth.QiniuAuthRequest{
		Method:      "POST",
		Path:        ImageCensorEndpoint,
		Host:        c.Host,
		ContentType: "application/json",
		Body:        bodyJSON,
	}

	token, err := auth.GenerateQiniuToken(c.AccessKey, c.SecretKey, authReq)
	if err != nil {
		return nil, fmt.Errorf("failed to generate authentication token: %w", err)
	}

	// Build HTTP request
	url := fmt.Sprintf("https://%s%s", c.Host, ImageCensorEndpoint)
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
	var censorResp ImageCensorResponse
	if err := json.Unmarshal(respBody, &censorResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w, response content: %s", err, string(respBody))
	}

	// Check HTTP status code
	if resp.StatusCode != http.StatusOK {
		return &censorResp, fmt.Errorf("HTTP status code error: %d, response: %s", resp.StatusCode, string(respBody))
	}

	return &censorResp, nil
}

// CensorImage is a convenience method to moderate a single image
// uri: image URI
// scenes: moderation scene list, e.g., []string{"pulp", "terror", "politician"}
func (c *ImageCensorClient) CensorImage(uri string, scenes []string) (*ImageCensorResponse, error) {
	req := ImageCensorRequest{
		Data: ImageCensorData{
			URI: uri,
		},
		Params: ImageCensorParams{
			Scenes: scenes,
		},
	}
	return c.Censor(req)
}
