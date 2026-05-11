package sms

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	baidusms "github.com/baidubce/bce-sdk-go/services/sms"
	"github.com/baidubce/bce-sdk-go/services/sms/api"
)

type BaiduConfig struct {
	AK          string `json:"ak"`
	SK          string `json:"sk"`
	SignatureID string `json:"signatureId,omitempty"`
	// InvokeId is kept for backward compatibility with older configs / naming.
	InvokeID string `json:"invokeId,omitempty"`
	Domain   string `json:"domain,omitempty"` // e.g. https://smsv3.bj.baidubce.com
}

type BaiduProvider struct {
	cfg BaiduConfig
}

func NewBaidu(cfg BaiduConfig) (*BaiduProvider, error) {
	if strings.TrimSpace(cfg.AK) == "" || strings.TrimSpace(cfg.SK) == "" {
		return nil, fmt.Errorf("%w: baidu requires ak/sk", ErrInvalidConfig)
	}
	sig := strings.TrimSpace(cfg.SignatureID)
	if sig == "" {
		sig = strings.TrimSpace(cfg.InvokeID)
	}
	if sig == "" {
		return nil, fmt.Errorf("%w: baidu requires signatureId (or invokeId)", ErrInvalidConfig)
	}
	return &BaiduProvider{cfg: cfg}, nil
}

func (p *BaiduProvider) Kind() ProviderKind { return ProviderBaidu }

func (p *BaiduProvider) Send(ctx context.Context, req SendRequest) (*SendResult, error) {
	_ = ctx
	if err := ValidateBasic(req); err != nil {
		return nil, err
	}
	if strings.TrimSpace(req.Message.Template) == "" {
		return nil, fmt.Errorf("%w: baidu requires template", ErrInvalidArgument)
	}

	endpoint := strings.TrimSpace(p.cfg.Domain)
	if endpoint == "" {
		endpoint = "https://smsv3.bj.baidubce.com"
	}
	if !strings.HasPrefix(endpoint, "http://") && !strings.HasPrefix(endpoint, "https://") {
		endpoint = "https://" + endpoint
	}

	cli, err := baidusms.NewClient(strings.TrimSpace(p.cfg.AK), strings.TrimSpace(p.cfg.SK), endpoint)
	if err != nil {
		return nil, err
	}

	var mobiles []string
	for _, to := range req.To {
		mobiles = append(mobiles, strings.TrimSpace(to.String()))
	}
	mobileJoined := strings.Join(mobiles, ",")

	sigID := strings.TrimSpace(p.cfg.SignatureID)
	if sigID == "" {
		sigID = strings.TrimSpace(p.cfg.InvokeID)
	}
	if alt := strings.TrimSpace(req.Message.SignName); alt != "" {
		sigID = alt
	}

	contentVar := map[string]interface{}{}
	for k, v := range req.Message.Data {
		contentVar[k] = v
	}

	args := &api.SendSmsArgs{
		Mobile:      mobileJoined,
		Template:    strings.TrimSpace(req.Message.Template),
		SignatureId: sigID,
		ContentVar:  contentVar,
	}

	res, err := cli.SendSms(args)
	raw := ""
	if b, mErr := json.Marshal(res); mErr == nil {
		raw = truncateRaw(string(b), 4000)
	}
	if err != nil {
		return &SendResult{Provider: p.Kind(), Accepted: false, Error: err.Error(), Raw: raw, SentAtUnix: nowUnix()}, err
	}
	if res == nil || strings.TrimSpace(res.Code) != "1000" {
		msg := ""
		if res != nil {
			msg = strings.TrimSpace(res.Message)
		}
		if msg == "" {
			msg = "provider rejected"
		}
		code := ""
		if res != nil {
			code = strings.TrimSpace(res.Code)
		}
		return &SendResult{Provider: p.Kind(), Accepted: false, Status: code, Error: msg, Raw: raw, SentAtUnix: nowUnix()}, errProviderRejected
	}
	msgID := ""
	if len(res.Data) > 0 {
		msgID = strings.TrimSpace(res.Data[0].MessageId)
	}
	return &SendResult{Provider: p.Kind(), MessageID: msgID, Accepted: true, Status: res.Code, Raw: raw, SentAtUnix: nowUnix()}, nil
}
