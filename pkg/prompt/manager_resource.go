package prompt

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/LingByte/SoulNexus/pkg/utils"
)

// resourceManager 管理资源
// 资源功能遵循这些启用机制:
// 1.默认情况下，资源功能是禁用的
// 2.注册第一个资源时，会自动启用资源功能，而不
// 附加配置
// 3.当启用资源功能但不存在资源时，ListResources将返回一个空的
// 资源列表而不是错误
// 4.客户端可以通过功能来确定服务器是否支持资源功能
// 初始化响应中的字段
// 这种设计简化了API的使用，不需要显式的配置参数来
// 启用或禁用资源功能。
type resourceManager struct {
	// Resource mapping table
	resources map[string]*registeredResource

	// Resource template mapping table
	templates map[string]*registerResourceTemplate

	// Mutex
	mu sync.RWMutex

	// Subscriber mapping table
	subscribers map[string][]chan *JSONRPCNotification

	// Subscriber mutex
	subMu sync.RWMutex

	// Order of resources
	resourcesOrder []string
}

// newResourceManager 创建新的资源管理器
// 注意:简单地创建资源管理器不会启用资源功能，
// 仅在添加第一个资源时启用。
func newResourceManager() *resourceManager {
	return &resourceManager{
		resources:   make(map[string]*registeredResource),
		templates:   make(map[string]*registerResourceTemplate),
		subscribers: make(map[string][]chan *JSONRPCNotification),
	}
}

// registerResource registers a resource
func (m *resourceManager) registerResource(resource *Resource, handler resourceHandler) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if resource == nil || resource.URI == "" {
		return
	}

	if _, exists := m.resources[resource.URI]; !exists {
		// Only add to order slice if it's a new resource
		m.resourcesOrder = append(m.resourcesOrder, resource.URI)
	}

	m.resources[resource.URI] = &registeredResource{
		Resource: resource,
		Handler:  handler,
	}
}

// registerTemplate registers a resource template
func (m *resourceManager) registerTemplate(template *ResourceTemplate, handler resourceTemplateHandler) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if template == nil {
		return fmt.Errorf("template cannot be nil")
	}

	if template.Name == "" {
		return fmt.Errorf("template name cannot be empty")
	}

	if template.URITemplate == nil {
		return fmt.Errorf("template URI cannot be empty")
	}

	if _, exists := m.templates[template.Name]; exists {
		return fmt.Errorf("template %s already exists", template.Name)
	}

	m.templates[template.Name] = &registerResourceTemplate{
		resourceTemplate: template,
		Handler:          handler,
	}

	return nil
}

// getResource retrieves a resource
func (m *resourceManager) getResource(uri string) (*Resource, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	registeredResource, exists := m.resources[uri]
	if !exists {
		return nil, false
	}
	return registeredResource.Resource, true
}

// getResources retrieves all resources
func (m *resourceManager) getResources() []*Resource {
	m.mu.RLock()
	defer m.mu.RUnlock()

	resources := make([]*Resource, 0, len(m.resources))
	for _, registeredResource := range m.resources {
		resources = append(resources, registeredResource.Resource)
	}
	return resources
}

// getTemplates retrieves all resource templates
func (m *resourceManager) getTemplates() []*ResourceTemplate {
	m.mu.RLock()
	defer m.mu.RUnlock()

	templates := make([]*ResourceTemplate, 0, len(m.templates))
	for _, template := range m.templates {
		templates = append(templates, template.resourceTemplate)
	}
	return templates
}

// subscribe subscribes to resource updates
func (m *resourceManager) subscribe(uri string) chan *JSONRPCNotification {
	m.subMu.Lock()
	defer m.subMu.Unlock()

	ch := make(chan *JSONRPCNotification, 10)
	m.subscribers[uri] = append(m.subscribers[uri], ch)
	return ch
}

// unsubscribe cancels a subscription
func (m *resourceManager) unsubscribe(uri string, ch chan *JSONRPCNotification) {
	m.subMu.Lock()
	defer m.subMu.Unlock()

	subs := m.subscribers[uri]
	for i, sub := range subs {
		if sub == ch {
			close(ch)
			subs = append(subs[:i], subs[i+1:]...)
			break
		}
	}
	if len(subs) == 0 {
		delete(m.subscribers, uri)
	} else {
		m.subscribers[uri] = subs
	}
}

// notifyUpdate notifies about resource updates
func (m *resourceManager) notifyUpdate(uri string) {
	m.subMu.RLock()
	subs := m.subscribers[uri]
	m.subMu.RUnlock()

	// Create jsonrpcNotification params with correct struct type
	notification := Notification{
		Method: "notifications/resources/updated",
		Params: NotificationParams{
			AdditionalFields: map[string]interface{}{
				"uri": uri,
			},
		},
	}

	jsonrpcNotification := newJSONRPCNotification(notification)

	for _, ch := range subs {
		select {
		case ch <- jsonrpcNotification:
		default:
			// Skip this subscriber if the channel is full
		}
	}
}

// handleListResources handles listing resources requests
func (m *resourceManager) handleListResources(ctx context.Context, req *JSONRPCRequest) (JSONRPCMessage, error) {
	resources := m.getResources()

	// Convert []*mcp.Resource to []mcp.Resource for the result
	resultResources := make([]Resource, len(resources))
	for i, resource := range resources {
		resultResources[i] = *resource
	}

	// Create result
	result := ListResourcesResult{
		Resources: resultResources,
	}

	// Return response
	return result, nil
}

// handleReadResource handles reading resource requests
func (m *resourceManager) handleReadResource(ctx context.Context, req *JSONRPCRequest) (JSONRPCMessage, error) {
	// Convert params to map for easier access
	paramsMap, ok := req.Params.(map[string]interface{})
	if !ok {
		return newJSONRPCErrorResponse(req.ID, ErrCodeInvalidParams, utils.ErrInvalidParams.Error(), nil), nil
	}

	// Get resource URI from parameters
	uri, ok := paramsMap["uri"].(string)
	if !ok {
		return newJSONRPCErrorResponse(req.ID, ErrCodeInvalidParams, utils.ErrMissingParams.Error(), nil), nil
	}

	// Get resource
	registeredResource, exists := m.resources[uri]
	if !exists {
		return newJSONRPCErrorResponse(
			req.ID,
			ErrCodeMethodNotFound,
			fmt.Sprintf("%v: %s", utils.ErrResourceNotFound, uri),
			nil,
		), nil
	}

	// Create resource read request
	readReq := &ReadResourceRequest{
		Params: struct {
			URI       string                 `json:"uri"`
			Arguments map[string]interface{} `json:"arguments,omitempty"`
		}{
			URI: uri,
		},
	}

	// Call resource handler
	content, err := registeredResource.Handler(ctx, readReq)
	if err != nil {
		return newJSONRPCErrorResponse(req.ID, ErrCodeInternal, err.Error(), nil), nil
	}

	// Create result
	result := ReadResourceResult{
		Contents: []ResourceContents{content},
	}

	return result, nil
}

// handleListTemplates handles listing templates requests
func (m *resourceManager) handleListTemplates(ctx context.Context, req *JSONRPCRequest) (JSONRPCMessage, error) {
	templates := m.getTemplates()

	// Convert []*mcp.ResourceTemplate to []mcp.ResourceTemplate for the result
	resultTemplates := make([]ResourceTemplate, len(templates))
	for i, template := range templates {
		resultTemplates[i] = *template
	}

	// Use map structure since ListResourceTemplatesResult might not be defined
	result := map[string]interface{}{
		"resourceTemplates": resultTemplates,
	}

	return result, nil
}

// handleSubscribe handles subscription requests
func (m *resourceManager) handleSubscribe(ctx context.Context, req *JSONRPCRequest) (JSONRPCMessage, error) {
	// Convert params to map for easier access
	paramsMap, ok := req.Params.(map[string]interface{})
	if !ok {
		return newJSONRPCErrorResponse(req.ID, ErrCodeInvalidParams, utils.ErrInvalidParams.Error(), nil), nil
	}

	// Get resource URI from parameters
	uri, ok := paramsMap["uri"].(string)
	if !ok {
		return newJSONRPCErrorResponse(req.ID, ErrCodeInvalidParams, utils.ErrMissingParams.Error(), nil), nil
	}

	// Check if resource exists
	_, exists := m.getResource(uri)
	if !exists {
		return newJSONRPCErrorResponse(req.ID, ErrCodeMethodNotFound, fmt.Sprintf("resource %s not found", uri), nil), nil
	}

	// subscribe to resource updates
	_ = m.subscribe(uri) // We're not using the channel directly in the response

	// Return success response
	result := map[string]interface{}{
		"uri":           uri,
		"subscribeTime": time.Now().UTC().Format(time.RFC3339),
	}

	return result, nil
}

// handleUnsubscribe handles unsubscription requests
func (m *resourceManager) handleUnsubscribe(ctx context.Context, req *JSONRPCRequest) (JSONRPCMessage, error) {
	// Convert params to map for easier access
	paramsMap, ok := req.Params.(map[string]interface{})
	if !ok {
		return newJSONRPCErrorResponse(req.ID, ErrCodeInvalidParams, utils.ErrInvalidParams.Error(), nil), nil
	}

	// Get resource URI from parameters
	uri, ok := paramsMap["uri"].(string)
	if !ok {
		return newJSONRPCErrorResponse(req.ID, ErrCodeInvalidParams, utils.ErrMissingParams.Error(), nil), nil
	}

	// unsubscribe from resource updates
	// Note: In real implementation, you need to locate the specific channel to unsubscribe
	// This is just a simplified implementation

	// Return success response
	result := map[string]interface{}{
		"uri":             uri,
		"unsubscribeTime": time.Now().UTC().Format(time.RFC3339),
	}

	return result, nil
}
