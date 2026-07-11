package server

// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

// World Info / Lorebook API handlers — lightweight keyword/regex-triggered
// context injection for role-play and conversational AI.

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/LingByte/SoulNexus/i18n"
	"github.com/LingByte/SoulNexus/internal/models/auth"
	svcmodels "github.com/LingByte/SoulNexus/internal/models/server"
	"github.com/LingByte/SoulNexus/pkg/response"
	"github.com/LingByte/SoulNexus/pkg/utils"
	"github.com/gin-gonic/gin"
)

// ---------------------------------------------------------------------------
// CRUD: World Info Entries
// ---------------------------------------------------------------------------

// GET /api/chat/world-info
func (h *Handlers) listWorldInfoEntries(c *gin.Context) {
	user := auth.CurrentUser(c)
	if user == nil {
		response.Fail(c, i18n.MsgUnauthorized, nil)
		return
	}

	agentIDStr := c.Query("agentId")
	var agentID int64
	if agentIDStr != "" {
		fmt.Sscanf(agentIDStr, "%d", &agentID)
	}

	userID := fmt.Sprintf("%d", user.ID)

	var entries []svcmodels.WorldInfoEntry
	query := h.db.Where("deleted_at IS NULL").
		Where("(user_id = ? OR group_id IN (SELECT group_id FROM group_members WHERE user_id = ?))", userID, userID)
	if agentID > 0 {
		query = query.Where("agent_id = ?", agentID)
	}
	if err := query.Order("\"order\" ASC, created_at DESC").Find(&entries).Error; err != nil {
		response.Fail(c, i18n.MsgWorldInfoListFailed, nil)
		return
	}

	response.Success(c, i18n.MsgWorldInfoListFetched, gin.H{
		"entries": entries,
		"total":   len(entries),
	})
}

// POST /api/chat/world-info
func (h *Handlers) createWorldInfoEntry(c *gin.Context) {
	user := auth.CurrentUser(c)
	if user == nil {
		response.Fail(c, i18n.MsgUnauthorized, nil)
		return
	}

	var entry svcmodels.WorldInfoEntry
	if err := c.ShouldBindJSON(&entry); err != nil {
		response.Fail(c, i18n.MsgInvalidParams, nil)
		return
	}

	entry.ID = utils.SnowflakeUtil.GenID()
	entry.UserID = fmt.Sprintf("%d", user.ID)

	if entry.Depth <= 0 {
		entry.Depth = 4
	}

	if err := h.db.Create(&entry).Error; err != nil {
		response.Fail(c, i18n.MsgWorldInfoCreateFailed, nil)
		return
	}

	response.Success(c, i18n.MsgWorldInfoCreated, gin.H{"entry": entry})
}

// PUT /api/chat/world-info/:id
func (h *Handlers) updateWorldInfoEntry(c *gin.Context) {
	user := auth.CurrentUser(c)
	if user == nil {
		response.Fail(c, i18n.MsgUnauthorized, nil)
		return
	}

	entryID := c.Param("id")
	userID := fmt.Sprintf("%d", user.ID)

	var existing svcmodels.WorldInfoEntry
	if err := h.db.Where("id = ? AND user_id = ?", entryID, userID).First(&existing).Error; err != nil {
		response.Fail(c, i18n.MsgWorldInfoUpdateFailed, nil)
		return
	}

	var updates map[string]interface{}
	if err := c.ShouldBindJSON(&updates); err != nil {
		response.Fail(c, i18n.MsgInvalidParams, nil)
		return
	}

	// prevent overwriting protected fields
	delete(updates, "id")
	delete(updates, "user_id")
	delete(updates, "created_at")

	if err := h.db.Model(&existing).Updates(updates).Error; err != nil {
		response.Fail(c, i18n.MsgWorldInfoUpdateFailed, nil)
		return
	}

	// reload
	h.db.Where("id = ?", entryID).First(&existing)
	response.Success(c, i18n.MsgWorldInfoUpdated, gin.H{"entry": existing})
}

// DELETE /api/chat/world-info/:id
func (h *Handlers) deleteWorldInfoEntry(c *gin.Context) {
	user := auth.CurrentUser(c)
	if user == nil {
		response.Fail(c, i18n.MsgUnauthorized, nil)
		return
	}

	entryID := c.Param("id")
	userID := fmt.Sprintf("%d", user.ID)

	if err := h.db.Where("id = ? AND user_id = ?", entryID, userID).Delete(&svcmodels.WorldInfoEntry{}).Error; err != nil {
		response.Fail(c, i18n.MsgWorldInfoDeleteFailed, nil)
		return
	}

	response.Success(c, i18n.MsgWorldInfoDeleted, nil)
}

// ---------------------------------------------------------------------------
// Activation: match world info entries against conversation context
// POST /api/chat/world-info/activate
// ---------------------------------------------------------------------------
type ActivateWorldInfoRequest struct {
	AgentID int64    `json:"agentId"`
	Text    string   `json:"text"`    // current user message or full conversation
	MaxScan int      `json:"maxScan"` // max depth to scan (default 100)
}

type ActivatedEntry struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Content string `json:"content"`
	Order   int    `json:"order"`
}

func (h *Handlers) activateWorldInfo(c *gin.Context) {
	user := auth.CurrentUser(c)
	if user == nil {
		response.Fail(c, i18n.MsgUnauthorized, nil)
		return
	}

	var req ActivateWorldInfoRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, i18n.MsgInvalidParams, nil)
		return
	}

	if req.MaxScan <= 0 {
		req.MaxScan = 100
	}

	userID := fmt.Sprintf("%d", user.ID)

	// fetch all enabled entries for this agent/user
	var entries []svcmodels.WorldInfoEntry
	query := h.db.Where("deleted_at IS NULL AND enabled = true").
		Where("(user_id = ? OR group_id IN (SELECT group_id FROM group_members WHERE user_id = ?))", userID, userID)
	if req.AgentID > 0 {
		query = query.Where("agent_id = ?", req.AgentID)
	}
	if err := query.Order("\"order\" ASC").Find(&entries).Error; err != nil {
		response.Fail(c, i18n.MsgWorldInfoActivateFailed, nil)
		return
	}

	textLower := strings.ToLower(req.Text)
	activated := make([]ActivatedEntry, 0)
	seen := make(map[string]bool)

	// First pass: constant entries
	for _, e := range entries {
		if e.Constant && !seen[e.ID] {
			activated = append(activated, ActivatedEntry{
				ID:      e.ID,
				Name:    e.Name,
				Content: e.Content,
				Order:   e.Order,
			})
			seen[e.ID] = true
		}
	}

	// Second pass: key/regex triggered entries (with chain/selective support)
	activatedKeys := make([]ActivatedEntry, 0)
	for _, e := range entries {
		if e.Constant || seen[e.ID] {
			continue
		}

		matched := false

		// Check regex trigger
		if e.Regex != "" {
			if re, err := regexp.Compile("(?i)" + e.Regex); err == nil {
				if re.MatchString(req.Text) {
					matched = true
				}
			}
		}

		// Check keyword triggers
		if !matched && e.Keys != "" {
			keywords := strings.Split(e.Keys, ",")
			for _, kw := range keywords {
				kw = strings.TrimSpace(strings.ToLower(kw))
				if kw != "" && strings.Contains(textLower, kw) {
					matched = true
					break
				}
			}
		}

		if matched && !seen[e.ID] {
			activatedKeys = append(activatedKeys, ActivatedEntry{
				ID:      e.ID,
				Name:    e.Name,
				Content: e.Content,
				Order:   e.Order,
			})
			seen[e.ID] = true

			// Chain: if Selective is true, scan subsequent entries triggered by this content
			if e.Selective && e.Depth > 0 {
				chainEntries := scanChainedEntries(entries, e, textLower, e.Depth, seen)
				activatedKeys = append(activatedKeys, chainEntries...)
			}
		}
	}

	activated = append(activated, activatedKeys...)

	// Limit scan depth
	if len(activated) > req.MaxScan {
		activated = activated[:req.MaxScan]
	}

	response.Success(c, i18n.MsgWorldInfoActivated, gin.H{
		"activated": activated,
		"total":     len(activated),
	})
}

// scanChainedEntries recursively scans entries that should chain from a triggering entry
func scanChainedEntries(entries []svcmodels.WorldInfoEntry, trigger svcmodels.WorldInfoEntry, text string, depth int, seen map[string]bool) []ActivatedEntry {
	if depth <= 0 {
		return nil
	}

	result := make([]ActivatedEntry, 0)
	for _, e := range entries {
		if seen[e.ID] || e.Constant || e.ID == trigger.ID {
			continue
		}

		matched := false
		if e.Regex != "" {
			if re, err := regexp.Compile("(?i)" + e.Regex); err == nil && re.MatchString(text) {
				matched = true
			}
		}
		if !matched && e.Keys != "" {
			for _, kw := range strings.Split(e.Keys, ",") {
				kw = strings.TrimSpace(strings.ToLower(kw))
				if kw != "" && strings.Contains(text, kw) {
					matched = true
					break
				}
			}
		}

		if matched {
			seen[e.ID] = true
			result = append(result, ActivatedEntry{
				ID:      e.ID,
				Name:    e.Name,
				Content: e.Content,
				Order:   e.Order,
			})
			if e.Selective && depth > 1 {
				result = append(result, scanChainedEntries(entries, e, text, depth-1, seen)...)
			}
		}
	}
	return result
}

// ---------------------------------------------------------------------------
// Build formatted world info context for injection into system prompt
// POST /api/chat/world-info/inject
// ---------------------------------------------------------------------------
type InjectWorldInfoRequest struct {
	AgentID int64  `json:"agentId"`
	Text    string `json:"text"`
	Format  string `json:"format"` // "prompt" (default) or "json"
}

func (h *Handlers) injectWorldInfo(c *gin.Context) {
	user := auth.CurrentUser(c)
	if user == nil {
		response.Fail(c, i18n.MsgUnauthorized, nil)
		return
	}

	var req InjectWorldInfoRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, i18n.MsgInvalidParams, nil)
		return
	}
	if req.Format == "" {
		req.Format = "prompt"
	}

	userID := fmt.Sprintf("%d", user.ID)

	// Fetch entries
	var entries []svcmodels.WorldInfoEntry
	query := h.db.Where("deleted_at IS NULL AND enabled = true").
		Where("(user_id = ? OR group_id IN (SELECT group_id FROM group_members WHERE user_id = ?))", userID, userID)
	if req.AgentID > 0 {
		query = query.Where("agent_id = ?", req.AgentID)
	}
	if err := query.Order("\"order\" ASC").Find(&entries).Error; err != nil {
		response.Fail(c, i18n.MsgWorldInfoInjectFailed, nil)
		return
	}

	// Activate matching entries
	activated, _ := activateEntriesForText(entries, req.Text)

	// Build output
	if req.Format == "json" {
		response.Success(c, i18n.MsgWorldInfoInjected, gin.H{"entries": activated})
		return
	}

	// Build prompt-formatted injection
	var sb strings.Builder
	sb.WriteString("[World Information]\n")
	for _, e := range activated {
		sb.WriteString(fmt.Sprintf("## %s\n%s\n\n", e.Name, e.Content))
	}
	response.Success(c, i18n.MsgWorldInfoInjected, gin.H{
		"text":    sb.String(),
		"entries": activated,
	})
}

// activateEntriesForText matches entries against text without DB dependency
func activateEntriesForText(entries []svcmodels.WorldInfoEntry, text string) ([]ActivatedEntry, map[string]bool) {
	textLower := strings.ToLower(text)
	activated := make([]ActivatedEntry, 0)
	seen := make(map[string]bool)

	// Constants first
	for _, e := range entries {
		if e.Constant && !seen[e.ID] {
			activated = append(activated, ActivatedEntry{ID: e.ID, Name: e.Name, Content: e.Content, Order: e.Order})
			seen[e.ID] = true
		}
	}

	// Triggered entries
	for _, e := range entries {
		if e.Constant || seen[e.ID] {
			continue
		}
		matched := false
		if e.Regex != "" {
			if re, err := regexp.Compile("(?i)" + e.Regex); err == nil && re.MatchString(text) {
				matched = true
			}
		}
		if !matched && e.Keys != "" {
			for _, kw := range strings.Split(e.Keys, ",") {
				if kw = strings.TrimSpace(strings.ToLower(kw)); kw != "" && strings.Contains(textLower, kw) {
					matched = true
					break
				}
			}
		}
		if matched {
			activated = append(activated, ActivatedEntry{ID: e.ID, Name: e.Name, Content: e.Content, Order: e.Order})
			seen[e.ID] = true
		}
	}

	return activated, seen
}
