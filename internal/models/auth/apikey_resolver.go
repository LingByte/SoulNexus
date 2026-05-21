package auth

// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

import (
	"context"
	"errors"
	"strings"

	"github.com/LingByte/SoulNexus/pkg/constants"
	"github.com/gin-gonic/gin"
)

// APIKeyUserResolver resolves a user from API key + secret without requiring Gin DB middleware.
// cmd/server registers a gRPC-backed resolver; cmd/auth uses the default DB lookup.
type APIKeyUserResolver func(ctx context.Context, apiKey, apiSecret string) (*User, error)

var apiKeyUserResolver APIKeyUserResolver

// SetAPIKeyUserResolver registers how AuthApiRequired resolves X-API-KEY / X-API-SECRET.
// Pass nil to fall back to database lookup (auth service).
func SetAPIKeyUserResolver(r APIKeyUserResolver) {
	apiKeyUserResolver = r
}

func apiKeySecretFromRequest(c *gin.Context) (apiKey, apiSecret string, ok bool) {
	if c == nil {
		return "", "", false
	}
	apiKey = strings.TrimSpace(c.GetHeader(constants.CREDENTIAL_API_KEY))
	apiSecret = strings.TrimSpace(c.GetHeader(constants.CREDENTIAL_API_SECRET))
	if apiKey != "" && apiSecret != "" {
		return apiKey, apiSecret, true
	}
	apiKey = strings.TrimSpace(c.Query("apiKey"))
	apiSecret = strings.TrimSpace(c.Query("apiSecret"))
	return apiKey, apiSecret, apiKey != "" && apiSecret != ""
}

func resolveUserByAPIKey(ctx context.Context, c *gin.Context, apiKey, apiSecret string) (*User, error) {
	if apiKeyUserResolver != nil {
		return apiKeyUserResolver(ctx, apiKey, apiSecret)
	}
	if c == nil {
		return nil, errors.New("missing gin context for api key auth")
	}
	return GetUserByAPIKey(c, apiKey, apiSecret)
}
