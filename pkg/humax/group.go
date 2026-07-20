// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

// Package humax bridges existing Gin handlers into Huma's OpenAPI surface.
//
// Routes still execute through Gin (so middleware using c.Next() keeps working).
// Every GET/POST/PUT/PATCH/DELETE is also registered on the Huma OpenAPI document
// so /api/docs and /openapi.json describe the full HTTP API.
package humax

import (
	"net/http"
	"strings"
	"unicode"

	"github.com/danielgtaylor/huma/v2"
	"github.com/gin-gonic/gin"
)

// Group mirrors the subset of gin.RouterGroup used by SoulNexus handlers,
// while documenting each route on the Huma OpenAPI object.
type Group struct {
	api  huma.API
	gin  *gin.RouterGroup
	base string // absolute path prefix, Gin style (/api/credentials)
}

// NewGroup mounts a Huma-documented group at absolutePath on engine
// (e.g. absolutePath="/api").
func NewGroup(api huma.API, engine *gin.Engine, absolutePath string) *Group {
	absolutePath = normalizePrefix(absolutePath)
	return &Group{
		api:  api,
		gin:  engine.Group(absolutePath),
		base: absolutePath,
	}
}

// Gin exposes the underlying RouterGroup for rare cases (e.g. wrapping).
func (g *Group) Gin() *gin.RouterGroup { return g.gin }

// Base returns the absolute path prefix.
func (g *Group) Base() string { return g.base }

// Use appends Gin middleware to this group.
func (g *Group) Use(middleware ...gin.HandlerFunc) *Group {
	g.gin.Use(middleware...)
	return g
}

// Group creates a sub-group with an optional middleware chain.
func (g *Group) Group(relativePath string, handlers ...gin.HandlerFunc) *Group {
	rel := strings.TrimPrefix(relativePath, "/")
	childGin := g.gin.Group(rel, handlers...)
	childBase := joinPath(g.base, rel)
	return &Group{api: g.api, gin: childGin, base: childBase}
}

func (g *Group) GET(relativePath string, handlers ...gin.HandlerFunc) {
	g.handle(http.MethodGet, relativePath, handlers...)
}
func (g *Group) POST(relativePath string, handlers ...gin.HandlerFunc) {
	g.handle(http.MethodPost, relativePath, handlers...)
}
func (g *Group) PUT(relativePath string, handlers ...gin.HandlerFunc) {
	g.handle(http.MethodPut, relativePath, handlers...)
}
func (g *Group) PATCH(relativePath string, handlers ...gin.HandlerFunc) {
	g.handle(http.MethodPatch, relativePath, handlers...)
}
func (g *Group) DELETE(relativePath string, handlers ...gin.HandlerFunc) {
	g.handle(http.MethodDelete, relativePath, handlers...)
}

func (g *Group) handle(method, relativePath string, handlers ...gin.HandlerFunc) {
	switch method {
	case http.MethodGet:
		g.gin.GET(relativePath, handlers...)
	case http.MethodPost:
		g.gin.POST(relativePath, handlers...)
	case http.MethodPut:
		g.gin.PUT(relativePath, handlers...)
	case http.MethodPatch:
		g.gin.PATCH(relativePath, handlers...)
	case http.MethodDelete:
		g.gin.DELETE(relativePath, handlers...)
	default:
		g.gin.Handle(method, relativePath, handlers...)
	}

	full := joinPath(g.base, strings.TrimPrefix(relativePath, "/"))
	if relativePath == "" || relativePath == "/" {
		full = g.base
	}
	document(g.api, method, full)
}

// GinPathToOpenAPI converts /api/users/:id → /api/users/{id}.
func GinPathToOpenAPI(p string) string {
	parts := strings.Split(p, "/")
	for i, part := range parts {
		if strings.HasPrefix(part, ":") && len(part) > 1 {
			parts[i] = "{" + part[1:] + "}"
		}
		if strings.HasPrefix(part, "*") && len(part) > 1 {
			parts[i] = "{" + part[1:] + "}"
		}
	}
	return strings.Join(parts, "/")
}

// OperationID builds a stable OpenAPI operationId.
func OperationID(method, oapiPath string) string {
	s := strings.ToLower(method) + oapiPath
	s = strings.ReplaceAll(s, "{", "")
	s = strings.ReplaceAll(s, "}", "")
	var b strings.Builder
	prevDash := false
	for _, r := range s {
		switch {
		case unicode.IsLetter(r) || unicode.IsDigit(r):
			b.WriteRune(r)
			prevDash = false
		default:
			if !prevDash {
				b.WriteByte('-')
				prevDash = true
			}
		}
	}
	id := strings.Trim(b.String(), "-")
	if id == "" {
		id = "op"
	}
	return id
}

// Summary is kept for compatibility; prefer HumanSummary for docs UI.
func Summary(method, oapiPath string) string {
	return HumanSummary(method, oapiPath)
}

// TagFromPath is kept for compatibility; prefer HumanTag.
func TagFromPath(oapiPath string) string {
	return HumanTag(oapiPath)
}

func normalizePrefix(p string) string {
	p = strings.TrimSpace(p)
	if p == "" {
		return "/"
	}
	if !strings.HasPrefix(p, "/") {
		p = "/" + p
	}
	return strings.TrimSuffix(p, "/")
}

func joinPath(base, rel string) string {
	base = strings.TrimSuffix(base, "/")
	rel = strings.TrimPrefix(rel, "/")
	if rel == "" {
		return base
	}
	return base + "/" + rel
}
