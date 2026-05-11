package sms

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
	"time"
)

type HuaweiConfig struct {
	AppKey    string `json:"appKey"`
	AppSecret string `json:"appSecret"`
	Sender    string `json:"sender"`
	Signature string `json:"signature,omitempty"`
	Endpoint  string `json:"endpoint,omitempty"` // default https://smsapi.cn-north-4.myhuaweicloud.com:443
}

type HuaweiProvider struct {
	cfg HuaweiConfig
}

func NewHuawei(cfg HuaweiConfig) (*HuaweiProvider, error) {
	if strings.TrimSpace(cfg.AppKey) == "" || strings.TrimSpace(cfg.AppSecret) == "" {
		return nil, fmt.Errorf("%w: huawei requires appKey/appSecret", ErrInvalidConfig)
	}
	if strings.TrimSpace(cfg.Sender) == "" {
		return nil, fmt.Errorf("%w: huawei requires sender", ErrInvalidConfig)
	}
	return &HuaweiProvider{cfg: cfg}, nil
}

func (p *HuaweiProvider) Kind() ProviderKind { return ProviderHuawei }

func (p *HuaweiProvider) Send(ctx context.Context, req SendRequest) (*SendResult, error) {
	ctx = ctxOrBackground(ctx)
	if err := ValidateBasic(req); err != nil {
		return nil, err
	}
	// Huawei typically uses template + data.
	if strings.TrimSpace(req.Message.Template) == "" {
		return nil, fmt.Errorf("%w: huawei requires template", ErrInvalidArgument)
	}

	to, err := firstRecipient(req)
	if err != nil {
		return nil, err
	}
	endpoint := strings.TrimSpace(p.cfg.Endpoint)
	if endpoint == "" {
		endpoint = "https://smsapi.cn-north-4.myhuaweicloud.com:443"
	}
	created := time.Now().UTC().Format("2006-01-02T15:04:05Z")
	nonce := randHex(16)
	digest := sha256B64(nonce + created + strings.TrimSpace(p.cfg.AppSecret))
	xwsse := fmt.Sprintf(`UsernameToken Username="%s",PasswordDigest="%s",Nonce="%s",Created="%s"`,
		strings.TrimSpace(p.cfg.AppKey), digest, nonce, created)
	auth := `WSSE realm="SDP",profile="UsernameToken",type="Appkey"`

	// Template paras ordered: use Extras.params if provided; else map values.
	var params []string
	if req.Extras != nil {
		if arr, ok := req.Extras["params"]; ok {
			b, _ := json.Marshal(arr)
			_ = json.Unmarshal(b, &params)
		}
	}
	if len(params) == 0 {
		for _, v := range req.Message.Data {
			params = append(params, v)
		}
	}
	paras := "[]"
	if b, err := json.Marshal(params); err == nil {
		paras = string(b)
	}

	form := url.Values{}
	form.Set("from", strings.TrimSpace(p.cfg.Sender))
	form.Set("to", to)
	form.Set("templateId", strings.TrimSpace(req.Message.Template))
	form.Set("templateParas", paras)
	sig := strings.TrimSpace(req.Message.SignName)
	if sig == "" {
		sig = strings.TrimSpace(p.cfg.Signature)
	}
	if sig != "" {
		form.Set("signature", sig)
	}

	headers := map[string]string{
		"Authorization": auth,
		"X-WSSE":        xwsse,
	}
	status, body, err := postForm(ctx, endpoint+"/sms/batchSendSms/v1", form, headers, "", "")
	raw := truncateRaw(string(body), 4000)
	if err != nil {
		return &SendResult{Provider: p.Kind(), Accepted: false, Error: err.Error(), Raw: raw, SentAtUnix: nowUnix()}, err
	}
	var r struct {
		Code        string `json:"code"`
		Description string `json:"description"`
		Result      []struct {
			Status   string `json:"status"`
			SmsMsgID string `json:"smsMsgId"`
		} `json:"result"`
	}
	_ = json.Unmarshal(body, &r)
	if !is2xx(status) || strings.TrimSpace(r.Code) != "000000" {
		msg := strings.TrimSpace(r.Description)
		if msg == "" {
			msg = "provider rejected"
		}
		return &SendResult{Provider: p.Kind(), Accepted: false, Status: r.Code, Error: msg, Raw: raw, SentAtUnix: time.Now().Unix()}, errProviderRejected
	}
	msgID := ""
	if len(r.Result) > 0 {
		msgID = strings.TrimSpace(r.Result[0].SmsMsgID)
	}
	return &SendResult{Provider: p.Kind(), MessageID: msgID, Accepted: true, Status: r.Code, Raw: raw, SentAtUnix: time.Now().Unix()}, nil
}
