package handlers

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"text/template"
	"time"

	"github.com/LingByte/SoulNexus"
	"github.com/LingByte/SoulNexus/internal/models"
	"github.com/LingByte/SoulNexus/pkg/config"
	"github.com/LingByte/SoulNexus/pkg/constants"
	"github.com/LingByte/SoulNexus/pkg/graph"
	"github.com/LingByte/SoulNexus/pkg/logger"
	"github.com/LingByte/SoulNexus/pkg/response"
	"github.com/LingByte/SoulNexus/pkg/utils"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// hashString 计算字符串的哈希值（用于灰度发布）
func hashString(s string) int {
	hash := sha256.Sum256([]byte(s))
	hashStr := hex.EncodeToString(hash[:])
	// 取前8个字符转换为整数
	val, _ := strconv.ParseInt(hashStr[:8], 16, 64)
	return int(val % 100)
}

// CreateAssistant create new assistant
func (h *Handlers) CreateAssistant(c *gin.Context) {
	var input struct {
		Name        string `json:"name" binding:"required"`
		Description string `json:"description"`
		Icon        string `json:"icon"`
		GroupID     *uint  `json:"groupId,omitempty"` // Organization ID, if set, creates a shared assistant for the organization
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		response.Fail(c, "Parameter error", nil)
		return
	}

	user := models.CurrentUser(c)

	// If an organization ID is specified, verify that the user has permission to create a shared assistant in that organization
	if input.GroupID != nil {
		var group models.Group
		if err := h.db.First(&group, *input.GroupID).Error; err != nil {
			response.Fail(c, "Organization does not exist", nil)
			return
		}
		// Check if the user is the creator or administrator of the organization
		if group.CreatorID != user.ID {
			var member models.GroupMember
			if err := h.db.Where("group_id = ? AND user_id = ? AND role = ?", *input.GroupID, user.ID, models.GroupRoleAdmin).First(&member).Error; err != nil {
				response.Fail(c, "Insufficient permissions", "Only creators or administrators can create organization-shared assistants")
				return
			}
		}
	}

	assistant := models.Assistant{
		UserID:       user.ID,
		GroupID:      input.GroupID,
		Name:         input.Name,
		Description:  input.Description,
		Icon:         input.Icon,
		SystemPrompt: "empty system prompt",
		PersonaTag:   "mentor",
		Temperature:  0.6,
		MaxTokens:    150,
		JsSourceID:   strconv.FormatInt(utils.SnowflakeUtil.NextID(), 20),
		Language:     "zh-cn",
		Speaker:      "101016",
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	if err := h.db.Create(&assistant).Error; err != nil {
		response.Fail(c, fmt.Sprintf("Failed to create assistant %s", assistant.Name), nil)
		return
	}
	utils.Sig().Emit(constants.AssistantCreate, user, h.db, assistant)
	response.Success(c, fmt.Sprintf("Successfully created assistant %s", assistant.Name), assistant)
}

// ListAssistants Query all assistants of the current user, including organization-shared assistants
func (h *Handlers) ListAssistants(c *gin.Context) {
	user := models.CurrentUser(c)
	if user == nil {
		response.Fail(c, "unauthorized", "User not logged in")
		return
	}
	var list []models.Assistant

	// Query user's own assistants and organization-shared assistants
	// 1. Assistants created by the user (user_id = ?)
	// 2. Organization-shared assistants (group_id IN (list of organization IDs the user belongs to))
	var groupIDs []uint
	h.db.Model(&models.GroupMember{}).
		Where("user_id = ?", user.ID).
		Pluck("group_id", &groupIDs)

	query := h.db.Model(&models.Assistant{})
	if len(groupIDs) > 0 {
		// User's own assistants OR organization-shared assistants
		query = query.Where("user_id = ? OR (group_id IN ? AND group_id IS NOT NULL)", user.ID, groupIDs)
	} else {
		// Only query user's own assistants
		query = query.Where("user_id = ?", user.ID)
	}

	if err := query.Order("created_at desc").Find(&list).Error; err != nil {
		response.Fail(c, "select assistants failed", nil)
		return
	}

	response.Success(c, "select assistants successful", list)
}

// GetAssistant Query a single assistant
func (h *Handlers) GetAssistant(c *gin.Context) {
	user := models.CurrentUser(c)
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	var assistant models.Assistant
	if err := h.db.First(&assistant, id).Error; err != nil {
		response.Fail(c, "not found", "this assistant is not exist")
		return
	}
	if user.ID != assistant.UserID {
		response.Fail(c, "permission denied", "you are not allowed to access this assistant")
		return
	}
	response.Success(c, "select assistant successful", assistant)
}

// UpdateAssistant Update assistant
func (h *Handlers) UpdateAssistant(c *gin.Context) {
	user := models.CurrentUser(c)
	if user == nil {
		response.Fail(c, "unauthorized", "User not logged in")
		return
	}

	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)

	var input struct {
		Name                 string   `json:"name"`
		Description          string   `json:"description"`
		Icon                 string   `json:"icon"`
		SystemPrompt         string   `json:"systemPrompt"`
		PersonaTag           string   `json:"persona_tag"`
		Temperature          float32  `json:"temperature"`
		MaxTokens            int      `json:"maxTokens"`
		Language             string   `json:"language"`
		Speaker              string   `json:"speaker"`
		VoiceCloneId         *int     `json:"voiceCloneId"`
		KnowledgeBaseId      *string  `json:"knowledgeBaseId"`
		TtsProvider          string   `json:"ttsProvider"`
		ApiKey               string   `json:"apiKey"`
		ApiSecret            string   `json:"apiSecret"`
		LLMModel             string   `json:"llmModel"` // LLM model name
		EnableGraphMemory    *bool    `json:"enableGraphMemory"`
		EnableVAD            *bool    `json:"enableVAD"`            // 是否启用VAD
		VADThreshold         *float64 `json:"vadThreshold"`         // VAD阈值
		VADConsecutiveFrames *int     `json:"vadConsecutiveFrames"` // VAD连续帧数
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		response.Fail(c, "invalid request", "parameter error")
		return
	}

	var assistant models.Assistant
	if err := h.db.First(&assistant, id).Error; err != nil {
		response.Fail(c, "not found", "Assistant does not exist.")
		return
	}

	if assistant.UserID != user.ID {
		response.Fail(c, "forbidden", "No permission to operate this assistant.")
		return
	}

	// Update fields
	updateData := map[string]interface{}{
		"updated_at": time.Now(),
	}

	// Only update non-empty fields
	if input.Name != "" {
		updateData["name"] = input.Name
	}
	if input.Description != "" {
		updateData["description"] = input.Description
	}
	if input.Icon != "" {
		updateData["icon"] = input.Icon
	}
	if input.SystemPrompt != "" {
		updateData["system_prompt"] = input.SystemPrompt
	}
	if input.PersonaTag != "" {
		updateData["persona_tag"] = input.PersonaTag
	}
	if input.Temperature != 0 {
		updateData["temperature"] = input.Temperature
	}
	if input.MaxTokens != 0 {
		updateData["max_tokens"] = input.MaxTokens
	}
	if input.Language != "" {
		updateData["language"] = input.Language
	}
	if input.Speaker != "" {
		updateData["speaker"] = input.Speaker
	}
	if input.VoiceCloneId != nil {
		updateData["voice_clone_id"] = input.VoiceCloneId
	}
	if input.KnowledgeBaseId != nil {
		updateData["knowledge_base_id"] = input.KnowledgeBaseId
	}
	if input.TtsProvider != "" {
		updateData["tts_provider"] = input.TtsProvider
	}
	if input.ApiKey != "" {
		updateData["api_key"] = input.ApiKey
	}
	if input.ApiSecret != "" {
		updateData["api_secret"] = input.ApiSecret
	}
	if input.LLMModel != "" {
		updateData["llm_model"] = input.LLMModel
	}
	if input.EnableGraphMemory != nil {
		updateData["enable_graph_memory"] = *input.EnableGraphMemory
	}
	if input.EnableVAD != nil {
		updateData["enable_vad"] = *input.EnableVAD
	}
	if input.VADThreshold != nil {
		updateData["vad_threshold"] = *input.VADThreshold
	}
	if input.VADConsecutiveFrames != nil {
		updateData["vad_consecutive_frames"] = *input.VADConsecutiveFrames
	}

	if err := h.db.Model(&assistant).Where("id = ?", id).Updates(updateData).Error; err != nil {
		response.Fail(c, "update failed", "Update failed")
		return
	}

	// Re-query the updated data
	if err := h.db.First(&assistant, id).Error; err != nil {
		response.Fail(c, "update failed", "Failed to query updated data")
		return
	}

	response.Success(c, "Update successful", assistant)
}

// UpdateAssistantJS Update assistant JS template
func (h *Handlers) UpdateAssistantJS(c *gin.Context) {
	user := models.CurrentUser(c)
	if user == nil {
		response.Fail(c, "unauthorized", "User not logged in")
		return
	}

	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)

	var input struct {
		JsSourceId string `json:"jsSourceId"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		response.Fail(c, "Parameter error", nil)
		return
	}

	var assistant models.Assistant
	if err := h.db.First(&assistant, id).Error; err != nil {
		response.Fail(c, "not found", "Assistant does not exist")
		return
	}

	if assistant.UserID != user.ID {
		response.Fail(c, "forbidden", "No permission to modify this assistant")
		return
	}

	// If a JS template ID is provided, verify that the template exists
	if input.JsSourceId != "" {
		_, err := models.GetJSTemplateByJsSourceID(h.db, input.JsSourceId)
		if err != nil {
			response.Fail(c, "Specified JS template does not exist", nil)
			return
		}
	}

	// Update JS template ID
	if err := h.db.Model(&assistant).Update("js_source_id", input.JsSourceId).Error; err != nil {
		response.Fail(c, "Update failed", nil)
		return
	}

	response.Success(c, "Update successful", nil)
}

// GetAssistantGraphData 获取助手在图数据库中的图数据
func (h *Handlers) GetAssistantGraphData(c *gin.Context) {
	user := models.CurrentUser(c)
	if user == nil {
		response.Fail(c, "unauthorized", "User not logged in")
		return
	}

	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	var assistant models.Assistant
	if err := h.db.First(&assistant, id).Error; err != nil {
		response.Fail(c, "not found", "Assistant does not exist")
		return
	}

	if assistant.UserID != user.ID {
		response.Fail(c, "forbidden", "No permission to access this assistant")
		return
	}

	// 检查是否启用了 Neo4j
	if !config.GlobalConfig.Services.KnowledgeBase.Neo4j.Enabled {
		response.Fail(c, "Neo4j not enabled", "Neo4j is not enabled in the system")
		return
	}

	// 检查助手是否启用了图记忆
	if !assistant.EnableGraphMemory {
		response.Fail(c, "Graph memory not enabled", "Graph memory is not enabled for this assistant")
		return
	}

	// 获取图数据
	store := graph.GetDefaultStore()
	if store == nil {
		response.Fail(c, "Graph store not available", "Graph store is not initialized")
		return
	}

	ctx := c.Request.Context()
	graphData, err := store.GetAssistantGraphData(ctx, id)
	if err != nil {
		logger.Error("Failed to get assistant graph data", zap.Error(err), zap.Int64("assistantID", id))
		response.Fail(c, "Failed to get graph data", err.Error())
		return
	}

	response.Success(c, "Graph data retrieved successfully", graphData)
}

// DeleteAssistant Delete assistant
func (h *Handlers) DeleteAssistant(c *gin.Context) {
	user := models.CurrentUser(c)
	if user == nil {
		response.Fail(c, "unauthorized", "User not logged in")
		return
	}

	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)

	var assistant models.Assistant
	if err := h.db.First(&assistant, id).Error; err != nil {
		response.Fail(c, "not found", "Assistant does not exist")
		return
	}

	if assistant.UserID != user.ID {
		response.Fail(c, "forbidden", "No permission to delete this assistant")
		return
	}

	if err := h.db.Delete(&assistant, id).Error; err != nil {
		response.Fail(c, "delete failed", "Delete failed")
		return
	}

	response.Success(c, "Delete successful", nil)
}

func (h *Handlers) ServeVoiceSculptorLoaderJS(c *gin.Context) {
	jsSourceID := c.Param("id")
	var assistant models.Assistant
	err := h.db.Where("js_source_id = ?", jsSourceID).First(&assistant).Error
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"code":  http.StatusNotFound,
			"error": "assistant is not exists",
			"data":  nil,
		})
		return
	}

	host := c.Request.Host
	scheme := "http"
	if c.Request.TLS != nil {
		scheme = "https"
	}
	baseURL := fmt.Sprintf("%s://%s%s", scheme, host, config.GlobalConfig.Server.APIPrefix)

	// Check if there is a bound JS template
	var templateContent string
	if assistant.JsSourceID != "" {
		// Try to get the bound JS template
		jsTemplate, err := models.GetJSTemplateByJsSourceID(h.db, assistant.JsSourceID)
		if err == nil && jsTemplate.Content != "" {
			// 检查是否有灰度版本
			activeVersion, err := models.GetActiveJSTemplateVersion(h.db, jsTemplate.ID)
			if err == nil && activeVersion != nil && activeVersion.Grayscale > 0 {
				// 使用灰度版本（根据用户ID或其他因素决定是否使用灰度版本）
				// 这里简化处理：如果灰度>0，使用版本内容；否则使用模板内容
				// 实际可以根据用户ID、IP等做更精细的灰度控制
				userHash := hashString(c.ClientIP() + c.GetHeader("User-Agent"))
				if userHash%100 < activeVersion.Grayscale {
					templateContent = activeVersion.Content
				} else {
					templateContent = jsTemplate.Content
				}
			} else {
				// Use the bound JS template
				templateContent = jsTemplate.Content
			}
		}
	}

	// If there is no bound JS template, use the default client.js
	if templateContent == "" {
		templateContent = LingEcho.AssistantJsModule
	}

	// Inject SDK at the beginning of the template content (if not already loaded)
	// 使用固定的CDN地址而不是本地地址
	sdkPath := "https://store.lingecho.com/uploads/buckets/default/lingecho-sdk.js"
	sdkInjection := fmt.Sprintf(`
// LingEcho SDK - auto load
(function() {
    // If SDK is already loaded, return
    if (typeof LingEchoSDK !== 'undefined' && window.lingEcho) {
        console.log('[LingEcho] SDK already loaded');
        window.__LINGECHO_SDK_READY__ = true;
        return;
    }
    
    // Asynchronously load SDK
    (function loadSDK() {
        const script = document.createElement('script');
        script.src = '%s';
        script.async = false; // Ensure execution order
        script.onload = function() {
            console.log('[LingEcho] SDK script loaded');
            // Wait for SDK class definition
            (function waitForSDKClass() {
                if (typeof LingEchoSDK !== 'undefined') {
                    // SDK class is loaded, wait for instance creation or manual creation
                    (function waitForInstance() {
                        if (window.lingEcho) {
                            console.log('[LingEcho] SDK instance ready');
                            window.__LINGECHO_SDK_READY__ = true;
                            // Trigger custom event
                            if (typeof window.dispatchEvent !== 'undefined') {
                                window.dispatchEvent(new Event('lingecho-sdk-ready'));
                            }
                            return;
                        }
                        // If SDK class is loaded but instance is not created, try to create
                        if (typeof SERVER_BASE !== 'undefined' || (typeof window !== 'undefined' && window.SERVER_BASE)) {
                            try {
                                const serverBase = typeof SERVER_BASE !== 'undefined' ? SERVER_BASE : window.SERVER_BASE;
                                const assistantName = typeof ASSISTANT_NAME !== 'undefined' ? ASSISTANT_NAME : (window.ASSISTANT_NAME || '');
                                window.lingEcho = new LingEchoSDK({
                                    baseURL: serverBase,
                                    assistantName: assistantName
                                });
                                window.__LINGECHO_SDK_READY__ = true;
                                console.log('[LingEcho] SDK instance created');
                                if (typeof window.dispatchEvent !== 'undefined') {
                                    window.dispatchEvent(new Event('lingecho-sdk-ready'));
                                }
                                return;
                            } catch (e) {
                                console.error('[LingEcho] Failed to create SDK instance:', e);
                            }
                        }
                        // Continue waiting
                        setTimeout(waitForInstance, 100);
                    })();
                } else {
                    // SDK class is not defined yet, continue waiting
                    setTimeout(waitForSDKClass, 100);
                }
            })();
        };
        script.onerror = function() {
            console.error('[LingEcho] Failed to load SDK script');
            window.__LINGECHO_SDK_ERROR__ = true;
        };
        // Insert at the beginning of head, ensuring priority loading
        const head = document.head || document.getElementsByTagName('head')[0];
        head.insertBefore(script, head.firstChild);
    })();
})();

`, sdkPath)

	// Combine SDK and template content
	fullTemplateContent := sdkInjection + templateContent

	tmpl, err := template.New("verification").Parse(fullTemplateContent)
	if err != nil {
		logger.Error("failed to parse verification template: ", zap.Error(err))
	}
	data := struct {
		BaseURL        string
		Name           string
		AssistantID    int64
		JsSourceID     string
		Description    string
		Language       string
		Speaker        string
		TtsProvider    string
		LLMModel       string
		Temperature    float32
		MaxTokens      int
		ASSISTANT_NAME string
		SERVER_BASE    string
	}{
		BaseURL:        baseURL,
		Name:           assistant.Name,
		AssistantID:    assistant.ID,
		JsSourceID:     assistant.JsSourceID,
		Description:    assistant.Description,
		Language:       assistant.Language,
		Speaker:        assistant.Speaker,
		TtsProvider:    assistant.TtsProvider,
		LLMModel:       assistant.LLMModel,
		Temperature:    assistant.Temperature,
		MaxTokens:      assistant.MaxTokens,
		ASSISTANT_NAME: assistant.Name,
		SERVER_BASE:    baseURL,
	}
	var body bytes.Buffer
	if err := tmpl.Execute(&body, data); err != nil {
		logger.Error("failed to render loader template: ", zap.Error(err))
	}

	c.Header("Content-Type", "application/javascript; charset=utf-8")
	c.String(http.StatusOK, body.String())
}

// ListAssistantTools Get all tools of the assistant
func (h *Handlers) ListAssistantTools(c *gin.Context) {
	user := models.CurrentUser(c)
	if user == nil {
		response.Fail(c, "unauthorized", "User not logged in")
		return
	}

	assistantID, _ := strconv.ParseInt(c.Param("id"), 10, 64)

	// Verify that the assistant exists and belongs to the current user
	var assistant models.Assistant
	if err := h.db.First(&assistant, assistantID).Error; err != nil {
		response.Fail(c, "not found", "Assistant does not exist")
		return
	}

	if assistant.UserID != user.ID {
		response.Fail(c, "forbidden", "No permission to access tools for this assistant")
		return
	}

	// Get all tools (including disabled ones)
	var tools []models.AssistantTool
	if err := h.db.Where("assistant_id = ?", assistantID).
		Order("created_at ASC").
		Find(&tools).Error; err != nil {
		response.Fail(c, "查询失败", "获取工具列表失败")
		return
	}

	response.Success(c, "Successfully retrieved tool list", tools)
}

// CreateAssistantTool Create a new tool
func (h *Handlers) CreateAssistantTool(c *gin.Context) {
	user := models.CurrentUser(c)
	if user == nil {
		response.Fail(c, "unauthorized", "User not logged in")
		return
	}

	assistantID, _ := strconv.ParseInt(c.Param("id"), 10, 64)

	// Verify that the assistant exists and belongs to the current user
	var assistant models.Assistant
	if err := h.db.First(&assistant, assistantID).Error; err != nil {
		response.Fail(c, "not found", "Assistant does not exist")
		return
	}

	if assistant.UserID != user.ID {
		response.Fail(c, "forbidden", "No permission to add tools for this assistant")
		return
	}

	var input struct {
		Name        string `json:"name" binding:"required"`
		Description string `json:"description" binding:"required"`
		Parameters  string `json:"parameters" binding:"required"` // JSON Schema格式
		Code        string `json:"code,omitempty"`                // 可选的代码实现（weather, calculator等）
		WebhookURL  string `json:"webhookUrl,omitempty"`          // Webhook URL（用于自定义工具执行）
		Enabled     bool   `json:"enabled"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		response.Fail(c, "Parameter error", err.Error())
		return
	}

	// Verify that name and description are not just whitespace
	if strings.TrimSpace(input.Name) == "" {
		response.Fail(c, "Parameter error", "Tool name cannot be empty")
		return
	}
	if strings.TrimSpace(input.Description) == "" {
		response.Fail(c, "Parameter error", "Tool description cannot be empty")
		return
	}

	// 验证name格式（只允许字母、数字、下划线、连字符）
	if matched, _ := regexp.MatchString(`^[a-zA-Z0-9_-]+$`, input.Name); !matched {
		response.Fail(c, "参数错误", "工具名称只能包含字母、数字、下划线和连字符")
		return
	}

	// Verify that Parameters is valid JSON Schema
	var paramsSchema map[string]interface{}
	if err := json.Unmarshal([]byte(input.Parameters), &paramsSchema); err != nil {
		response.Fail(c, "Parameter error", "Parameters must be valid JSON format")
		return
	}

	// Verify JSON Schema basic structure
	if schemaType, ok := paramsSchema["type"].(string); !ok || schemaType != "object" {
		response.Fail(c, "Parameter error", "Parameters must be JSON Schema format, type must be 'object'")
		return
	}

	// Verify properties field (if exists)
	if properties, ok := paramsSchema["properties"].(map[string]interface{}); ok {
		// properties can be empty, but if it has values, they should be validated
		for key, prop := range properties {
			if propMap, ok := prop.(map[string]interface{}); ok {
				// Verify that each property has a type field
				if _, hasType := propMap["type"]; !hasType {
					response.Fail(c, "Parameter error", fmt.Sprintf("Property '%s' is missing type field", key))
					return
				}
			}
		}
	}

	// Verify required field (if exists)
	if required, ok := paramsSchema["required"].([]interface{}); ok {
		// Verify that values in required are strings
		for _, req := range required {
			if _, ok := req.(string); !ok {
				response.Fail(c, "Parameter error", "Values in required array must be strings")
				return
			}
		}
	}

	// Verify: If webhook_url is provided, code should be empty or "webhook"
	if input.WebhookURL != "" {
		if input.Code != "" && input.Code != "webhook" {
			response.Fail(c, "Parameter error", "When using webhook, code should be empty or set to 'webhook'")
			return
		}
		// Verify webhook URL format
		if !isValidURL(input.WebhookURL) {
			response.Fail(c, "Parameter error", "webhookUrl must be a valid HTTP/HTTPS URL")
			return
		}
	}

	tool := models.AssistantTool{
		AssistantID: assistantID,
		Name:        input.Name,
		Description: input.Description,
		Parameters:  input.Parameters,
		Code:        input.Code,
		WebhookURL:  input.WebhookURL,
		Enabled:     input.Enabled,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	if err := models.CreateAssistantTool(h.db, &tool); err != nil {
		response.Fail(c, "Creation failed", err.Error())
		return
	}

	response.Success(c, "Tool created successfully", tool)
}

// UpdateAssistantTool Update tool
func (h *Handlers) UpdateAssistantTool(c *gin.Context) {
	user := models.CurrentUser(c)
	if user == nil {
		response.Fail(c, "unauthorized", "User not logged in")
		return
	}

	assistantID, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	toolID, _ := strconv.ParseInt(c.Param("toolId"), 10, 64)

	// Verify that the assistant exists and belongs to the current user
	var assistant models.Assistant
	if err := h.db.First(&assistant, assistantID).Error; err != nil {
		response.Fail(c, "not found", "Assistant does not exist")
		return
	}
	if assistant.UserID != user.ID {
		response.Fail(c, "forbidden", "No permission to modify tools for this assistant")
		return
	}

	// Verify that the tool exists and belongs to the assistant
	if exists, err := models.IsAssistantToolOwner(h.db, toolID, assistantID); err != nil || !exists {
		response.Fail(c, "not found", "Tool does not exist")
		return
	}

	var input struct {
		Name        string `json:"name"`
		Description string `json:"description"`
		Parameters  string `json:"parameters"` // JSON Schema format
		Code        string `json:"code,omitempty"`
		WebhookURL  string `json:"webhookUrl,omitempty"`
		Enabled     *bool  `json:"enabled"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		response.Fail(c, "Parameter error", err.Error())
		return
	}

	updates := make(map[string]interface{})
	if input.Name != "" {
		// Verify that name is not just whitespace
		if strings.TrimSpace(input.Name) == "" {
			response.Fail(c, "Parameter error", "Tool name cannot be empty")
			return
		}
		// Verify name format
		if matched, _ := regexp.MatchString(`^[a-zA-Z0-9_-]+$`, input.Name); !matched {
			response.Fail(c, "Parameter error", "Tool name can only contain letters, numbers, underscores, and hyphens")
			return
		}
		updates["name"] = input.Name
	}
	if input.Description != "" {
		// Verify that description is not just whitespace
		if strings.TrimSpace(input.Description) == "" {
			response.Fail(c, "Parameter error", "Tool description cannot be empty")
			return
		}
		updates["description"] = input.Description
	}
	if input.Parameters != "" {
		// Verify that Parameters is valid JSON Schema
		var paramsJSON map[string]interface{}
		if err := json.Unmarshal([]byte(input.Parameters), &paramsJSON); err != nil {
			response.Fail(c, "Parameter error", "Parameters must be valid JSON format")
			return
		}

		// Verify JSON Schema basic structure
		if schemaType, ok := paramsJSON["type"].(string); !ok || schemaType != "object" {
			response.Fail(c, "Parameter error", "Parameters must be JSON Schema format, type must be 'object'")
			return
		}

		// Verify properties field (if exists)
		if properties, ok := paramsJSON["properties"].(map[string]interface{}); ok {
			for key, prop := range properties {
				if propMap, ok := prop.(map[string]interface{}); ok {
					if _, hasType := propMap["type"]; !hasType {
						response.Fail(c, "Parameter error", fmt.Sprintf("Property '%s' is missing type field", key))
						return
					}
				}
			}
		}

		// Verify required field (if exists)
		if required, ok := paramsJSON["required"].([]interface{}); ok {
			for _, req := range required {
				if _, ok := req.(string); !ok {
					response.Fail(c, "Parameter error", "Values in required array must be strings")
					return
				}
			}
		}

		updates["parameters"] = input.Parameters
	}
	if input.Code != "" {
		updates["code"] = input.Code
	}
	if input.WebhookURL != "" {
		// Verify webhook URL format
		if !isValidURL(input.WebhookURL) {
			response.Fail(c, "Parameter error", "webhookUrl must be a valid HTTP/HTTPS URL")
			return
		}
		// If webhook_url is provided, code should be empty or "webhook"
		if input.Code != "" && input.Code != "webhook" {
			response.Fail(c, "Parameter error", "When using webhook, code should be empty or set to 'webhook'")
			return
		}
		updates["webhook_url"] = input.WebhookURL
	}
	if input.Enabled != nil {
		updates["enabled"] = *input.Enabled
	}

	if len(updates) == 0 {
		response.Fail(c, "Parameter error", "No fields to update")
		return
	}

	if err := models.UpdateAssistantTool(h.db, toolID, assistantID, updates); err != nil {
		response.Fail(c, "Update failed", err.Error())
		return
	}

	// Get the updated tool
	tool, err := models.GetAssistantToolByID(h.db, toolID, assistantID)
	if err != nil {
		response.Fail(c, "Query failed", "Failed to get updated tool")
		return
	}

	response.Success(c, "Tool updated successfully", tool)
}

// DeleteAssistantTool Delete tool
func (h *Handlers) DeleteAssistantTool(c *gin.Context) {
	user := models.CurrentUser(c)
	if user == nil {
		response.Fail(c, "unauthorized", "User not logged in")
		return
	}

	assistantID, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	toolID, _ := strconv.ParseInt(c.Param("toolId"), 10, 64)

	// Verify that the assistant exists and belongs to the current user
	var assistant models.Assistant
	if err := h.db.First(&assistant, assistantID).Error; err != nil {
		response.Fail(c, "not found", "Assistant does not exist")
		return
	}
	if assistant.UserID != user.ID {
		response.Fail(c, "forbidden", "No permission to delete tools for this assistant")
		return
	}

	// Verify that the tool exists and belongs to the assistant
	if exists, err := models.IsAssistantToolOwner(h.db, toolID, assistantID); err != nil || !exists {
		response.Fail(c, "not found", "Tool does not exist")
		return
	}

	if err := models.DeleteAssistantTool(h.db, toolID, assistantID); err != nil {
		response.Fail(c, "Deletion failed", err.Error())
		return
	}

	response.Success(c, "Tool deleted successfully", nil)
}

// TestAssistantTool Test tool execution
func (h *Handlers) TestAssistantTool(c *gin.Context) {
	user := models.CurrentUser(c)
	if user == nil {
		response.Fail(c, "unauthorized", "User not logged in")
		return
	}

	assistantID, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	toolID, _ := strconv.ParseInt(c.Param("toolId"), 10, 64)

	// Verify that the assistant exists and belongs to the current user
	var assistant models.Assistant
	if err := h.db.First(&assistant, assistantID).Error; err != nil {
		response.Fail(c, "not found", "Assistant does not exist")
		return
	}
	if assistant.UserID != user.ID {
		response.Fail(c, "forbidden", "No permission to test tools for this assistant")
		return
	}

	// Verify that the tool exists and belongs to the assistant
	tool, err := models.GetAssistantToolByID(h.db, toolID, assistantID)
	if err != nil {
		response.Fail(c, "not found", "Tool does not exist")
		return
	}

	var input struct {
		Args map[string]interface{} `json:"args" binding:"required"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		response.Fail(c, "Parameter error", err.Error())
		return
	}

	// Directly call the execution logic in assistant_tools.go to test the tool
	testResult, err := h.executeToolForTest(tool, input.Args)
	if err != nil {
		response.Fail(c, "Tool execution failed", err.Error())
		return
	}

	response.Success(c, "Tool test successful", gin.H{
		"result": testResult,
		"tool":   tool,
	})
}

// executeToolForTest Execute tool test (using execution logic in assistant_tools.go)
func (h *Handlers) executeToolForTest(tool *models.AssistantTool, args map[string]interface{}) (string, error) {
	// Call the executeToolCode method in assistant_tools.go
	// Since executeToolCode requires assistantID, we use 0 as a placeholder (test scenario)
	return h.executeToolCode(*tool, 0, args)
}

// isValidURL Validate URL format
func isValidURL(urlStr string) bool {
	parsedURL, err := url.Parse(urlStr)
	if err != nil {
		return false
	}
	// Only allow HTTP and HTTPS protocols
	return parsedURL.Scheme == "http" || parsedURL.Scheme == "https"
}
