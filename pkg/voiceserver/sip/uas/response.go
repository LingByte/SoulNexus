package uas

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/LingByte/SoulNexus/pkg/voiceserver/sip/stack"
)

// NewResponse builds a SIP response with common headers copied from the request (From, To, Call-ID, CSeq, Via).
// body and contentType may be empty for no body (Content-Length: 0).
func NewResponse(req *stack.Message, status int, reason, body, contentType string) (*stack.Message, error) {
	if req == nil || !req.IsRequest {
		return nil, fmt.Errorf("sip/uas: need a SIP request")
	}
	if status < 100 || status > 699 {
		return nil, fmt.Errorf("sip/uas: invalid status %d", status)
	}
	reason = strings.TrimSpace(reason)
	if reason == "" {
		reason = "OK"
	}
	resp := &stack.Message{
		IsRequest:    false,
		Version:      "SIP/2.0",
		StatusCode:   status,
		StatusText:   reason,
		Headers:      map[string]string{},
		HeadersMulti: map[string][]string{},
	}
	for _, h := range []string{"Via", "From", "To", "Call-ID", "CSeq"} {
		if v := req.GetHeader(h); v != "" {
			resp.SetHeader(h, v)
		}
	}
	body = strings.ReplaceAll(body, "\r\n", "\n")
	body = strings.ReplaceAll(body, "\r", "\n")
	resp.Body = body
	if strings.TrimSpace(contentType) != "" {
		resp.SetHeader("Content-Type", strings.TrimSpace(contentType))
	}
	resp.SetHeader("Content-Length", strconv.Itoa(stack.BodyBytesLen(body)))
	return resp, nil
}

// ErrorResponse returns a minimal final error response (3xx–6xx) with optional Reason header text in StatusText.
func ErrorResponse(req *stack.Message, status int, reason string) (*stack.Message, error) {
	if reason == "" {
		switch status {
		case 400:
			reason = "Bad Request"
		case 403:
			reason = "Forbidden"
		case 404:
			reason = "Not Found"
		case 405:
			reason = "Method Not Allowed"
		case 486:
			reason = "Busy Here"
		case 488:
			reason = "Not Acceptable Here"
		case 500:
			reason = "Server Internal Error"
		case 503:
			reason = "Service Unavailable"
		default:
			reason = "Error"
		}
	}
	return NewResponse(req, status, reason, "", "")
}
