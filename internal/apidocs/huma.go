// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

// Package apidocs mounts Huma OpenAPI 3.1 docs on the existing Gin engine.
package apidocs

import (
	"context"
	"net/http"
	"strings"

	"github.com/LingByte/SoulNexus/pkg/humax"
	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/adapters/humagin"
	"github.com/gin-gonic/gin"
)

// Options configures the Huma OpenAPI surface.
type Options struct {
	Title     string
	Version   string
	DocsPath  string // e.g. /api/docs
	APIPrefix string // e.g. /api
}

type metaOutput struct {
	Body struct {
		Name        string   `json:"name" example:"SoulNexus" doc:"Product name"`
		Version     string   `json:"version" example:"1.0.0" doc:"API docs version"`
		DocsPath    string   `json:"docs_path" doc:"Interactive OpenAPI UI path"`
		OpenAPIJSON string   `json:"openapi_json" example:"/openapi.json" doc:"Download OpenAPI 3.1 JSON"`
		OpenAPIYAML string   `json:"openapi_yaml" example:"/openapi.yaml" doc:"Download OpenAPI 3.1 YAML"`
		Health      []string `json:"health_probes" doc:"Orchestrator probe paths (Gin)"`
		Bodies      int      `json:"registered_bodies" doc:"How many typed JSON bodies are registered"`
		Note        string   `json:"note" doc:"Migration note"`
	}
}

// Mount wires Huma onto the Gin engine (custom Scalar docs + OpenAPI spec).
func Mount(r *gin.Engine, opts Options) huma.API {
	if opts.Title == "" {
		opts.Title = "SoulNexus API"
	}
	if opts.Version == "" {
		opts.Version = "1.0.0"
	}
	docsPath := strings.TrimSpace(opts.DocsPath)
	if docsPath == "" {
		docsPath = "/api/docs"
	}

	humax.RegisterAllBodies()
	// Handler-local BindJSON bodies: call handlers.RegisterOpenAPIBodies from Register().

	cfg := huma.DefaultConfig(opts.Title, opts.Version)
	// Disable built-in docs page — we serve a SoulNexus-themed Scalar shell instead.
	cfg.DocsPath = ""
	cfg.OpenAPIPath = "/openapi"
	cfg.SchemasPath = "/schemas"
	cfg.Info.Description = humax.InfoDescription(opts.APIPrefix)
	if opts.APIPrefix != "" {
		cfg.Servers = []*huma.Server{{URL: "/", Description: "与站点同源（业务 API 前缀 " + opts.APIPrefix + "）"}}
	}

	api := humagin.New(r, cfg)
	humax.EnsureSecuritySchemes(api)
	mountCustomDocs(r, docsPath, opts.Title, "/openapi.json", opts.APIPrefix)

	huma.Register(api, huma.Operation{
		OperationID:   "get-api-meta",
		Method:        http.MethodGet,
		Path:          "/api/v1/meta",
		Summary:       "文档元数据",
		Description:   "返回 OpenAPI 导出地址与健康检查路径，便于集成方发现文档入口。",
		Tags:          []string{"元信息"},
		DefaultStatus: http.StatusOK,
	}, func(ctx context.Context, _ *struct{}) (*metaOutput, error) {
		out := &metaOutput{}
		out.Body.Name = opts.Title
		out.Body.Version = opts.Version
		out.Body.DocsPath = docsPath
		out.Body.OpenAPIJSON = "/openapi.json"
		out.Body.OpenAPIYAML = "/openapi.yaml"
		out.Body.Health = []string{"/healthz", "/livez", "/readyz", "/ready"}
		out.Body.Bodies = humax.DebugBodyCount()
		out.Body.Note = "带类型的 JSON 请求体来自 RegisterJSONBody / genbodies；其余需对照 handler。"
		return out, nil
	})
	humax.EnsureTag(api, "元信息")

	return api
}
