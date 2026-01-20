package handlers

import (
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/LingByte/SoulNexus/internal/models"
	workflowdef "github.com/LingByte/SoulNexus/internal/workflow"
	"github.com/LingByte/SoulNexus/pkg/events"
	"github.com/LingByte/SoulNexus/pkg/logger"
	"github.com/LingByte/SoulNexus/pkg/response"
	"github.com/LingByte/SoulNexus/pkg/websocket"
	runtimewf "github.com/LingByte/SoulNexus/pkg/workflow"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// WorkflowLogSender implements LogSender interface for WebSocket log streaming
type WorkflowLogSender struct {
	hub    *websocket.Hub
	userID string
}

func (w *WorkflowLogSender) SendLog(log runtimewf.ExecutionLog) error {
	if w.hub == nil {
		return nil
	}
	message := &websocket.Message{
		Type:      "workflow_log",
		Data:      log,
		Timestamp: time.Now().Unix(),
		To:        w.userID,
	}
	// Send message to hub's broadcast channel (non-blocking)
	select {
	case w.hub.GetBroadcastChannel() <- message:
		// Message sent successfully
	default:
		// If broadcast channel is full, skip sending (non-blocking)
		// This prevents blocking the workflow execution
	}
	return nil
}

func (h *Handlers) registerWorkflowRoutes(r *gin.RouterGroup) {
	workflows := r.Group("/workflows")
	workflows.Use(models.AuthRequired)
	{
		defs := workflows.Group("/definitions")
		defs.POST("", h.CreateWorkflowDefinition)
		defs.GET("", h.ListWorkflowDefinitions)
		defs.GET("/:id", h.GetWorkflowDefinition)
		defs.PUT("/:id", h.UpdateWorkflowDefinition)
		defs.DELETE("/:id", h.DeleteWorkflowDefinition)
		defs.POST("/:id/run", h.RunWorkflowDefinition)
		defs.POST("/:id/nodes/:nodeId/test", h.TestWorkflowNode)

		// Event management routes
		workflows.GET("/events/types", h.GetAvailableEventTypes)

		// Version management routes
		defs.GET("/:id/versions", h.ListWorkflowVersions)
		defs.GET("/:id/versions/:versionId", h.GetWorkflowVersion)
		defs.POST("/:id/versions/:versionId/rollback", h.RollbackWorkflowVersion)
		defs.GET("/:id/versions/compare", h.CompareWorkflowVersions)
	}
}

type workflowDefinitionInput struct {
	Name        string               `json:"name" binding:"required"`
	Slug        string               `json:"slug" binding:"required"`
	Description string               `json:"description"`
	Status      string               `json:"status"`
	Definition  models.WorkflowGraph `json:"definition" binding:"required"`
	Settings    models.JSONMap       `json:"settings"`
	Triggers    models.JSONMap       `json:"triggers"`
	Tags        []string             `json:"tags"`
	Version     uint                 `json:"version"`
	GroupID     interface{}          `json:"groupId,omitempty"` // 组织ID，如果设置则表示这是组织共享的工作流
}

// CreateWorkflowDefinition stores a new workflow template.
func (h *Handlers) CreateWorkflowDefinition(c *gin.Context) {
	user := models.CurrentUser(c)
	if user == nil {
		response.Fail(c, "unauthorized", "User not logged in")
		return
	}

	var input workflowDefinitionInput
	if err := c.ShouldBindJSON(&input); err != nil {
		response.Fail(c, "invalid payload", err.Error())
		return
	}

	// 获取组织ID（从请求体获取）
	var groupID *uint
	if groupIDVal, ok := input.GroupID.(float64); ok && groupIDVal > 0 {
		uid := uint(groupIDVal)
		groupID = &uid
		// 验证用户是否有权限访问该组织
		var group models.Group
		if err := h.db.Where("id = ?", uid).First(&group).Error; err != nil {
			response.Fail(c, "organization not found", nil)
			return
		}
		// 检查用户是否是组织成员或创建者
		if group.CreatorID != user.ID {
			var member models.GroupMember
			if err := h.db.Where("group_id = ? AND user_id = ?", uid, user.ID).First(&member).Error; err != nil {
				response.Fail(c, "insufficient permissions", "You are not a member of this organization")
				return
			}
		}
	}

	if err := validateWorkflowGraph(input.Definition); err != nil {
		response.Fail(c, "invalid workflow definition", err.Error())
		return
	}

	if strings.TrimSpace(input.Status) == "" {
		input.Status = "draft"
	}
	if input.Version == 0 {
		input.Version = 1
	}

	def := models.WorkflowDefinition{
		UserID:      user.ID,
		GroupID:     groupID,
		Name:        input.Name,
		Slug:        input.Slug,
		Description: input.Description,
		Status:      input.Status,
		Definition:  input.Definition,
		Settings:    input.Settings,
		Triggers:    input.Triggers,
		Tags:        models.StringArray(input.Tags),
		Version:     input.Version,
		CreatedBy:   user.Email,
		UpdatedBy:   user.Email,
	}

	if err := h.db.Create(&def).Error; err != nil {
		response.Fail(c, "failed to create workflow definition", err.Error())
		return
	}

	// Save initial version to history
	initialVersion := models.WorkflowVersion{
		DefinitionID: def.ID,
		Version:      def.Version,
		Name:         def.Name,
		Slug:         def.Slug,
		Description:  def.Description,
		Status:       def.Status,
		Definition:   def.Definition,
		Settings:     def.Settings,
		Triggers:     def.Triggers,
		Tags:         def.Tags,
		CreatedBy:    def.CreatedBy,
		UpdatedBy:    def.UpdatedBy,
		ChangeNote:   "初始版本",
	}
	// Ignore error for initial version save (non-critical)
	_ = h.db.Create(&initialVersion).Error

	response.Success(c, "workflow definition created", def)
}

// ListWorkflowDefinitions returns definitions with optional status filter.
func (h *Handlers) ListWorkflowDefinitions(c *gin.Context) {
	user := models.CurrentUser(c)
	if user == nil {
		response.Fail(c, "unauthorized", "User not logged in")
		return
	}

	var (
		status  = c.Query("status")
		keyword = c.Query("keyword")
		list    []models.WorkflowDefinition
	)

	// 获取用户所属的组织ID列表
	var groupIDs []uint
	var groupMembers []models.GroupMember
	if err := h.db.Where("user_id = ?", user.ID).Find(&groupMembers).Error; err == nil {
		for _, member := range groupMembers {
			groupIDs = append(groupIDs, member.GroupID)
		}
	}
	// 获取用户创建的组织ID
	var userGroups []models.Group
	if err := h.db.Where("creator_id = ?", user.ID).Find(&userGroups).Error; err == nil {
		for _, group := range userGroups {
			groupIDs = append(groupIDs, group.ID)
		}
	}

	query := h.db.Model(&models.WorkflowDefinition{})

	// 查询：用户自己的工作流 + 组织共享的工作流
	if len(groupIDs) > 0 {
		query = query.Where("user_id = ? OR (group_id IS NOT NULL AND group_id IN (?))", user.ID, groupIDs)
	} else {
		query = query.Where("user_id = ?", user.ID)
	}

	if status != "" {
		query = query.Where("status = ?", status)
	}
	if keyword != "" {
		like := "%" + keyword + "%"
		query = query.Where("name LIKE ? OR slug LIKE ?", like, like)
	}

	if err := query.Order("updated_at desc").Find(&list).Error; err != nil {
		response.Fail(c, "failed to list workflow definitions", err.Error())
		return
	}

	response.Success(c, "ok", list)
}

// GetWorkflowDefinition returns a single definition.
func (h *Handlers) GetWorkflowDefinition(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		response.Fail(c, "invalid id", nil)
		return
	}

	var def models.WorkflowDefinition
	if err := h.db.First(&def, id).Error; err != nil {
		response.Fail(c, "workflow definition not found", err.Error())
		return
	}

	response.Success(c, "ok", def)
}

// UpdateWorkflowDefinition updates definition fields.
func (h *Handlers) UpdateWorkflowDefinition(c *gin.Context) {
	user := models.CurrentUser(c)
	if user == nil {
		response.Fail(c, "unauthorized", nil)
		return
	}

	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		response.Fail(c, "invalid id", nil)
		return
	}

	var def models.WorkflowDefinition
	if err := h.db.First(&def, id).Error; err != nil {
		response.Fail(c, "workflow definition not found", err.Error())
		return
	}

	var input struct {
		Name        *string               `json:"name"`
		Description *string               `json:"description"`
		Status      *string               `json:"status"`
		Definition  *models.WorkflowGraph `json:"definition"`
		Settings    *models.JSONMap       `json:"settings"`
		Triggers    *models.JSONMap       `json:"triggers"`
		Tags        []string              `json:"tags"`
		Version     uint                  `json:"version" binding:"required"`
		ChangeNote  string                `json:"changeNote"`        // 版本变更说明
		GroupID     interface{}           `json:"groupId,omitempty"` // 组织ID，如果设置则表示这是组织共享的工作流
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		response.Fail(c, "invalid payload", err.Error())
		return
	}

	if input.Version == 0 {
		response.Fail(c, "invalid payload", "version is required")
		return
	}

	if input.Definition != nil {
		if err := validateWorkflowGraph(*input.Definition); err != nil {
			response.Fail(c, "invalid workflow definition", err.Error())
			return
		}
	}

	if input.Version != def.Version {
		response.Fail(c, "version conflict", fmt.Sprintf("expected version %d", def.Version))
		return
	}

	// 检查权限：只有创建者或组织管理员可以更新
	if def.UserID != user.ID {
		if def.GroupID == nil {
			response.Fail(c, "insufficient permissions", nil)
			return
		}
		// 检查用户是否是组织创建者或管理员
		var group models.Group
		if err := h.db.Where("id = ?", *def.GroupID).First(&group).Error; err != nil {
			response.Fail(c, "organization not found", nil)
			return
		}
		if group.CreatorID != user.ID {
			var member models.GroupMember
			if err := h.db.Where("group_id = ? AND user_id = ? AND role = ?", *def.GroupID, user.ID, models.GroupRoleAdmin).First(&member).Error; err != nil {
				response.Fail(c, "insufficient permissions", "Only creator or admin can update organization-shared workflows")
				return
			}
		}
	}

	// 如果更新了 GroupID，验证权限
	if input.GroupID != nil {
		var groupID *uint
		if groupIDVal, ok := input.GroupID.(float64); ok && groupIDVal > 0 {
			uid := uint(groupIDVal)
			groupID = &uid
			var group models.Group
			if err := h.db.Where("id = ?", uid).First(&group).Error; err != nil {
				response.Fail(c, "organization not found", nil)
				return
			}
			if group.CreatorID != user.ID {
				var member models.GroupMember
				if err := h.db.Where("group_id = ? AND user_id = ?", uid, user.ID).First(&member).Error; err != nil {
					response.Fail(c, "insufficient permissions", "You are not a member of this organization")
					return
				}
			}
			def.GroupID = groupID
		} else {
			// 如果传入 null，表示取消组织共享
			def.GroupID = nil
		}
	}

	// Save current version to history before updating
	versionHistory := models.WorkflowVersion{
		DefinitionID: def.ID,
		Version:      def.Version,
		Name:         def.Name,
		Slug:         def.Slug,
		Description:  def.Description,
		Status:       def.Status,
		Definition:   def.Definition,
		Settings:     def.Settings,
		Triggers:     def.Triggers,
		Tags:         def.Tags,
		CreatedBy:    def.CreatedBy,
		UpdatedBy:    def.UpdatedBy,
		ChangeNote:   input.ChangeNote,
	}
	if err := h.db.Create(&versionHistory).Error; err != nil {
		// Log error but don't fail the update (version history is non-critical)
		logger.Error("failed to save version history", zap.Error(err), zap.Uint("definition_id", def.ID), zap.Uint("version", def.Version))
		// Continue with update even if version history save fails
	}

	if input.Name != nil {
		def.Name = *input.Name
	}
	if input.Description != nil {
		def.Description = *input.Description
	}
	if input.Status != nil && *input.Status != "" {
		def.Status = *input.Status
	}
	if input.Definition != nil {
		def.Definition = *input.Definition
	}
	if input.Settings != nil {
		def.Settings = *input.Settings
	}
	if input.Triggers != nil {
		def.Triggers = *input.Triggers
	}
	if input.Tags != nil {
		def.Tags = models.StringArray(input.Tags)
	}
	def.UpdatedBy = user.Email
	oldVersion := def.Version
	def.Version++

	updates := map[string]interface{}{
		"name":        def.Name,
		"description": def.Description,
		"status":      def.Status,
		"definition":  def.Definition,
		"settings":    def.Settings,
		"triggers":    def.Triggers,
		"tags":        def.Tags,
		"updated_by":  def.UpdatedBy,
		"version":     def.Version,
	}

	tx := h.db.Model(&models.WorkflowDefinition{}).
		Where("id = ? AND version = ?", def.ID, oldVersion).
		Updates(updates)
	if tx.Error != nil {
		response.Fail(c, "failed to update workflow definition", tx.Error.Error())
		return
	}
	if tx.RowsAffected == 0 {
		response.Fail(c, "version conflict", "workflow definition was updated by others")
		return
	}

	response.Success(c, "workflow definition updated", def)
}

// DeleteWorkflowDefinition deletes a workflow definition.
func (h *Handlers) DeleteWorkflowDefinition(c *gin.Context) {
	user := models.CurrentUser(c)
	if user == nil {
		response.Fail(c, "unauthorized", "User not logged in")
		return
	}

	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		response.Fail(c, "invalid id", nil)
		return
	}

	var def models.WorkflowDefinition
	if err := h.db.First(&def, id).Error; err != nil {
		response.Fail(c, "workflow definition not found", err.Error())
		return
	}

	// 检查权限：只有创建者或组织管理员可以删除
	if def.UserID != user.ID {
		if def.GroupID == nil {
			response.Fail(c, "insufficient permissions", nil)
			return
		}
		// 检查用户是否是组织创建者或管理员
		var group models.Group
		if err := h.db.Where("id = ?", *def.GroupID).First(&group).Error; err != nil {
			response.Fail(c, "organization not found", nil)
			return
		}
		if group.CreatorID != user.ID {
			var member models.GroupMember
			if err := h.db.Where("group_id = ? AND user_id = ? AND role = ?", *def.GroupID, user.ID, models.GroupRoleAdmin).First(&member).Error; err != nil {
				response.Fail(c, "insufficient permissions", "Only creator or admin can delete organization-shared workflows")
				return
			}
		}
	}

	// 软删除：使用 GORM 的 Delete 方法
	if err := h.db.Delete(&def).Error; err != nil {
		response.Fail(c, "failed to delete workflow definition", err.Error())
		return
	}

	response.Success(c, "workflow definition deleted", map[string]string{
		"message": "Workflow definition deleted successfully",
	})
}

var allowedNodeTypes = map[string]struct{}{
	"start":     {},
	"end":       {},
	"task":      {},
	"gateway":   {},
	"event":     {},
	"subflow":   {},
	"condition": {},
	"parallel":  {},
	"wait":      {},
	"timer":     {},
	"script":    {},
}

func validateWorkflowGraph(graph models.WorkflowGraph) error {
	if len(graph.Nodes) == 0 {
		return errors.New("workflow must contain at least one node")
	}

	nodeTypes := make(map[string]string, len(graph.Nodes))
	startCount := 0
	endCount := 0

	for _, node := range graph.Nodes {
		if node.ID == "" {
			return errors.New("node id cannot be empty")
		}
		if _, exists := nodeTypes[node.ID]; exists {
			return fmt.Errorf("duplicate node id %s", node.ID)
		}
		normalizedType := strings.ToLower(node.Type)
		if _, ok := allowedNodeTypes[normalizedType]; !ok {
			return fmt.Errorf("unsupported node type %s", node.Type)
		}
		nodeTypes[node.ID] = normalizedType
		if normalizedType == "start" {
			startCount++
		}
		if normalizedType == "end" {
			endCount++
		}
	}

	if startCount != 1 {
		return fmt.Errorf("workflow must contain exactly one start node, got %d", startCount)
	}
	if endCount == 0 {
		return errors.New("workflow must contain at least one end node")
	}

	for _, edge := range graph.Edges {
		if edge.Source == "" || edge.Target == "" {
			return errors.New("edge source and target cannot be empty")
		}
		sourceType, ok := nodeTypes[edge.Source]
		if !ok {
			return fmt.Errorf("edge references unknown source node %s", edge.Source)
		}
		if _, ok := nodeTypes[edge.Target]; !ok {
			return fmt.Errorf("edge references unknown target node %s", edge.Target)
		}

		switch edge.Type {
		case models.WorkflowEdgeTypeTrue, models.WorkflowEdgeTypeFalse:
			if sourceType != "gateway" && sourceType != "condition" {
				return fmt.Errorf("edge type %s allowed only for gateway/condition nodes", edge.Type)
			}
			// Note: condition type is deprecated but kept for backward compatibility
		case models.WorkflowEdgeTypeBranch:
			if sourceType != "parallel" {
				return fmt.Errorf("branch edge allowed only for parallel nodes")
			}
		}
	}

	return nil
}

// RunWorkflowDefinition executes a workflow definition and creates an instance.
func (h *Handlers) RunWorkflowDefinition(c *gin.Context) {
	user := models.CurrentUser(c)
	if user == nil {
		response.Fail(c, "unauthorized", "User not logged in")
		return
	}

	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		response.Fail(c, "invalid id", nil)
		return
	}

	var def models.WorkflowDefinition
	if err := h.db.First(&def, id).Error; err != nil {
		response.Fail(c, "workflow definition not found", err.Error())
		return
	}

	// Parse initial parameters from request body (optional)
	var input struct {
		Parameters map[string]interface{} `json:"parameters"`
	}
	// Parse request body (empty body is allowed for parameters)
	if err := c.ShouldBindJSON(&input); err != nil {
		// Check if error is EOF (empty body), which is acceptable
		if err.Error() == "EOF" {
			// Empty body is allowed, continue with empty parameters
		} else {
			response.Fail(c, "invalid payload", err.Error())
			return
		}
	}

	// Build runtime workflow from definition
	runtimeWf, err := workflowdef.BuildRuntimeWorkflow(&def)
	if err != nil {
		response.Fail(c, "failed to build workflow", err.Error())
		return
	}

	// Set initial parameters to workflow context
	// Context is already initialized by BuildRuntimeWorkflow, so we can directly set parameters
	if input.Parameters != nil && len(input.Parameters) > 0 {
		if runtimeWf.Context == nil {
			// Context should already be initialized by BuildRuntimeWorkflow
			// But if it's nil, create a new one (shouldn't happen, but safety check)
			runtimeWf.Context = runtimewf.NewWorkflowContext(fmt.Sprintf("definition-%d", def.ID))
		}
		// Initialize Parameters map if nil
		if runtimeWf.Context.Parameters == nil {
			runtimeWf.Context.Parameters = make(map[string]interface{})
		}
		for k, v := range input.Parameters {
			runtimeWf.Context.Parameters[k] = v
		}
	}

	// Set up WebSocket log sender if WebSocket hub is available
	if h.wsHub != nil && runtimeWf.Context != nil {
		userID := fmt.Sprintf("%d", user.ID)
		runtimeWf.Context.LogSender = &WorkflowLogSender{
			hub:    h.wsHub,
			userID: userID,
		}
	}

	// Create workflow instance
	now := time.Now()
	instance := models.WorkflowInstance{
		DefinitionID:   def.ID,
		DefinitionName: def.Name,
		Status:         "running",
		StartedAt:      &now,
		ContextData:    make(models.JSONMap),
		ResultData:     make(models.JSONMap),
	}

	if err := h.db.Create(&instance).Error; err != nil {
		response.Fail(c, "failed to create workflow instance", err.Error())
		return
	}

	// Execute workflow
	execErr := runtimeWf.Execute()

	// Update instance with execution result
	completedAt := time.Now()
	instance.CompletedAt = &completedAt

	if execErr != nil {
		instance.Status = "failed"
		instance.ResultData = models.JSONMap{
			"error": execErr.Error(),
		}
	} else {
		instance.Status = "completed"
		// Store workflow context data as result
		if runtimeWf.Context != nil {
			instance.ContextData = runtimeWf.Context.NodeData
			instance.ResultData = models.JSONMap{
				"success": true,
				"context": runtimeWf.Context.NodeData,
			}
		}
	}

	// Update current node if available
	if runtimeWf.Context != nil && runtimeWf.Context.CurrentNode != "" {
		instance.CurrentNodeID = runtimeWf.Context.CurrentNode
	}

	if err := h.db.Save(&instance).Error; err != nil {
		response.Fail(c, "failed to update workflow instance", err.Error())
		return
	}

	// Prepare response with logs
	responseData := map[string]interface{}{
		"instance": instance,
	}
	if runtimeWf.Context != nil && len(runtimeWf.Context.Logs) > 0 {
		responseData["logs"] = runtimeWf.Context.Logs
	}

	if execErr != nil {
		response.Fail(c, "workflow execution failed", execErr.Error())
		return
	}

	response.Success(c, "workflow executed successfully", responseData)
}

// TestWorkflowNode tests a single node with provided inputs
func (h *Handlers) TestWorkflowNode(c *gin.Context) {
	user := models.CurrentUser(c)
	if user == nil {
		response.Fail(c, "unauthorized", "User not logged in")
		return
	}

	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		response.Fail(c, "invalid id", nil)
		return
	}

	nodeID := c.Param("nodeId")
	if nodeID == "" {
		response.Fail(c, "invalid node id", nil)
		return
	}

	var def models.WorkflowDefinition
	if err := h.db.First(&def, id).Error; err != nil {
		response.Fail(c, "workflow definition not found", err.Error())
		return
	}

	// Find the node in definition
	var targetNode *models.WorkflowNodeSchema
	for i := range def.Definition.Nodes {
		if def.Definition.Nodes[i].ID == nodeID {
			targetNode = &def.Definition.Nodes[i]
			break
		}
	}

	if targetNode == nil {
		response.Fail(c, "node not found", fmt.Sprintf("Node %s not found in workflow", nodeID))
		return
	}

	// Parse input parameters from request body
	var input struct {
		Parameters map[string]interface{} `json:"parameters"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		// Empty body is allowed
		if err.Error() != "EOF" {
			response.Fail(c, "invalid payload", err.Error())
			return
		}
	}

	// Create a minimal workflow context for testing
	ctx := runtimewf.NewWorkflowContext(fmt.Sprintf("test-node-%s", nodeID))
	if input.Parameters != nil {
		// If input.Parameters has a "parameters" key, merge it into ctx.Parameters
		// This allows both formats: {"parameters": {"value": true}} and {"value": true}
		if nestedParams, ok := input.Parameters["parameters"].(map[string]interface{}); ok {
			ctx.Parameters = nestedParams
		} else {
			ctx.Parameters = input.Parameters
		}
	}

	// Set up WebSocket log sender if available
	if h.wsHub != nil {
		userID := fmt.Sprintf("%d", user.ID)
		ctx.LogSender = &WorkflowLogSender{
			hub:    h.wsHub,
			userID: userID,
		}
	}

	// Build runtime node from definition
	runtimeNode, err := workflowdef.BuildRuntimeNode(targetNode, &def.Definition)
	if err != nil {
		response.Fail(c, "failed to build node", err.Error())
		return
	}

	// For gateway nodes (and condition nodes for backward compatibility), we need to process edges to set up next node IDs
	if targetNode.Type == "gateway" || targetNode.Type == "condition" {
		// Find edges from this node
		var edgeNextNodes []string
		for _, edge := range def.Definition.Edges {
			if edge.Source == nodeID {
				edgeNextNodes = append(edgeNextNodes, edge.Target)
				// Assign edge metadata to the node (for gateway/condition nodes)
				workflowdef.AssignEdgeMetadata(runtimeNode, edge)
			}
		}
		if runtimeNode.Base() != nil {
			runtimeNode.Base().NextNodes = edgeNextNodes
		}
	} else {
		// For other nodes, just collect next nodes from edges
		var edgeNextNodes []string
		for _, edge := range def.Definition.Edges {
			if edge.Source == nodeID {
				edgeNextNodes = append(edgeNextNodes, edge.Target)
			}
		}
		if runtimeNode.Base() != nil {
			runtimeNode.Base().NextNodes = edgeNextNodes
		}
	}

	// Execute the node
	ctx.CurrentNode = nodeID
	ctx.SetNodeStatus(nodeID, runtimewf.NodeStatusRunning, nil)
	ctx.AddLog("info", fmt.Sprintf("Testing node: %s", targetNode.Name), nodeID, targetNode.Name)

	nextNodes, execErr := runtimeNode.Run(ctx)

	// Update node status
	if execErr != nil {
		ctx.SetNodeStatus(nodeID, runtimewf.NodeStatusFailed, execErr)
		ctx.AddLog("error", fmt.Sprintf("Node test failed: %s", execErr.Error()), nodeID, targetNode.Name)
	} else {
		ctx.SetNodeStatus(nodeID, runtimewf.NodeStatusCompleted, nil)
		ctx.AddLog("success", fmt.Sprintf("Node test completed: %s", targetNode.Name), nodeID, targetNode.Name)
	}

	// Prepare response
	responseData := map[string]interface{}{
		"nodeId":    nodeID,
		"nodeName":  targetNode.Name,
		"status":    ctx.GetNodeStatus(nodeID),
		"nextNodes": nextNodes,
		"context":   ctx.NodeData,
		"logs":      ctx.Logs,
	}

	if execErr != nil {
		responseData["error"] = execErr.Error()
		response.Fail(c, "node test failed", responseData)
		return
	}

	response.Success(c, "node test completed", responseData)
}

// ListWorkflowVersions returns all historical versions of a workflow definition.
func (h *Handlers) ListWorkflowVersions(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		response.Fail(c, "invalid id", nil)
		return
	}

	// Verify workflow definition exists
	var def models.WorkflowDefinition
	if err := h.db.First(&def, id).Error; err != nil {
		response.Fail(c, "workflow definition not found", err.Error())
		return
	}

	var versions []models.WorkflowVersion
	if err := h.db.Where("definition_id = ?", id).
		Order("version desc").
		Find(&versions).Error; err != nil {
		response.Fail(c, "failed to list workflow versions", err.Error())
		return
	}

	response.Success(c, "ok", versions)
}

// GetWorkflowVersion returns a specific version of a workflow definition.
func (h *Handlers) GetWorkflowVersion(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		response.Fail(c, "invalid id", nil)
		return
	}

	versionId, err := strconv.Atoi(c.Param("versionId"))
	if err != nil {
		response.Fail(c, "invalid version id", nil)
		return
	}

	// Verify workflow definition exists
	var def models.WorkflowDefinition
	if err := h.db.First(&def, id).Error; err != nil {
		response.Fail(c, "workflow definition not found", err.Error())
		return
	}

	var version models.WorkflowVersion
	if err := h.db.Where("definition_id = ? AND id = ?", id, versionId).
		First(&version).Error; err != nil {
		response.Fail(c, "workflow version not found", err.Error())
		return
	}

	response.Success(c, "ok", version)
}

// RollbackWorkflowVersion restores a workflow definition to a specific version.
func (h *Handlers) RollbackWorkflowVersion(c *gin.Context) {
	user := models.CurrentUser(c)
	if user == nil {
		response.Fail(c, "unauthorized", nil)
		return
	}

	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		response.Fail(c, "invalid id", nil)
		return
	}

	versionId, err := strconv.Atoi(c.Param("versionId"))
	if err != nil {
		response.Fail(c, "invalid version id", nil)
		return
	}

	// Get current workflow definition
	var def models.WorkflowDefinition
	if err := h.db.First(&def, id).Error; err != nil {
		response.Fail(c, "workflow definition not found", err.Error())
		return
	}

	// Get the version to rollback to
	var version models.WorkflowVersion
	if err := h.db.Where("definition_id = ? AND id = ?", id, versionId).
		First(&version).Error; err != nil {
		response.Fail(c, "workflow version not found", err.Error())
		return
	}

	// Save current version to history before rollback
	currentVersionHistory := models.WorkflowVersion{
		DefinitionID: def.ID,
		Version:      def.Version,
		Name:         def.Name,
		Slug:         def.Slug,
		Description:  def.Description,
		Status:       def.Status,
		Definition:   def.Definition,
		Settings:     def.Settings,
		Triggers:     def.Triggers,
		Tags:         def.Tags,
		CreatedBy:    def.CreatedBy,
		UpdatedBy:    def.UpdatedBy,
		ChangeNote:   fmt.Sprintf("Rollback to version %d", version.Version),
	}
	if err := h.db.Create(&currentVersionHistory).Error; err != nil {
		response.Fail(c, "failed to save version history", err.Error())
		return
	}

	// Restore from version
	def.Name = version.Name
	def.Description = version.Description
	def.Status = version.Status
	def.Definition = version.Definition
	def.Settings = version.Settings
	def.Triggers = version.Triggers
	def.Tags = version.Tags
	def.UpdatedBy = user.Email
	def.Version++

	updates := map[string]interface{}{
		"name":        def.Name,
		"description": def.Description,
		"status":      def.Status,
		"definition":  def.Definition,
		"settings":    def.Settings,
		"triggers":    def.Triggers,
		"tags":        def.Tags,
		"updated_by":  def.UpdatedBy,
		"version":     def.Version,
	}

	if err := h.db.Model(&models.WorkflowDefinition{}).
		Where("id = ?", def.ID).
		Updates(updates).Error; err != nil {
		response.Fail(c, "failed to rollback workflow definition", err.Error())
		return
	}

	response.Success(c, "workflow definition rolled back", def)
}

// CompareWorkflowVersions compares two versions of a workflow definition.
func (h *Handlers) CompareWorkflowVersions(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		response.Fail(c, "invalid id", nil)
		return
	}

	// Get version IDs from query parameters
	version1Id := c.Query("version1")
	version2Id := c.Query("version2")

	if version1Id == "" || version2Id == "" {
		response.Fail(c, "invalid parameters", "version1 and version2 are required")
		return
	}

	v1Id, err := strconv.Atoi(version1Id)
	if err != nil {
		response.Fail(c, "invalid version1 id", nil)
		return
	}

	v2Id, err := strconv.Atoi(version2Id)
	if err != nil {
		response.Fail(c, "invalid version2 id", nil)
		return
	}

	// Verify workflow definition exists
	var def models.WorkflowDefinition
	if err := h.db.First(&def, id).Error; err != nil {
		response.Fail(c, "workflow definition not found", err.Error())
		return
	}

	// Get both versions
	var v1, v2 models.WorkflowVersion
	if err := h.db.Where("definition_id = ? AND id = ?", id, v1Id).
		First(&v1).Error; err != nil {
		response.Fail(c, "workflow version 1 not found", err.Error())
		return
	}

	if err := h.db.Where("definition_id = ? AND id = ?", id, v2Id).
		First(&v2).Error; err != nil {
		response.Fail(c, "workflow version 2 not found", err.Error())
		return
	}

	// Compare versions and build diff
	diff := buildWorkflowDiff(&v1, &v2)

	response.Success(c, "ok", map[string]interface{}{
		"version1": v1,
		"version2": v2,
		"diff":     diff,
	})
}

// buildWorkflowDiff compares two workflow versions and returns differences.
func buildWorkflowDiff(v1, v2 *models.WorkflowVersion) map[string]interface{} {
	diff := make(map[string]interface{})

	// Compare basic fields
	if v1.Name != v2.Name {
		diff["name"] = map[string]interface{}{
			"old": v1.Name,
			"new": v2.Name,
		}
	}
	if v1.Description != v2.Description {
		diff["description"] = map[string]interface{}{
			"old": v1.Description,
			"new": v2.Description,
		}
	}
	if v1.Status != v2.Status {
		diff["status"] = map[string]interface{}{
			"old": v1.Status,
			"new": v2.Status,
		}
	}

	// Compare nodes
	nodesDiff := compareNodes(v1.Definition.Nodes, v2.Definition.Nodes)
	if len(nodesDiff) > 0 {
		diff["nodes"] = nodesDiff
	}

	// Compare edges
	edgesDiff := compareEdges(v1.Definition.Edges, v2.Definition.Edges)
	if len(edgesDiff) > 0 {
		diff["edges"] = edgesDiff
	}

	// Compare settings (simplified - just check if different)
	if !mapsEqual(v1.Settings, v2.Settings) {
		diff["settings"] = map[string]interface{}{
			"old": v1.Settings,
			"new": v2.Settings,
		}
	}

	// Compare triggers
	if !mapsEqual(v1.Triggers, v2.Triggers) {
		diff["triggers"] = map[string]interface{}{
			"old": v1.Triggers,
			"new": v2.Triggers,
		}
	}

	return diff
}

// compareNodes compares two node arrays and returns differences.
func compareNodes(nodes1, nodes2 []models.WorkflowNodeSchema) map[string]interface{} {
	diff := make(map[string]interface{})
	added := []models.WorkflowNodeSchema{}
	removed := []models.WorkflowNodeSchema{}
	modified := []map[string]interface{}{}

	// Create maps for easier lookup
	nodes1Map := make(map[string]models.WorkflowNodeSchema)
	for _, n := range nodes1 {
		nodes1Map[n.ID] = n
	}

	nodes2Map := make(map[string]models.WorkflowNodeSchema)
	for _, n := range nodes2 {
		nodes2Map[n.ID] = n
	}

	// Find added and modified nodes
	for id, n2 := range nodes2Map {
		if n1, exists := nodes1Map[id]; !exists {
			added = append(added, n2)
		} else if !nodesEqual(n1, n2) {
			modified = append(modified, map[string]interface{}{
				"id":  id,
				"old": n1,
				"new": n2,
			})
		}
	}

	// Find removed nodes
	for id, n1 := range nodes1Map {
		if _, exists := nodes2Map[id]; !exists {
			removed = append(removed, n1)
		}
	}

	if len(added) > 0 {
		diff["added"] = added
	}
	if len(removed) > 0 {
		diff["removed"] = removed
	}
	if len(modified) > 0 {
		diff["modified"] = modified
	}

	return diff
}

// compareEdges compares two edge arrays and returns differences.
func compareEdges(edges1, edges2 []models.WorkflowEdgeSchema) map[string]interface{} {
	diff := make(map[string]interface{})
	added := []models.WorkflowEdgeSchema{}
	removed := []models.WorkflowEdgeSchema{}
	modified := []map[string]interface{}{}

	// Create maps for easier lookup
	edges1Map := make(map[string]models.WorkflowEdgeSchema)
	for _, e := range edges1 {
		edges1Map[e.ID] = e
	}

	edges2Map := make(map[string]models.WorkflowEdgeSchema)
	for _, e := range edges2 {
		edges2Map[e.ID] = e
	}

	// Find added and modified edges
	for id, e2 := range edges2Map {
		if e1, exists := edges1Map[id]; !exists {
			added = append(added, e2)
		} else if !edgesEqual(e1, e2) {
			modified = append(modified, map[string]interface{}{
				"id":  id,
				"old": e1,
				"new": e2,
			})
		}
	}

	// Find removed edges
	for id, e1 := range edges1Map {
		if _, exists := edges2Map[id]; !exists {
			removed = append(removed, e1)
		}
	}

	if len(added) > 0 {
		diff["added"] = added
	}
	if len(removed) > 0 {
		diff["removed"] = removed
	}
	if len(modified) > 0 {
		diff["modified"] = modified
	}

	return diff
}

// nodesEqual checks if two nodes are equal.
func nodesEqual(n1, n2 models.WorkflowNodeSchema) bool {
	return n1.ID == n2.ID &&
		n1.Name == n2.Name &&
		n1.Type == n2.Type &&
		n1.Description == n2.Description
}

// edgesEqual checks if two edges are equal.
func edgesEqual(e1, e2 models.WorkflowEdgeSchema) bool {
	return e1.ID == e2.ID &&
		e1.Source == e2.Source &&
		e1.Target == e2.Target &&
		e1.Type == e2.Type &&
		e1.Condition == e2.Condition
}

// mapsEqual checks if two JSON maps are equal (simplified comparison).
func mapsEqual(m1, m2 models.JSONMap) bool {
	if len(m1) != len(m2) {
		return false
	}
	for k, v1 := range m1 {
		if v2, ok := m2[k]; !ok {
			return false
		} else {
			// Simple comparison - for complex nested structures, might need deep comparison
			if fmt.Sprintf("%v", v1) != fmt.Sprintf("%v", v2) {
				return false
			}
		}
	}
	return true
}

// GetAvailableEventTypes 获取系统中可用的事件类型
func (h *Handlers) GetAvailableEventTypes(c *gin.Context) {
	// 1. 从事件总线获取所有发布过的事件类型
	bus := events.GetEventBus()
	publishedTypes := bus.GetPublishedEventTypes()

	// 2. 从所有工作流定义中提取事件节点发布的事件类型
	publishedFromWorkflows := make(map[string]bool)
	listenedFromWorkflows := make(map[string]bool)

	var workflows []models.WorkflowDefinition
	if err := h.db.Where("status IN ?", []string{"active", "draft"}).Find(&workflows).Error; err == nil {
		for _, wf := range workflows {
			// 提取事件节点发布的事件类型
			for _, node := range wf.Definition.Nodes {
				if node.Type == "event" {
					if props := node.Properties; props != nil {
						if eventType, ok := props["event_type"]; ok && eventType != "" {
							publishedFromWorkflows[eventType] = true
						} else if eventType, ok := props["eventType"]; ok && eventType != "" {
							publishedFromWorkflows[eventType] = true
						}
					}
				}
			}

			// 提取触发器配置中监听的事件类型
			if wf.Triggers != nil {
				var triggerConfig workflowdef.WorkflowTriggerConfig
				if triggerBytes, err := json.Marshal(wf.Triggers); err == nil {
					if err := json.Unmarshal(triggerBytes, &triggerConfig); err == nil {
						if triggerConfig.Event != nil && triggerConfig.Event.Enabled {
							for _, eventType := range triggerConfig.Event.Events {
								if eventType != "" {
									listenedFromWorkflows[eventType] = true
								}
							}
						}
					}
				}
			}
		}
	}

	// 合并所有事件类型
	allEventTypes := make(map[string]map[string]interface{})

	// 添加发布过的事件类型
	for eventType, firstPublished := range publishedTypes {
		if allEventTypes[eventType] == nil {
			allEventTypes[eventType] = make(map[string]interface{})
		}
		allEventTypes[eventType]["first_published"] = firstPublished
		allEventTypes[eventType]["source"] = "published"
	}

	// 添加工作流中定义的事件类型
	for eventType := range publishedFromWorkflows {
		if allEventTypes[eventType] == nil {
			allEventTypes[eventType] = make(map[string]interface{})
		}
		allEventTypes[eventType]["source"] = "workflow_node"
		if _, ok := allEventTypes[eventType]["first_published"]; !ok {
			allEventTypes[eventType]["first_published"] = nil
		}
	}

	// 添加工作流监听的事件类型
	for eventType := range listenedFromWorkflows {
		if allEventTypes[eventType] == nil {
			allEventTypes[eventType] = make(map[string]interface{})
		}
		if source, ok := allEventTypes[eventType]["source"]; ok {
			allEventTypes[eventType]["source"] = fmt.Sprintf("%s,workflow_trigger", source)
		} else {
			allEventTypes[eventType]["source"] = "workflow_trigger"
		}
		if _, ok := allEventTypes[eventType]["first_published"]; !ok {
			allEventTypes[eventType]["first_published"] = nil
		}
	}

	// 转换为列表格式
	result := make([]map[string]interface{}, 0, len(allEventTypes))
	for eventType, info := range allEventTypes {
		result = append(result, map[string]interface{}{
			"type":            eventType,
			"first_published": info["first_published"],
			"source":          info["source"],
		})
	}

	response.Success(c, "ok", map[string]interface{}{
		"event_types": result,
		"total":       len(result),
	})
}
