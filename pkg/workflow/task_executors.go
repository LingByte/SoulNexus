package workflow

// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/LingByte/SoulNexus/pkg/utils/xhttp"
)

// TaskExecutor defines the interface for task executors
type TaskExecutor interface {
	Execute(ctx *WorkflowContext, config map[string]interface{}, inputs map[string]interface{}) (map[string]interface{}, error)
	GetTaskType() string
}

// TaskExecutorRegistry manages all task executors
var taskExecutorRegistry = make(map[string]TaskExecutor)

// RegisterTaskExecutor registers a task executor
func RegisterTaskExecutor(executor TaskExecutor) {
	taskExecutorRegistry[executor.GetTaskType()] = executor
}

// GetTaskExecutor gets a task executor by type
func GetTaskExecutor(taskType string) TaskExecutor {
	return taskExecutorRegistry[taskType]
}

// HTTPTaskExecutor handles HTTP request tasks
type HTTPTaskExecutor struct{}

func (e *HTTPTaskExecutor) GetTaskType() string {
	return "http"
}

func (e *HTTPTaskExecutor) Execute(ctx *WorkflowContext, config map[string]interface{}, inputs map[string]interface{}) (map[string]interface{}, error) {
	if ctx != nil {
		ctx.AddLog("info", "Executing HTTP task", "", "")
	}

	// Parse configuration
	method := "GET"
	if m, ok := config["method"].(string); ok {
		method = strings.ToUpper(m)
	}

	url := ""
	if u, ok := config["url"].(string); ok {
		url = u
	} else {
		return nil, fmt.Errorf("HTTP task requires 'url' configuration")
	}

	// Resolve URL from context if it contains template variables
	url = resolveTemplate(url, inputs, ctx)

	timeout := 10 * time.Second
	if t, ok := config["timeout"].(string); ok {
		if parsed, err := time.ParseDuration(t); err == nil {
			timeout = parsed
		}
	} else if t, ok := config["timeout"].(float64); ok {
		timeout = time.Duration(t) * time.Second
	}

	// Parse headers
	headers := make(map[string]string)
	if h, ok := config["headers"].(map[string]interface{}); ok {
		for k, v := range h {
			if str, ok := v.(string); ok {
				headers[k] = resolveTemplate(str, inputs, ctx)
			}
		}
	}

	// Execute HTTP request
	var respBody []byte
	var err error
	var statusCode int

	switch method {
	case "GET":
		// Build query parameters from inputs or config
		params := make(map[string]interface{})
		if p, ok := config["params"].(map[string]interface{}); ok {
			params = p
		} else {
			// Use inputs as query parameters
			params = inputs
		}
		// Resolve template variables in params
		resolvedParams := make(map[string]interface{})
		for k, v := range params {
			if str, ok := v.(string); ok {
				resolvedParams[k] = resolveTemplate(str, inputs, ctx)
			} else {
				resolvedParams[k] = v
			}
		}
		headerOptions := make([]*xhttp.HeaderOption, 0)
		for k, v := range headers {
			headerOptions = append(headerOptions, &xhttp.HeaderOption{Key: k, Value: v})
		}
		respBody, err = xhttp.Get(url, resolvedParams, headerOptions...)
		if err == nil {
			statusCode = 200 // xhttp.Get doesn't return status code, assume 200 if no error
		}

	case "POST", "PUT", "PATCH":
		// Build request body
		var body interface{}
		if b, ok := config["body"].(map[string]interface{}); ok {
			body = b
		} else if b, ok := config["body"].(string); ok {
			// Try to parse as JSON, or use as template
			var parsed map[string]interface{}
			if err := json.Unmarshal([]byte(b), &parsed); err == nil {
				body = parsed
			} else {
				// Use as template string
				body = resolveTemplate(b, inputs, ctx)
			}
		} else {
			// Use inputs as body
			body = inputs
		}

		// Resolve template variables in body
		body = resolveTemplateInValue(body, inputs, ctx)

		// Convert body to JSON
		jsonBody, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request body: %w", err)
		}

		// Create custom HTTP client with timeout
		client := &http.Client{Timeout: timeout}
		var req *http.Request
		req, err = http.NewRequest(method, url, bytes.NewBuffer(jsonBody))
		if err != nil {
			return nil, fmt.Errorf("failed to create request: %w", err)
		}

		// Set headers
		for k, v := range headers {
			req.Header.Set(k, v)
		}
		if _, ok := headers["Content-Type"]; !ok {
			req.Header.Set("Content-Type", "application/json")
		}

		var resp *http.Response
		resp, err = client.Do(req)
		if err != nil {
			return nil, fmt.Errorf("HTTP request failed: %w", err)
		}
		defer resp.Body.Close()

		statusCode = resp.StatusCode
		respBody, err = io.ReadAll(resp.Body)
		if err != nil {
			return nil, fmt.Errorf("failed to read response: %w", err)
		}

	case "DELETE":
		client := &http.Client{Timeout: timeout}
		req, err := http.NewRequest("DELETE", url, nil)
		if err != nil {
			return nil, fmt.Errorf("failed to create request: %w", err)
		}

		// Set headers
		for k, v := range headers {
			req.Header.Set(k, v)
		}

		var resp *http.Response
		resp, err = client.Do(req)
		if err != nil {
			return nil, fmt.Errorf("HTTP request failed: %w", err)
		}
		defer resp.Body.Close()

		statusCode = resp.StatusCode
		respBody, err = io.ReadAll(resp.Body)
		if err != nil {
			return nil, fmt.Errorf("failed to read response: %w", err)
		}

	default:
		return nil, fmt.Errorf("unsupported HTTP method: %s", method)
	}

	if err != nil {
		if ctx != nil {
			ctx.AddLog("error", fmt.Sprintf("HTTP request failed: %v", err), "", "")
		}
		return nil, err
	}

	// Parse response
	var responseData interface{}
	if len(respBody) > 0 {
		if err := json.Unmarshal(respBody, &responseData); err != nil {
			// If not JSON, return as string
			responseData = string(respBody)
		}
	}

	result := map[string]interface{}{
		"statusCode": statusCode,
		"body":       responseData,
		"raw":        string(respBody),
		"response": map[string]interface{}{
			"statusCode": statusCode,
			"body":       responseData,
			"raw":        string(respBody),
		},
	}

	// Also store response data in context with node-specific key for easy access
	if ctx != nil {
		// Store under a key that includes the task name/ID for easy reference
		// This allows accessing as: context.response or context.{nodeId}.response
		if ctx.NodeData == nil {
			ctx.NodeData = make(map[string]interface{})
		}
		// Store the full response structure
		ctx.NodeData["response"] = result["response"]
		// Also store body directly for convenience
		if bodyMap, ok := responseData.(map[string]interface{}); ok {
			// If body is a map, merge its keys into context for direct access
			for k, v := range bodyMap {
				ctx.NodeData[k] = v
			}
		}

		ctx.AddLog("success", fmt.Sprintf("HTTP %s request completed with status %d", method, statusCode), "", "")
		// Log response structure for debugging
		if bodyMap, ok := responseData.(map[string]interface{}); ok {
			ctx.AddLog("debug", fmt.Sprintf("Response body keys: %v", getMapKeys(bodyMap)), "", "")
		}
	}

	return result, nil
}

// DataTransformTaskExecutor handles data transformation tasks
type DataTransformTaskExecutor struct{}

func (e *DataTransformTaskExecutor) GetTaskType() string {
	return "transform"
}

func (e *DataTransformTaskExecutor) Execute(ctx *WorkflowContext, config map[string]interface{}, inputs map[string]interface{}) (map[string]interface{}, error) {
	if ctx != nil {
		ctx.AddLog("info", "Executing data transform task", "", "")
	}

	operation := "copy"
	if op, ok := config["operation"].(string); ok {
		operation = op
	}

	result := make(map[string]interface{})

	switch operation {
	case "copy":
		// Simply copy inputs to outputs
		for k, v := range inputs {
			result[k] = v
		}

	case "select":
		// Select specific fields
		if fields, ok := config["fields"].([]interface{}); ok {
			for _, field := range fields {
				if fieldStr, ok := field.(string); ok {
					if val, ok := inputs[fieldStr]; ok {
						result[fieldStr] = val
					}
				}
			}
		}

	case "map":
		// Map fields to new names
		if mapping, ok := config["mapping"].(map[string]interface{}); ok {
			for newKey, oldKey := range mapping {
				if oldKeyStr, ok := oldKey.(string); ok {
					if val, ok := inputs[oldKeyStr]; ok {
						result[newKey] = val
					}
				}
			}
		}

	case "merge":
		// Merge with additional data
		for k, v := range inputs {
			result[k] = v
		}
		if additional, ok := config["data"].(map[string]interface{}); ok {
			for k, v := range additional {
				result[k] = resolveTemplateInValue(v, inputs, ctx)
			}
		}

	case "filter":
		// Filter based on condition (simple key-value match for now)
		if condition, ok := config["condition"].(map[string]interface{}); ok {
			for k, v := range inputs {
				// Simple filter: include if key exists in condition and value matches
				if condVal, ok := condition[k]; ok {
					if condVal == v {
						result[k] = v
					}
				} else {
					result[k] = v
				}
			}
		} else {
			// No condition, copy all
			for k, v := range inputs {
				result[k] = v
			}
		}

	default:
		return nil, fmt.Errorf("unsupported transform operation: %s", operation)
	}

	if ctx != nil {
		ctx.AddLog("success", fmt.Sprintf("Data transform completed: %s", operation), "", "")
	}

	return result, nil
}

// SetVariableTaskExecutor handles variable setting tasks
type SetVariableTaskExecutor struct{}

func (e *SetVariableTaskExecutor) GetTaskType() string {
	return "set_variable"
}

func (e *SetVariableTaskExecutor) Execute(ctx *WorkflowContext, config map[string]interface{}, inputs map[string]interface{}) (map[string]interface{}, error) {
	if ctx != nil {
		ctx.AddLog("info", "Executing set variable task", "", "")
	}

	if ctx == nil {
		return nil, fmt.Errorf("workflow context is required for set_variable task")
	}

	// Get variables to set
	variables := make(map[string]interface{})
	if vars, ok := config["variables"].(map[string]interface{}); ok {
		variables = vars
	} else {
		// Support simple key-value pairs
		for k, v := range config {
			if k != "task_type" && k != "type" {
				variables[k] = resolveTemplateInValue(v, inputs, ctx)
			}
		}
	}

	// Set variables in context
	if ctx.NodeData == nil {
		ctx.NodeData = make(map[string]interface{})
	}
	for k, v := range variables {
		ctx.NodeData[k] = v
		if ctx != nil {
			ctx.AddLog("debug", fmt.Sprintf("Set variable: %s = %v", k, v), "", "")
		}
	}

	result := map[string]interface{}{
		"variables": variables,
	}

	if ctx != nil {
		ctx.AddLog("success", fmt.Sprintf("Set %d variable(s)", len(variables)), "", "")
	}

	return result, nil
}

// DelayTaskExecutor handles delay/wait tasks
type DelayTaskExecutor struct{}

func (e *DelayTaskExecutor) GetTaskType() string {
	return "delay"
}

func (e *DelayTaskExecutor) Execute(ctx *WorkflowContext, config map[string]interface{}, inputs map[string]interface{}) (map[string]interface{}, error) {
	if ctx != nil {
		ctx.AddLog("info", "Executing delay task", "", "")
	}

	duration := 1 * time.Second
	if d, ok := config["duration"].(string); ok {
		if parsed, err := time.ParseDuration(d); err == nil {
			duration = parsed
		}
	} else if d, ok := config["duration"].(float64); ok {
		duration = time.Duration(d) * time.Second
	} else if d, ok := config["duration"].(int); ok {
		duration = time.Duration(d) * time.Second
	}

	if ctx != nil {
		ctx.AddLog("debug", fmt.Sprintf("Waiting for %v", duration), "", "")
	}

	time.Sleep(duration)

	result := map[string]interface{}{
		"delayed":  true,
		"duration": duration.String(),
	}

	if ctx != nil {
		ctx.AddLog("success", fmt.Sprintf("Delay completed: %v", duration), "", "")
	}

	return result, nil
}

// LogTaskExecutor handles logging tasks
type LogTaskExecutor struct{}

func (e *LogTaskExecutor) GetTaskType() string {
	return "log"
}

func (e *LogTaskExecutor) Execute(ctx *WorkflowContext, config map[string]interface{}, inputs map[string]interface{}) (map[string]interface{}, error) {
	level := "info"
	if l, ok := config["level"].(string); ok {
		level = l
	}

	message := ""
	if m, ok := config["message"].(string); ok {
		message = resolveTemplate(m, inputs, ctx)
	} else {
		// Use inputs as message
		messageBytes, _ := json.Marshal(inputs)
		message = string(messageBytes)
	}

	if ctx != nil {
		ctx.AddLog(level, message, "", "")
	} else {
		fmt.Printf("[%s] %s\n", level, message)
	}

	result := map[string]interface{}{
		"logged":  true,
		"level":   level,
		"message": message,
	}

	return result, nil
}

// Helper functions

// getMapKeys returns all keys from a map (for debugging)
func getMapKeys(m map[string]interface{}) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}

// resolveTemplate resolves template variables in a string
// Supports {{variable}} or {{context.key}} syntax
func resolveTemplate(template string, inputs map[string]interface{}, ctx *WorkflowContext) string {
	result := template
	// Simple template replacement: {{key}} or {{context.key}}
	for {
		start := strings.Index(result, "{{")
		if start == -1 {
			break
		}
		end := strings.Index(result[start:], "}}")
		if end == -1 {
			break
		}
		end += start

		varName := strings.TrimSpace(result[start+2 : end])
		var value interface{}
		var found bool

		// Check if it's a context variable
		if strings.HasPrefix(varName, "context.") {
			key := strings.TrimPrefix(varName, "context.")
			if ctx != nil {
				value, found = ctx.ResolveValue("context." + key)
			}
		} else if strings.HasPrefix(varName, "parameters.") {
			key := strings.TrimPrefix(varName, "parameters.")
			if ctx != nil {
				value, found = ctx.ResolveValue("parameters." + key)
			}
		} else {
			// Check inputs first
			value, found = inputs[varName]
			if !found && ctx != nil {
				value, found = ctx.ResolveValue(varName)
			}
		}

		var strValue string
		if found {
			switch v := value.(type) {
			case string:
				strValue = v
			case int, int64, float64:
				strValue = fmt.Sprintf("%v", v)
			case bool:
				strValue = strconv.FormatBool(v)
			default:
				if jsonBytes, err := json.Marshal(v); err == nil {
					strValue = string(jsonBytes)
				} else {
					strValue = fmt.Sprintf("%v", v)
				}
			}
		} else {
			strValue = ""
		}

		result = result[:start] + strValue + result[end+2:]
	}

	return result
}

// resolveTemplateInValue resolves template variables in any value type
func resolveTemplateInValue(value interface{}, inputs map[string]interface{}, ctx *WorkflowContext) interface{} {
	switch v := value.(type) {
	case string:
		return resolveTemplate(v, inputs, ctx)
	case map[string]interface{}:
		result := make(map[string]interface{})
		for k, val := range v {
			result[k] = resolveTemplateInValue(val, inputs, ctx)
		}
		return result
	case []interface{}:
		result := make([]interface{}, len(v))
		for i, val := range v {
			result[i] = resolveTemplateInValue(val, inputs, ctx)
		}
		return result
	default:
		return v
	}
}

// Initialize task executors
func init() {
	RegisterTaskExecutor(&HTTPTaskExecutor{})
	RegisterTaskExecutor(&DataTransformTaskExecutor{})
	RegisterTaskExecutor(&SetVariableTaskExecutor{})
	RegisterTaskExecutor(&DelayTaskExecutor{})
	RegisterTaskExecutor(&LogTaskExecutor{})
}
