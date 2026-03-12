package handlers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/LingByte/SoulNexus/internal/models"
	workflowdef "github.com/LingByte/SoulNexus/internal/workflow"
	"github.com/LingByte/SoulNexus/pkg/llm"
	"github.com/LingByte/SoulNexus/pkg/llm/apis"
	"github.com/LingByte/SoulNexus/pkg/logger"
	"go.uber.org/zap"
)

// LoadAssistantToolsToHandler loads assistant tools from database and registers them to LLMHandler
// Each assistant has an independent tool set, and tool names are unique within an assistant
func (h *Handlers) LoadAssistantToolsToHandler(handler *llm.LLMHandler, assistantID int64) error {
	// Get all enabled tools for the assistant
	tools, err := models.GetAssistantTools(h.db, assistantID)
	if err != nil {
		return fmt.Errorf("failed to load assistant tools: %w", err)
	}

	logger.Info("Loading assistant tools",
		zap.Int64("assistantID", assistantID),
		zap.Int("toolCount", len(tools)))

	// Register callback function for each tool
	for _, tool := range tools {
		if !tool.Enabled {
			logger.Debug("Skipping disabled tool",
				zap.String("toolName", tool.Name),
				zap.Int64("assistantID", assistantID))
			continue
		}

		// Parse Parameters JSON
		parameters := json.RawMessage(tool.Parameters)

		// Create callback function
		callback := h.createToolCallback(tool, assistantID)

		// Register tool (tool names are unique within an assistant, different assistants can have tools with the same name)
		handler.RegisterFunctionTool(tool.Name, tool.Description, parameters, callback)

		logger.Info("Registered assistant tool",
			zap.String("toolName", tool.Name),
			zap.Int64("assistantID", assistantID))
	}

	// Load and register workflows that can be called by this assistant
	if err := h.LoadWorkflowToolsToHandler(handler, assistantID); err != nil {
		logger.Warn("Failed to load workflow tools",
			zap.Int64("assistantID", assistantID),
			zap.Error(err))
		// Don't fail the entire loading process if workflow loading fails
	}

	return nil
}

// LoadWorkflowToolsToHandler loads workflows that can be called by the assistant as tools
func (h *Handlers) LoadWorkflowToolsToHandler(handler *llm.LLMHandler, assistantID int64) error {
	// Get all active workflows
	var workflows []models.WorkflowDefinition
	if err := h.db.Where("status = ?", "active").Find(&workflows).Error; err != nil {
		return fmt.Errorf("failed to load workflows: %w", err)
	}

	workflowCount := 0
	for _, wf := range workflows {
		// Parse trigger config
		config, err := workflowdef.ParseTriggerConfig(&wf)
		if err != nil {
			continue
		}

		// Check if this workflow can be called by this assistant
		if !config.CanBeCalledByAssistant(assistantID) {
			continue
		}

		// Build tool parameters from workflow start node inputs
		var parameters json.RawMessage
		var startNode *models.WorkflowNodeSchema
		for i := range wf.Definition.Nodes {
			if wf.Definition.Nodes[i].Type == "start" {
				startNode = &wf.Definition.Nodes[i]
				break
			}
		}

		// Build parameters schema from start node input map
		if startNode != nil && len(startNode.InputMap) > 0 {
			paramsSchema := make(map[string]interface{})
			properties := make(map[string]interface{})
			required := []string{}

			for alias := range startNode.InputMap {
				properties[alias] = map[string]interface{}{
					"type":        "string",
					"description": fmt.Sprintf("Input parameter: %s", alias),
				}
				required = append(required, alias)
			}

			paramsSchema["type"] = "object"
			paramsSchema["properties"] = properties
			if len(required) > 0 {
				paramsSchema["required"] = required
			}

			parameters, _ = json.Marshal(paramsSchema)
		} else {
			// No inputs, use empty parameters
			parameters = json.RawMessage(`{"type":"object","properties":{}}`)
		}

		// Build description
		description := wf.Description
		if description == "" {
			description = fmt.Sprintf("Execute workflow: %s", wf.Name)
		}
		if config.Assistant != nil && config.Assistant.Description != "" {
			description = config.Assistant.Description
		}

		// Create tool name (use workflow slug or name)
		toolName := fmt.Sprintf("workflow_%s", wf.Slug)
		if toolName == "" {
			toolName = fmt.Sprintf("workflow_%d", wf.ID)
		}

		// Create callback
		callback := h.createWorkflowCallback(wf.ID, assistantID)

		// Register as tool
		handler.RegisterFunctionTool(toolName, description, parameters, callback)
		workflowCount++

		logger.Info("Registered workflow as tool",
			zap.String("toolName", toolName),
			zap.Uint("workflowId", wf.ID),
			zap.Int64("assistantID", assistantID))
	}

	logger.Info("Loaded workflow tools",
		zap.Int64("assistantID", assistantID),
		zap.Int("workflowCount", workflowCount))

	return nil
}

// createWorkflowCallback creates a callback function for workflow execution
func (h *Handlers) createWorkflowCallback(workflowID uint, assistantID int64) llm.FunctionToolCallback {
	return func(args map[string]interface{}, llmService interface{}) (string, error) {
		// Use trigger manager to execute workflow
		triggerManager := workflowdef.NewWorkflowTriggerManager(h.db)
		instance, err := triggerManager.TriggerWorkflow(
			workflowID,
			args,
			fmt.Sprintf("assistant:%d", assistantID),
		)

		if err != nil {
			return "", fmt.Errorf("workflow execution failed: %w", err)
		}

		// Format result
		result := fmt.Sprintf("Workflow executed successfully. Instance ID: %d, Status: %s", instance.ID, instance.Status)
		if instance.ResultData != nil {
			if success, ok := instance.ResultData["success"].(bool); ok && success {
				if context, ok := instance.ResultData["context"].(map[string]interface{}); ok {
					// Try to extract meaningful result
					if resultData, ok := context["workflow_result"].(map[string]interface{}); ok {
						resultJSON, _ := json.Marshal(resultData)
						result = fmt.Sprintf("Workflow completed successfully. Result: %s", string(resultJSON))
					}
				}
			}
		}

		return result, nil
	}
}

// createToolCallback creates a callback function for AssistantTool
func (h *Handlers) createToolCallback(tool models.AssistantTool, assistantID int64) llm.FunctionToolCallback {
	return func(args map[string]interface{}, llmService interface{}) (string, error) {
		// If the tool defines Code, different processing logic can be executed based on Code
		if tool.Code != "" {
			return h.executeToolCode(tool, assistantID, args)
		}

		// Default response: return tool name and parameters
		result := fmt.Sprintf("Tool '%s' executed. Args: %v", tool.Name, args)
		logger.Info("Tool executed",
			zap.String("toolName", tool.Name),
			zap.Int64("assistantID", assistantID),
			zap.Any("args", args))

		return result, nil
	}
}

// executeToolCode executes the tool's code logic
// Different processing logic can be implemented here based on the value of tool.Code
// For example: Code can be identifiers like "weather", "calculator", etc., then call corresponding processing functions
// If webhook_url is provided, call the webhook
func (h *Handlers) executeToolCode(tool models.AssistantTool, assistantID int64, args map[string]interface{}) (string, error) {
	// Prioritize checking if there is a webhook URL
	if tool.WebhookURL != "" {
		return h.handleWebhookTool(tool, args)
	}

	// Execute different logic based on Code type
	switch tool.Code {
	case "weather":
		// Can call weather API
		return h.handleWeatherTool(tool, args)
	case "calculator":
		// Can perform calculations
		return h.handleCalculatorTool(tool, args)
	default:
		// Default handling: if there is no webhook and no predefined code, return error
		return "", fmt.Errorf("Tool '%s' has no execution method configured (need to set code or webhookUrl)", tool.Name)
	}
}

// handleWeatherTool handles weather tool
func (h *Handlers) handleWeatherTool(tool models.AssistantTool, args map[string]interface{}) (string, error) {
	location, ok := args["location"].(string)
	if !ok || location == "" {
		return "", fmt.Errorf("location parameter is required")
	}

	unit := "celsius"
	if u, ok := args["unit"].(string); ok && u != "" {
		unit = u
	}

	// Use real weather API service
	weatherManager := apis.NewWeatherManager()
	weatherData, err := weatherManager.GetCurrentWeather(location, unit)
	if err != nil {
		return "", fmt.Errorf("Failed to get weather information: %w", err)
	}

	result := fmt.Sprintf(
		"Current weather in %s:\n"+
			"- Weather condition: %s\n"+
			"- Temperature: %.1f%s\n"+
			"- Humidity: %d%%\n"+
			"- Wind speed: %.1f km/h\n"+
			"- Pressure: %d hPa\n"+
			"- Visibility: %d km\n"+
			"- UV index: %.1f\n"+
			"- Update time: %s",
		weatherData.Location,
		weatherData.Description,
		weatherData.Temperature,
		weatherData.Unit,
		weatherData.Humidity,
		weatherData.WindSpeed,
		weatherData.Pressure,
		weatherData.Visibility,
		weatherData.UVIndex,
		weatherData.Timestamp,
	)

	logger.Info("Weather tool executed",
		zap.String("toolName", tool.Name),
		zap.String("location", location),
		zap.String("unit", unit))

	return result, nil
}

// handleWebhookTool handles Webhook tool
func (h *Handlers) handleWebhookTool(tool models.AssistantTool, args map[string]interface{}) (string, error) {
	if tool.WebhookURL == "" {
		return "", fmt.Errorf("webhook URL not configured")
	}

	// Create HTTP client, set timeout
	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	// Prepare request body
	requestBody := map[string]interface{}{
		"tool": map[string]interface{}{
			"id":          tool.ID,
			"name":        tool.Name,
			"description": tool.Description,
		},
		"args": args,
	}

	jsonData, err := json.Marshal(requestBody)
	if err != nil {
		return "", fmt.Errorf("Failed to serialize request parameters: %w", err)
	}

	// Create HTTP request
	req, err := http.NewRequest("POST", tool.WebhookURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return "", fmt.Errorf("Failed to create HTTP request: %w", err)
	}

	// Set request headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "LingEcho-Tool-Webhook/1.0")

	// Send request
	logger.Info("Calling webhook",
		zap.String("toolName", tool.Name),
		zap.String("webhookURL", tool.WebhookURL),
		zap.Any("args", args))

	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("Failed to call webhook: %w", err)
	}
	defer resp.Body.Close()

	// Read response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("Failed to read webhook response: %w", err)
	}

	// Check HTTP status code
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("webhook returned error status code %d: %s", resp.StatusCode, string(body))
	}

	// Try to parse JSON response
	var webhookResponse struct {
		Result  string `json:"result"`
		Error   string `json:"error,omitempty"`
		Message string `json:"message,omitempty"`
	}

	if err := json.Unmarshal(body, &webhookResponse); err == nil {
		// If parsing succeeds, prioritize returning result field
		if webhookResponse.Error != "" {
			return "", fmt.Errorf("webhook returned error: %s", webhookResponse.Error)
		}
		if webhookResponse.Result != "" {
			return webhookResponse.Result, nil
		}
		if webhookResponse.Message != "" {
			return webhookResponse.Message, nil
		}
	}

	// If unable to parse JSON, directly return response body as string
	result := strings.TrimSpace(string(body))
	if result == "" {
		return "", fmt.Errorf("webhook returned empty response")
	}

	logger.Info("Webhook executed successfully",
		zap.String("toolName", tool.Name),
		zap.String("webhookURL", tool.WebhookURL),
		zap.Int("statusCode", resp.StatusCode))

	return result, nil
}

// handleCalculatorTool handles calculator tool
func (h *Handlers) handleCalculatorTool(tool models.AssistantTool, args map[string]interface{}) (string, error) {
	expression, ok := args["expression"].(string)
	if !ok || expression == "" {
		return "", fmt.Errorf("expression parameter is required")
	}

	// Use simple expression evaluation
	// Note: Here we use a simple parser that only supports basic arithmetic operations
	// For more complex expressions, third-party libraries like go-expr or govaluate can be used
	result, err := evaluateSimpleExpression(expression)
	if err != nil {
		return "", fmt.Errorf("Calculation failed: %w", err)
	}

	logger.Info("Calculator tool executed",
		zap.String("toolName", tool.Name),
		zap.String("expression", expression),
		zap.Float64("result", result))

	return fmt.Sprintf("Calculation result: %s = %.2f", expression, result), nil
}

// evaluateSimpleExpression calculates simple mathematical expressions
// Supports: +, -, *, /, (), numbers (including decimals)
func evaluateSimpleExpression(expr string) (float64, error) {
	// Remove all spaces
	expr = strings.ReplaceAll(expr, " ", "")
	if expr == "" {
		return 0, fmt.Errorf("Expression is empty")
	}

	// Use Go's expr package or a simple recursive descent parser
	// Implement a simple version here that supports basic operations
	return evaluateExpression(expr)
}

// evaluateExpression recursively calculates expressions
func evaluateExpression(expr string) (float64, error) {
	// Handle parentheses
	for {
		left := strings.LastIndex(expr, "(")
		if left == -1 {
			break
		}
		right := strings.Index(expr[left:], ")")
		if right == -1 {
			return 0, fmt.Errorf("Mismatched parentheses")
		}
		right += left

		// Calculate expression inside parentheses
		innerResult, err := evaluateExpression(expr[left+1 : right])
		if err != nil {
			return 0, err
		}

		// Replace parentheses expression with result
		expr = expr[:left] + fmt.Sprintf("%.10f", innerResult) + expr[right+1:]
	}

	// Handle multiplication and division
	for {
		mulIndex := strings.Index(expr, "*")
		divIndex := strings.Index(expr, "/")

		var opIndex int
		var op byte
		if mulIndex != -1 && divIndex != -1 {
			if mulIndex < divIndex {
				opIndex = mulIndex
				op = '*'
			} else {
				opIndex = divIndex
				op = '/'
			}
		} else if mulIndex != -1 {
			opIndex = mulIndex
			op = '*'
		} else if divIndex != -1 {
			opIndex = divIndex
			op = '/'
		} else {
			break
		}

		// Find left operand
		leftStart := findLeftOperand(expr, opIndex)
		// Find right operand
		rightEnd := findRightOperand(expr, opIndex)

		leftVal, err := strconv.ParseFloat(expr[leftStart:opIndex], 64)
		if err != nil {
			return 0, fmt.Errorf("Invalid left operand: %s", expr[leftStart:opIndex])
		}

		rightVal, err := strconv.ParseFloat(expr[opIndex+1:rightEnd], 64)
		if err != nil {
			return 0, fmt.Errorf("Invalid right operand: %s", expr[opIndex+1:rightEnd])
		}

		var result float64
		if op == '*' {
			result = leftVal * rightVal
		} else {
			if rightVal == 0 {
				return 0, fmt.Errorf("Division by zero")
			}
			result = leftVal / rightVal
		}

		// Replace expression
		expr = expr[:leftStart] + fmt.Sprintf("%.10f", result) + expr[rightEnd:]
	}

	// Handle addition and subtraction
	for {
		plusIndex := strings.Index(expr, "+")
		minusIndex := strings.Index(expr, "-")

		// Skip leading negative sign
		if minusIndex == 0 {
			minusIndex = strings.Index(expr[1:], "-")
			if minusIndex != -1 {
				minusIndex++
			}
		}

		var opIndex int
		var op byte
		if plusIndex != -1 && minusIndex != -1 {
			if plusIndex < minusIndex {
				opIndex = plusIndex
				op = '+'
			} else {
				opIndex = minusIndex
				op = '-'
			}
		} else if plusIndex != -1 {
			opIndex = plusIndex
			op = '+'
		} else if minusIndex != -1 && minusIndex > 0 {
			opIndex = minusIndex
			op = '-'
		} else {
			break
		}

		// Find left operand
		leftStart := findLeftOperand(expr, opIndex)
		// Find right operand
		rightEnd := findRightOperand(expr, opIndex)

		leftVal, err := strconv.ParseFloat(expr[leftStart:opIndex], 64)
		if err != nil {
			return 0, fmt.Errorf("Invalid left operand: %s", expr[leftStart:opIndex])
		}

		rightVal, err := strconv.ParseFloat(expr[opIndex+1:rightEnd], 64)
		if err != nil {
			return 0, fmt.Errorf("Invalid right operand: %s", expr[opIndex+1:rightEnd])
		}

		var result float64
		if op == '+' {
			result = leftVal + rightVal
		} else {
			result = leftVal - rightVal
		}

		// Replace expression
		expr = expr[:leftStart] + fmt.Sprintf("%.10f", result) + expr[rightEnd:]
	}

	// Final result
	result, err := strconv.ParseFloat(expr, 64)
	if err != nil {
		return 0, fmt.Errorf("Invalid expression: %s", expr)
	}

	return result, nil
}

// findLeftOperand finds the starting position of the left operand
func findLeftOperand(expr string, opIndex int) int {
	start := opIndex - 1
	for start >= 0 && (expr[start] >= '0' && expr[start] <= '9' || expr[start] == '.' || expr[start] == '-') {
		start--
	}
	return start + 1
}

// findRightOperand finds the ending position of the right operand
func findRightOperand(expr string, opIndex int) int {
	end := opIndex + 1
	// Handle negative sign
	if end < len(expr) && expr[end] == '-' {
		end++
	}
	for end < len(expr) && (expr[end] >= '0' && expr[end] <= '9' || expr[end] == '.') {
		end++
	}
	return end
}

// ReloadAssistantTools reloads assistant tools
// Note: Since LLMHandler is created for each request, this method is mainly used for testing or management scenarios
func (h *Handlers) ReloadAssistantTools(handler *llm.LLMHandler, assistantID int64) error {
	// Directly load new tools, if name conflicts occur, they will be overwritten
	return h.LoadAssistantToolsToHandler(handler, assistantID)
}
