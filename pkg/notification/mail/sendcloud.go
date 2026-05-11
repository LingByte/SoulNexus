package mail

// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"gorm.io/gorm"
)

// SendCloudConfig holds SendCloud API credentials.
type SendCloudConfig struct {
	APIUser  string
	APIKey   string
	From     string
	FromName string
}

// SendCloudClient implements MailProvider via SendCloud HTTP API.
type SendCloudClient struct {
	Config SendCloudConfig
	Client *http.Client
	sender ParsedSender
}

// SendCloudWebhookEvent is a normalized webhook payload (JSON or form).
type SendCloudWebhookEvent struct {
	Event      string `json:"event"`
	MessageID  string `json:"messageId"`
	Email      string `json:"email"`
	Timestamp  int64  `json:"timestamp"`
	SmtpStatus string `json:"smtpStatus"`
	SmtpError  string `json:"smtpError"`
}

type sendCloudSendResponse struct {
	Result    bool   `json:"result"`
	Message   string `json:"message"`
	MessageID string `json:"messageId"`
	Data      struct {
		MessageID string `json:"messageId"`
	} `json:"data"`
	Info struct {
		EmailIDList []string `json:"emailIdList"`
		MessageID   string   `json:"messageId"`
	} `json:"info"`
}

// NewSendCloudClient creates a SendCloud provider with default HTTP timeout.
func NewSendCloudClient(config SendCloudConfig) (*SendCloudClient, error) {
	p, err := ParseMailSender(config.From, config.FromName)
	if err != nil {
		return nil, err
	}
	return &SendCloudClient{
		Config: config,
		Client: &http.Client{Timeout: 30 * time.Second},
		sender: p,
	}, nil
}

// Kind implements MailProvider.
func (s *SendCloudClient) Kind() string {
	return ProviderSendCloud
}

// SendHTMLWith sends HTML mail with variable substitution.
func (s *SendCloudClient) SendHTMLWith(to, subject, htmlBody string, vars map[string]any) (string, error) {
	return s.sendMail(to, subject, htmlBody, vars, "html")
}

// SendTextWith sends plain text mail with variable substitution.
func (s *SendCloudClient) SendTextWith(to, subject, textBody string, vars map[string]any) (string, error) {
	return s.sendMail(to, subject, textBody, vars, "plainText")
}

// sendMail is the shared implementation for SendHTMLWith and SendTextWith.
func (s *SendCloudClient) sendMail(to, subject, body string, vars map[string]any, bodyType string) (string, error) {
	const apiURL = "https://api.sendcloud.net/apiv2/mail/send"
	data := url.Values{}
	data.Set("apiUser", s.Config.APIUser)
	data.Set("apiKey", s.Config.APIKey)
	data.Set("to", to)
	data.Set("from", s.sender.Envelope)
	if s.sender.Display != "" {
		data.Set("fromName", s.sender.Display)
	}
	data.Set("subject", ReplacePlaceholders(subject, vars))
	data.Set(bodyType, ReplacePlaceholders(body, vars))

	req, err := http.NewRequest(http.MethodPost, apiURL, strings.NewReader(data.Encode()))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := s.Client.Do(req)
	if err != nil {
		return "", fmt.Errorf("sendcloud request: %w", err)
	}
	defer resp.Body.Close()

	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("sendcloud read body: %w", err)
	}

	var result sendCloudSendResponse
	if err := json.Unmarshal(responseBody, &result); err != nil {
		return "", fmt.Errorf("sendcloud json: %w", err)
	}

	if resp.StatusCode/100 != 2 {
		msg := strings.TrimSpace(result.Message)
		if msg == "" {
			msg = strings.TrimSpace(string(responseBody))
		}
		if msg == "" {
			msg = resp.Status
		}
		return "", fmt.Errorf("sendcloud http %d: %s", resp.StatusCode, msg)
	}
	if !result.Result {
		msg := strings.TrimSpace(result.Message)
		if msg == "" {
			msg = "request failed"
		}
		return "", fmt.Errorf("sendcloud: %s", msg)
	}
	if len(result.Info.EmailIDList) > 0 && result.Info.EmailIDList[0] != "" {
		return result.Info.EmailIDList[0], nil
	}
	if result.Info.MessageID != "" {
		return result.Info.MessageID, nil
	}
	if result.Data.MessageID != "" {
		return result.Data.MessageID, nil
	}
	if result.MessageID != "" {
		return result.MessageID, nil
	}
	return "", nil
}

// ParseSendCloudWebhookEvent parses JSON or x-www-form-urlencoded webhook bodies.
func ParseSendCloudWebhookEvent(data []byte) (*SendCloudWebhookEvent, error) {
	var event SendCloudWebhookEvent
	if err := json.Unmarshal(data, &event); err == nil && (event.Event != "" || event.MessageID != "") {
		return &event, nil
	}

	params, err := url.ParseQuery(string(data))
	if err != nil {
		return nil, fmt.Errorf("webhook parse: %w", err)
	}
	messageID := params.Get("messageId")
	if messageID == "" {
		messageID = params.Get("emailId")
	}
	if strings.Contains(messageID, "@") {
		parts := strings.Split(messageID, "@")
		if len(parts) > 0 {
			messageID = parts[0]
		}
	}
	event = SendCloudWebhookEvent{
		Event:      params.Get("event"),
		MessageID:  messageID,
		Email:      params.Get("recipient"),
		SmtpStatus: params.Get("smtpStatus"),
		SmtpError:  params.Get("smtpError"),
	}
	if event.Email == "" {
		if emailID := params.Get("emailId"); strings.Contains(emailID, "@") {
			parts := strings.Split(emailID, "@")
			if len(parts) >= 2 {
				event.Email = strings.Join(parts[1:], "@")
			}
		}
	}
	if ts := params.Get("timestamp"); ts != "" {
		if t, err := time.Parse("2006-01-02 15:04:05", ts); err == nil {
			event.Timestamp = t.Unix()
		}
	}
	return &event, nil
}

// ApplySendCloudWebhookToMailLog maps a webhook to MailLog status and updates the row (SendCloud only).
func ApplySendCloudWebhookToMailLog(db *gorm.DB, raw []byte) error {
	ev, err := ParseSendCloudWebhookEvent(raw)
	if err != nil {
		return err
	}
	if ev.MessageID == "" {
		return nil
	}
	status := SendCloudEventToStatus(ev.Event)
	errMsg := ""
	if ev.SmtpError != "" {
		errMsg = ev.SmtpError
	} else if ev.SmtpStatus != "" {
		errMsg = ev.SmtpStatus
	}
	return UpdateMailLogStatusByMessageID(db, ev.MessageID, string(ProviderSendCloud), status, errMsg)
}
