package notification

import (
	"context"
	"errors"
	"testing"
)

// mockAliyunSMSClient is a mock implementation of AliyunSMSClient
type mockAliyunSMSClient struct {
	sendFunc func(ctx context.Context, phone, sign, template string, params map[string]string) error
}

func (m *mockAliyunSMSClient) Send(ctx context.Context, phone, sign, template string, params map[string]string) error {
	if m.sendFunc != nil {
		return m.sendFunc(ctx, phone, sign, template, params)
	}
	return nil
}

func TestNewAliyunSMS(t *testing.T) {
	cfg := AliyunSMSConfig{
		AccessKeyId:     "test-key",
		AccessKeySecret: "test-secret",
		SignName:        "TestSign",
		TemplateCode:    "SMS_123456",
		Endpoint:        "cn-hangzhou",
	}

	mockClient := &mockAliyunSMSClient{}
	sms := NewAliyunSMS(cfg, mockClient)

	if sms == nil {
		t.Fatal("NewAliyunSMS returned nil")
	}
	if sms.cfg.AccessKeyId != cfg.AccessKeyId {
		t.Errorf("Expected AccessKeyId %s, got %s", cfg.AccessKeyId, sms.cfg.AccessKeyId)
	}
	if sms.cli != mockClient {
		t.Error("Client not set correctly")
	}
}

func TestAliyunSMS_SendCode_Success(t *testing.T) {
	cfg := AliyunSMSConfig{
		AccessKeyId:     "test-key",
		AccessKeySecret: "test-secret",
		SignName:        "TestSign",
		TemplateCode:    "SMS_123456",
	}

	mockClient := &mockAliyunSMSClient{
		sendFunc: func(ctx context.Context, phone, sign, template string, params map[string]string) error {
			if phone != "13800138000" {
				t.Errorf("Expected phone 13800138000, got %s", phone)
			}
			if sign != "TestSign" {
				t.Errorf("Expected sign TestSign, got %s", sign)
			}
			if template != "SMS_123456" {
				t.Errorf("Expected template SMS_123456, got %s", template)
			}
			if params["code"] != "123456" {
				t.Errorf("Expected code 123456, got %s", params["code"])
			}
			return nil
		},
	}

	sms := NewAliyunSMS(cfg, mockClient)
	err := sms.SendCode(context.Background(), "13800138000", "123456")
	if err != nil {
		t.Errorf("SendCode failed: %v", err)
	}
}

func TestAliyunSMS_SendCode_ClientError(t *testing.T) {
	cfg := AliyunSMSConfig{
		SignName:     "TestSign",
		TemplateCode: "SMS_123456",
	}

	mockClient := &mockAliyunSMSClient{
		sendFunc: func(ctx context.Context, phone, sign, template string, params map[string]string) error {
			return errors.New("send failed")
		},
	}

	sms := NewAliyunSMS(cfg, mockClient)
	err := sms.SendCode(context.Background(), "13800138000", "123456")
	if err == nil {
		t.Error("Expected error, got nil")
	}
}

func TestAliyunSMS_SendCode_NoClient(t *testing.T) {
	cfg := AliyunSMSConfig{
		SignName:     "TestSign",
		TemplateCode: "SMS_123456",
	}

	sms := NewAliyunSMS(cfg, nil)
	err := sms.SendCode(context.Background(), "13800138000", "123456")
	if err == nil {
		t.Error("Expected error when client is nil, got nil")
	}
	if err.Error() != "AliyunSMSClient not configured" {
		t.Errorf("Expected 'AliyunSMSClient not configured', got %v", err)
	}
}

func TestAliyunSMSConfig_Structure(t *testing.T) {
	cfg := AliyunSMSConfig{
		AccessKeyId:     "key",
		AccessKeySecret: "secret",
		SignName:        "Sign",
		TemplateCode:    "Template",
		Endpoint:        "cn-hangzhou",
	}

	if cfg.AccessKeyId == "" {
		t.Error("AccessKeyId should not be empty")
	}
	if cfg.SignName == "" {
		t.Error("SignName should not be empty")
	}
}
