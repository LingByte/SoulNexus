package utils

import (
	"net/http"
	"testing"
)

func TestError_StatusCode(t *testing.T) {
	tests := []struct {
		name string
		code int
		want int
	}{
		{
			name: "Unauthorized status code",
			code: http.StatusUnauthorized,
			want: http.StatusUnauthorized,
		},
		{
			name: "Not found status code",
			code: http.StatusNotFound,
			want: http.StatusNotFound,
		},
		{
			name: "Forbidden status code",
			code: http.StatusForbidden,
			want: http.StatusForbidden,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := Error{
				Code: tt.code,
			}
			if got := e.StatusCode(); got != tt.want {
				t.Errorf("Error.StatusCode() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestError_Error(t *testing.T) {
	tests := []struct {
		name    string
		code    int
		message string
		want    string
	}{
		{
			name:    "Unauthorized error message",
			code:    http.StatusUnauthorized,
			message: "unauthorized",
			want:    "[401] unauthorized",
		},
		{
			name:    "Not found error message",
			code:    http.StatusNotFound,
			message: "attachment not exist",
			want:    "[404] attachment not exist",
		},
		{
			name:    "Forbidden error message",
			code:    http.StatusForbidden,
			message: "not attachment owner",
			want:    "[403] not attachment owner",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := Error{
				Code:    tt.code,
				Message: tt.message,
			}
			if got := e.Error(); got != tt.want {
				t.Errorf("Error.Error() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestPredefinedErrors(t *testing.T) {
	tests := []struct {
		name     string
		err      *Error
		expected int
		message  string
	}{
		{
			name:     "ErrUnauthorized",
			err:      ErrUnauthorized,
			expected: http.StatusUnauthorized,
			message:  "unauthorized",
		},
		{
			name:     "ErrAttachmentNotExist",
			err:      ErrAttachmentNotExist,
			expected: http.StatusNotFound,
			message:  "attachment not exist",
		},
		{
			name:     "ErrNotAttachmentOwner",
			err:      ErrNotAttachmentOwner,
			expected: http.StatusForbidden,
			message:  "not attachment owner",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.err.StatusCode() != tt.expected {
				t.Errorf("Expected status code %d, got %d", tt.expected, tt.err.StatusCode())
			}
			if tt.err.Message != tt.message {
				t.Errorf("Expected message '%s', got '%s'", tt.message, tt.err.Message)
			}
		})
	}
}
