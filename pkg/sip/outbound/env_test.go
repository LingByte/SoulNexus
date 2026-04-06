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

func TestTransferDialTargetFromEnv_WebSeat(t *testing.T) {
	_ = os.Setenv(EnvSIPTransferReqURI, "")
	_ = os.Setenv(EnvSIPTransferSigAddr, "")
	_ = os.Setenv(EnvSIPTransferNumber, "web")
	_ = os.Setenv(EnvSIPTransferHost, "")
	defer func() {
		_ = os.Unsetenv(EnvSIPTransferReqURI)
		_ = os.Unsetenv(EnvSIPTransferSigAddr)
		_ = os.Unsetenv(EnvSIPTransferNumber)
		_ = os.Unsetenv(EnvSIPTransferHost)
	}()

	dt, ok := TransferDialTargetFromEnv()
	if !ok || !dt.WebSeat {
		t.Fatalf("expected WebSeat ok, got ok=%v dt=%+v", ok, dt)
	}
	if dt.RequestURI != "" || dt.SignalingAddr != "" {
		t.Fatalf("expected empty SIP fields, got %+v", dt)
	}
}
