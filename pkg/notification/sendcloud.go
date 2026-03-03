package notification

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// SendCloudConfig SendCloud configuration
type SendCloudConfig struct {
	APIUser string // API User
	APIKey  string // API Key
	From    string // Sender email address
}

// SendCloudClient SendCloud API client
type SendCloudClient struct {
	Config SendCloudConfig
	Client *http.Client
}

// SendCloudWebhookEvent SendCloud webhook event
type SendCloudWebhookEvent struct {
	Event      string `json:"event"`      // 1=delivered, 3=spam, 4=invalid, 5=soft_bounce, 10=click, 11=open, 12=unsubscribe, 18=request
	MessageID  string `json:"messageId"`  // Message ID
	Email      string `json:"email"`      // Recipient email
	Timestamp  int64  `json:"timestamp"`  // Event timestamp
	SmtpStatus string `json:"smtpStatus"` // SMTP status code
	SmtpError  string `json:"smtpError"`  // SMTP error message
}

// NewSendCloudClient creates SendCloud client instance
func NewSendCloudClient(config SendCloudConfig) *SendCloudClient {
	return &SendCloudClient{
		Config: config,
		Client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// Send sends email via SendCloud API
func (s *SendCloudClient) Send(to, subject, body string) (string, error) {
	return s.SendHTML(to, subject, body)
}

// SendHTML sends HTML email via SendCloud API using form-data
func (s *SendCloudClient) SendHTML(to, subject, htmlBody string) (string, error) {
	apiURL := "https://api.sendcloud.net/apiv2/mail/send"

	// Use form-data format instead of JSON
	data := url.Values{}
	data.Set("apiUser", s.Config.APIUser)
	data.Set("apiKey", s.Config.APIKey)
	data.Set("to", to)
	data.Set("from", s.Config.From)
	data.Set("subject", subject)
	data.Set("html", htmlBody)

	req, err := http.NewRequest("POST", apiURL, strings.NewReader(data.Encode()))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := s.Client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %w", err)
	}

	// Parse JSON response
	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		return "", fmt.Errorf("failed to parse response: %w", err)
	}

	// Check result field
	if resultBool, ok := result["result"].(bool); ok && !resultBool {
		if msg, ok := result["message"].(string); ok {
			return "", fmt.Errorf("sendcloud error: %s", msg)
		}
		return "", fmt.Errorf("sendcloud error: request failed")
	}

	// Extract message ID from response - try multiple possible field names
	messageID := ""

	// Try from info.emailIdList first (array of email IDs)
	if info, ok := result["info"].(map[string]interface{}); ok {
		if emailIdList, ok := info["emailIdList"].([]interface{}); ok && len(emailIdList) > 0 {
			if firstId, ok := emailIdList[0].(string); ok {
				messageID = firstId
			}
		}
		// Try from info.messageId
		if messageID == "" {
			if msgId, ok := info["messageId"].(string); ok && msgId != "" {
				messageID = msgId
			}
		}
	}

	// Try from data.messageId
	if messageID == "" {
		if dataObj, ok := result["data"].(map[string]interface{}); ok {
			if msgId, ok := dataObj["messageId"].(string); ok && msgId != "" {
				messageID = msgId
			}
		}
	}

	// Try from top-level messageId
	if messageID == "" {
		if msgId, ok := result["messageId"].(string); ok && msgId != "" {
			messageID = msgId
		}
	}

	return messageID, nil
}

// SendBatch sends batch emails via SendCloud API
func (s *SendCloudClient) SendBatch(recipients []string, subject, htmlBody string) ([]string, error) {
	apiURL := "https://api.sendcloud.net/apiv2/mail/send"

	// Convert recipients to semicolon-separated string
	toList := strings.Join(recipients, ";")

	data := url.Values{}
	data.Set("apiUser", s.Config.APIUser)
	data.Set("apiKey", s.Config.APIKey)
	data.Set("to", toList)
	data.Set("from", s.Config.From)
	data.Set("subject", subject)
	data.Set("html", htmlBody)

	req, err := http.NewRequest("POST", apiURL, strings.NewReader(data.Encode()))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := s.Client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	if resultBool, ok := result["result"].(bool); ok && !resultBool {
		if msg, ok := result["message"].(string); ok {
			return nil, fmt.Errorf("sendcloud error: %s", msg)
		}
		return nil, fmt.Errorf("sendcloud error: request failed")
	}

	// For batch send, we return empty message IDs as SendCloud doesn't provide them in batch response
	messageIDs := make([]string, len(recipients))
	return messageIDs, nil
}

// GetSendStatus gets email send status from SendCloud
func (s *SendCloudClient) GetSendStatus(messageID string) (map[string]interface{}, error) {
	apiURL := "https://api.sendcloud.net/apiv2/mail/detail"

	data := url.Values{}
	data.Set("apiUser", s.Config.APIUser)
	data.Set("apiKey", s.Config.APIKey)
	data.Set("messageId", messageID)

	req, err := http.NewRequest("POST", apiURL, strings.NewReader(data.Encode()))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := s.Client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return result, nil
}

// ParseSendCloudWebhookEvent parses SendCloud webhook event
// SendCloud sends webhook as form-data, not JSON
func ParseSendCloudWebhookEvent(data []byte) (*SendCloudWebhookEvent, error) {
	// First try to parse as JSON (for compatibility)
	var event SendCloudWebhookEvent
	if err := json.Unmarshal(data, &event); err == nil {
		return &event, nil
	}

	// If JSON parsing fails, try to parse as form-data
	// SendCloud sends webhook as: event=request&messageId=xxx or event=deliver&emailId=xxx
	params, err := url.ParseQuery(string(data))
	if err != nil {
		return nil, fmt.Errorf("failed to parse webhook event as JSON or form-data: %w", err)
	}

	// Get messageId from either messageId or emailId field
	messageID := params.Get("messageId")
	if messageID == "" {
		messageID = params.Get("emailId")
	}

	// Extract message ID from emailId if it contains the full format
	// emailId format: 1771668254649_238461_319875_25585.sc-10-9-138-42-inbound0@19511899044@163.com
	// We need to extract just the message part before the first @
	if messageID != "" && strings.Contains(messageID, "@") {
		parts := strings.Split(messageID, "@")
		if len(parts) > 0 {
			messageID = parts[0]
		}
	}

	event = SendCloudWebhookEvent{
		Event:      params.Get("event"),
		MessageID:  messageID,
		Email:      params.Get("recipient"), // recipient field for deliver/open events
		SmtpStatus: params.Get("smtpStatus"),
		SmtpError:  params.Get("smtpError"),
	}

	// If email is empty, try to get it from the emailId
	if event.Email == "" && params.Get("emailId") != "" {
		emailID := params.Get("emailId")
		if strings.Contains(emailID, "@") {
			parts := strings.Split(emailID, "@")
			if len(parts) >= 2 {
				event.Email = strings.Join(parts[1:], "@")
			}
		}
	}

	// Parse timestamp if present
	if timestampStr := params.Get("timestamp"); timestampStr != "" {
		if ts, err := time.Parse("2006-01-02 15:04:05", timestampStr); err == nil {
			event.Timestamp = ts.Unix()
		}
	}

	return &event, nil
}

// EventTypeToStatus converts SendCloud event type to email status
func EventTypeToStatus(eventType string) string {
	switch eventType {
	case "1":
		return "delivered"
	case "3":
		return "spam"
	case "4":
		return "invalid"
	case "5":
		return "soft_bounce"
	case "10":
		return "clicked"
	case "11":
		return "opened"
	case "12":
		return "unsubscribed"
	case "18":
		return "sent"
	default:
		return "unknown"
	}
}
