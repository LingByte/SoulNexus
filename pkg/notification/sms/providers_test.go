package sms

import (
	"context"
	"errors"
	"strings"
	"testing"
)

// --- constructor validation (invalid configs) ---

func TestNewAliyun_invalid(t *testing.T) {
	t.Parallel()
	tests := []AliyunConfig{
		{AccessKeySecret: "s", SignName: "sign"},
		{AccessKeyID: "id", SignName: "sign"},
		{AccessKeyID: "id", AccessKeySecret: "s"},
	}
	for _, cfg := range tests {
		if _, err := NewAliyun(cfg); err == nil || !errors.Is(err, ErrInvalidConfig) {
			t.Errorf("cfg=%+v err=%v", cfg, err)
		}
	}
}

func TestNewTencent_invalid(t *testing.T) {
	t.Parallel()
	if _, err := NewTencent(TencentConfig{SecretID: "a", SecretKey: "b", SignName: "s"}); err == nil {
		t.Error("expected error")
	}
}

func TestNewHuawei_invalid(t *testing.T) {
	t.Parallel()
	if _, err := NewHuawei(HuaweiConfig{AppKey: "k"}); err == nil {
		t.Error("expected error")
	}
}

func TestNewYunpian_invalid(t *testing.T) {
	t.Parallel()
	if _, err := NewYunpian(YunpianConfig{}); err == nil {
		t.Error("expected error")
	}
}

func TestNewSubmail_invalid(t *testing.T) {
	t.Parallel()
	if _, err := NewSubmail(SubmailConfig{AppID: "id"}); err == nil {
		t.Error("expected error")
	}
}

func TestNewLuosimao_invalid(t *testing.T) {
	t.Parallel()
	if _, err := NewLuosimao(LuosimaoConfig{}); err == nil {
		t.Error("expected error")
	}
}

func TestNewYuntongxun_invalid(t *testing.T) {
	t.Parallel()
	if _, err := NewYuntongxun(YuntongxunConfig{AppID: "a", AccountSID: "s"}); err == nil {
		t.Error("expected error")
	}
}

func TestNewHuyi_invalid(t *testing.T) {
	t.Parallel()
	if _, err := NewHuyi(HuyiConfig{APIID: "id"}); err == nil {
		t.Error("expected error")
	}
}

func TestNewJuhe_invalid(t *testing.T) {
	t.Parallel()
	if _, err := NewJuhe(JuheConfig{}); err == nil {
		t.Error("expected error")
	}
}

func TestNewBaidu_invalid(t *testing.T) {
	t.Parallel()
	if _, err := NewBaidu(BaiduConfig{AK: "ak", SK: "sk"}); err == nil {
		t.Error("expected signatureId error")
	}
	if _, err := NewBaidu(BaiduConfig{AK: "ak"}); err == nil {
		t.Error("expected error")
	}
}

func TestNewBaidu_invokeIDFallback(t *testing.T) {
	t.Parallel()
	p, err := NewBaidu(BaiduConfig{AK: "ak", SK: "sk", InvokeID: "legacy"})
	if err != nil {
		t.Fatalf("NewBaidu: %v", err)
	}
	if p.Kind() != ProviderBaidu {
		t.Errorf("Kind = %q", p.Kind())
	}
}

func TestNewHuaxin_invalid(t *testing.T) {
	t.Parallel()
	if _, err := NewHuaxin(HuaxinConfig{UserID: "u"}); err == nil {
		t.Error("expected error")
	}
}

func TestNewChuanglan_invalid(t *testing.T) {
	t.Parallel()
	if _, err := NewChuanglan(ChuanglanConfig{Account: "a"}); err == nil {
		t.Error("expected error")
	}
}

func TestNewRongcloud_invalid(t *testing.T) {
	t.Parallel()
	if _, err := NewRongcloud(RongcloudConfig{AppKey: "k"}); err == nil {
		t.Error("expected error")
	}
}

func TestNewTwilio_invalid(t *testing.T) {
	t.Parallel()
	if _, err := NewTwilio(TwilioConfig{AccountSID: "s", Token: "t"}); err == nil {
		t.Error("expected error")
	}
}

func TestNewTiniyo_invalid(t *testing.T) {
	t.Parallel()
	if _, err := NewTiniyo(TiniyoConfig{AccountSID: "s", Token: "t"}); err == nil {
		t.Error("expected error")
	}
}

func TestNewUCloud_invalid(t *testing.T) {
	t.Parallel()
	base := UCloudConfig{PublicKey: "pk", PrivateKey: "sk", ProjectID: "p", Region: "cn-bj2"}
	tests := []UCloudConfig{
		{PrivateKey: base.PrivateKey, ProjectID: base.ProjectID, Region: base.Region},
		{PublicKey: base.PublicKey, ProjectID: base.ProjectID, Region: base.Region},
		{PublicKey: base.PublicKey, PrivateKey: base.PrivateKey, Region: base.Region},
		{PublicKey: base.PublicKey, PrivateKey: base.PrivateKey, ProjectID: base.ProjectID},
	}
	for _, cfg := range tests {
		if _, err := NewUCloud(cfg); err == nil {
			t.Errorf("expected error for cfg=%+v", cfg)
		}
	}
}

func TestNewNetease_invalid(t *testing.T) {
	t.Parallel()
	if _, err := NewNetease(NeteaseConfig{AppKey: "k"}); err == nil {
		t.Error("expected error")
	}
}

// --- Send preflight (no network) ---

func validTo() []PhoneNumber {
	return []PhoneNumber{{Number: "13800138000"}}
}

func TestProviderSend_validateBasic(t *testing.T) {
	t.Parallel()
	yp, err := NewYunpian(YunpianConfig{APIKey: "k"})
	if err != nil {
		t.Fatal(err)
	}
	ap, err := NewAliyun(AliyunConfig{AccessKeyID: "id", AccessKeySecret: "sec", SignName: "s"})
	if err != nil {
		t.Fatal(err)
	}
	for _, p := range []Provider{yp, ap} {
		_, err := p.Send(context.Background(), SendRequest{})
		if !errors.Is(err, ErrInvalidArgument) {
			t.Errorf("%s Send empty: err=%v", p.Kind(), err)
		}
	}
}

func TestAliyunSend_requiresTemplate(t *testing.T) {
	t.Parallel()
	p, err := NewAliyun(AliyunConfig{AccessKeyID: "id", AccessKeySecret: "sec", SignName: "s"})
	if err != nil {
		t.Fatal(err)
	}
	_, err = p.Send(context.Background(), SendRequest{To: validTo(), Message: Message{Content: "hi"}})
	if err == nil || !errors.Is(err, ErrInvalidArgument) {
		t.Errorf("err = %v", err)
	}
}

func TestTencentSend_requiresTemplate(t *testing.T) {
	t.Parallel()
	p, err := NewTencent(TencentConfig{SdkAppID: "1", SecretID: "id", SecretKey: "key", SignName: "s"})
	if err != nil {
		t.Fatal(err)
	}
	_, err = p.Send(context.Background(), SendRequest{To: validTo(), Message: Message{Content: "hi"}})
	if !errors.Is(err, ErrInvalidArgument) {
		t.Errorf("err = %v", err)
	}
}

func TestHuaweiSend_requiresTemplate(t *testing.T) {
	t.Parallel()
	p, err := NewHuawei(HuaweiConfig{AppKey: "k", AppSecret: "s", Sender: "10086"})
	if err != nil {
		t.Fatal(err)
	}
	_, err = p.Send(context.Background(), SendRequest{To: validTo(), Message: Message{Content: "hi"}})
	if !errors.Is(err, ErrInvalidArgument) {
		t.Errorf("err = %v", err)
	}
}

func TestYunpianSend_requiresContent(t *testing.T) {
	t.Parallel()
	p, err := NewYunpian(YunpianConfig{APIKey: "k"})
	if err != nil {
		t.Fatal(err)
	}
	_, err = p.Send(context.Background(), SendRequest{To: validTo(), Message: Message{Template: "T1"}})
	if !errors.Is(err, ErrInvalidArgument) {
		t.Errorf("err = %v", err)
	}
}

func TestSubmailSend_requiresContentOrTemplate(t *testing.T) {
	t.Parallel()
	p, err := NewSubmail(SubmailConfig{AppID: "id", AppKey: "key"})
	if err != nil {
		t.Fatal(err)
	}
	_, err = p.Send(context.Background(), SendRequest{To: validTo(), Message: Message{}})
	if !errors.Is(err, ErrInvalidArgument) {
		t.Errorf("err = %v", err)
	}
}

func TestLuosimaoSend_requiresContent(t *testing.T) {
	t.Parallel()
	p, err := NewLuosimao(LuosimaoConfig{APIKey: "k"})
	if err != nil {
		t.Fatal(err)
	}
	_, err = p.Send(context.Background(), SendRequest{To: validTo(), Message: Message{Template: "T"}})
	if !errors.Is(err, ErrInvalidArgument) {
		t.Errorf("err = %v", err)
	}
}

func TestYuntongxunSend_requiresTemplate(t *testing.T) {
	t.Parallel()
	p, err := NewYuntongxun(YuntongxunConfig{AppID: "a", AccountSID: "sid", AccountToken: "tok"})
	if err != nil {
		t.Fatal(err)
	}
	_, err = p.Send(context.Background(), SendRequest{To: validTo(), Message: Message{Content: "c"}})
	if !errors.Is(err, ErrInvalidArgument) {
		t.Errorf("err = %v", err)
	}
}

func TestHuyiSend_requiresContentOrTemplate(t *testing.T) {
	t.Parallel()
	p, err := NewHuyi(HuyiConfig{APIID: "id", APIKey: "key"})
	if err != nil {
		t.Fatal(err)
	}
	_, err = p.Send(context.Background(), SendRequest{To: validTo(), Message: Message{}})
	if !errors.Is(err, ErrInvalidArgument) {
		t.Errorf("err = %v", err)
	}
}

func TestJuheSend_requiresTemplate(t *testing.T) {
	t.Parallel()
	p, err := NewJuhe(JuheConfig{AppKey: "k"})
	if err != nil {
		t.Fatal(err)
	}
	_, err = p.Send(context.Background(), SendRequest{To: validTo(), Message: Message{Content: "c"}})
	if !errors.Is(err, ErrInvalidArgument) {
		t.Errorf("err = %v", err)
	}
}

func TestBaiduSend_requiresTemplate(t *testing.T) {
	t.Parallel()
	p, err := NewBaidu(BaiduConfig{AK: "ak", SK: "sk", SignatureID: "sig"})
	if err != nil {
		t.Fatal(err)
	}
	_, err = p.Send(context.Background(), SendRequest{To: validTo(), Message: Message{Content: "c"}})
	if !errors.Is(err, ErrInvalidArgument) {
		t.Errorf("err = %v", err)
	}
}

func TestHuaxinSend_requiresBaseURL(t *testing.T) {
	t.Parallel()
	p, err := NewHuaxin(HuaxinConfig{UserID: "u", Password: "p"})
	if err != nil {
		t.Fatal(err)
	}
	_, err = p.Send(context.Background(), SendRequest{To: validTo(), Message: Message{Content: "c"}})
	if !errors.Is(err, ErrInvalidConfig) {
		t.Errorf("err = %v", err)
	}
}

func TestChuanglanSend_requiresContent(t *testing.T) {
	t.Parallel()
	p, err := NewChuanglan(ChuanglanConfig{Account: "a", Password: "p"})
	if err != nil {
		t.Fatal(err)
	}
	_, err = p.Send(context.Background(), SendRequest{To: validTo(), Message: Message{}})
	if !errors.Is(err, ErrInvalidArgument) {
		t.Errorf("err = %v", err)
	}
}

func TestRongcloudSend_requiresTemplate(t *testing.T) {
	t.Parallel()
	p, err := NewRongcloud(RongcloudConfig{AppKey: "k", AppSecret: "s"})
	if err != nil {
		t.Fatal(err)
	}
	_, err = p.Send(context.Background(), SendRequest{To: validTo(), Message: Message{Content: "c"}})
	if !errors.Is(err, ErrInvalidArgument) {
		t.Errorf("err = %v", err)
	}
}

func TestTwilioSend_requiresContent(t *testing.T) {
	t.Parallel()
	p, err := NewTwilio(TwilioConfig{AccountSID: "sid", Token: "tok", From: "+1"})
	if err != nil {
		t.Fatal(err)
	}
	_, err = p.Send(context.Background(), SendRequest{To: validTo(), Message: Message{Template: "T"}})
	if !errors.Is(err, ErrInvalidArgument) {
		t.Errorf("err = %v", err)
	}
}

func TestTiniyoSend_requiresContent(t *testing.T) {
	t.Parallel()
	p, err := NewTiniyo(TiniyoConfig{AccountSID: "sid", Token: "tok", From: "+1"})
	if err != nil {
		t.Fatal(err)
	}
	_, err = p.Send(context.Background(), SendRequest{To: validTo(), Message: Message{Template: "T"}})
	if !errors.Is(err, ErrInvalidArgument) {
		t.Errorf("err = %v", err)
	}
}

func TestUCloudSend_requiresTemplate(t *testing.T) {
	t.Parallel()
	p, err := NewUCloud(UCloudConfig{PublicKey: "pk", PrivateKey: "sk", ProjectID: "proj", Region: "cn-bj2"})
	if err != nil {
		t.Fatal(err)
	}
	_, err = p.Send(context.Background(), SendRequest{To: validTo(), Message: Message{Content: "c"}})
	if !errors.Is(err, ErrInvalidArgument) {
		t.Errorf("err = %v", err)
	}
}

func TestNeteaseSend_requiresTemplate(t *testing.T) {
	t.Parallel()
	p, err := NewNetease(NeteaseConfig{AppKey: "k", AppSecret: "s"})
	if err != nil {
		t.Fatal(err)
	}
	_, err = p.Send(context.Background(), SendRequest{To: validTo(), Message: Message{Content: "c"}})
	if !errors.Is(err, ErrInvalidArgument) {
		t.Errorf("err = %v", err)
	}
}

// --- UCloud signing helpers ---

func TestUcloudSignString(t *testing.T) {
	t.Parallel()
	params := map[string]any{
		"Action":    "SendUSMSMessage",
		"PublicKey": "pub",
		"Region":    "cn-bj2",
		"Signature": "ignored",
	}
	got := ucloudSignString(params, "private")
	if !strings.Contains(got, "ActionSendUSMSMessage") {
		t.Errorf("sign string = %q", got)
	}
	if !strings.HasSuffix(got, "private") {
		t.Errorf("missing private key suffix: %q", got)
	}
}

func TestUcloudEncodeValue(t *testing.T) {
	t.Parallel()
	if ucloudEncodeValue(true) != "true" || ucloudEncodeValue(false) != "false" {
		t.Error("bool encode")
	}
	if ucloudEncodeValue(42) != "42" {
		t.Errorf("int = %q", ucloudEncodeValue(42))
	}
}

func TestUcloudFlattenParams(t *testing.T) {
	t.Parallel()
	out := ucloudFlattenParams(map[string]any{
		"PhoneNumbers.0": "+861",
		"TemplateId":     "tpl",
		"Signature":      "skip",
	})
	if out["PhoneNumbers.0"] != "+861" || out["TemplateId"] != "tpl" {
		t.Errorf("flat = %+v", out)
	}
	if _, ok := out["Signature"]; ok {
		t.Error("Signature should be excluded")
	}
}

// --- Rongcloud / Netease checksum pattern (sha1Hex) ---

func TestRongcloudSignature(t *testing.T) {
	t.Parallel()
	secret, nonce, ts := "sec", "abc12345", "1700000000"
	sig := sha1Hex(secret + nonce + ts)
	if len(sig) != 40 {
		t.Errorf("sha1 hex len = %d", len(sig))
	}
}

func TestNeteaseChecksum(t *testing.T) {
	t.Parallel()
	checksum := sha1Hex("secret" + "nonce123" + "1700000000")
	if len(checksum) != 40 {
		t.Errorf("checksum len = %d", len(checksum))
	}
}
