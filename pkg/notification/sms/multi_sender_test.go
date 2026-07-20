package sms

import (
	"context"
	"errors"
	"testing"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type stubProvider struct {
	kind     ProviderKind
	accepted bool
	sendErr  error
	calls    int
}

func (s *stubProvider) Kind() ProviderKind { return s.kind }

func (s *stubProvider) Send(ctx context.Context, req SendRequest) (*SendResult, error) {
	s.calls++
	if s.sendErr != nil {
		return nil, s.sendErr
	}
	return &SendResult{
		Provider:  s.kind,
		Accepted:  s.accepted,
		MessageID: "stub-id",
		Raw:       `{}`,
	}, nil
}

func TestNewMultiSender_validation(t *testing.T) {
	t.Parallel()
	_, err := NewMultiSender(nil, nil, "")
	if err == nil {
		t.Fatal("expected error for empty channels")
	}
	p := &stubProvider{kind: ProviderYunpian, accepted: true}
	ms, err := NewMultiSender([]SenderChannel{{Label: "a", Provider: p}}, nil, "1.2.3.4", WithSMSLogUserID(9))
	if err != nil {
		t.Fatalf("NewMultiSender: %v", err)
	}
	if ms.rt.userID != 9 {
		t.Errorf("userID = %d", ms.rt.userID)
	}
}

func TestMultiSender_Send_success(t *testing.T) {
	t.Parallel()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	if err := db.AutoMigrate(&SMSLog{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	ok := &stubProvider{kind: ProviderYunpian, accepted: true}
	fail := &stubProvider{kind: ProviderTwilio, accepted: false}
	ms, err := NewMultiSender([]SenderChannel{
		{Label: "fail-first", Provider: fail},
		{Label: "ok", Provider: ok},
	}, db, "127.0.0.1")
	if err != nil {
		t.Fatalf("NewMultiSender: %v", err)
	}
	req := SendRequest{
		To:      []PhoneNumber{{Number: "13800138000", CountryCode: 86}},
		Message: Message{Content: "test"},
	}
	if err := ms.Send(context.Background(), req); err != nil {
		t.Fatalf("Send: %v", err)
	}
	if ok.calls == 0 {
		t.Error("expected failover to second channel")
	}
	var count int64
	db.Model(&SMSLog{}).Count(&count)
	if count != 1 {
		t.Errorf("sms log count = %d, want 1", count)
	}
}

func TestMultiSender_Send_allFail(t *testing.T) {
	t.Parallel()
	p := &stubProvider{kind: ProviderYunpian, accepted: false}
	ms, err := NewMultiSender([]SenderChannel{{Label: "x", Provider: p}}, nil, "")
	if err != nil {
		t.Fatalf("NewMultiSender: %v", err)
	}
	err = ms.Send(context.Background(), SendRequest{
		To:      []PhoneNumber{{Number: "13800138000"}},
		Message: Message{Content: "hi"},
	})
	if err == nil {
		t.Fatal("expected error")
	}
	if !errors.Is(err, errProviderRejected) {
		t.Errorf("err = %v", err)
	}
}

func TestMultiSender_Send_validateBasic(t *testing.T) {
	t.Parallel()
	p := &stubProvider{kind: ProviderYunpian, accepted: true}
	ms, _ := NewMultiSender([]SenderChannel{{Label: "x", Provider: p}}, nil, "")
	err := ms.Send(context.Background(), SendRequest{})
	if !errors.Is(err, ErrInvalidArgument) {
		t.Errorf("err = %v", err)
	}
	if p.calls != 0 {
		t.Error("provider should not be called")
	}
}
