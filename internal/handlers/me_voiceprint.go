package handlers

import (
	"context"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/LingByte/SoulNexus/internal/constants"
	"github.com/LingByte/SoulNexus/internal/models"
	"github.com/LingByte/SoulNexus/pkg/i18n"
	"github.com/LingByte/SoulNexus/pkg/middleware"
	"github.com/LingByte/SoulNexus/pkg/response"
	"github.com/LingByte/SoulNexus/pkg/voice/voiceprintconfig"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

func (h *Handlers) getMyVoiceprint(c *gin.Context) {
	if middleware.AuthPlatformAdminID(c) > 0 {
		response.Render(c, response.NewI18n(response.CodeForbidden, i18n.KeyForbidden))
		return
	}
	tid := middleware.CurrentTenantID(c)
	uid := middleware.AuthUserID(c)
	u, err := models.GetAuthenticatedTenantUser(h.db, uid, tid)
	if err != nil {
		response.Render(c, response.Err(response.CodeUnauthorized))
		return
	}
	if !models.TenantUserHasVoiceprint(u) {
		response.SuccessI18n(c, i18n.KeySuccess, gin.H{"enrolled": false})
		return
	}
	row, err := models.GetAccountVoiceprintForUser(h.db, tid, uid)
	if err != nil {
		_, _ = models.UpdateTenantUser(h.db, uid, map[string]any{"voiceprint_id": nil}, middleware.AuditOperator(c))
		response.SuccessI18n(c, i18n.KeySuccess, gin.H{"enrolled": false})
		return
	}
	response.SuccessI18n(c, i18n.KeySuccess, gin.H{
		"enrolled": true,
		"profile":  row,
	})
}

func (h *Handlers) enrollMyVoiceprint(c *gin.Context) {
	if middleware.AuthPlatformAdminID(c) > 0 {
		response.Render(c, response.NewI18n(response.CodeForbidden, i18n.KeyForbidden))
		return
	}
	tid := middleware.CurrentTenantID(c)
	uid := middleware.AuthUserID(c)
	operator := middleware.AuditOperator(c)

	slug, _, ok := voiceprintconfig.ResolveEnabled()
	if !ok {
		response.Render(c, response.Wrap(response.CodeBadRequest, "voiceprint not enabled", fmt.Errorf("VOICEPRINT_PROVIDER not configured")))
		return
	}

	var audio []byte
	if file, err := c.FormFile("audio"); err == nil && file != nil {
		fh, err := file.Open()
		if err != nil {
			response.Render(c, response.Wrap(response.CodeBadRequest, "open audio failed", err))
			return
		}
		defer fh.Close()
		audio, err = io.ReadAll(io.LimitReader(fh, 8<<20))
		if err != nil {
			response.Render(c, response.Wrap(response.CodeBadRequest, "read audio failed", err))
			return
		}
	} else if audioURL := strings.TrimSpace(c.PostForm("audioUrl")); audioURL != "" {
		var err error
		audio, err = fetchVoiceprintAudioURL(audioURL)
		if err != nil {
			response.Render(c, response.Wrap(response.CodeBadRequest, "fetch audio url failed", err))
			return
		}
	} else {
		response.Render(c, response.Wrap(response.CodeBadRequest, "audio file or audioUrl required", fmt.Errorf("missing audio")))
		return
	}

	displayName := strings.TrimSpace(c.PostForm("name"))
	if displayName == "" {
		displayName = "账号声纹"
	}

	if u, err := models.GetAuthenticatedTenantUser(h.db, uid, tid); err == nil && models.TenantUserHasVoiceprint(u) {
		if old, err := models.GetAccountVoiceprint(h.db, tid, *u.VoiceprintID); err == nil {
			bridge, berr := voiceprintconfig.NewBridge()
			if berr == nil {
				ctx, cancel := context.WithTimeout(c.Request.Context(), 30*time.Second)
				_ = bridge.Delete(ctx, tid, nil, old.FeatureID)
				cancel()
				bridge.Close()
			}
			_ = models.DeleteVoiceprintProfile(h.db, tid, old.ID)
		}
		_, _ = models.UpdateTenantUser(h.db, uid, map[string]any{"voiceprint_id": nil}, operator)
	}

	featureID := uuid.NewString()
	row := &models.VoiceprintProfile{
		TenantID:  tid,
		Scene:     models.VoiceprintSceneAccount,
		Name:      displayName,
		Provider:  slug,
		FeatureID: featureID,
		Status:    models.VoiceprintStatusActive,
	}
	if err := models.CreateVoiceprintProfile(h.db, row); err != nil {
		response.Render(c, response.Wrap(response.CodeBadRequest, "save profile failed", err))
		return
	}

	bridge, err := voiceprintconfig.NewBridge()
	if err != nil {
		_ = models.DeleteVoiceprintProfile(h.db, tid, row.ID)
		response.Render(c, response.Wrap(response.CodeBadRequest, "voiceprint service unavailable", err))
		return
	}
	defer bridge.Close()

	ctx, cancel := context.WithTimeout(c.Request.Context(), 45*time.Second)
	defer cancel()
	enrolledID, err := bridge.Enroll(ctx, tid, nil, row.ID, featureID, displayName, "", audio)
	if err != nil {
		_ = models.DeleteVoiceprintProfile(h.db, tid, row.ID)
		response.Render(c, response.Wrap(response.CodeBadRequest, err.Error(), err))
		return
	}
	row.FeatureID = enrolledID
	if _, err := models.UpdateTenantUser(h.db, uid, map[string]any{"voiceprint_id": row.ID}, operator); err != nil {
		_ = models.DeleteVoiceprintProfile(h.db, tid, row.ID)
		response.Render(c, response.Wrap(response.CodeBadRequest, "bind user voiceprint failed", err))
		return
	}
	h.recordOpChange(c, OpLogEntry{
		TenantID: tid, Action: constants.OpActionCreate,
		Resource: constants.OpResourceVoiceprint, ResourceID: row.ID, ResourceName: row.Name,
		Summary: fmt.Sprintf("Created account voiceprint %s", row.Name),
	}, nil, row)
	response.SuccessI18n(c, i18n.KeySuccess, row)
}

func (h *Handlers) deleteMyVoiceprint(c *gin.Context) {
	if middleware.AuthPlatformAdminID(c) > 0 {
		response.Render(c, response.NewI18n(response.CodeForbidden, i18n.KeyForbidden))
		return
	}
	tid := middleware.CurrentTenantID(c)
	uid := middleware.AuthUserID(c)
	operator := middleware.AuditOperator(c)

	u, err := models.GetAuthenticatedTenantUser(h.db, uid, tid)
	if err != nil || !models.TenantUserHasVoiceprint(u) {
		response.Render(c, response.NewI18n(response.CodeNotFound, i18n.KeyNotFound))
		return
	}
	row, err := models.GetAccountVoiceprint(h.db, tid, *u.VoiceprintID)
	if err != nil {
		_, _ = models.UpdateTenantUser(h.db, uid, map[string]any{"voiceprint_id": nil}, operator)
		response.SuccessI18n(c, i18n.KeySuccess, gin.H{"deleted": true})
		return
	}

	bridge, err := voiceprintconfig.NewBridge()
	if err == nil {
		ctx, cancel := context.WithTimeout(c.Request.Context(), 30*time.Second)
		_ = bridge.Delete(ctx, tid, nil, row.FeatureID)
		cancel()
		bridge.Close()
	}
	_ = models.DeleteVoiceprintProfile(h.db, tid, row.ID)
	_, _ = models.UpdateTenantUser(h.db, uid, map[string]any{"voiceprint_id": nil}, operator)
	response.SuccessI18n(c, i18n.KeySuccess, gin.H{"deleted": true})
}
