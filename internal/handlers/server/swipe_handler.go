package server

// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

// Swipe/多候选回复 + Chat Branching API handlers

import (
	"fmt"
	"log"
	"strings"

	"github.com/LingByte/SoulNexus/i18n"
	"github.com/LingByte/SoulNexus/internal/models/auth"
	svcmodels "github.com/LingByte/SoulNexus/internal/models/server"
	"github.com/LingByte/SoulNexus/pkg/llm"
	"github.com/LingByte/SoulNexus/pkg/response"
	"github.com/LingByte/SoulNexus/pkg/utils"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// ---------------------------------------------------------------------------
// Swipe: 获取某条 user message 下的所有候选 assistant 回复
// GET /api/chat/messages/:id/alternatives
// ---------------------------------------------------------------------------
type MessageAlternative struct {
	ID             string `json:"id"`
	Content        string `json:"content"`
	BranchIndex    int    `json:"branchIndex"`
	IsActiveBranch bool   `json:"isActiveBranch"`
	TokenCount     int    `json:"tokenCount"`
	Model          string `json:"model"`
	Provider       string `json:"provider"`
	CreatedAt      string `json:"createdAt"`
}

func (h *Handlers) getMessageAlternatives(c *gin.Context) {
	user := auth.CurrentUser(c)
	if user == nil {
		response.Fail(c, i18n.MsgUnauthorized, nil)
		return
	}

	parentID := c.Param("id")
	if parentID == "" {
		response.Fail(c, i18n.MsgInvalidParams, nil)
		return
	}

	// verify the parent message exists and belongs to user
	var parentMsg svcmodels.ChatMessage
	if err := h.db.Where("id = ? AND role = ?", parentID, "user").First(&parentMsg).Error; err != nil {
		response.Fail(c, i18n.MsgNotFound, nil)
		return
	}

	// verify session belongs to user
	var session svcmodels.ChatSession
	if err := h.db.Where("id = ? AND user_id = ?", parentMsg.SessionID, fmt.Sprintf("%d", user.ID)).First(&session).Error; err != nil {
		response.Fail(c, i18n.MsgForbidden, nil)
		return
	}

	// fetch all assistant replies under this parent
	var replies []svcmodels.ChatMessage
	if err := h.db.Where("parent_message_id = ? AND role = ?", parentID, "assistant").
		Order("branch_index ASC").Find(&replies).Error; err != nil {
		response.Fail(c, i18n.MsgChatAlternativesFailed, nil)
		return
	}

	alternatives := make([]MessageAlternative, 0, len(replies))
	for _, r := range replies {
		alternatives = append(alternatives, MessageAlternative{
			ID:             r.ID,
			Content:        r.Content,
			BranchIndex:    r.BranchIndex,
			IsActiveBranch: r.IsActiveBranch,
			TokenCount:     r.TokenCount,
			Model:          r.Model,
			Provider:       r.Provider,
			CreatedAt:      r.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		})
	}

	response.Success(c, i18n.MsgChatAlternativesFetched, gin.H{
		"parentId":     parentID,
		"alternatives": alternatives,
	})
}

// ---------------------------------------------------------------------------
// Swipe: 切换到指定候选回复 (set it as active branch)
// POST /api/chat/messages/:id/activate
// ---------------------------------------------------------------------------
func (h *Handlers) activateMessageAlternative(c *gin.Context) {
	user := auth.CurrentUser(c)
	if user == nil {
		response.Fail(c, i18n.MsgUnauthorized, nil)
		return
	}

	messageID := c.Param("id")
	if messageID == "" {
		response.Fail(c, i18n.MsgInvalidParams, nil)
		return
	}

	var targetMsg svcmodels.ChatMessage
	if err := h.db.Where("id = ? AND role = ?", messageID, "assistant").First(&targetMsg).Error; err != nil {
		response.Fail(c, i18n.MsgNotFound, nil)
		return
	}

	// verify ownership
	var session svcmodels.ChatSession
	if err := h.db.Where("id = ? AND user_id = ?", targetMsg.SessionID, fmt.Sprintf("%d", user.ID)).First(&session).Error; err != nil {
		response.Fail(c, i18n.MsgForbidden, nil)
		return
	}

	// transaction: deactivate all siblings, activate this one
	tx := h.db.Begin()

	// deactivate all siblings under same parent
	if targetMsg.ParentMessageID != nil && *targetMsg.ParentMessageID != "" {
		if err := tx.Model(&svcmodels.ChatMessage{}).
			Where("parent_message_id = ? AND role = ?", *targetMsg.ParentMessageID, "assistant").
			Update("is_active_branch", false).Error; err != nil {
			tx.Rollback()
			response.Fail(c, i18n.MsgChatActivateFailed, nil)
			return
		}
	}

	// activate this one
	if err := tx.Model(&targetMsg).Update("is_active_branch", true).Error; err != nil {
		tx.Rollback()
		response.Fail(c, i18n.MsgChatActivateFailed, nil)
		return
	}

	tx.Commit()

	response.Success(c, i18n.MsgChatAlternativeActivated, gin.H{
		"messageId":     messageID,
		"isActiveBranch": true,
	})
}

// ---------------------------------------------------------------------------
// Swipe: 为某条 user message 生成新的候选回复 (regenerate)
// POST /api/chat/messages/:id/regenerate
// ---------------------------------------------------------------------------
type RegenerateRequest struct {
	Temperature *float32 `json:"temperature"`
	MaxTokens   *int     `json:"maxTokens"`
	Model       string   `json:"model"`
}

func (h *Handlers) regenerateMessage(c *gin.Context) {
	user := auth.CurrentUser(c)
	if user == nil {
		response.Fail(c, i18n.MsgUnauthorized, nil)
		return
	}

	parentID := c.Param("id")
	if parentID == "" {
		response.Fail(c, i18n.MsgInvalidParams, nil)
		return
	}

	// fetch parent user message
	var parentMsg svcmodels.ChatMessage
	if err := h.db.Where("id = ? AND role = ?", parentID, "user").First(&parentMsg).Error; err != nil {
		response.Fail(c, i18n.MsgNotFound, nil)
		return
	}

	// verify session belongs to user
	var session svcmodels.ChatSession
	if err := h.db.Where("id = ? AND user_id = ?", parentMsg.SessionID, fmt.Sprintf("%d", user.ID)).First(&session).Error; err != nil {
		response.Fail(c, i18n.MsgForbidden, nil)
		return
	}

	// parse request body
	var req RegenerateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		req = RegenerateRequest{}
	}

	// determine next branch index
	var maxIndex int
	h.db.Model(&svcmodels.ChatMessage{}).
		Where("parent_message_id = ? AND role = ?", parentID, "assistant").
		Select("COALESCE(MAX(branch_index), -1)").
		Scan(&maxIndex)
	newBranchIndex := maxIndex + 1

	// Inject WorldInfo and Persona into system prompt before building context
	sysPrompt := session.SystemPrompt
	if sysPrompt != "" {
		sysPrompt = h.injectWorldInfoForChat(c.Request.Context(), session.AgentID, parentMsg.Content, sysPrompt, fmt.Sprintf("%d", user.ID))
		sysPrompt = h.injectPersonaForChat(c.Request.Context(), fmt.Sprintf("%d", user.ID), sysPrompt)
	}

	// build conversation context: all active messages up to the parent
	messages, err := buildActiveConversationContext(h.db, session.ID, parentID, sysPrompt)
	if err != nil {
		response.Fail(c, i18n.MsgChatRegenerateFailed, nil)
		return
	}

	// get agent info for LLM config
	var agent svcmodels.Agent
	if err := h.db.Where("id = ?", session.AgentID).First(&agent).Error; err != nil {
		response.Fail(c, i18n.MsgChatRegenerateFailed, nil)
		return
	}

	// get API credentials
	apiKey := c.GetHeader("X-Api-Key")
	apiSecret := c.GetHeader("X-Api-Secret")

	temperature := float32(0.7)
	if req.Temperature != nil {
		temperature = *req.Temperature
	} else if agent.Temperature > 0 {
		temperature = agent.Temperature
	}

	maxTokens := 1024
	if req.MaxTokens != nil {
		maxTokens = *req.MaxTokens
	} else if agent.MaxTokens > 0 {
		maxTokens = agent.MaxTokens
	}

	model := session.Model
	if req.Model != "" {
		model = req.Model
	}

	provider := session.Provider

	// Build QueryOptions
	queryOpts := &llm.QueryOptions{
		Model:       model,
		N:           1,
		MaxTokens:   maxTokens,
		Temperature: temperature,
		SessionID:   session.ID,
		UserID:      fmt.Sprintf("%d", user.ID),
		RequestType: "regenerate",
		Messages:    messages,
	}

	// Get LLM handler
	llmHandler, err := getLLMHandler(c, apiKey, apiSecret, provider, queryOpts)
	if err != nil {
		response.Fail(c, i18n.MsgChatRegenerateFailed, nil)
		return
	}

	// Execute query
	resp, err := llmHandler.QueryWithOptions(parentMsg.Content, queryOpts)
	if err != nil {
		response.Fail(c, i18n.MsgChatRegenerateFailed, nil)
		return
	}

	// Extract response content
	responseContent := ""
	if len(resp.Choices) > 0 {
		responseContent = resp.Choices[0].Content
	}

	tokenCount := 0
	if resp.Usage != nil {
		tokenCount = resp.Usage.TotalTokens
	}

	// Create new assistant message with branch info
	newMsg := svcmodels.ChatMessage{
		ID:              utils.SnowflakeUtil.GenID(),
		SessionID:       session.ID,
		Role:            "assistant",
		Content:         responseContent,
		TokenCount:      tokenCount,
		Model:           model,
		Provider:        provider,
		RequestID:       resp.RequestID,
		ParentMessageID: &parentID,
		BranchIndex:     newBranchIndex,
		IsActiveBranch:  true,
	}

	// Deactivate all existing siblings
	tx := h.db.Begin()
	if err := tx.Model(&svcmodels.ChatMessage{}).
		Where("parent_message_id = ? AND role = ?", parentID, "assistant").
		Update("is_active_branch", false).Error; err != nil {
		tx.Rollback()
		response.Fail(c, i18n.MsgChatRegenerateFailed, nil)
		return
	}
	if err := tx.Create(&newMsg).Error; err != nil {
		tx.Rollback()
		response.Fail(c, i18n.MsgChatRegenerateFailed, nil)
		return
	}
	tx.Commit()

	response.Success(c, i18n.MsgChatRegenerated, gin.H{
		"id":            newMsg.ID,
		"content":       responseContent,
		"branchIndex":   newBranchIndex,
		"isActiveBranch": true,
		"tokenCount":    tokenCount,
		"model":         model,
		"provider":      provider,
		"createdAt":     newMsg.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
	})
}

// ---------------------------------------------------------------------------
// Chat Branching: 从某条消息创建分支 (新时间线)
// POST /api/chat/messages/:id/branch
// ---------------------------------------------------------------------------
type BranchRequest struct {
	Title *string `json:"title"`
}

func (h *Handlers) branchFromMessage(c *gin.Context) {
	user := auth.CurrentUser(c)
	if user == nil {
		response.Fail(c, i18n.MsgUnauthorized, nil)
		return
	}

	messageID := c.Param("id")
	if messageID == "" {
		response.Fail(c, i18n.MsgInvalidParams, nil)
		return
	}

	var anchorMsg svcmodels.ChatMessage
	if err := h.db.Where("id = ?", messageID).First(&anchorMsg).Error; err != nil {
		response.Fail(c, i18n.MsgNotFound, nil)
		return
	}

	// verify ownership
	var session svcmodels.ChatSession
	if err := h.db.Where("id = ? AND user_id = ?", anchorMsg.SessionID, fmt.Sprintf("%d", user.ID)).First(&session).Error; err != nil {
		response.Fail(c, i18n.MsgForbidden, nil)
		return
	}

	// parse request
	var req BranchRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		req = BranchRequest{}
	}

	// create new session as branch
	branchSessionID := utils.SnowflakeUtil.GenID()
	title := fmt.Sprintf("Branch: %s", anchorMsg.Content)
	if req.Title != nil && *req.Title != "" {
		title = *req.Title
	}
	if len(title) > 100 {
		title = title[:100]
	}

	branchSession := svcmodels.ChatSession{
		ID:           branchSessionID,
		UserID:       fmt.Sprintf("%d", user.ID),
		AgentID:      session.AgentID,
		Title:        title,
		Provider:     session.Provider,
		Model:        session.Model,
		SystemPrompt: session.SystemPrompt,
		Status:       "active",
	}
	if err := h.db.Create(&branchSession).Error; err != nil {
		response.Fail(c, i18n.MsgChatBranchFailed, nil)
		return
	}

	// copy all ancestor messages (up to the anchor) into the new session
	ancestors, err := buildAncestorMessages(h.db, anchorMsg.SessionID, messageID)
	if err != nil {
		log.Printf("Failed to build ancestor messages: %v", err)
		// non-fatal: branch still works without history copy
	}

	branchMessages := make([]svcmodels.ChatMessage, 0, len(ancestors))
	for _, m := range ancestors {
		copyMsg := svcmodels.ChatMessage{
			ID:              utils.SnowflakeUtil.GenID(),
			SessionID:       branchSessionID,
			Role:            m.Role,
			Content:         m.Content,
			TokenCount:      m.TokenCount,
			Model:           m.Model,
			Provider:        m.Provider,
			RequestID:       m.RequestID,
			ParentMessageID: nil, // rebuilt in new session
			BranchIndex:     0,
			IsActiveBranch:  true,
			CreatedAt:       m.CreatedAt,
			UpdatedAt:       m.UpdatedAt,
		}
		branchMessages = append(branchMessages, copyMsg)
	}
	if len(branchMessages) > 0 {
		if err := h.db.Create(&branchMessages).Error; err != nil {
			log.Printf("Failed to copy ancestor messages to branch: %v", err)
		}
	}

	response.Success(c, i18n.MsgChatBranchCreated, gin.H{
		"branchSessionId": branchSessionID,
		"title":           title,
		"anchorMessageId": messageID,
	})
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

// buildActiveConversationContext returns the active-branch message chain up to (and including) the parentID.
// sysPrompt allows the caller to inject enriched system prompt (WorldInfo + Persona) ahead of time.
func buildActiveConversationContext(db *gorm.DB, sessionID, parentID, sysPrompt string) ([]llm.ChatMessage, error) {
	// get all messages in session ordered by time
	var allMessages []svcmodels.ChatMessage
	if err := db.Where("session_id = ?", sessionID).Order("created_at ASC").Find(&allMessages).Error; err != nil {
		return nil, err
	}

	// build tree: only include active branches
	activeMap := make(map[string]bool)
	for _, m := range allMessages {
		if m.Role == "assistant" && m.IsActiveBranch {
			if m.ParentMessageID != nil {
				activeMap[*m.ParentMessageID] = true // mark the parent's active child
			}
		}
	}

	result := make([]llm.ChatMessage, 0, len(allMessages))
	stopped := false
	for _, m := range allMessages {
		if stopped {
			break
		}
		// For user messages: include if it's the parent or before it
		// For assistant messages: include only active branch
		if m.Role == "user" {
			result = append(result, llm.ChatMessage{Role: "user", Content: m.Content})
			if m.ID == parentID {
				// include the parent user message but stop before adding children
				// (we're generating a new child)
				continue
			}
		} else if m.Role == "assistant" {
			if m.ParentMessageID != nil && activeMap[*m.ParentMessageID] {
				result = append(result, llm.ChatMessage{Role: "assistant", Content: m.Content})
			}
		}

		// stop at parent
		if m.ID == parentID {
			break
		}
	}

	// Prepend system prompt if provided (caller has already injected WorldInfo/Persona)
	if strings.TrimSpace(sysPrompt) != "" {
		sysMsg := llm.ChatMessage{Role: "system", Content: sysPrompt}
		result = append([]llm.ChatMessage{sysMsg}, result...)
	} else {
		// fallback: read from session
		var session svcmodels.ChatSession
		if err := db.Where("id = ?", sessionID).First(&session).Error; err == nil && session.SystemPrompt != "" {
			sysMsg := llm.ChatMessage{Role: "system", Content: session.SystemPrompt}
			result = append([]llm.ChatMessage{sysMsg}, result...)
		}
	}

	return result, nil
}

// buildAncestorMessages returns all messages from the beginning up to (and including) anchorID
func buildAncestorMessages(db *gorm.DB, sessionID, anchorID string) ([]svcmodels.ChatMessage, error) {
	var messages []svcmodels.ChatMessage
	if err := db.Where("session_id = ?", sessionID).Order("created_at ASC").Find(&messages).Error; err != nil {
		return nil, err
	}

	result := make([]svcmodels.ChatMessage, 0, len(messages))
	for _, m := range messages {
		result = append(result, m)
		if m.ID == anchorID {
			break
		}
	}
	return result, nil
}

// getLLMHandler creates an LLM handler based on provider credentials
func getLLMHandler(ctx *gin.Context, apiKey, apiSecret, provider string, opts *llm.QueryOptions) (llm.LLMHandler, error) {
	systemPrompt := ""
	// Find system prompt in messages
	for _, m := range opts.Messages {
		if m.Role == "system" {
			systemPrompt = m.Content
			break
		}
	}

	llmOpts := &llm.LLMOptions{
		Provider:     provider,
		ApiKey:       apiKey,
		SystemPrompt: systemPrompt,
	}

	handler, err := llm.NewProviderHandler(ctx.Request.Context(), provider, llmOpts)
	if err != nil {
		return nil, err
	}

	return handler, nil
}
