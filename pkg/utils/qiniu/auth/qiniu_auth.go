package auth

import (
	"crypto/hmac"
	"crypto/sha1"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"
)

// QiniuAuthRequest qiniu authorization request
type QiniuAuthRequest struct {
	Method      string // HTTP method, supports POST, PUT, GET, must be in all uppercase
	Path        string // Request path
	RawQuery    string // URL query parameters (excluding ?)
	Host        string // Request domain name, for example: atlab.ai
	ContentType string // Request content type, optional
	Body        []byte // Request Body, optional
}

// GenerateQiniuToken  generates a Qiniu authentication token
// accessKey: Qiniu Access Key
// secretKey: Qiniu Secret Key
// req: Authentication request parameters
// Return: QiniuToken string, in the format of "Qiniu AccessKey:encodedSign"
func GenerateQiniuToken(accessKey, secretKey string, req QiniuAuthRequest) (string, error) {
	// Step 1: Construct the Data to be signed
	data, err := buildSignData(req)
	if err != nil {
		return "", fmt.Errorf("failed to construct the signature string: %w", err)
	}
	// Step 2: Calculate the HMAC-SHA1 signature and perform URL-safe Base64 encoding on the signature result
	sign := hmac.New(sha1.New, []byte(secretKey))
	sign.Write([]byte(data))
	digest := sign.Sum(nil)
	encodedSign := base64.URLEncoding.EncodeToString(digest)
	// Step 3: Combine the Qiniu identifier with the AccessKey and encodedSign to obtain the management credential
	qiniuToken := fmt.Sprintf("Qiniu %s:%s", accessKey, encodedSign)
	return qiniuToken, nil
}

// buildSignData Constructs the Data string to be signed
func buildSignData(req QiniuAuthRequest) (string, error) {
	if req.Method == "" {
		return "", errors.New("method must not be empty")
	}
	if req.Path == "" {
		return "", errors.New("path must not be empty")
	}
	if req.Host == "" {
		return "", errors.New("host must not be empty")
	}

	var builder strings.Builder
	// Method + " " + Path
	builder.WriteString(strings.ToUpper(req.Method))
	builder.WriteString(" ")
	builder.WriteString(req.Path)
	// If RawQuery exists, add '?' <RawQuery>
	if req.RawQuery != "" {
		builder.WriteString("?")
		builder.WriteString(req.RawQuery)
	}
	// \nHost: <Host>
	builder.WriteString("\nHost: ")
	builder.WriteString(req.Host)
	// If ContentType exists, add "\nContent-Type: <ContentType>"
	if req.ContentType != "" {
		builder.WriteString("\nContent-Type: ")
		builder.WriteString(req.ContentType)
	}
	// add \n\n
	builder.WriteString("\n\n")
	// Determine whether to add bodyStr
	// Condition: Content-Length exists and Body is not empty, and at the same time, Content-Type exists and is not empty and does not equal "application/octet-stream"
	contentLength := len(req.Body)
	bodyStr := ""
	if contentLength > 0 &&
		req.ContentType != "" &&
		req.ContentType != "application/octet-stream" {
		bodyStr = string(req.Body)
		builder.WriteString(bodyStr)
	}
	return builder.String(), nil
}

// BuildQiniuAuthRequest build QiniuAuthRequest
func BuildQiniuAuthRequest(method, path, host string) *QiniuAuthRequest {
	return &QiniuAuthRequest{
		Method: method,
		Path:   path,
		Host:   host,
	}
}

// SetRawQuery set RawQuery
func (r *QiniuAuthRequest) SetRawQuery(rawQuery string) *QiniuAuthRequest {
	r.RawQuery = rawQuery
	return r
}

// SetContentType set ContentType
func (r *QiniuAuthRequest) SetContentType(contentType string) *QiniuAuthRequest {
	r.ContentType = contentType
	return r
}

// SetBody set Body
func (r *QiniuAuthRequest) SetBody(body []byte) *QiniuAuthRequest {
	r.Body = body
	return r
}

// SetBodyString set Body
func (r *QiniuAuthRequest) SetBodyString(bodyStr string) *QiniuAuthRequest {
	r.Body = []byte(bodyStr)
	return r
}
