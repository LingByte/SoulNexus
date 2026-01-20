package prompt

import (
	"context"
	"errors"
	"testing"

	"github.com/LingByte/SoulNexus/pkg/utils"
	"github.com/getkin/kin-openapi/openapi3"
	"github.com/stretchr/testify/assert"
)

// mockToolHandler is a mock implementation of a toolHandler
func mockToolHandler(expectedResult *CallToolResult, expectedError error) toolHandler {
	return func(ctx context.Context, req *CallToolRequest) (*CallToolResult, error) {
		return expectedResult, expectedError
	}
}

func TestRegisterAndGetTool(t *testing.T) {
	manager := newToolManager()

	tool := &Tool{
		Name: "echo",
		InputSchema: &openapi3.Schema{
			Type: &openapi3.Types{openapi3.TypeObject},
		},
	}

	handler := mockToolHandler(&CallToolResult{
		Content: []Content{NewTextContent("ok")},
	}, nil)

	manager.registerTool(tool, handler)

	retrieved, exists := manager.getTool("echo")
	assert.True(t, exists)
	assert.Equal(t, "echo", retrieved.Name)
}

func TestHandleListTools(t *testing.T) {
	manager := newToolManager()

	manager.registerTool(&Tool{
		Name: "tool1",
		InputSchema: &openapi3.Schema{
			Type: &openapi3.Types{openapi3.TypeObject},
		},
	}, nil)

	manager.registerTool(&Tool{
		Name: "tool2",
		InputSchema: &openapi3.Schema{
			Type: &openapi3.Types{openapi3.TypeObject},
		},
	}, nil)

	resp, err := manager.handleListTools(context.Background(), &JSONRPCRequest{}, *utils.NewSession())
	assert.NoError(t, err)

	result := resp.(ListToolsResult)
	assert.Len(t, result.Tools, 2)
}

func TestHandleCallTool_Success(t *testing.T) {
	manager := newToolManager()

	manager.registerTool(&Tool{
		Name: "echo",
		InputSchema: &openapi3.Schema{
			Type: &openapi3.Types{openapi3.TypeObject},
		},
	}, mockToolHandler(&CallToolResult{
		Content: []Content{NewTextContent("echoed")},
	}, nil))

	req := &JSONRPCRequest{
		ID: "1",
		Params: map[string]interface{}{
			"name": "echo",
			"arguments": map[string]interface{}{
				"message": "hi",
			},
		},
	}

	resp, err := manager.handleCallTool(context.Background(), req, *utils.NewSession())
	assert.NoError(t, err)

	result := resp.(*CallToolResult)
	assert.False(t, result.IsError)
	assert.Equal(t, "echoed", result.Content[0].(TextContent).Text)
}

func TestHandleCallTool_InvalidParams(t *testing.T) {
	manager := newToolManager()

	req := &JSONRPCRequest{
		ID:     "2",
		Params: "not-a-map",
	}

	resp, _ := manager.handleCallTool(context.Background(), req, *utils.NewSession())
	assert.Contains(t, resp.(*JSONRPCError).Error.Message, "invalid parameters")
}

func TestHandleCallTool_MissingName(t *testing.T) {
	manager := newToolManager()

	req := &JSONRPCRequest{
		ID:     "3",
		Params: map[string]interface{}{},
	}

	resp, _ := manager.handleCallTool(context.Background(), req, *utils.NewSession())
	assert.Contains(t, resp.(*JSONRPCError).Error.Message, "missing tool name")
}

func TestHandleCallTool_ToolNotFound(t *testing.T) {
	manager := newToolManager()

	req := &JSONRPCRequest{
		ID: "4",
		Params: map[string]interface{}{
			"name": "nonexistent",
		},
	}

	resp, _ := manager.handleCallTool(context.Background(), req, *utils.NewSession())
	assert.Contains(t, resp.(*JSONRPCError).Error.Message, "tool not found")
}

func TestHandleCallTool_HandlerError(t *testing.T) {
	manager := newToolManager()

	manager.registerTool(&Tool{
		Name: "fail",
		InputSchema: &openapi3.Schema{
			Type: &openapi3.Types{openapi3.TypeObject},
		},
	}, mockToolHandler(nil, errors.New("boom")))

	req := &JSONRPCRequest{
		ID: "5",
		Params: map[string]interface{}{
			"name": "fail",
		},
	}

	resp, _ := manager.handleCallTool(context.Background(), req, *utils.NewSession())
	assert.Contains(t, resp.(*JSONRPCError).Error.Message, "tool execution failed")
}

func TestToolManager_WithServerProvider(t *testing.T) {
	manager := newToolManager()

	mockProvider := &mockServerProvider{}
	manager.withServerProvider(mockProvider)

	assert.Equal(t, mockProvider, manager.serverProvider)
}

func TestToolManager_WithToolListFilter(t *testing.T) {
	manager := newToolManager()

	filter := func(ctx context.Context, tools []*Tool) []*Tool {
		return tools[:1] // Return only first tool
	}
	manager.withToolListFilter(filter)

	assert.NotNil(t, manager.toolListFilter)
}

func TestToolManager_WithMethodNameModifier(t *testing.T) {
	manager := newToolManager()

	modifier := func(ctx context.Context, method, toolName string) {
		// Do nothing
	}
	manager.withMethodNameModifier(modifier)

	assert.NotNil(t, manager.methodNameModifier)
}

func TestHandleListTools_WithFilter(t *testing.T) {
	manager := newToolManager()

	manager.registerTool(&Tool{
		Name: "tool1",
		InputSchema: &openapi3.Schema{
			Type: &openapi3.Types{openapi3.TypeObject},
		},
	}, nil)

	manager.registerTool(&Tool{
		Name: "tool2",
		InputSchema: &openapi3.Schema{
			Type: &openapi3.Types{openapi3.TypeObject},
		},
	}, nil)

	filter := func(ctx context.Context, tools []*Tool) []*Tool {
		// Return only first tool
		return tools[:1]
	}
	manager.withToolListFilter(filter)

	resp, err := manager.handleListTools(context.Background(), &JSONRPCRequest{}, *utils.NewSession())
	assert.NoError(t, err)

	result := resp.(ListToolsResult)
	assert.Len(t, result.Tools, 1)
}

func TestHandleCallTool_WithProgressToken(t *testing.T) {
	manager := newToolManager()

	manager.registerTool(&Tool{
		Name: "test",
		InputSchema: &openapi3.Schema{
			Type: &openapi3.Types{openapi3.TypeObject},
		},
	}, mockToolHandler(&CallToolResult{
		Content: []Content{NewTextContent("ok")},
	}, nil))

	req := &JSONRPCRequest{
		ID: "1",
		Params: map[string]interface{}{
			"name": "test",
			"_meta": map[string]interface{}{
				"progressToken": "token123",
			},
		},
	}

	resp, err := manager.handleCallTool(context.Background(), req, *utils.NewSession())
	assert.NoError(t, err)
	assert.NotNil(t, resp)
}

func TestHandleCallTool_InvalidArguments(t *testing.T) {
	manager := newToolManager()

	manager.registerTool(&Tool{
		Name: "test",
		InputSchema: &openapi3.Schema{
			Type: &openapi3.Types{openapi3.TypeObject},
		},
	}, nil)

	req := &JSONRPCRequest{
		ID: "1",
		Params: map[string]interface{}{
			"name":      "test",
			"arguments": "not-a-map",
		},
	}

	resp, _ := manager.handleCallTool(context.Background(), req, *utils.NewSession())
	errResp, ok := resp.(*JSONRPCError)
	assert.True(t, ok)
	assert.Equal(t, ErrCodeInvalidParams, errResp.Error.Code)
}

func TestGetTools(t *testing.T) {
	manager := newToolManager()

	manager.registerTool(&Tool{
		Name: "tool1",
		InputSchema: &openapi3.Schema{
			Type: &openapi3.Types{openapi3.TypeObject},
		},
	}, nil)

	tools := manager.getTools("")
	assert.Len(t, tools, 1)
	assert.Equal(t, "tool1", tools[0].Name)
}

func TestRegisterTool_NilTool(t *testing.T) {
	manager := newToolManager()
	manager.registerTool(nil, nil)

	tools := manager.getTools("")
	assert.Empty(t, tools)
}

func TestRegisterTool_EmptyName(t *testing.T) {
	manager := newToolManager()
	manager.registerTool(&Tool{
		Name: "",
		InputSchema: &openapi3.Schema{
			Type: &openapi3.Types{openapi3.TypeObject},
		},
	}, nil)

	tools := manager.getTools("")
	assert.Empty(t, tools)
}

// mockServerProvider is a mock implementation of serverProvider
type mockServerProvider struct{}

func (m *mockServerProvider) withContext(ctx context.Context) context.Context {
	return ctx
}
