package prompt

import (
	"context"
	"errors"
	"testing"
	"time"
)

// mockContent 用于模拟 ResourceContents 实现
type mockContent struct {
	Data string
}

func (mockContent) isResourceContents() {}

// mockHandler 返回一个简单的 mockContent
func mockHandler(ctx context.Context, req *ReadResourceRequest) (ResourceContents, error) {
	if req.Params.URI == "invalid" {
		return nil, errors.New("resource not found")
	}
	return mockContent{Data: "mock data"}, nil
}

func TestResourceManager_RegisterAndGet(t *testing.T) {
	manager := newResourceManager()
	resource := &Resource{
		Name: "test",
		URI:  "res://test",
	}

	manager.registerResource(resource, mockHandler)
	got, exists := manager.getResource("res://test")
	if !exists {
		t.Fatal("expected resource to exist")
	}
	if got.Name != "test" {
		t.Errorf("expected name 'test', got '%s'", got.Name)
	}
}

func TestResourceManager_ReadResource(t *testing.T) {
	manager := newResourceManager()
	resource := &Resource{Name: "test", URI: "res://test"}
	manager.registerResource(resource, mockHandler)

	req := &JSONRPCRequest{
		ID:      "1",
		JSONRPC: "2.0",
		Params: map[string]interface{}{
			"uri": "res://test",
		},
	}

	resp, err := manager.handleReadResource(context.Background(), req)
	if err != nil {
		t.Fatal(err)
	}
	result, ok := resp.(ReadResourceResult)
	if !ok {
		t.Fatal("expected ReadResourceResult type")
	}
	if len(result.Contents) != 1 {
		t.Errorf("expected 1 content, got %d", len(result.Contents))
	}
}

func TestResourceManager_SubscribeUnsubscribe(t *testing.T) {
	manager := newResourceManager()
	uri := "res://test"
	ch := manager.subscribe(uri)

	if len(manager.subscribers[uri]) != 1 {
		t.Fatal("subscriber not added")
	}

	manager.unsubscribe(uri, ch)

	if _, ok := manager.subscribers[uri]; ok {
		t.Fatal("subscriber not removed")
	}
}

func TestResourceManager_NotifyUpdate(t *testing.T) {
	manager := newResourceManager()
	uri := "res://update"
	ch := manager.subscribe(uri)

	// 使用 goroutine 监听通知
	done := make(chan bool)
	go func() {
		select {
		case n := <-ch:
			if n.Method != "notifications/resources/updated" {
				t.Errorf("unexpected method: %s", n.Method)
			}
			done <- true
		case <-time.After(1 * time.Second):
			t.Error("notification not received")
			done <- false
		}
	}()

	manager.notifyUpdate(uri)
	<-done
}

func TestResourceManager_ListResources(t *testing.T) {
	manager := newResourceManager()
	manager.registerResource(&Resource{Name: "a", URI: "res://a"}, mockHandler)
	manager.registerResource(&Resource{Name: "b", URI: "res://b"}, mockHandler)

	req := &JSONRPCRequest{
		ID:      "list",
		JSONRPC: "2.0",
	}

	resp, err := manager.handleListResources(context.Background(), req)
	if err != nil {
		t.Fatal(err)
	}
	result, ok := resp.(ListResourcesResult)
	if !ok {
		t.Fatal("unexpected result type")
	}
	if len(result.Resources) != 2 {
		t.Errorf("expected 2 resources, got %d", len(result.Resources))
	}
}

func TestResourceManager_RegisterTemplate(t *testing.T) {
	manager := newResourceManager()

	template := &ResourceTemplate{
		Name:        "test-template",
		URITemplate: &URITemplate{},
		Description: "Test template",
	}

	handler := func(ctx context.Context, req *ReadResourceRequest) ([]ResourceContents, error) {
		return []ResourceContents{mockContent{Data: "data"}}, nil
	}

	err := manager.registerTemplate(template, handler)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	templates := manager.getTemplates()
	if len(templates) != 1 {
		t.Errorf("Expected 1 template, got %d", len(templates))
	}
}

func TestResourceManager_RegisterTemplate_NilTemplate(t *testing.T) {
	manager := newResourceManager()

	err := manager.registerTemplate(nil, nil)
	if err == nil {
		t.Fatal("Expected error for nil template")
	}
}

func TestResourceManager_RegisterTemplate_EmptyName(t *testing.T) {
	manager := newResourceManager()

	template := &ResourceTemplate{
		Name:        "",
		URITemplate: &URITemplate{},
	}

	err := manager.registerTemplate(template, nil)
	if err == nil {
		t.Fatal("Expected error for empty name")
	}
}

func TestResourceManager_RegisterTemplate_NilURI(t *testing.T) {
	manager := newResourceManager()

	template := &ResourceTemplate{
		Name:        "test",
		URITemplate: nil,
	}

	err := manager.registerTemplate(template, nil)
	if err == nil {
		t.Fatal("Expected error for nil URI template")
	}
}

func TestResourceManager_RegisterTemplate_Duplicate(t *testing.T) {
	manager := newResourceManager()

	template := &ResourceTemplate{
		Name:        "test",
		URITemplate: &URITemplate{},
	}

	err := manager.registerTemplate(template, nil)
	if err != nil {
		t.Fatalf("First registration should succeed, got %v", err)
	}

	err = manager.registerTemplate(template, nil)
	if err == nil {
		t.Fatal("Expected error for duplicate template")
	}
}

func TestResourceManager_GetTemplates(t *testing.T) {
	manager := newResourceManager()

	template1 := &ResourceTemplate{
		Name:        "template1",
		URITemplate: &URITemplate{},
	}
	template2 := &ResourceTemplate{
		Name:        "template2",
		URITemplate: &URITemplate{},
	}

	manager.registerTemplate(template1, nil)
	manager.registerTemplate(template2, nil)

	templates := manager.getTemplates()
	if len(templates) != 2 {
		t.Errorf("Expected 2 templates, got %d", len(templates))
	}
}

func TestResourceManager_HandleListTemplates(t *testing.T) {
	manager := newResourceManager()

	template := &ResourceTemplate{
		Name:        "test",
		URITemplate: &URITemplate{},
	}
	manager.registerTemplate(template, nil)

	req := &JSONRPCRequest{
		ID: "1",
	}

	resp, err := manager.handleListTemplates(context.Background(), req)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	result, ok := resp.(map[string]interface{})
	if !ok {
		t.Fatal("Expected map result")
	}

	templates, ok := result["resourceTemplates"].([]ResourceTemplate)
	if !ok {
		t.Fatal("Expected resourceTemplates array")
	}
	if len(templates) != 1 {
		t.Errorf("Expected 1 template, got %d", len(templates))
	}
}

func TestResourceManager_HandleSubscribe(t *testing.T) {
	manager := newResourceManager()
	manager.registerResource(&Resource{Name: "test", URI: "res://test"}, mockHandler)

	req := &JSONRPCRequest{
		ID: "1",
		Params: map[string]interface{}{
			"uri": "res://test",
		},
	}

	resp, err := manager.handleSubscribe(context.Background(), req)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	result, ok := resp.(map[string]interface{})
	if !ok {
		t.Fatal("Expected map result")
	}

	if result["uri"] != "res://test" {
		t.Errorf("Expected uri 'res://test', got %v", result["uri"])
	}
}

func TestResourceManager_HandleSubscribe_ResourceNotFound(t *testing.T) {
	manager := newResourceManager()

	req := &JSONRPCRequest{
		ID: "1",
		Params: map[string]interface{}{
			"uri": "res://nonexistent",
		},
	}

	resp, err := manager.handleSubscribe(context.Background(), req)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	errResp, ok := resp.(*JSONRPCError)
	if !ok {
		t.Fatal("Expected error response")
	}
	if errResp.Error.Code != ErrCodeMethodNotFound {
		t.Errorf("Expected method not found error, got %d", errResp.Error.Code)
	}
}

func TestResourceManager_HandleUnsubscribe(t *testing.T) {
	manager := newResourceManager()

	req := &JSONRPCRequest{
		ID: "1",
		Params: map[string]interface{}{
			"uri": "res://test",
		},
	}

	resp, err := manager.handleUnsubscribe(context.Background(), req)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	result, ok := resp.(map[string]interface{})
	if !ok {
		t.Fatal("Expected map result")
	}

	if result["uri"] != "res://test" {
		t.Errorf("Expected uri 'res://test', got %v", result["uri"])
	}
}

func TestResourceManager_HandleReadResource_InvalidParams(t *testing.T) {
	manager := newResourceManager()

	req := &JSONRPCRequest{
		ID:     "1",
		Params: "not-a-map",
	}

	resp, err := manager.handleReadResource(context.Background(), req)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	errResp, ok := resp.(*JSONRPCError)
	if !ok {
		t.Fatal("Expected error response")
	}
	if errResp.Error.Code != ErrCodeInvalidParams {
		t.Errorf("Expected invalid params error, got %d", errResp.Error.Code)
	}
}

func TestResourceManager_HandleReadResource_MissingURI(t *testing.T) {
	manager := newResourceManager()

	req := &JSONRPCRequest{
		ID:     "1",
		Params: map[string]interface{}{},
	}

	resp, err := manager.handleReadResource(context.Background(), req)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	errResp, ok := resp.(*JSONRPCError)
	if !ok {
		t.Fatal("Expected error response")
	}
	if errResp.Error.Code != ErrCodeInvalidParams {
		t.Errorf("Expected invalid params error, got %d", errResp.Error.Code)
	}
}

func TestResourceManager_HandleReadResource_NotFound(t *testing.T) {
	manager := newResourceManager()

	req := &JSONRPCRequest{
		ID: "1",
		Params: map[string]interface{}{
			"uri": "res://nonexistent",
		},
	}

	resp, err := manager.handleReadResource(context.Background(), req)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	errResp, ok := resp.(*JSONRPCError)
	if !ok {
		t.Fatal("Expected error response")
	}
	if errResp.Error.Code != ErrCodeMethodNotFound {
		t.Errorf("Expected method not found error, got %d", errResp.Error.Code)
	}
}

func TestResourceManager_RegisterResource_Nil(t *testing.T) {
	manager := newResourceManager()
	manager.registerResource(nil, nil)

	resources := manager.getResources()
	if len(resources) != 0 {
		t.Errorf("Expected 0 resources, got %d", len(resources))
	}
}

func TestResourceManager_RegisterResource_EmptyURI(t *testing.T) {
	manager := newResourceManager()
	manager.registerResource(&Resource{URI: ""}, nil)

	resources := manager.getResources()
	if len(resources) != 0 {
		t.Errorf("Expected 0 resources, got %d", len(resources))
	}
}

func TestResourceManager_Unsubscribe_NonExistent(t *testing.T) {
	manager := newResourceManager()
	ch := make(chan *JSONRPCNotification, 10)

	// Should not panic
	manager.unsubscribe("res://nonexistent", ch)
}

func TestResourceManager_NotifyUpdate_NoSubscribers(t *testing.T) {
	manager := newResourceManager()

	// Should not panic
	manager.notifyUpdate("res://test")
}
