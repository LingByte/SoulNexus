package video

import (
	"strings"
	"testing"
)

func TestGetVideoCensor_UnknownKind(t *testing.T) {
	_, err := GetVideoCensor("unknown")
	if err == nil {
		t.Fatal("expected error for unknown kind")
	}
	if !strings.Contains(err.Error(), "unknown video censor kind") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestGetVideoCensor_MissingCredentials(t *testing.T) {
	tests := []struct {
		kind string
		want string
	}{
		{KindQiniu, "qiniu requires accessKey and secretKey"},
		{KindQCloud, "qcloud requires secretID and secretKey"},
		{KindAliyun, "aliyun requires accessKeyID and accessKeySecret"},
	}
	for _, tt := range tests {
		t.Run(tt.kind, func(t *testing.T) {
			_, err := GetVideoCensor(tt.kind)
			if err == nil {
				t.Fatal("expected error for missing credentials")
			}
			if !strings.Contains(err.Error(), tt.want) {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

func TestGetVideoCensor_InvalidCredentialTypes(t *testing.T) {
	tests := []struct {
		kind string
		cred interface{}
		want string
	}{
		{KindQiniu, 123, "invalid accessKey type"},
		{KindQCloud, 123, "invalid secretID type"},
		{KindAliyun, 123, "invalid accessKeyID type"},
	}
	for _, tt := range tests {
		t.Run(tt.kind, func(t *testing.T) {
			_, err := GetVideoCensor(tt.kind, tt.cred, "secret")
			if err == nil {
				t.Fatal("expected error for invalid credential type")
			}
			if !strings.Contains(err.Error(), tt.want) {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

func TestGetVideoCensor_InvalidSecondCredentialType(t *testing.T) {
	tests := []struct {
		kind string
		want string
	}{
		{KindQiniu, "invalid secretKey type"},
		{KindQCloud, "invalid secretKey type"},
		{KindAliyun, "invalid accessKeySecret type"},
	}
	for _, tt := range tests {
		t.Run(tt.kind, func(t *testing.T) {
			_, err := GetVideoCensor(tt.kind, "key", 123)
			if err == nil {
				t.Fatal("expected error for invalid credential type")
			}
			if !strings.Contains(err.Error(), tt.want) {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

func TestGetVideoCensor_ValidCredentials(t *testing.T) {
	t.Run("qiniu", func(t *testing.T) {
		c, err := GetVideoCensor(KindQiniu, "ak", "sk")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if c == nil {
			t.Fatal("expected non-nil censor")
		}
	})

	t.Run("qcloud", func(t *testing.T) {
		c, err := GetVideoCensor(KindQCloud, "sid", "sk")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if c == nil {
			t.Fatal("expected non-nil censor")
		}
	})

	t.Run("aliyun", func(t *testing.T) {
		c, err := GetVideoCensor(KindAliyun, "akid", "aksecret")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if c == nil {
			t.Fatal("expected non-nil censor")
		}
	})
}
