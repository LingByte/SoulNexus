package outbound

import (
	"os"
	"testing"
)

func TestDialTargetFromEnv_BuildsFromTargetAndHost(t *testing.T) {
	_ = os.Setenv("SIP_TARGET_NUMBER", "1001")
	_ = os.Setenv("SIP_OUTBOUND_HOST", "pbx.example.com")
	_ = os.Setenv("SIP_OUTBOUND_PORT", "5060")
	_ = os.Setenv("SIP_SIGNALING_ADDR", "")
	defer func() {
		_ = os.Unsetenv("SIP_TARGET_NUMBER")
		_ = os.Unsetenv("SIP_OUTBOUND_HOST")
		_ = os.Unsetenv("SIP_OUTBOUND_PORT")
		_ = os.Unsetenv("SIP_SIGNALING_ADDR")
		_ = os.Unsetenv("SIP_OUTBOUND_REQUEST_URI")
	}()

	dt, ok := DialTargetFromEnv()
	if !ok {
		t.Fatal("expected ok")
	}
	if dt.RequestURI != "sip:1001@pbx.example.com:5060" {
		t.Fatalf("RequestURI: %q", dt.RequestURI)
	}
	if dt.SignalingAddr != "pbx.example.com:5060" {
		t.Fatalf("SignalingAddr: %q", dt.SignalingAddr)
	}
}
