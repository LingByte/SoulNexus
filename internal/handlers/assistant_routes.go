package handlers

// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

import (
	"github.com/LingByte/SoulNexus/pkg/humax"
	"github.com/LingByte/SoulNexus/pkg/middleware"
)

// registerAssistantRoutes mounts assistants, voices, clones, voiceprints,
// tenant org, NLU, assistant-tools, and MCP market under the API root.
func (h *Handlers) registerAssistantRoutes(r *humax.Group) {
	h.registerAssistantsCRUDRoutes(r)
	h.registerTenantOrgRoutes(r)
	h.registerTenantNLURoutes(r)
	h.registerTenantAssistantToolRoutes(r)
	h.registerTenantMcpMarketRoutes(r)
}

func (h *Handlers) registerAssistantsCRUDRoutes(r *humax.Group) {
	read := r.Group("")
	read.Use(middleware.RequireTenantPermissionAny("api.assistants.read", "menu.res.assistant"))
	{
		read.GET("/assistants", h.listAssistants)
		read.GET("/assistants/:id", h.getAssistant)
		read.GET("/assistants/:id/members", h.listAssistantMembers)
		read.GET("/assistants/:id/versions", h.listAssistantVersions)
		read.GET("/assistants/:id/diff", h.diffAssistantVersions)
		read.GET("/voices", h.listVoiceCatalog)
		read.POST("/voices/preview", h.previewVoice)
		read.GET("/tenant-voice-providers", h.getTenantVoiceProviders)
		read.GET("/voice-clones/config", h.getVoiceCloneConfig)
		read.GET("/voice-clones", h.listVoiceClones)
		read.GET("/voice-clones/training-texts", h.getVoiceCloneTrainingTexts)
		read.GET("/voice-clones/:id", h.getVoiceClone)
		read.GET("/voiceprints/config", h.getVoiceprintConfig)
		read.GET("/voiceprints/self-test", h.voiceprintSelfTest)
		read.GET("/voiceprints", h.listVoiceprints)
		read.GET("/voiceprints/:id", h.getVoiceprint)
		read.GET("/voiceprints/:id/speaker", h.getVoiceprintSpeaker)
		read.GET("/voice-synthesis-history", h.listVoiceSynthesisHistory)
	}
	write := r.Group("")
	write.Use(middleware.RequireTenantPermissionAny("api.assistants.write", "menu.res.assistant"))
	{
		write.POST("/assistants", h.createAssistant)
		write.PUT("/assistants/:id", h.updateAssistant)
		write.PUT("/assistants/:id/settings", h.patchAssistantSettings)
		write.POST("/assistants/:id/avatar", h.uploadAssistantAvatar)
		write.PUT("/assistants/:id/members", h.updateAssistantMembers)
		write.POST("/assistants/:id/members", h.addAssistantMembers)
		write.DELETE("/assistants/:id/members/:userId", h.removeAssistantMember)
		write.DELETE("/assistants/:id", h.deleteAssistant)
		write.POST("/assistants/:id/publish", h.publishAssistant)
		write.POST("/assistants/:id/rollback", h.rollbackAssistant)
		write.POST("/assistants/import-from-tenant", h.importAssistantFromTenant)
		write.POST("/voice-clones", h.createVoiceClone)
		write.POST("/voice-clones/:id/audio", h.submitVoiceCloneAudio)
		write.POST("/voice-clones/:id/sync", h.syncVoiceCloneStatus)
		write.POST("/voice-clones/:id/preview", h.previewVoiceClone)
		write.POST("/voice-clones/:id/synthesize", h.synthesizeVoiceClone)
		write.DELETE("/voice-clones/:id", h.deleteVoiceClone)
		write.DELETE("/voice-synthesis-history/:id", h.deleteVoiceSynthesisHistory)
		write.POST("/voiceprints", h.createVoiceprint)
		write.POST("/voiceprints/identify", h.identifyVoiceprint)
		write.PUT("/voiceprints/:id/assistant", h.bindVoiceprintAssistant)
		write.PUT("/voiceprints/:id/speaker", h.upsertVoiceprintSpeaker)
		write.DELETE("/voiceprints/:id", h.deleteVoiceprint)
	}
}
