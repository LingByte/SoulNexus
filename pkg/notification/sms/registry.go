package sms

import (
	"encoding/json"
	"fmt"
)

// NewProviderFromKind returns a Provider instance for the given kind.
// This is a light factory to keep call sites simple; it does not implement routing/strategy (avoid over-encapsulation).
func NewProviderFromKind(kind ProviderKind, cfg any) (Provider, error) {
	switch kind {
	case ProviderAliyun:
		c, ok := cfg.(AliyunConfig)
		if !ok {
			return nil, fmt.Errorf("%w: aliyun config type mismatch", ErrInvalidConfig)
		}
		return NewAliyun(c)
	case ProviderTencent:
		c, ok := cfg.(TencentConfig)
		if !ok {
			return nil, fmt.Errorf("%w: tencent config type mismatch", ErrInvalidConfig)
		}
		return NewTencent(c)
	case ProviderHuawei:
		c, ok := cfg.(HuaweiConfig)
		if !ok {
			return nil, fmt.Errorf("%w: huawei config type mismatch", ErrInvalidConfig)
		}
		return NewHuawei(c)
	case ProviderYunpian:
		c, ok := cfg.(YunpianConfig)
		if !ok {
			return nil, fmt.Errorf("%w: yunpian config type mismatch", ErrInvalidConfig)
		}
		return NewYunpian(c)
	case ProviderSubmail:
		c, ok := cfg.(SubmailConfig)
		if !ok {
			return nil, fmt.Errorf("%w: submail config type mismatch", ErrInvalidConfig)
		}
		return NewSubmail(c)
	case ProviderLuosimao:
		c, ok := cfg.(LuosimaoConfig)
		if !ok {
			return nil, fmt.Errorf("%w: luosimao config type mismatch", ErrInvalidConfig)
		}
		return NewLuosimao(c)
	case ProviderYuntongxun:
		c, ok := cfg.(YuntongxunConfig)
		if !ok {
			return nil, fmt.Errorf("%w: yuntongxun config type mismatch", ErrInvalidConfig)
		}
		return NewYuntongxun(c)
	case ProviderHuyi:
		c, ok := cfg.(HuyiConfig)
		if !ok {
			return nil, fmt.Errorf("%w: huyi config type mismatch", ErrInvalidConfig)
		}
		return NewHuyi(c)
	case ProviderJuhe:
		c, ok := cfg.(JuheConfig)
		if !ok {
			return nil, fmt.Errorf("%w: juhe config type mismatch", ErrInvalidConfig)
		}
		return NewJuhe(c)
	case ProviderBaidu:
		c, ok := cfg.(BaiduConfig)
		if !ok {
			return nil, fmt.Errorf("%w: baidu config type mismatch", ErrInvalidConfig)
		}
		return NewBaidu(c)
	case ProviderHuaxin:
		c, ok := cfg.(HuaxinConfig)
		if !ok {
			return nil, fmt.Errorf("%w: huaxin config type mismatch", ErrInvalidConfig)
		}
		return NewHuaxin(c)
	case ProviderChuanglan:
		c, ok := cfg.(ChuanglanConfig)
		if !ok {
			return nil, fmt.Errorf("%w: chuanglan config type mismatch", ErrInvalidConfig)
		}
		return NewChuanglan(c)
	case ProviderRongcloud:
		c, ok := cfg.(RongcloudConfig)
		if !ok {
			return nil, fmt.Errorf("%w: rongcloud config type mismatch", ErrInvalidConfig)
		}
		return NewRongcloud(c)
	case ProviderTwilio:
		c, ok := cfg.(TwilioConfig)
		if !ok {
			return nil, fmt.Errorf("%w: twilio config type mismatch", ErrInvalidConfig)
		}
		return NewTwilio(c)
	case ProviderTiniyo:
		c, ok := cfg.(TiniyoConfig)
		if !ok {
			return nil, fmt.Errorf("%w: tiniyo config type mismatch", ErrInvalidConfig)
		}
		return NewTiniyo(c)
	case ProviderUCloud:
		c, ok := cfg.(UCloudConfig)
		if !ok {
			return nil, fmt.Errorf("%w: ucloud config type mismatch", ErrInvalidConfig)
		}
		return NewUCloud(c)
	case ProviderNeteaseYunx:
		c, ok := cfg.(NeteaseConfig)
		if !ok {
			return nil, fmt.Errorf("%w: netease config type mismatch", ErrInvalidConfig)
		}
		return NewNetease(c)
	default:
		return nil, fmt.Errorf("%w: unknown provider %q", ErrInvalidConfig, kind)
	}
}

// NewProviderFromKindMap unmarshals a map config into the right provider config struct.
// This is used by notification_channels.config_json where config is persisted as JSON.
func NewProviderFromKindMap(kind ProviderKind, cfg map[string]any) (Provider, error) {
	b, err := json.Marshal(cfg)
	if err != nil {
		return nil, err
	}
	switch kind {
	case ProviderYunpian:
		var c YunpianConfig
		_ = json.Unmarshal(b, &c)
		return NewYunpian(c)
	case ProviderLuosimao:
		var c LuosimaoConfig
		_ = json.Unmarshal(b, &c)
		return NewLuosimao(c)
	case ProviderTwilio:
		var c TwilioConfig
		_ = json.Unmarshal(b, &c)
		return NewTwilio(c)
	case ProviderHuyi:
		var c HuyiConfig
		_ = json.Unmarshal(b, &c)
		return NewHuyi(c)
	case ProviderJuhe:
		var c JuheConfig
		_ = json.Unmarshal(b, &c)
		return NewJuhe(c)
	case ProviderChuanglan:
		var c ChuanglanConfig
		_ = json.Unmarshal(b, &c)
		return NewChuanglan(c)
	case ProviderSubmail:
		var c SubmailConfig
		_ = json.Unmarshal(b, &c)
		return NewSubmail(c)
	case ProviderTencent:
		var c TencentConfig
		_ = json.Unmarshal(b, &c)
		return NewTencent(c)
	case ProviderAliyun:
		var c AliyunConfig
		_ = json.Unmarshal(b, &c)
		return NewAliyun(c)
	case ProviderHuawei:
		var c HuaweiConfig
		_ = json.Unmarshal(b, &c)
		return NewHuawei(c)
	case ProviderRongcloud:
		var c RongcloudConfig
		_ = json.Unmarshal(b, &c)
		return NewRongcloud(c)
	case ProviderNeteaseYunx:
		var c NeteaseConfig
		_ = json.Unmarshal(b, &c)
		return NewNetease(c)
	case ProviderYuntongxun:
		var c YuntongxunConfig
		_ = json.Unmarshal(b, &c)
		return NewYuntongxun(c)
	case ProviderBaidu:
		var c BaiduConfig
		_ = json.Unmarshal(b, &c)
		return NewBaidu(c)
	case ProviderUCloud:
		var c UCloudConfig
		_ = json.Unmarshal(b, &c)
		return NewUCloud(c)
	case ProviderTiniyo:
		var c TiniyoConfig
		_ = json.Unmarshal(b, &c)
		return NewTiniyo(c)
	case ProviderHuaxin:
		var c HuaxinConfig
		_ = json.Unmarshal(b, &c)
		return NewHuaxin(c)
	default:
		return nil, fmt.Errorf("%w: provider %q not wired yet", ErrNotImplemented, kind)
	}
}
