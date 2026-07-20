package sms

import (
	"errors"
	"testing"
)

func TestNewProviderFromKind_valid(t *testing.T) {
	t.Parallel()
	cases := []struct {
		kind ProviderKind
		cfg  any
	}{
		{ProviderAliyun, AliyunConfig{AccessKeyID: "id", AccessKeySecret: "sec", SignName: "sign"}},
		{ProviderTencent, TencentConfig{SdkAppID: "1", SecretID: "id", SecretKey: "key", SignName: "sign"}},
		{ProviderHuawei, HuaweiConfig{AppKey: "k", AppSecret: "s", Sender: "10086"}},
		{ProviderYunpian, YunpianConfig{APIKey: "key"}},
		{ProviderSubmail, SubmailConfig{AppID: "id", AppKey: "key"}},
		{ProviderLuosimao, LuosimaoConfig{APIKey: "key"}},
		{ProviderYuntongxun, YuntongxunConfig{AppID: "a", AccountSID: "sid", AccountToken: "tok"}},
		{ProviderHuyi, HuyiConfig{APIID: "id", APIKey: "key"}},
		{ProviderJuhe, JuheConfig{AppKey: "key"}},
		{ProviderBaidu, BaiduConfig{AK: "ak", SK: "sk", SignatureID: "sig"}},
		{ProviderHuaxin, HuaxinConfig{UserID: "u", Password: "p"}},
		{ProviderChuanglan, ChuanglanConfig{Account: "a", Password: "p"}},
		{ProviderRongcloud, RongcloudConfig{AppKey: "k", AppSecret: "s"}},
		{ProviderTwilio, TwilioConfig{AccountSID: "sid", Token: "tok", From: "+1000"}},
		{ProviderTiniyo, TiniyoConfig{AccountSID: "sid", Token: "tok", From: "+1000"}},
		{ProviderUCloud, UCloudConfig{PublicKey: "pk", PrivateKey: "sk", ProjectID: "proj", Region: "cn-bj2"}},
		{ProviderNeteaseYunx, NeteaseConfig{AppKey: "k", AppSecret: "s"}},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(string(tc.kind), func(t *testing.T) {
			t.Parallel()
			p, err := NewProviderFromKind(tc.kind, tc.cfg)
			if err != nil {
				t.Fatalf("NewProviderFromKind: %v", err)
			}
			if p.Kind() != tc.kind {
				t.Errorf("Kind = %q, want %q", p.Kind(), tc.kind)
			}
		})
	}
}

func TestNewProviderFromKind_typeMismatch(t *testing.T) {
	t.Parallel()
	_, err := NewProviderFromKind(ProviderAliyun, TencentConfig{})
	if err == nil || !errors.Is(err, ErrInvalidConfig) {
		t.Errorf("expected ErrInvalidConfig, got %v", err)
	}
}

func TestNewProviderFromKind_unknown(t *testing.T) {
	t.Parallel()
	_, err := NewProviderFromKind(ProviderKind("unknown"), nil)
	if err == nil || !errors.Is(err, ErrInvalidConfig) {
		t.Errorf("expected ErrInvalidConfig, got %v", err)
	}
}

func TestNewProviderFromKindMap_yunpian(t *testing.T) {
	t.Parallel()
	p, err := NewProviderFromKindMap(ProviderYunpian, map[string]any{"apiKey": "test-key"})
	if err != nil {
		t.Fatalf("NewProviderFromKindMap: %v", err)
	}
	if p.Kind() != ProviderYunpian {
		t.Errorf("Kind = %q", p.Kind())
	}
}

func TestNewProviderFromKindMap_unknown(t *testing.T) {
	t.Parallel()
	_, err := NewProviderFromKindMap(ProviderKind("nope"), map[string]any{})
	if err == nil || !errors.Is(err, ErrNotImplemented) {
		t.Errorf("expected ErrNotImplemented, got %v", err)
	}
}
