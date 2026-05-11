package sms

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"sort"
	"strconv"
	"strings"
)

// UCloudConfig configures UCloud SMS (USMS) API access.
// See https://docs.ucloud.cn/api/usms-api/send_usms_message
type UCloudConfig struct {
	PublicKey  string `json:"publicKey"`
	PrivateKey string `json:"privateKey"`
	ProjectID  string `json:"projectId"`
	Region     string `json:"region"` // e.g. cn-bj2
	APIBase    string `json:"apiBase,omitempty"`
}

type UCloudProvider struct {
	cfg UCloudConfig
}

func NewUCloud(cfg UCloudConfig) (*UCloudProvider, error) {
	if strings.TrimSpace(cfg.PublicKey) == "" || strings.TrimSpace(cfg.PrivateKey) == "" {
		return nil, fmt.Errorf("%w: ucloud requires publicKey/privateKey", ErrInvalidConfig)
	}
	if strings.TrimSpace(cfg.ProjectID) == "" {
		return nil, fmt.Errorf("%w: ucloud requires projectId", ErrInvalidConfig)
	}
	if strings.TrimSpace(cfg.Region) == "" {
		return nil, fmt.Errorf("%w: ucloud requires region", ErrInvalidConfig)
	}
	return &UCloudProvider{cfg: cfg}, nil
}

func (p *UCloudProvider) Kind() ProviderKind { return ProviderUCloud }

func (p *UCloudProvider) Send(ctx context.Context, req SendRequest) (*SendResult, error) {
	ctx = ctxOrBackground(ctx)
	if err := ValidateBasic(req); err != nil {
		return nil, err
	}

	tpl := strings.TrimSpace(req.Message.Template)
	if tpl == "" {
		return nil, fmt.Errorf("%w: ucloud requires templateId", ErrInvalidArgument)
	}

	params := map[string]any{
		"Action":    "SendUSMSMessage",
		"PublicKey": strings.TrimSpace(p.cfg.PublicKey),
		"ProjectId": strings.TrimSpace(p.cfg.ProjectID),
		"Region":    strings.TrimSpace(p.cfg.Region),
	}
	for i, to := range req.To {
		params[fmt.Sprintf("PhoneNumbers.%d", i)] = strings.TrimSpace(to.String())
	}
	params["TemplateId"] = tpl
	if len(req.Message.Data) > 0 {
		var ordered []string
		if req.Extras != nil {
			if arr, ok := req.Extras["templateParamOrder"]; ok {
				b, _ := json.Marshal(arr)
				_ = json.Unmarshal(b, &ordered)
			}
		}
		if len(ordered) == 0 {
			keys := make([]string, 0, len(req.Message.Data))
			for k := range req.Message.Data {
				keys = append(keys, k)
			}
			sort.Strings(keys)
			for _, k := range keys {
				ordered = append(ordered, k)
			}
		}
		for i, k := range ordered {
			params[fmt.Sprintf("TemplateParams.%d", i)] = strings.TrimSpace(req.Message.Data[k])
		}
	}
	sig := strings.TrimSpace(req.Message.SignName)
	if sig != "" {
		params["SigContent"] = sig
	}

	signStr := ucloudSignString(params, strings.TrimSpace(p.cfg.PrivateKey))
	params["Signature"] = sha1Hex(signStr)

	base := strings.TrimSpace(p.cfg.APIBase)
	if base == "" {
		base = "https://api.ucloud.cn"
	}
	u, err := url.Parse(base)
	if err != nil {
		return nil, fmt.Errorf("%w: invalid apiBase", ErrInvalidConfig)
	}
	q := u.Query()
	for k, v := range ucloudFlattenParams(params) {
		q.Set(k, v)
	}
	u.RawQuery = q.Encode()

	status, body, err := getURL(ctx, u.String(), nil, "", "")
	raw := truncateRaw(string(body), 4000)
	if err != nil {
		return &SendResult{Provider: p.Kind(), Accepted: false, Error: err.Error(), Raw: raw, SentAtUnix: nowUnix()}, err
	}

	var r struct {
		Action    string `json:"Action"`
		RetCode   int    `json:"RetCode"`
		Message   string `json:"Message"`
		SessionNo string `json:"SessionNo"`
	}
	_ = json.Unmarshal(body, &r)
	if !is2xx(status) || r.RetCode != 0 {
		msg := strings.TrimSpace(r.Message)
		if msg == "" {
			msg = "provider rejected"
		}
		return &SendResult{
			Provider:   p.Kind(),
			Accepted:   false,
			Status:     strconv.Itoa(r.RetCode),
			Error:      msg,
			Raw:        raw,
			SentAtUnix: nowUnix(),
		}, errProviderRejected
	}
	msgID := strings.TrimSpace(r.SessionNo)
	return &SendResult{Provider: p.Kind(), MessageID: msgID, Accepted: true, Status: "0", Raw: raw, SentAtUnix: nowUnix()}, nil
}

func ucloudSignString(params map[string]any, privateKey string) string {
	keys := make([]string, 0, len(params))
	for k := range params {
		if k == "Signature" {
			continue
		}
		keys = append(keys, k)
	}
	sort.Strings(keys)
	var sb strings.Builder
	for _, k := range keys {
		sb.WriteString(k)
		sb.WriteString(ucloudEncodeValue(params[k]))
	}
	sb.WriteString(privateKey)
	return sb.String()
}

func ucloudEncodeValue(v any) string {
	switch t := v.(type) {
	case nil:
		return ""
	case string:
		return t
	case bool:
		if t {
			return "true"
		}
		return "false"
	case float64:
		if t == float64(int64(t)) {
			return strconv.FormatInt(int64(t), 10)
		}
		return strconv.FormatFloat(t, 'f', -1, 64)
	case int:
		return strconv.Itoa(t)
	case int64:
		return strconv.FormatInt(t, 10)
	case uint64:
		return strconv.FormatUint(t, 10)
	case map[string]string:
		b, _ := json.Marshal(t)
		return string(b)
	case map[string]any:
		b, _ := json.Marshal(t)
		return string(b)
	default:
		b, _ := json.Marshal(t)
		return string(b)
	}
}

// ucloudFlattenParams turns nested maps/slices into UCloud-style dotted keys for query encoding.
func ucloudFlattenParams(params map[string]any) map[string]string {
	out := map[string]string{}
	for k, v := range params {
		if k == "Signature" {
			continue
		}
		ucloudFlattenKey("", k, v, out)
	}
	return out
}

func ucloudFlattenKey(prefix, key string, v any, out map[string]string) {
	full := key
	if prefix != "" {
		full = prefix + "." + key
	}
	switch t := v.(type) {
	case nil:
		return
	case map[string]any:
		for ck, cv := range t {
			ucloudFlattenKey(full, ck, cv, out)
		}
	case map[string]string:
		for ck, cv := range t {
			ucloudFlattenKey(full, ck, cv, out)
		}
	case []any:
		for i, item := range t {
			ucloudFlattenKey(full, strconv.Itoa(i), item, out)
		}
	case []string:
		for i, item := range t {
			ucloudFlattenKey(full, strconv.Itoa(i), item, out)
		}
	default:
		out[full] = ucloudEncodeValue(t)
	}
}
