package auth

import (
	"strings"
	"testing"
)

func TestGenerateQiniuToken(t *testing.T) {
	accessKey := "test_access_key"
	secretKey := "test_secret_key"

	tests := []struct {
		name    string
		req     QiniuAuthRequest
		wantErr bool
	}{
		{
			name: "POST request with Body and ContentType",
			req: QiniuAuthRequest{
				Method:      "POST",
				Path:        "/v1/face/detect",
				Host:        "argus.atlab.ai",
				ContentType: "application/json",
				Body:        []byte(`{"data": {"uri":"http://img3.redocn.com/20120528/Redocn_2012052817213559.jpg"}}`),
			},
			wantErr: false,
		},
		{
			name: "GET request with RawQuery",
			req: QiniuAuthRequest{
				Method:   "GET",
				Path:     "/v1/resource",
				RawQuery: "param1=value1&param2=value2",
				Host:     "atlab.ai",
			},
			wantErr: false,
		},
		{
			name: "PUT request without Body",
			req: QiniuAuthRequest{
				Method:      "PUT",
				Path:        "/v1/update",
				Host:        "atlab.ai",
				ContentType: "application/json",
			},
			wantErr: false,
		},
		{
			name: "POST request with ContentType application/octet-stream (should not contain Body)",
			req: QiniuAuthRequest{
				Method:      "POST",
				Path:        "/v1/upload",
				Host:        "atlab.ai",
				ContentType: "application/octet-stream",
				Body:        []byte("binary data"),
			},
			wantErr: false,
		},
		{
			name: "Missing Method",
			req: QiniuAuthRequest{
				Path: "/v1/test",
				Host: "atlab.ai",
			},
			wantErr: true,
		},
		{
			name: "Missing Path",
			req: QiniuAuthRequest{
				Method: "GET",
				Host:   "atlab.ai",
			},
			wantErr: true,
		},
		{
			name: "Missing Host",
			req: QiniuAuthRequest{
				Method: "GET",
				Path:   "/v1/test",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			token, err := GenerateQiniuToken(accessKey, secretKey, tt.req)
			if (err != nil) != tt.wantErr {
				t.Errorf("GenerateQiniuToken() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && token == "" {
				t.Error("GenerateQiniuToken() returned token is empty")
			}
			if !tt.wantErr {
				// Validate token format
				if !strings.HasPrefix(token, "Qiniu ") {
					t.Error("GenerateQiniuToken() token format error, should start with 'Qiniu '")
				}
				if !strings.Contains(token, accessKey+":") {
					t.Error("GenerateQiniuToken() token should contain AccessKey")
				}
			}
		})
	}
}

func TestBuildSignData(t *testing.T) {
	tests := []struct {
		name    string
		req     QiniuAuthRequest
		want    string
		wantErr bool
	}{
		{
			name: "POST request example (face recognition)",
			req: QiniuAuthRequest{
				Method:      "POST",
				Path:        "/v1/face/detect",
				Host:        "argus.atlab.ai",
				ContentType: "application/json",
				Body:        []byte(`{"data": {"uri":"http://img3.redocn.com/20120528/Redocn_2012052817213559.jpg"}}`),
			},
			want: `POST /v1/face/detect
Host: argus.atlab.ai
Content-Type: application/json

{"data": {"uri":"http://img3.redocn.com/20120528/Redocn_2012052817213559.jpg"}}`,
			wantErr: false,
		},
		{
			name: "GET request with RawQuery",
			req: QiniuAuthRequest{
				Method:   "GET",
				Path:     "/v1/resource",
				RawQuery: "param1=value1&param2=value2",
				Host:     "atlab.ai",
			},
			want: `GET /v1/resource?param1=value1&param2=value2
Host: atlab.ai

`,
			wantErr: false,
		},
		{
			name: "Without ContentType and Body",
			req: QiniuAuthRequest{
				Method: "GET",
				Path:   "/v1/test",
				Host:   "atlab.ai",
			},
			want: `GET /v1/test
Host: atlab.ai

`,
			wantErr: false,
		},
		{
			name: "ContentType is application/octet-stream (should not contain Body)",
			req: QiniuAuthRequest{
				Method:      "POST",
				Path:        "/v1/upload",
				Host:        "atlab.ai",
				ContentType: "application/octet-stream",
				Body:        []byte("binary data"),
			},
			want: `POST /v1/upload
Host: atlab.ai
Content-Type: application/octet-stream

`,
			wantErr: false,
		},
		{
			name: "Empty Body (should not contain Body)",
			req: QiniuAuthRequest{
				Method:      "POST",
				Path:        "/v1/test",
				Host:        "atlab.ai",
				ContentType: "application/json",
				Body:        []byte{},
			},
			want: `POST /v1/test
Host: atlab.ai
Content-Type: application/json

`,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := buildSignData(tt.req)
			if (err != nil) != tt.wantErr {
				t.Errorf("buildSignData() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("buildSignData() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestBuildQiniuAuthRequest(t *testing.T) {
	req := BuildQiniuAuthRequest("POST", "/v1/face/detect", "argus.atlab.ai").
		SetContentType("application/json").
		SetBodyString(`{"data": {"uri":"http://img3.redocn.com/20120528/Redocn_2012052817213559.jpg"}}`)

	if req.Method != "POST" {
		t.Error("Method setting error")
	}
	if req.Path != "/v1/face/detect" {
		t.Error("Path setting error")
	}
	if req.Host != "argus.atlab.ai" {
		t.Error("Host setting error")
	}
	if req.ContentType != "application/json" {
		t.Error("ContentType setting error")
	}
}
