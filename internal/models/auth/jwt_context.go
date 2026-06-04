package auth

// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

import (
	"errors"
	"net/http"

	"github.com/LingByte/SoulNexus/internal/models"
	"github.com/LingByte/SoulNexus/pkg/constants"
	"github.com/LingByte/SoulNexus/pkg/response"
	"github.com/LingByte/SoulNexus/pkg/utils"
	"github.com/gin-gonic/gin"
)

// UserFromAccessPayload builds the minimal Gin context user from JWT claims (no DB).
func UserFromAccessPayload(p *utils.AccessPayload) *User {
	if p == nil || p.UserID == 0 {
		return nil
	}
	return &User{
		BaseModel:      models.BaseModel{ID: p.UserID},
		Email:          p.Email,
		RoleSlugs:      RoleSlugsFromJWTClaim(p.Role),
		PermissionKeys: append([]string(nil), p.Perms...),
		Status:         UserStatusActive,
	}
}

func setUserFromJWT(c *gin.Context, token string) {
	km := utils.JWTKeyManager()
	if km == nil {
		response.AbortWithJSONError(c, http.StatusInternalServerError, errors.New("jwt key manager not initialized"))
		return
	}
	p, err := utils.ParseAccessTokenWithKey(token, km)
	if err != nil || p == nil || p.UserID == 0 {
		response.AbortWithJSONError(c, http.StatusUnauthorized, utils.ErrInvalidToken)
		return
	}
	u := UserFromAccessPayload(p)
	c.Set(constants.UserField, u)
	c.Next()
}
