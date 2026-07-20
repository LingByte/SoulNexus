package mail

import (
	"errors"
	"fmt"
	"testing"
)

func TestIsPermanentMailError(t *testing.T) {
	cases := []struct {
		err  error
		want bool
	}{
		// permanent — keyword matches
		{errors.New("sendcloud: 额度不足"), true},
		{errors.New("sendcloud http 403: forbidden"), true},
		{fmt.Errorf("smtp: authentication failed"), true},
		{errors.New("quota exceeded"), true},
		{errors.New("insufficient balance"), true},
		{errors.New("余额不足"), true},
		{errors.New("account disabled"), true},
		{errors.New("account suspended"), true},
		{errors.New("unauthorized: invalid api key"), true},
		{errors.New("apikey invalid"), true},
		{errors.New("密钥错误"), true},
		{fmt.Errorf("auth failed"), true},
		{errors.New("recipient rejected: blacklisted"), true},
		{errors.New("黑名单"), true},
		{errors.New("invalid recipient"), true},
		{errors.New("invalid from address"), true},
		{errors.New("invalid sender"), true},
		{errors.New("收件人不存在"), true},
		{errors.New("rate limit exceeded"), true},
		{errors.New("too many requests"), true},
		{errors.New("请求过于频繁"), true},

		// permanent — HTTP status codes in message
		{errors.New("sendcloud http 401: unauthorized"), true},
		{errors.New("sendcloud:  401: unauthorized"), true},
		{errors.New("provider  403: access denied"), true},
		{errors.New("http 402 payment required"), true},

		// NOT permanent — transient
		{errors.New("connection reset by peer"), false},
		{errors.New("sendcloud request: timeout"), false},
		{errors.New("temporary failure"), false},

		// edge cases
		{nil, false},
		{errors.New(""), false},
	}
	for _, tc := range cases {
		if got := isPermanentMailError(tc.err); got != tc.want {
			t.Errorf("isPermanentMailError(%v) = %v, want %v", tc.err, got, tc.want)
		}
	}
}

func TestIsPermanentMailError_allKeywords(t *testing.T) {
	// verify every keyword in the list triggers a match
	for _, kw := range permanentMailErrorKeywords {
		err := fmt.Errorf("some error: %s happened", kw)
		if !isPermanentMailError(err) {
			t.Errorf("keyword %q should be detected as permanent", kw)
		}
	}
}

func TestIsPermanentMailError_caseInsensitive(t *testing.T) {
	// all matching is case-insensitive
	err := errors.New("QUOTA EXCEEDED: please top up")
	if !isPermanentMailError(err) {
		t.Error("uppercase quota should match")
	}
	err = errors.New("Account DISABLED")
	if !isPermanentMailError(err) {
		t.Error("mixed case account disabled should match")
	}
}
