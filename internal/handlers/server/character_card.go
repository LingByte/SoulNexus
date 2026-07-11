package server

// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

import (
	"bytes"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"hash/crc32"
	"image/png"
	"io"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/LingByte/SoulNexus/internal/models/auth"
	svcmodels "github.com/LingByte/SoulNexus/internal/models/server"
	"github.com/LingByte/SoulNexus/pkg/response"
	"github.com/LingByte/SoulNexus/pkg/stores"
	"github.com/LingByte/SoulNexus/pkg/utils"
	"github.com/gin-gonic/gin"
	"gopkg.in/yaml.v3"
)

// ==================== Character Card Spec ====================

// CharacterCardV2 represents SillyTavern Character Card V2 JSON spec.
type CharacterCardV2 struct {
	Spec        string `json:"spec" yaml:"spec"`
	SpecVersion string `json:"spec_version" yaml:"spec_version"`
	Data        struct {
		Name             string `json:"name" yaml:"name"`
		Description      string `json:"description" yaml:"description"`
		Personality      string `json:"personality" yaml:"personality"`
		Scenario         string `json:"scenario" yaml:"scenario"`
		FirstMes         string `json:"first_mes" yaml:"first_mes"`
		MesExample       string `json:"mes_example" yaml:"mes_example"`
		CreatorNotes     string `json:"creator_notes" yaml:"creator_notes"`
		SystemPrompt     string `json:"system_prompt" yaml:"system_prompt"`
		PostHistoryInst  string `json:"post_history_instructions" yaml:"post_history_instructions"`
		Tags             []string `json:"tags" yaml:"tags"`
		Creator          string `json:"creator" yaml:"creator"`
		CharacterVersion string `json:"character_version" yaml:"character_version"`
		AlternateGreetings []string `json:"alternate_greetings" yaml:"alternate_greetings"`
	} `json:"data" yaml:"data"`
}

// SoulNexusCard is our own export format (JSON).
type SoulNexusCard struct {
	Spec        string          `json:"spec"`
	SpecVersion string          `json:"spec_version"`
	Agent       json.RawMessage `json:"agent"`
}

// SoulNexusImportPayload is the input for multi-format import.
type SoulNexusImportPayload struct {
	Format      string `json:"format"`      // "json" | "png" | "yaml"
	Data        string `json:"data"`        // base64 encoded content (for PNG) or raw JSON/YAML string
	GroupID     *uint  `json:"groupId"`     // target org (nil = personal)
	OverrideName string `json:"overrideName"`
}

// exportPayload is the input for export format selection.
type exportPayload struct {
	Format string `json:"format"` // "json" | "png"
}

const (
	characterCardSpecV2 = "chara_card_v2"
	characterCardSpecV3 = "chara_card_v3"
	soulNexusCardSpec   = "soulnx_card_v1"

	pngChunkType = "chara" // PNG tEXt chunk keyword for character data
)

// ==================== Import Character Card ====================

// handleImportCharacterCard imports a character card from JSON/PNG/YAML format.
func (h *Handlers) handleImportCharacterCard(c *gin.Context) {
	user := auth.CurrentUser(c)
	if user == nil {
		response.Fail(c, "unauthorized", "User not logged in")
		return
	}

	// Support both multipart file upload and JSON body
	contentType := c.ContentType()
	var rawData []byte
	var detectedFormat string

	if strings.Contains(contentType, "multipart/form-data") {
		// File upload mode
		file, header, err := c.Request.FormFile("file")
		if err != nil {
			response.Fail(c, "missing file", err.Error())
			return
		}
		defer file.Close()

		rawData, err = io.ReadAll(file)
		if err != nil {
			response.Fail(c, "failed to read file", err.Error())
			return
		}

		ext := strings.ToLower(filepath.Ext(header.Filename))
		switch ext {
		case ".png":
			detectedFormat = "png"
		case ".json":
			detectedFormat = "json"
		case ".yaml", ".yml":
			detectedFormat = "yaml"
		default:
			// Try to detect by content
			detectedFormat = detectFormat(rawData)
		}
	} else {
		// JSON body mode
		var payload SoulNexusImportPayload
		if err := c.ShouldBindJSON(&payload); err != nil {
			response.Fail(c, "invalid payload", err.Error())
			return
		}
		detectedFormat = payload.Format
		if payload.Data != "" {
			// Try base64 decode first
			decoded, err := base64.StdEncoding.DecodeString(payload.Data)
			if err != nil {
				rawData = []byte(payload.Data)
			} else {
				rawData = decoded
			}
		}
	}

	if len(rawData) == 0 {
		response.Fail(c, "empty data", "No data provided for import")
		return
	}

	// Get target group ID
	gid, err := svcmodels.ResolveWriteGroupID(h.db, user.ID, nil)
	if err != nil {
		response.Fail(c, "failed to resolve group", err.Error())
		return
	}

	// Parse and convert to agent
	agent, err := parseCharacterCard(rawData, detectedFormat)
	if err != nil {
		response.Fail(c, "parse failed", err.Error())
		return
	}

	// Check if we should use form body for group override
	overrideGroupID := c.PostForm("groupId")
	if overrideGroupID != "" {
		if parsed, err := strconv.ParseUint(overrideGroupID, 10, 64); err == nil {
			g := uint(parsed)
			gid, err = svcmodels.ResolveWriteGroupID(h.db, user.ID, &g)
			if err != nil {
				response.Fail(c, "invalid group", err.Error())
				return
			}
		}
	}

	overrideName := c.PostForm("overrideName")
	if overrideName != "" {
		agent.Name = overrideName
	}

	// Fill required fields
	agent.GroupID = gid
	agent.CreatedBy = user.ID
	agent.CreatedAt = time.Now()
	agent.UpdatedAt = time.Now()
	agent.JsSourceID = utils.SnowflakeUtil.GenID()
	if agent.SystemPrompt == "" {
		agent.SystemPrompt = "empty system prompt"
	}
	if agent.PersonaTag == "" {
		agent.PersonaTag = "assistant"
	}
	if agent.Temperature == 0 {
		agent.Temperature = 0.6
	}
	if agent.MaxTokens == 0 {
		agent.MaxTokens = 150
	}
	if agent.Speaker == "" {
		agent.Speaker = "101016"
	}
	if agent.SpecVersion == "" {
		agent.SpecVersion = detectedFormat
	}

	if err = h.db.Create(&agent).Error; err != nil {
		response.Fail(c, "failed to create agent from character card", err.Error())
		return
	}

	utils.Sig().Emit("agent:imported", user, h.db, &agent)
	response.Success(c, "Character card imported successfully", agent)
}

// detectFormat detects the format of raw data.
func detectFormat(data []byte) string {
	// Check for PNG header
	if len(data) >= 8 && data[0] == 0x89 && data[1] == 0x50 && data[2] == 0x4E && data[3] == 0x47 {
		return "png"
	}
	// Try JSON
	if strings.TrimSpace(string(data))[0] == '{' {
		return "json"
	}
	// Default to YAML
	return "yaml"
}

// parseCharacterCard parses raw data from any supported format and returns an Agent.
func parseCharacterCard(data []byte, format string) (svcmodels.Agent, error) {
	switch strings.ToLower(format) {
	case "png":
		return parsePNGCharacterCard(data)
	case "json":
		return parseJSONCharacterCard(data)
	case "yaml", "yml":
		return parseYAMLCharacterCard(data)
	default:
		return svcmodels.Agent{}, fmt.Errorf("unsupported format: %s", format)
	}
}

// parsePNGCharacterCard extracts character card JSON from PNG tEXt chunks.
func parsePNGCharacterCard(data []byte) (svcmodels.Agent, error) {
	img, err := png.Decode(bytes.NewReader(data))
	if err != nil {
		return svcmodels.Agent{}, fmt.Errorf("invalid PNG: %w", err)
	}
	_ = img // Use the image for verification

	// Re-read the PNG to extract tEXt chunks manually
	reader := bytes.NewReader(data)

	// Skip PNG signature (8 bytes)
	sig := make([]byte, 8)
	if _, err := io.ReadFull(reader, sig); err != nil {
		return svcmodels.Agent{}, fmt.Errorf("failed to read PNG signature: %w", err)
	}
	if sig[0] != 0x89 || sig[1] != 0x50 || sig[2] != 0x4E || sig[3] != 0x47 {
		return svcmodels.Agent{}, errors.New("not a valid PNG file")
	}

	var charaJSON string
	for {
		// Read chunk length (4 bytes, big-endian)
		var length uint32
		if err := binary.Read(reader, binary.BigEndian, &length); err != nil {
			break
		}

		// Read chunk type (4 bytes)
		chunkType := make([]byte, 4)
		if _, err := io.ReadFull(reader, chunkType); err != nil {
			break
		}

		// Read chunk data
		chunkData := make([]byte, length)
		if _, err := io.ReadFull(reader, chunkData); err != nil {
			break
		}

		// Skip CRC (4 bytes)
		crcBuf := make([]byte, 4)
		if _, err := io.ReadFull(reader, crcBuf); err != nil {
			break
		}

		// Check for tEXt chunk with keyword "chara"
		if string(chunkType) == "tEXt" {
			// tEXt chunk format: keyword\0text
			nullIdx := bytes.IndexByte(chunkData, 0)
			if nullIdx > 0 {
				keyword := string(chunkData[:nullIdx])
				text := string(chunkData[nullIdx+1:])
				if keyword == pngChunkType || keyword == "ccv3" {
					charaJSON = text
					break
				}
			}
		}

		// Check for IEND (end of image)
		if string(chunkType) == "IEND" {
			break
		}
	}

	if charaJSON == "" {
		return svcmodels.Agent{}, errors.New("no character card data found in PNG (missing 'chara' tEXt chunk)")
	}

	// Try base64 decode first (SillyTavern convention)
	decoded, err := base64.StdEncoding.DecodeString(charaJSON)
	if err != nil {
		decoded = []byte(charaJSON)
	}

	return parseJSONCharacterCard(decoded)
}

// parseJSONCharacterCard parses JSON character card (V2 or SoulNexus format).
func parseJSONCharacterCard(data []byte) (svcmodels.Agent, error) {
	// Try SoulNexus format first
	var snxCard SoulNexusCard
	if err := json.Unmarshal(data, &snxCard); err == nil {
		if snxCard.Spec == soulNexusCardSpec {
			var agent svcmodels.Agent
			if err := json.Unmarshal(snxCard.Agent, &agent); err == nil {
				return agent, nil
			}
		}
	}

	// Try Character Card V2
	var cc CharacterCardV2
	if err := json.Unmarshal(data, &cc); err != nil {
		return svcmodels.Agent{}, fmt.Errorf("invalid JSON character card: %w", err)
	}

	return convertCharacterCardV2ToAgent(&cc), nil
}

// parseYAMLCharacterCard parses YAML character card.
func parseYAMLCharacterCard(data []byte) (svcmodels.Agent, error) {
	var cc CharacterCardV2
	if err := yaml.Unmarshal(data, &cc); err != nil {
		return svcmodels.Agent{}, fmt.Errorf("invalid YAML character card: %w", err)
	}
	return convertCharacterCardV2ToAgent(&cc), nil
}

// convertCharacterCardV2ToAgent converts a Character Card V2 to Agent model.
func convertCharacterCardV2ToAgent(cc *CharacterCardV2) svcmodels.Agent {
	agent := svcmodels.Agent{
		Name:             cc.Data.Name,
		Description:      cc.Data.Description,
		Personality:      cc.Data.Personality,
		Scenario:         cc.Data.Scenario,
		SystemPrompt:     cc.Data.SystemPrompt,
		OpeningStatement: cc.Data.FirstMes,
		CreatorNote:      cc.Data.CreatorNotes,
		SpecVersion:      cc.SpecVersion,
	}

	if agent.SystemPrompt == "" {
		agent.SystemPrompt = cc.Data.Personality
	}

	// Convert example dialogues to JSON
	if cc.Data.MesExample != "" {
		// Simple conversion: wrap in an array of {role, content}
		dialogues := []map[string]string{
			{"role": "example", "content": cc.Data.MesExample},
		}
		if b, err := json.Marshal(dialogues); err == nil {
			agent.ExampleDialogues = string(b)
		}
	}

	// Join tags
	if len(cc.Data.Tags) > 0 {
		agent.Tags = strings.Join(cc.Data.Tags, ",")
	}

	// Set visibility default
	agent.Visibility = "group"

	return agent
}

// ==================== Export Character Card ====================

// handleExportCharacterCard exports an agent as a character card (JSON or PNG).
func (h *Handlers) handleExportCharacterCard(c *gin.Context) {
	user := auth.CurrentUser(c)
	if user == nil {
		response.Fail(c, "unauthorized", "User not logged in")
		return
	}

	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		response.Fail(c, "invalid id", "Invalid agent ID")
		return
	}

	var agent svcmodels.Agent
	if err = h.db.First(&agent, id).Error; err != nil {
		response.Fail(c, "agent not found", err.Error())
		return
	}

	// Check access
	if !svcmodels.UserIsGroupMember(h.db, user.ID, agent.GroupID) {
		response.Fail(c, "permission denied", "You do not have access to this agent")
		return
	}

	format := strings.ToLower(c.Query("format"))
	if format == "" {
		// Check JSON body for format
		var payload exportPayload
		if c.ShouldBindJSON(&payload) == nil && payload.Format != "" {
			format = payload.Format
		}
	}
	if format == "" {
		format = "json" // default
	}

	switch format {
	case "png":
		exportPNGCharacterCard(c, &agent)
	case "json":
		exportJSONCharacterCard(c, &agent)
	default:
		response.Fail(c, "unsupported format", "Supported formats: json, png")
	}
}

// exportJSONCharacterCard exports agent as SoulNexus JSON format.
func exportJSONCharacterCard(c *gin.Context, agent *svcmodels.Agent) {
	agentJSON, err := json.Marshal(agent)
	if err != nil {
		response.Fail(c, "serialize failed", err.Error())
		return
	}

	card := SoulNexusCard{
		Spec:        soulNexusCardSpec,
		SpecVersion: "v1",
		Agent:       agentJSON,
	}

	output, err := json.MarshalIndent(card, "", "  ")
	if err != nil {
		response.Fail(c, "serialize failed", err.Error())
		return
	}

	c.Header("Content-Type", "application/json; charset=utf-8")
	c.Header("Content-Disposition", fmt.Sprintf(`attachment; filename="%s.soulnx.json"`, safeFilename(agent.Name)))
	c.Data(http.StatusOK, "application/json; charset=utf-8", output)
}

// exportPNGCharacterCard exports agent as PNG with embedded character card.
func exportPNGCharacterCard(c *gin.Context, agent *svcmodels.Agent) {
	// Build Character Card V2 JSON
	cc := agentToCharacterCardV2(agent)
	ccJSON, err := json.Marshal(cc)
	if err != nil {
		response.Fail(c, "serialize failed", err.Error())
		return
	}

	// Base64 encode (SillyTavern convention)
	encoded := base64.StdEncoding.EncodeToString(ccJSON)

	// Create or load avatar image
	var imgData []byte
	if agent.AvatarURL != "" {
		st := stores.Default()
		// Try to read avatar from store
		if key := extractStoreKey(agent.AvatarURL); key != "" {
			rc, _, readErr := st.Read(key)
			if readErr == nil {
				imgData, _ = io.ReadAll(rc)
				rc.Close()
			}
		}
	}

	if len(imgData) == 0 {
		// Generate a simple placeholder PNG (100x100 blue square)
		imgData = generatePlaceholderPNG()
	}

	// Embed character card into PNG
	pngData, err := embedTextChunk(imgData, pngChunkType, encoded)
	if err != nil {
		response.Fail(c, "embed failed", err.Error())
		return
	}

	c.Header("Content-Type", "image/png")
	c.Header("Content-Disposition", fmt.Sprintf(`attachment; filename="%s.png"`, safeFilename(agent.Name)))
	c.Data(http.StatusOK, "image/png", pngData)
}

// agentToCharacterCardV2 converts Agent to CharacterCardV2.
func agentToCharacterCardV2(agent *svcmodels.Agent) CharacterCardV2 {
	cc := CharacterCardV2{
		Spec:        characterCardSpecV2,
		SpecVersion: "v2",
	}
	cc.Data.Name = agent.Name
	cc.Data.Description = agent.Description
	cc.Data.Personality = agent.Personality
	cc.Data.Scenario = agent.Scenario
	cc.Data.FirstMes = agent.OpeningStatement
	cc.Data.SystemPrompt = agent.SystemPrompt
	cc.Data.CreatorNotes = agent.CreatorNote

	if agent.Tags != "" {
		cc.Data.Tags = strings.Split(agent.Tags, ",")
		for i := range cc.Data.Tags {
			cc.Data.Tags[i] = strings.TrimSpace(cc.Data.Tags[i])
		}
	}

	// Parse example dialogues
	if agent.ExampleDialogues != "" {
		var dialogues []map[string]string
		if err := json.Unmarshal([]byte(agent.ExampleDialogues), &dialogues); err == nil && len(dialogues) > 0 {
			if content, ok := dialogues[0]["content"]; ok {
				cc.Data.MesExample = content
			}
		}
	}

	return cc
}

// ==================== Avatar Upload ====================

// handleUploadAgentAvatar handles agent avatar image upload.
func (h *Handlers) handleUploadAgentAvatar(c *gin.Context) {
	user := auth.CurrentUser(c)
	if user == nil {
		response.Fail(c, "unauthorized", "User not logged in")
		return
	}

	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		response.Fail(c, "invalid id", "Invalid agent ID")
		return
	}

	var agent svcmodels.Agent
	if err = h.db.First(&agent, id).Error; err != nil {
		response.Fail(c, "agent not found", err.Error())
		return
	}

	if !svcmodels.CanManageTenantResource(h.db, user.ID, agent.GroupID, agent.CreatedBy) {
		response.Fail(c, "permission denied", "No permission to update this agent")
		return
	}

	file, header, err := c.Request.FormFile("avatar")
	if err != nil {
		response.Fail(c, "missing file", err.Error())
		return
	}
	defer file.Close()

	// Validate file type
	ext := strings.ToLower(filepath.Ext(header.Filename))
	allowedExts := map[string]bool{".png": true, ".jpg": true, ".jpeg": true, ".gif": true, ".webp": true}
	if !allowedExts[ext] {
		response.Fail(c, "invalid file type", "Only PNG/JPG/GIF/WEBP images are allowed")
		return
	}

	// Validate file size (max 5MB)
	var buf bytes.Buffer
	tee := io.TeeReader(file, &buf)
	limited := io.LimitReader(tee, 5*1024*1024)
	if _, err = io.ReadAll(limited); err != nil {
		response.Fail(c, "file too large", "Maximum file size is 5MB")
		return
	}

	// Generate storage key
	fileName := fmt.Sprintf("agent-avatars/%d_%d%s", agent.ID, time.Now().Unix(), ext)

	st := stores.Default()
	if err = st.Write(fileName, &buf); err != nil {
		response.Fail(c, "upload failed", err.Error())
		return
	}

	avatarURL := st.PublicURL(fileName)

	// Update agent avatar
	if err = h.db.Model(&agent).Update("avatar_url", avatarURL).Error; err != nil {
		response.Fail(c, "update agent failed", err.Error())
		return
	}

	response.Success(c, "Avatar uploaded successfully", gin.H{
		"avatarUrl": avatarURL,
		"agentId":   agent.ID,
	})
}

// ==================== Market APIs ====================

// handleMarketListAgents lists public agents on the market.
func (h *Handlers) handleMarketListAgents(c *gin.Context) {
	page, pageSize := utils.ParsePagination(c)
	search := strings.TrimSpace(c.Query("search"))
	sortBy := strings.TrimSpace(c.Query("sortBy")) // download_count, rating, created_at
	if sortBy == "" {
		sortBy = "download_count"
	}

	query := h.db.Model(&svcmodels.Agent{}).Where("visibility = ?", "public")

	if search != "" {
		like := "%" + search + "%"
		query = query.Where("name LIKE ? OR description LIKE ? OR tags LIKE ? OR personality LIKE ?", like, like, like, like)
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		response.Fail(c, "list market agents failed", err.Error())
		return
	}

	orderClause := "download_count DESC"
	switch sortBy {
	case "rating":
		orderClause = "rating DESC"
	case "created_at", "newest":
		orderClause = "created_at DESC"
	case "download_count":
		orderClause = "download_count DESC"
	}

	var agents []svcmodels.Agent
	if err := query.Order(orderClause).Offset((page - 1) * pageSize).Limit(pageSize).Find(&agents).Error; err != nil {
		response.Fail(c, "list market agents failed", err.Error())
		return
	}

	response.Success(c, "market agents fetched", gin.H{
		"agents":   agents,
		"total":    total,
		"page":     page,
		"pageSize": pageSize,
	})
}

// handleMarketGetAgent gets a public agent detail (increment download count).
func (h *Handlers) handleMarketGetAgent(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		response.Fail(c, "invalid id", "Invalid agent ID")
		return
	}

	var agent svcmodels.Agent
	if err = h.db.First(&agent, id).Error; err != nil {
		response.Fail(c, "agent not found", err.Error())
		return
	}

	if agent.Visibility != "public" {
		response.Fail(c, "not available", "This agent is not public")
		return
	}

	response.Success(c, "agent fetched", agent)
}

// handleMarketForkAgent forks a public agent to the user's own group.
func (h *Handlers) handleMarketForkAgent(c *gin.Context) {
	user := auth.CurrentUser(c)
	if user == nil {
		response.Fail(c, "unauthorized", "User not logged in")
		return
	}

	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		response.Fail(c, "invalid id", "Invalid agent ID")
		return
	}

	var source svcmodels.Agent
	if err = h.db.First(&source, id).Error; err != nil {
		response.Fail(c, "agent not found", err.Error())
		return
	}

	if source.Visibility != "public" {
		response.Fail(c, "not available", "This agent is not public")
		return
	}

	// Resolve target group
	var input struct {
		GroupID *uint `json:"groupId"`
	}
	c.ShouldBindJSON(&input)

	gid, err := svcmodels.ResolveWriteGroupID(h.db, user.ID, input.GroupID)
	if err != nil {
		response.Fail(c, "failed to resolve group", err.Error())
		return
	}

	// Clone the agent
	forked := source
	forked.ID = 0 // Let DB auto-increment
	forked.GroupID = gid
	forked.CreatedBy = user.ID
	forked.ForkedFrom = &source.ID
	forked.Visibility = "group"
	forked.DownloadCount = 0
	forked.Rating = 0
	forked.RatingCount = 0
	forked.JsSourceID = utils.SnowflakeUtil.GenID()
	forked.Name = source.Name + " (Fork)"
	forked.CreatedAt = time.Now()
	forked.UpdatedAt = time.Now()

	if err = h.db.Create(&forked).Error; err != nil {
		response.Fail(c, "fork failed", err.Error())
		return
	}

	// Increment download count on source
	h.db.Model(&source).UpdateColumn("download_count", source.DownloadCount+1)

	response.Success(c, "Agent forked successfully", forked)
}

// handleMarketRateAgent rates a public agent.
func (h *Handlers) handleMarketRateAgent(c *gin.Context) {
	user := auth.CurrentUser(c)
	if user == nil {
		response.Fail(c, "unauthorized", "User not logged in")
		return
	}

	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		response.Fail(c, "invalid id", "Invalid agent ID")
		return
	}

	var input struct {
		Score float64 `json:"score" binding:"required"`
	}
	if err = c.ShouldBindJSON(&input); err != nil {
		response.Fail(c, "invalid score", err.Error())
		return
	}

	if input.Score < 0 || input.Score > 5 {
		response.Fail(c, "invalid score", "Score must be between 0 and 5")
		return
	}

	var agent svcmodels.Agent
	if err = h.db.First(&agent, id).Error; err != nil {
		response.Fail(c, "agent not found", err.Error())
		return
	}

	if agent.Visibility != "public" {
		response.Fail(c, "not available", "This agent is not public")
		return
	}

	// Simple average rating update (no per-user tracking for now)
	newCount := agent.RatingCount + 1
	newRating := (agent.Rating*float64(agent.RatingCount) + input.Score) / float64(newCount)

	if err = h.db.Model(&agent).Updates(map[string]any{
		"rating":       newRating,
		"rating_count": newCount,
	}).Error; err != nil {
		response.Fail(c, "rate failed", err.Error())
		return
	}

	response.Success(c, "Rated successfully", gin.H{
		"rating":      newRating,
		"ratingCount": newCount,
	})
}

// handleMarketShareAgent generates a share link + preview metadata for a public agent.
// GET /api/market/agents/:id/share
func (h *Handlers) handleMarketShareAgent(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		response.Fail(c, "invalid id", "Invalid agent ID")
		return
	}

	var agent svcmodels.Agent
	if err = h.db.First(&agent, id).Error; err != nil {
		response.Fail(c, "agent not found", err.Error())
		return
	}

	if agent.Visibility != "public" {
		response.Fail(c, "not available", "This agent is not public")
		return
	}

	// Build share URL (frontend handles routing)
	baseURL := c.GetHeader("X-Forwarded-Host")
	if baseURL == "" {
		baseURL = c.Request.Host
	}
	scheme := "https"
	if c.Request.TLS == nil {
		scheme = "http"
	}
	shareURL := fmt.Sprintf("%s://%s/market?open=%d", scheme, baseURL, id)

	response.Success(c, "share link generated", gin.H{
		"url":          shareURL,
		"title":        agent.Name,
		"description":  truncateStr(agent.Description, 200),
		"avatar":       agent.AvatarURL,
		"rating":       agent.Rating,
		"ratingCount":  agent.RatingCount,
		"downloadCount": agent.DownloadCount,
		"agentId":      agent.ID,
	})
}

func truncateStr(s string, maxLen int) string {
	runes := []rune(s)
	if len(runes) <= maxLen {
		return s
	}
	return string(runes[:maxLen]) + "..."
}

// ==================== PNG Embed / Extract Utilities ====================

// embedTextChunk embeds a tEXt chunk into a PNG image.
func embedTextChunk(pngData []byte, keyword, text string) ([]byte, error) {
	// Validate PNG
	if len(pngData) < 8 || pngData[0] != 0x89 || pngData[1] != 0x50 || pngData[2] != 0x4E || pngData[3] != 0x47 {
		return nil, errors.New("not a valid PNG file")
	}

	var buf bytes.Buffer

	// Write PNG signature
	buf.Write(pngData[:8])

	reader := bytes.NewReader(pngData[8:])

	for {
		// Read chunk length
		var length uint32
		if err := binary.Read(reader, binary.BigEndian, &length); err != nil {
			break
		}

		// Read chunk type
		chunkType := make([]byte, 4)
		if _, err := io.ReadFull(reader, chunkType); err != nil {
			break
		}

		// Read chunk data
		chunkData := make([]byte, length)
		if _, err := io.ReadFull(reader, chunkData); err != nil {
			break
		}

		// Read CRC
		crcBuf := make([]byte, 4)
		if _, err := io.ReadFull(reader, crcBuf); err != nil {
			break
		}

		// Insert tEXt chunk before IDAT
		if string(chunkType) == "IDAT" {
			// Build tEXt chunk: keyword\0text
			tEXtData := append([]byte(keyword), 0)
			tEXtData = append(tEXtData, []byte(text)...)

			// Write tEXt chunk
			if err := binary.Write(&buf, binary.BigEndian, uint32(len(tEXtData))); err != nil {
				return nil, err
			}
			buf.Write([]byte("tEXt"))
			buf.Write(tEXtData)

			// CRC for tEXt
			crc := crc32.NewIEEE()
			crc.Write([]byte("tEXt"))
			crc.Write(tEXtData)
			if err := binary.Write(&buf, binary.BigEndian, crc.Sum32()); err != nil {
				return nil, err
			}
		}

		// Write original chunk
		if err := binary.Write(&buf, binary.BigEndian, length); err != nil {
			return nil, err
		}
		buf.Write(chunkType)
		buf.Write(chunkData)
		buf.Write(crcBuf)
	}

	return buf.Bytes(), nil
}

// generatePlaceholderPNG generates a simple 100x100 PNG placeholder image.
func generatePlaceholderPNG() []byte {
	// Create a minimal 1x1 PNG pixel
	var buf bytes.Buffer
	// 1x1 blue pixel
	imgData := []byte{
		137, 80, 78, 71, 13, 10, 26, 10, // PNG signature
		0, 0, 0, 13, // IHDR length
		73, 72, 68, 82, // IHDR
		0, 0, 0, 1, // width = 1
		0, 0, 0, 1, // height = 1
		8, 2, 0, 0, 0, // bit depth=8, color type=2 (RGB)
		144, 119, 83, 222, // IHDR CRC
		0, 0, 0, 12, // IDAT length
		73, 68, 65, 84, // IDAT
		120, 156, 98, 248, 207, 192, 0, 0, 6, 16, 2, 89, // compressed data
		127, 99, 28, 12, // IDAT CRC
		0, 0, 0, 0, // IEND length
		73, 69, 78, 68, // IEND
		174, 66, 96, 130, // IEND CRC
	}
	buf.Write(imgData)
	return buf.Bytes()
}

// safeFilename sanitizes a string for use as a filename.
func safeFilename(name string) string {
	// Replace problematic characters
	replacer := strings.NewReplacer(
		"/", "_", "\\", "_", ":", "_", "*", "_",
		"?", "_", "\"", "_", "<", "_", ">", "_", "|", "_",
		" ", "_",
	)
	safe := replacer.Replace(name)
	if safe == "" {
		safe = "character"
	}
	if len(safe) > 100 {
		safe = safe[:100]
	}
	return safe
}

// extractStoreKey extracts the storage key from a URL.
func extractStoreKey(url string) string {
	// Handle common patterns like /media/xxx or full URLs
	if idx := strings.Index(url, "/media/"); idx >= 0 {
		return url[idx+len("/media/"):]
	}
	if idx := strings.Index(url, "agent-avatars/"); idx >= 0 {
		return url[idx:]
	}
	return ""
}

// extractStoreKey extracts the storage key from a URL.
