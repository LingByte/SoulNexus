package outbound

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

type fakeDialer struct {
	gotReq DialRequest
	callID string
	err    error
}

func (f *fakeDialer) Dial(_ context.Context, req DialRequest) (string, error) {
	f.gotReq = req
	if f.callID == "" {
		f.callID = "cid-1"
	}
	return f.callID, f.err
}

func TestDialHTTPAPI_OK_RequestURI(t *testing.T) {
	fd := &fakeDialer{}
	api := &dialHTTPAPI{token: "abc", dialer: fd}

	body := `{"request_uri":"sip:1001@10.0.0.1:5060","signaling_addr":"10.0.0.1:5060","media_profile":"ai_voice","scenario":"campaign"}`
	req := httptest.NewRequest(http.MethodPost, "/sip/v1/outbound/dial", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-API-Token", "abc")
	w := httptest.NewRecorder()
	api.handleDial(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
	}
	if fd.gotReq.Target.RequestURI != "sip:1001@10.0.0.1:5060" {
		t.Fatalf("uri=%q", fd.gotReq.Target.RequestURI)
	}
}

func TestDialHTTPAPI_Unauthorized(t *testing.T) {
	fd := &fakeDialer{}
	api := &dialHTTPAPI{token: "abc", dialer: fd}
	req := httptest.NewRequest(http.MethodPost, "/sip/v1/outbound/dial", strings.NewReader(`{}`))
	w := httptest.NewRecorder()
	api.handleDial(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("status=%d", w.Code)
	}
}

func TestDialHTTPAPI_OK_TargetHost(t *testing.T) {
	fd := &fakeDialer{}
	api := &dialHTTPAPI{dialer: fd}
	body := `{"target_number":"10086","outbound_host":"192.168.1.2","outbound_port":5062}`
	req := httptest.NewRequest(http.MethodPost, "/sip/v1/outbound/dial", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	api.handleDial(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
	}
	if fd.gotReq.Target.SignalingAddr != "192.168.1.2:5062" {
		t.Fatalf("signaling=%q", fd.gotReq.Target.SignalingAddr)
	}
	var resp map[string]any
	_ = json.Unmarshal(w.Body.Bytes(), &resp)
	if resp["call_id"] == "" {
		t.Fatalf("empty call_id")
	}
}

