package server

// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

// User Persona API handlers — user identity/profile definition for role-play scenarios.

import (
	"fmt"
	"strings"

	"github.com/LingByte/SoulNexus/i18n"
	"github.com/LingByte/SoulNexus/internal/models/auth"
	svcmodels "github.com/LingByte/SoulNexus/internal/models/server"
	"github.com/LingByte/SoulNexus/pkg/response"
	"github.com/LingByte/SoulNexus/pkg/utils"
	"github.com/gin-gonic/gin"
)

// ---------------------------------------------------------------------------
// CRUD: User Personas
// ---------------------------------------------------------------------------

// GET /api/chat/personas
func (h *Handlers) listPersonas(c *gin.Context) {
	user := auth.CurrentUser(c)
	if user == nil {
		response.Fail(c, i18n.MsgUnauthorized, nil)
		return
	}

	userID := fmt.Sprintf("%d", user.ID)

	var personas []svcmodels.UserPersona
	if err := h.db.Where("user_id = ? AND deleted_at IS NULL", userID).
		Order("is_default DESC, created_at DESC").
		Find(&personas).Error; err != nil {
		response.Fail(c, i18n.MsgPersonaListFailed, nil)
		return
	}

	response.Success(c, i18n.MsgPersonaListFetched, gin.H{
		"personas": personas,
		"total":    len(personas),
	})
}

// POST /api/chat/personas
func (h *Handlers) createPersona(c *gin.Context) {
	user := auth.CurrentUser(c)
	if user == nil {
		response.Fail(c, i18n.MsgUnauthorized, nil)
		return
	}

	var persona svcmodels.UserPersona
	if err := c.ShouldBindJSON(&persona); err != nil {
		response.Fail(c, i18n.MsgInvalidParams, nil)
		return
	}

	persona.ID = utils.SnowflakeUtil.GenID()
	persona.UserID = fmt.Sprintf("%d", user.ID)

	// If this is the first persona or marked as default, set as default
	var count int64
	h.db.Model(&svcmodels.UserPersona{}).Where("user_id = ? AND deleted_at IS NULL", persona.UserID).Count(&count)
	if count == 0 || persona.IsDefault {
		// unset other defaults
		if persona.IsDefault {
			h.db.Model(&svcmodels.UserPersona{}).
				Where("user_id = ? AND deleted_at IS NULL", persona.UserID).
				Update("is_default", false)
		} else if count == 0 {
			persona.IsDefault = true
		}
	}

	if err := h.db.Create(&persona).Error; err != nil {
		response.Fail(c, i18n.MsgPersonaCreateFailed, nil)
		return
	}

	response.Success(c, i18n.MsgPersonaCreated, gin.H{"persona": persona})
}

// PUT /api/chat/personas/:id
func (h *Handlers) updatePersona(c *gin.Context) {
	user := auth.CurrentUser(c)
	if user == nil {
		response.Fail(c, i18n.MsgUnauthorized, nil)
		return
	}

	personaID := c.Param("id")
	userID := fmt.Sprintf("%d", user.ID)

	var existing svcmodels.UserPersona
	if err := h.db.Where("id = ? AND user_id = ?", personaID, userID).First(&existing).Error; err != nil {
		response.Fail(c, i18n.MsgPersonaUpdateFailed, nil)
		return
	}

	var updates map[string]interface{}
	if err := c.ShouldBindJSON(&updates); err != nil {
		response.Fail(c, i18n.MsgInvalidParams, nil)
		return
	}

	// Prevent overwriting protected fields
	delete(updates, "id")
	delete(updates, "user_id")
	delete(updates, "created_at")

	// If setting as default, unset other defaults
	if isDefault, ok := updates["is_default"]; ok {
		if b, ok := isDefault.(bool); ok && b {
			h.db.Model(&svcmodels.UserPersona{}).
				Where("user_id = ? AND id != ? AND deleted_at IS NULL", userID, personaID).
				Update("is_default", false)
		}
	}

	if err := h.db.Model(&existing).Updates(updates).Error; err != nil {
		response.Fail(c, i18n.MsgPersonaUpdateFailed, nil)
		return
	}

	h.db.Where("id = ?", personaID).First(&existing)
	response.Success(c, i18n.MsgPersonaUpdated, gin.H{"persona": existing})
}

// DELETE /api/chat/personas/:id
func (h *Handlers) deletePersona(c *gin.Context) {
	user := auth.CurrentUser(c)
	if user == nil {
		response.Fail(c, i18n.MsgUnauthorized, nil)
		return
	}

	personaID := c.Param("id")
	userID := fmt.Sprintf("%d", user.ID)

	var persona svcmodels.UserPersona
	if err := h.db.Where("id = ? AND user_id = ?", personaID, userID).First(&persona).Error; err != nil {
		response.Fail(c, i18n.MsgPersonaDeleteFailed, nil)
		return
	}

	if err := h.db.Delete(&persona).Error; err != nil {
		response.Fail(c, i18n.MsgPersonaDeleteFailed, nil)
		return
	}

	response.Success(c, i18n.MsgPersonaDeleted, nil)
}

// ---------------------------------------------------------------------------
// Set default persona
// PUT /api/chat/personas/:id/default
// ---------------------------------------------------------------------------
func (h *Handlers) setDefaultPersona(c *gin.Context) {
	user := auth.CurrentUser(c)
	if user == nil {
		response.Fail(c, i18n.MsgUnauthorized, nil)
		return
	}

	personaID := c.Param("id")
	userID := fmt.Sprintf("%d", user.ID)

	var persona svcmodels.UserPersona
	if err := h.db.Where("id = ? AND user_id = ?", personaID, userID).First(&persona).Error; err != nil {
		response.Fail(c, i18n.MsgPersonaDefaultSetFailed, nil)
		return
	}

	// unset all defaults for this user
	h.db.Model(&svcmodels.UserPersona{}).
		Where("user_id = ? AND deleted_at IS NULL", userID).
		Update("is_default", false)

	// set this one as default
	h.db.Model(&persona).Update("is_default", true)

	response.Success(c, i18n.MsgPersonaDefaultSet, gin.H{"personaId": personaID})
}

// ---------------------------------------------------------------------------
// Build persona prompt injection
// GET /api/chat/personas/:id/inject
// ---------------------------------------------------------------------------
func (h *Handlers) injectPersona(c *gin.Context) {
	user := auth.CurrentUser(c)
	if user == nil {
		response.Fail(c, i18n.MsgUnauthorized, nil)
		return
	}

	personaID := c.Param("id")
	userID := fmt.Sprintf("%d", user.ID)

	// If no ID specified, use the default persona
	var persona svcmodels.UserPersona
	if personaID == "default" {
		if err := h.db.Where("user_id = ? AND is_default = true AND deleted_at IS NULL", userID).First(&persona).Error; err != nil {
			response.Fail(c, i18n.MsgPersonaInjectFailed, nil)
			return
		}
	} else {
		if err := h.db.Where("id = ? AND user_id = ? AND deleted_at IS NULL", personaID, userID).First(&persona).Error; err != nil {
			response.Fail(c, i18n.MsgPersonaInjectFailed, nil)
			return
		}
	}

	// Build prompt injection
	var sb strings.Builder
	sb.WriteString("[User Persona]\n")
	sb.WriteString(fmt.Sprintf("Name: %s\n", persona.Name))
	if persona.Description != "" {
		sb.WriteString(fmt.Sprintf("Description: %s\n", persona.Description))
	}
	if persona.Personality != "" {
		sb.WriteString(fmt.Sprintf("Personality: %s\n", persona.Personality))
	}
	if persona.Scenario != "" {
		sb.WriteString(fmt.Sprintf("Scenario: %s\n", persona.Scenario))
	}

	response.Success(c, i18n.MsgPersonaInjected, gin.H{
		"text":    sb.String(),
		"persona": persona,
	})
}
