package gateway

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
)

// MergeDialogPayloadQuery merges a JSON payload object into the dialog URL query as payload=.
func MergeDialogPayloadQuery(raw string, payload []byte) (string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", fmt.Errorf("gateway: empty dialog URL")
	}
	ps := strings.TrimSpace(string(payload))
	if len(payload) == 0 || ps == "" || ps == "null" {
		return raw, nil
	}
	if !json.Valid(payload) {
		return "", fmt.Errorf("gateway: invalid payload JSON")
	}
	u, err := url.Parse(raw)
	if err != nil {
		return "", fmt.Errorf("gateway: parse URL: %w", err)
	}
	q := u.Query()
	q.Set("payload", ps)
	u.RawQuery = q.Encode()
	return u.String(), nil
}

// MergeDialogQueryParams adds apiKey/apiSecret/agentId query parameters.
func MergeDialogQueryParams(raw, apiKey, apiSecret, agentID string) (string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", fmt.Errorf("gateway: empty dialog URL")
	}
	u, err := url.Parse(raw)
	if err != nil {
		return "", fmt.Errorf("gateway: parse URL: %w", err)
	}
	q := u.Query()
	if v := strings.TrimSpace(apiKey); v != "" {
		q.Set("apiKey", v)
	}
	if v := strings.TrimSpace(apiSecret); v != "" {
		q.Set("apiSecret", v)
	}
	if v := strings.TrimSpace(agentID); v != "" {
		q.Set("agentId", v)
	}
	u.RawQuery = q.Encode()
	return u.String(), nil
}

func appendCallIDQuery(raw, callID string) (string, error) {
	u, err := url.Parse(raw)
	if err != nil {
		return "", fmt.Errorf("gateway: parse URL: %w", err)
	}
	q := u.Query()
	q.Set("call_id", callID)
	u.RawQuery = q.Encode()
	return u.String(), nil
}

// RedactDialogDialURL redacts sensitive query values for logs.
func RedactDialogDialURL(u string) string {
	parsed, err := url.Parse(u)
	if err != nil {
		return "<invalid-dialog-url>"
	}
	q := parsed.Query()
	if q.Get("apiKey") != "" {
		q.Set("apiKey", "***")
	}
	if q.Get("apiSecret") != "" {
		q.Set("apiSecret", "***")
	}
	if q.Get("payload") != "" {
		q.Set("payload", "***")
	}
	parsed.RawQuery = q.Encode()
	return parsed.String()
}
