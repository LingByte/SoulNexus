package sms

import (
	"context"
	"encoding/xml"
	"fmt"
	"net/url"
	"strings"
)

// HuaxinConfig is for 华信/短信通 HTTP gateway (sms.aspx style).
type HuaxinConfig struct {
	UserID   string `json:"userId"`
	Password string `json:"password"`
	Account  string `json:"account"`
	// BaseURL e.g. http://sms.example.com:8080  (path /sms.aspx appended)
	BaseURL string `json:"baseUrl"`
	IP      string `json:"ip,omitempty"` // legacy alias for host in BaseURL
	ExtNo   string `json:"extNo,omitempty"`
}

type HuaxinProvider struct {
	cfg HuaxinConfig
}

func NewHuaxin(cfg HuaxinConfig) (*HuaxinProvider, error) {
	if strings.TrimSpace(cfg.UserID) == "" || strings.TrimSpace(cfg.Password) == "" {
		return nil, fmt.Errorf("%w: huaxin requires userId/password", ErrInvalidConfig)
	}
	if strings.TrimSpace(cfg.Account) == "" {
		cfg.Account = strings.TrimSpace(cfg.UserID)
	}
	return &HuaxinProvider{cfg: cfg}, nil
}

func (p *HuaxinProvider) Kind() ProviderKind { return ProviderHuaxin }

func (p *HuaxinProvider) Send(ctx context.Context, req SendRequest) (*SendResult, error) {
	ctx = ctxOrBackground(ctx)
	if err := ValidateBasic(req); err != nil {
		return nil, err
	}
	content := strings.TrimSpace(req.Message.Content)
	if content == "" {
		if strings.TrimSpace(req.Message.Template) == "" {
			return nil, fmt.Errorf("%w: huaxin requires content", ErrInvalidArgument)
		}
		content = strings.TrimSpace(req.Message.Template)
		for k, v := range req.Message.Data {
			content = strings.ReplaceAll(content, "${"+k+"}", v)
			content = strings.ReplaceAll(content, "{"+k+"}", v)
		}
	}
	to, err := firstRecipient(req)
	if err != nil {
		return nil, err
	}

	endpoint := strings.TrimSpace(p.cfg.BaseURL)
	if endpoint == "" && strings.TrimSpace(p.cfg.IP) != "" {
		endpoint = "http://" + strings.TrimSpace(p.cfg.IP)
	}
	if endpoint == "" {
		return nil, fmt.Errorf("%w: huaxin requires baseUrl", ErrInvalidConfig)
	}
	endpoint = strings.TrimRight(endpoint, "/")
	if !strings.HasSuffix(strings.ToLower(endpoint), "/sms.aspx") {
		endpoint += "/sms.aspx"
	}

	form := url.Values{}
	form.Set("action", "send")
	form.Set("userid", strings.TrimSpace(p.cfg.UserID))
	form.Set("account", strings.TrimSpace(p.cfg.Account))
	form.Set("password", strings.TrimSpace(p.cfg.Password))
	form.Set("mobile", to)
	form.Set("content", content)
	form.Set("sendTime", "")
	form.Set("extno", strings.TrimSpace(p.cfg.ExtNo))

	status, body, err := postForm(ctx, endpoint, form, nil, "", "")
	raw := truncateRaw(string(body), 4000)
	if err != nil {
		return &SendResult{Provider: p.Kind(), Accepted: false, Error: err.Error(), Raw: raw, SentAtUnix: nowUnix()}, err
	}

	// Typical XML: <returnsms><returnstatus>Success</returnstatus><message>ok</message><taskID>…</taskID></returnsms>
	var xr struct {
		ReturnStatus  string `xml:"returnstatus"`
		Message       string `xml:"message"`
		TaskID        string `xml:"taskID"`
		SuccessCounts string `xml:"successCounts"`
	}
	_ = xml.Unmarshal(body, &xr)
	st := strings.ToLower(strings.TrimSpace(xr.ReturnStatus))
	ok := st == "success" || st == "ok"
	if !is2xx(status) || !ok {
		msg := strings.TrimSpace(xr.Message)
		if msg == "" {
			msg = "provider rejected"
		}
		return &SendResult{Provider: p.Kind(), Accepted: false, Status: xr.ReturnStatus, Error: msg, Raw: raw, SentAtUnix: nowUnix()}, errProviderRejected
	}
	return &SendResult{
		Provider: p.Kind(), MessageID: strings.TrimSpace(xr.TaskID),
		Accepted: true, Status: xr.ReturnStatus, Raw: raw, SentAtUnix: nowUnix(),
	}, nil
}
