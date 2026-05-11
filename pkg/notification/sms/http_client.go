package sms

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

var httpClient = &http.Client{Timeout: 12 * time.Second}

func postForm(ctx context.Context, endpoint string, form url.Values, headers map[string]string, basicUser, basicPass string) (status int, body []byte, err error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, strings.NewReader(form.Encode()))
	if err != nil {
		return 0, nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	if basicUser != "" || basicPass != "" {
		req.SetBasicAuth(basicUser, basicPass)
	}
	resp, err := httpClient.Do(req)
	if err != nil {
		return 0, nil, err
	}
	defer resp.Body.Close()
	b, _ := io.ReadAll(resp.Body)
	return resp.StatusCode, b, nil
}

func getURL(ctx context.Context, endpoint string, headers map[string]string, basicUser, basicPass string) (status int, body []byte, err error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return 0, nil, err
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	if basicUser != "" || basicPass != "" {
		req.SetBasicAuth(basicUser, basicPass)
	}
	resp, err := httpClient.Do(req)
	if err != nil {
		return 0, nil, err
	}
	defer resp.Body.Close()
	b, _ := io.ReadAll(resp.Body)
	return resp.StatusCode, b, nil
}

func postJSON(ctx context.Context, endpoint string, jsonBody []byte, headers map[string]string, basicUser, basicPass string) (status int, body []byte, err error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, strings.NewReader(string(jsonBody)))
	if err != nil {
		return 0, nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	if basicUser != "" || basicPass != "" {
		req.SetBasicAuth(basicUser, basicPass)
	}
	resp, err := httpClient.Do(req)
	if err != nil {
		return 0, nil, err
	}
	defer resp.Body.Close()
	b, _ := io.ReadAll(resp.Body)
	return resp.StatusCode, b, nil
}

func jsonString(v any) string {
	b, err := json.Marshal(v)
	if err != nil {
		return ""
	}
	return string(b)
}

func is2xx(code int) bool { return code >= 200 && code < 300 }

func truncateRaw(s string, max int) string {
	s = strings.TrimSpace(s)
	if max <= 0 || len(s) <= max {
		return s
	}
	return s[:max] + "…"
}

func ctxOrBackground(ctx context.Context) context.Context {
	if ctx == nil {
		return context.Background()
	}
	return ctx
}

func firstRecipient(req SendRequest) (string, error) {
	if len(req.To) == 0 {
		return "", ErrInvalidArgument
	}
	n := strings.TrimSpace(req.To[0].Number)
	if n == "" {
		return "", ErrInvalidArgument
	}
	return n, nil
}

func normalizeContent(content, fallbackSignature string) string {
	c := strings.TrimSpace(content)
	if c == "" {
		return ""
	}
	// If content already includes Chinese signature brackets, keep.
	if strings.Contains(c, "【") && strings.Contains(c, "】") {
		return c
	}
	sig := strings.TrimSpace(fallbackSignature)
	if sig == "" {
		return c
	}
	return "【" + sig + "】" + c
}

var errProviderRejected = errors.New("sms: provider rejected")
