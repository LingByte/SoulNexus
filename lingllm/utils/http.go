package utils

import (
	"net/http"
	"strings"
)

var reservedHTTPHeaders = map[string]struct{}{
	"authorization": {},
	"content-type":  {},
}

// ApplyCustomHeaders attaches user-defined HTTP headers, skipping reserved headers.
func ApplyCustomHeaders(req *http.Request, headers map[string]string) {
	if req == nil || len(headers) == 0 {
		return
	}
	for key, value := range headers {
		key = strings.TrimSpace(key)
		if key == "" {
			continue
		}
		if _, reserved := reservedHTTPHeaders[strings.ToLower(key)]; reserved {
			continue
		}
		req.Header.Set(key, value)
	}
}
