package sms

import (
	"errors"
	"testing"
)

func TestPhoneNumber_String(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		p    PhoneNumber
		want string
	}{
		{"empty", PhoneNumber{}, ""},
		{"number only", PhoneNumber{Number: "13800138000"}, "13800138000"},
		{"with country code", PhoneNumber{Number: "13800138000", CountryCode: 86}, "+8613800138000"},
		{"zero country code", PhoneNumber{Number: "5551234", CountryCode: 0}, "5551234"},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := tt.p.String(); got != tt.want {
				t.Errorf("String() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestValidateBasic(t *testing.T) {
	t.Parallel()
	valid := SendRequest{
		To:      []PhoneNumber{{Number: "13800138000"}},
		Message: Message{Content: "hello"},
	}
	if err := ValidateBasic(valid); err != nil {
		t.Errorf("valid request: %v", err)
	}

	tests := []struct {
		name string
		req  SendRequest
	}{
		{"empty recipients", SendRequest{Message: Message{Content: "x"}}},
		{"empty number", SendRequest{To: []PhoneNumber{{Number: "  "}}, Message: Message{Content: "x"}}},
		{"no content or template", SendRequest{To: []PhoneNumber{{Number: "1"}}, Message: Message{}}},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := ValidateBasic(tt.req)
			if err == nil {
				t.Fatal("expected error")
			}
			if !errors.Is(err, ErrInvalidArgument) {
				t.Errorf("error = %v, want ErrInvalidArgument", err)
			}
		})
	}
}

func TestValidateBasic_templateOnly(t *testing.T) {
	t.Parallel()
	req := SendRequest{
		To:      []PhoneNumber{{Number: "13800138000"}},
		Message: Message{Template: "SMS_001"},
	}
	if err := ValidateBasic(req); err != nil {
		t.Errorf("template-only: %v", err)
	}
}
