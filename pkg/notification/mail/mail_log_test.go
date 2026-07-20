package mail

import (
	"errors"
	"testing"

	"github.com/LingByte/SoulNexus/pkg/constants"
)

func TestIsFourByteEncodingErr(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{"nil", nil, false},
		{"Error 1366", errors.New("Error 1366: Incorrect string value"), true},
		{"Error 3988", errors.New("Error 3988: Conversion from collation"), true},
		{"Incorrect string value", errors.New("Incorrect string value: '\\xF0\\x9F' for column"), true},
		{"random error", errors.New("connection refused"), false},
	}
	for _, tt := range tests {
		got := isFourByteEncodingErr(tt.err)
		if got != tt.want {
			t.Errorf("%s: isFourByteEncodingErr(%v) = %v, want %v", tt.name, tt.err, got, tt.want)
		}
	}
}

func TestSanitizeForMailLog(t *testing.T) {
	// log without 4-byte chars stays unchanged
	log := &MailLog{
		Subject:     "hello",
		HtmlBody:    "<p>world</p>",
		ErrorMsg:    "some error",
		ChannelName: "ch1",
	}
	sanitizeForMailLog(log)
	if log.Subject != "hello" {
		t.Errorf("subject changed: %q", log.Subject)
	}
	if log.HtmlBody != "<p>world</p>" {
		t.Errorf("htmlBody changed: %q", log.HtmlBody)
	}

	// log with emoji gets stripped
	log2 := &MailLog{
		Subject:     "hello 😀 world",
		HtmlBody:    "<p>🎉</p>",
		ErrorMsg:    "oops 😱",
		ChannelName: "chan 😈",
	}
	sanitizeForMailLog(log2)
	if log2.Subject == "hello 😀 world" {
		t.Errorf("emoji not stripped from subject")
	}
	if log2.HtmlBody == "<p>🎉</p>" {
		t.Errorf("emoji not stripped from htmlBody")
	}
	if log2.ErrorMsg == "oops 😱" {
		t.Errorf("emoji not stripped from error_msg")
	}
	if log2.ChannelName == "chan 😈" {
		t.Errorf("emoji not stripped from channel_name")
	}
}

func TestMailLog_TableName(t *testing.T) {
	ml := MailLog{}
	if got := ml.TableName(); got != constants.MAIL_LOGS_TABLE_NAME {
		t.Errorf("TableName() = %q, want %q", got, constants.MAIL_LOGS_TABLE_NAME)
	}
}

func TestEnsureMailLogsUtf8mb4_nilDB(t *testing.T) {
	// nil db should return immediately without panic
	ensureMailLogsUtf8mb4(nil)
}

func TestEnsureMailLogsUtf8mb4_alreadyAttempted(t *testing.T) {
	// set the flag; the next call should return immediately
	saved := alterAttempted
	alterAttempted = true
	ensureMailLogsUtf8mb4(nil)
	alterAttempted = saved
}
