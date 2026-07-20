package xhttp

// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

const HTTP_REQUEST_TIME_OUT_SECOND = time.Second * 10

// HeaderOption is a single HTTP request header.
type HeaderOption struct {
	Key   string
	Value string
}

func getQueryURL(params map[string]interface{}) string {
	if len(params) == 0 {
		return ""
	}
	var buffer bytes.Buffer
	buffer.WriteString("?")
	first := true
	for key, val := range params {
		if val == nil {
			continue
		}
		if !first {
			buffer.WriteByte('&')
		}
		first = false
		buffer.WriteString(fmt.Sprintf("%s=%v", key, val))
	}
	if buffer.Len() <= 1 {
		return ""
	}
	return buffer.String()
}

// Get performs an HTTP GET with optional query params and headers.
func Get(url string, params map[string]interface{}, headerOptions ...*HeaderOption) ([]byte, error) {
	url += getQueryURL(params)
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	for _, option := range headerOptions {
		if option == nil {
			continue
		}
		req.Header.Set(option.Key, option.Value)
	}
	client := http.Client{Timeout: HTTP_REQUEST_TIME_OUT_SECOND}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	return io.ReadAll(resp.Body)
}

// Post performs an HTTP POST with a JSON body.
func Post(url string, params map[string]interface{}, headerOptions ...*HeaderOption) ([]byte, error) {
	var body io.Reader
	if len(params) > 0 {
		jsonBuf, err := json.Marshal(params)
		if err != nil {
			return nil, err
		}
		body = bytes.NewReader(jsonBuf)
	}
	req, err := http.NewRequest(http.MethodPost, url, body)
	if err != nil {
		return nil, err
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	for _, option := range headerOptions {
		if option == nil {
			continue
		}
		req.Header.Set(option.Key, option.Value)
	}
	client := http.Client{Timeout: HTTP_REQUEST_TIME_OUT_SECOND}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	return io.ReadAll(resp.Body)
}
