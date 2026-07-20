// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

package apidocs

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestMount_MetaAndOpenAPI(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	Mount(r, Options{
		Title:     "SoulNexus Test",
		Version:   "0.0.1",
		DocsPath:  "/api/docs",
		APIPrefix: "/api",
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/meta", nil)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("meta status=%d body=%s", w.Code, w.Body.String())
	}
	var body map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body["name"] != "SoulNexus Test" {
		t.Fatalf("unexpected body: %v", body)
	}

	w2 := httptest.NewRecorder()
	req2 := httptest.NewRequest(http.MethodGet, "/openapi.json", nil)
	r.ServeHTTP(w2, req2)
	if w2.Code != http.StatusOK {
		t.Fatalf("openapi status=%d", w2.Code)
	}

	w3 := httptest.NewRecorder()
	req3 := httptest.NewRequest(http.MethodGet, "/api/docs", nil)
	r.ServeHTTP(w3, req3)
	if w3.Code != http.StatusOK {
		t.Fatalf("docs status=%d", w3.Code)
	}
	if !strings.Contains(w3.Body.String(), "@scalar/api-reference") {
		t.Fatalf("docs UI should use Scalar, body snippet: %s", w3.Body.String()[:min(200, w3.Body.Len())])
	}
	if !strings.Contains(w3.Body.String(), "lx-docs") {
		t.Fatal("docs UI should use lx-docs shell")
	}
	if !strings.Contains(w3.Body.String(), "docs.css") {
		t.Fatal("docs UI should link extracted docs.css")
	}
	if !strings.Contains(w3.Body.String(), "导出 JSON") {
		t.Fatal("docs UI should expose OpenAPI JSON download")
	}
	if strings.Contains(w3.Body.String(), `"darkMode":true`) {
		t.Fatal("docs UI must be light mode (web :root style), not dark")
	}
	htmlBody := w3.Body.String()
	if !strings.Contains(htmlBody, "disabled") || !strings.Contains(htmlBody, "agent") {
		t.Fatal("Ask AI / Scalar agent must be disabled by default")
	}
	if !strings.Contains(htmlBody, "/api/system/init") {
		t.Fatal("docs UI should load brand from /api/system/init")
	}
	if !strings.Contains(htmlBody, "lx-docs-logo") {
		t.Fatal("docs UI should render site logo img")
	}
	if !strings.Contains(htmlBody, "/api/docs/assets/logo.png") {
		t.Fatal("docs UI should use embedded logo asset, not /icon-lingyu.png on API origin")
	}
	if strings.Contains(htmlBody, `src="/icon-lingyu.png"`) {
		t.Fatal("must not request /icon-lingyu.png from API server (404/429 loop)")
	}

	w4 := httptest.NewRecorder()
	req4 := httptest.NewRequest(http.MethodGet, "/api/docs/assets/logo.png", nil)
	r.ServeHTTP(w4, req4)
	if w4.Code != http.StatusOK || w4.Body.Len() < 100 {
		t.Fatalf("embedded logo status=%d len=%d", w4.Code, w4.Body.Len())
	}
}
