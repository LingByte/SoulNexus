package mediagen

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"strings"
	"time"
)

const seedreamMinPixels = 3_686_400

type SeedreamClient struct {
	cfg  Config
	http *http.Client
}

func NewSeedreamClient(cfg Config) *SeedreamClient {
	return &SeedreamClient{
		cfg: cfg,
		http: &http.Client{
			Timeout: 3 * time.Minute,
		},
	}
}

type seedreamRequest struct {
	Model                      string `json:"model"`
	Prompt                     string `json:"prompt"`
	Size                       string `json:"size"`
	Image                      string `json:"image,omitempty"`
	SequentialImageGeneration  string `json:"sequential_image_generation"`
	ResponseFormat             string `json:"response_format"`
	OutputFormat               string `json:"output_format"`
	Watermark                  bool   `json:"watermark"`
}

type seedreamResponse struct {
	Data []struct {
		URL string `json:"url"`
	} `json:"data"`
	Error *struct {
		Message string `json:"message"`
		Code    string `json:"code"`
	} `json:"error"`
}

func (c *SeedreamClient) GenerateTextToImage(ctx context.Context, prompt string, width, height int) (string, error) {
	return c.generate(ctx, prompt, width, height, "")
}

func (c *SeedreamClient) GenerateImageToImage(ctx context.Context, prompt string, width, height int, imageDataURI string) (string, error) {
	return c.generate(ctx, prompt, width, height, imageDataURI)
}

func (c *SeedreamClient) generate(ctx context.Context, prompt string, width, height int, imageDataURI string) (string, error) {
	if !c.cfg.Configured() {
		return "", ErrAPIKeyMissing
	}
	body := seedreamRequest{
		Model:                     c.cfg.SeedreamModel,
		Prompt:                    prompt,
		Size:                      resolveSeedreamSize(width, height),
		SequentialImageGeneration: "disabled",
		ResponseFormat:            "url",
		OutputFormat:              "png",
		Watermark:                 false,
	}
	if imageDataURI != "" {
		body.Image = imageDataURI
	}
	payload, err := json.Marshal(body)
	if err != nil {
		return "", err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.cfg.SeedreamURL, bytes.NewReader(payload))
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+c.cfg.APIKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.http.Do(req)
	if err != nil {
		return "", &UpstreamError{Message: "调用 Seedream API 失败: " + err.Error()}
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", upstreamHTTPError("Seedream API 请求失败", resp.StatusCode, raw)
	}
	var parsed seedreamResponse
	if err := json.Unmarshal(raw, &parsed); err != nil {
		return "", &UpstreamError{Message: "Seedream API 响应解析失败"}
	}
	if parsed.Error != nil && strings.TrimSpace(parsed.Error.Message) != "" {
		return "", &UpstreamError{Message: "Seedream API 返回错误: " + parsed.Error.Message}
	}
	if len(parsed.Data) == 0 || strings.TrimSpace(parsed.Data[0].URL) == "" {
		return "", ErrNoImageData
	}
	return strings.TrimSpace(parsed.Data[0].URL), nil
}

func resolveSeedreamSize(width, height int) string {
	if width <= 0 {
		width = 1024
	}
	if height <= 0 {
		height = 1024
	}
	pixels := int64(width) * int64(height)
	if pixels >= seedreamMinPixels {
		return fmt.Sprintf("%dx%d", width, height)
	}
	scale := math.Sqrt(float64(seedreamMinPixels) / float64(pixels))
	scaledW := int(math.Ceil(float64(width) * scale))
	scaledH := int(math.Ceil(float64(height) * scale))
	return fmt.Sprintf("%dx%d", scaledW, scaledH)
}
