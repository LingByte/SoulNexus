// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

package voiceprint

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"strings"
	"time"
)

const volcengineRTCService = "rtc"

func signVolcengineRequest(req *http.Request, accessKey, secretKey, region string, body []byte) error {
	if strings.TrimSpace(accessKey) == "" || strings.TrimSpace(secretKey) == "" {
		return fmt.Errorf("volcengine access_key and secret_key are required")
	}
	if strings.TrimSpace(region) == "" {
		region = "cn-north-1"
	}
	now := time.Now().UTC()
	date := now.Format("20060102T150405Z")
	authDate := date[:8]
	req.Header.Set("X-Date", date)

	payloadHash := hex.EncodeToString(hashSHA256Bytes(body))
	req.Header.Set("X-Content-Sha256", payloadHash)
	if req.Header.Get("Content-Type") == "" {
		req.Header.Set("Content-Type", "application/json")
	}

	path := req.URL.Path
	if path == "" {
		path = "/"
	}
	queryString := strings.ReplaceAll(req.URL.RawQuery, "+", "%20")
	signedHeaders := []string{"content-type", "host", "x-content-sha256", "x-date"}
	var headerLines []string
	for _, h := range signedHeaders {
		switch h {
		case "host":
			headerLines = append(headerLines, "host:"+req.Host)
		default:
			headerLines = append(headerLines, h+":"+strings.TrimSpace(req.Header.Get(h)))
		}
	}
	headerString := strings.Join(headerLines, "\n")
	canonical := strings.Join([]string{
		req.Method,
		path,
		queryString,
		headerString + "\n",
		strings.Join(signedHeaders, ";"),
		payloadHash,
	}, "\n")

	hashedCanonical := hex.EncodeToString(hashSHA256Bytes([]byte(canonical)))
	credentialScope := authDate + "/" + region + "/" + volcengineRTCService + "/request"
	stringToSign := strings.Join([]string{
		"HMAC-SHA256",
		date,
		credentialScope,
		hashedCanonical,
	}, "\n")

	signingKey := volcengineSigningKey(secretKey, authDate, region, volcengineRTCService)
	signature := hex.EncodeToString(hmacSHA256Bytes(signingKey, stringToSign))
	req.Header.Set("Authorization", "HMAC-SHA256 Credential="+accessKey+"/"+credentialScope+
		", SignedHeaders="+strings.Join(signedHeaders, ";")+
		", Signature="+signature)
	return nil
}

func hashSHA256Bytes(data []byte) []byte {
	sum := sha256.Sum256(data)
	return sum[:]
}

func hmacSHA256Bytes(key []byte, content string) []byte {
	mac := hmac.New(sha256.New, key)
	_, _ = mac.Write([]byte(content))
	return mac.Sum(nil)
}

func volcengineSigningKey(secretKey, date, region, service string) []byte {
	kDate := hmacSHA256Bytes([]byte(secretKey), date)
	kRegion := hmacSHA256Bytes(kDate, region)
	kService := hmacSHA256Bytes(kRegion, service)
	return hmacSHA256Bytes(kService, "request")
}
