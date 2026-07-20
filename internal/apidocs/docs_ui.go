// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

package apidocs

import (
	_ "embed"
	"encoding/json"
	"html"
	"net/http"
	"os"
	"strings"

	"github.com/LingByte/SoulNexus"
	"github.com/gin-gonic/gin"
)

// mountCustomDocs serves a light-theme Scalar UI matching web/src/index.css (:root).
func mountCustomDocs(r *gin.Engine, docsPath, title, openAPIJSON, apiPrefix string) {
	docsPath = strings.TrimSuffix(docsPath, "/")
	apiPrefix = strings.TrimSuffix(strings.TrimSpace(apiPrefix), "/")
	cssPath := docsPath + "/assets/docs.css"
	logoPath := docsPath + "/assets/logo.png"

	r.GET(cssPath, func(c *gin.Context) {
		c.Header("Content-Type", "text/css; charset=utf-8")
		c.Header("Cache-Control", "public, max-age=300")
		c.Data(http.StatusOK, "text/css; charset=utf-8", SoulNexus.DocsCSS)
	})
	r.GET(logoPath, func(c *gin.Context) {
		c.Header("Content-Type", "image/png")
		c.Header("Cache-Control", "public, max-age=86400")
		c.Data(http.StatusOK, "image/png", SoulNexus.DefaultLogoPNG)
	})

	cfg := scalarConfig()
	cfgJSON, _ := json.Marshal(cfg)
	htmlPage := customDocsHTML(title, openAPIJSON, cssPath, logoPath, string(cfgJSON), apiPrefix)

	handler := func(c *gin.Context) {
		c.Header("Content-Type", "text/html; charset=utf-8")
		c.Header("Cache-Control", "no-store")
		c.String(http.StatusOK, htmlPage)
	}
	r.GET(docsPath, handler)
	r.GET(docsPath+"/", handler)
}

func scalarConfig() map[string]any {
	cfg := map[string]any{
		"theme":              "default",
		"layout":             "modern",
		"darkMode":           false,
		"forceDarkModeState": "light",
		"hideModels":         true,
		"hideDownloadButton": true,
		"persistAuth":        true,
		"showSidebar":        true,
		"authentication": map[string]any{
			"preferredSecurityScheme": "BearerAuth",
		},
		"metaData": map[string]any{
			"title": "API 文档",
		},
		// Ask AI is Scalar's hosted Agent — not swappable for our own LLM.
		// Default: off. Opt-in only with SCALAR_AGENT_KEY (and not DISABLED).
		"agent": map[string]any{"disabled": true},
	}

	disabled := envTruthy("SCALAR_AGENT_DISABLED")
	agentKey := strings.TrimSpace(os.Getenv("SCALAR_AGENT_KEY"))
	if !disabled && agentKey != "" {
		cfg["agent"] = map[string]any{"key": agentKey}
	}
	return cfg
}

func envTruthy(key string) bool {
	v := strings.TrimSpace(os.Getenv(key))
	return strings.EqualFold(v, "1") || strings.EqualFold(v, "true") || strings.EqualFold(v, "yes")
}

func customDocsHTML(title, openAPIJSON, cssPath, logoPath, configJSON, apiPrefix string) string {
	escTitle := html.EscapeString(title)
	escURL := html.EscapeString(openAPIJSON)
	escCSS := html.EscapeString(cssPath)
	escLogo := html.EscapeString(logoPath)
	escCfg := html.EscapeString(configJSON)
	return `<!doctype html>
<html lang="zh-CN">
<head>
  <meta charset="utf-8" />
  <meta name="viewport" content="width=device-width, initial-scale=1" />
  <meta name="color-scheme" content="light" />
  <meta name="referrer" content="no-referrer" />
  <title>` + escTitle + ` · API</title>
  <link rel="stylesheet" href="` + escCSS + `" />
</head>
<body class="lx-docs">
  <header class="lx-docs-topbar">
    <div class="brand">
      <img id="lx-docs-logo" class="brand-logo" src="` + escLogo + `" alt="" width="28" height="28" />
      <div class="brand-text">
        <strong id="lx-docs-name">` + escTitle + `</strong>
        <span>接口文档 · 与控制台同源鉴权</span>
      </div>
    </div>
    <div class="actions">
      <a class="btn-ghost" href="` + escURL + `" download="soulnexus.openapi.json">导出 JSON</a>
      <a class="btn-ghost" href="/openapi.yaml" download="soulnexus.openapi.yaml">导出 YAML</a>
      <a class="btn-primary" href="#tag/introduction">使用说明</a>
    </div>
  </header>
  <main class="lx-docs-main">
    <script id="api-reference" data-url="` + escURL + `" data-configuration="` + escCfg + `"></script>
  </main>
  <script src="https://unpkg.com/@scalar/api-reference@1.44.20/dist/browser/standalone.js"
    crossorigin integrity="sha384-tMz7GAo6dMy55x9tLFtH+sHtogji6Scmb+feBR31TAHmvSPRUTboK9H3M5NFaP4R"></script>
  <script>
(function () {
  var initURL = ` + jsonString(apiPrefix+"/system/init") + `;
  var fallbackLogo = ` + jsonString(logoPath) + `;
  var logoEl = document.getElementById('lx-docs-logo');
  var nameEl = document.getElementById('lx-docs-name');
  function useFallbackLogo() {
    if (!logoEl) return;
    logoEl.onerror = null;
    if (logoEl.getAttribute('src') !== fallbackLogo) {
      logoEl.src = fallbackLogo;
    }
  }
  if (logoEl) {
    logoEl.onerror = useFallbackLogo;
  }
  fetch(initURL + (initURL.indexOf('?') >= 0 ? '&' : '?') + '_t=' + Date.now(), {
    credentials: 'same-origin',
    headers: { 'Accept': 'application/json' }
  }).then(function (r) { return r.json(); }).then(function (j) {
    var d = (j && j.data) ? j.data : j;
    if (!d || typeof d !== 'object') return;
    var name = (d.SITE_NAME || '').trim();
    var logo = (d.SITE_LOGO_URL || '').trim();
    if (name && nameEl) {
      nameEl.textContent = name;
      document.title = name + ' · API';
      if (logoEl) logoEl.alt = name;
    }
    if (!logoEl) return;
    // Frontend-only default path is not served by the API process.
    if (!logo || logo === '/icon-lingyu.png' || logo.indexOf('icon-lingyu.png') >= 0) {
      useFallbackLogo();
      return;
    }
    logoEl.onerror = useFallbackLogo;
    logoEl.src = logo;
  }).catch(function () { useFallbackLogo(); });
})();
  </script>
</body>
</html>`
}

func jsonString(s string) string {
	b, _ := json.Marshal(s)
	return string(b)
}
