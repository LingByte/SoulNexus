package providers

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/LingByte/SoulNexus/pkg/utils/common"
	"go.uber.org/zap"
)

type mockChatLLM struct {
	tools map[string]LLMFunctionToolCallback
}

func (m *mockChatLLM) Query(string, string) (string, error)                          { return "", nil }
func (m *mockChatLLM) QueryWithOptions(string, LLMQueryOptions) (string, error)      { return "", nil }
func (m *mockChatLLM) QueryStream(string, LLMQueryOptions, func(string, bool) error) (string, error) {
	return "", nil
}
func (m *mockChatLLM) RegisterFunctionTool(name, _ string, _ interface{}, cb LLMFunctionToolCallback) {
	if m.tools == nil {
		m.tools = make(map[string]LLMFunctionToolCallback)
	}
	m.tools[name] = cb
}
func (m *mockChatLLM) RegisterFunctionToolDefinition(*LLMFunctionToolDefinition) {}
func (m *mockChatLLM) GetFunctionTools() []interface{}                             { return nil }
func (m *mockChatLLM) ListFunctionTools() []string                                 { return nil }
func (m *mockChatLLM) GetLastUsage() (LLMUsage, bool)                              { return LLMUsage{}, false }
func (m *mockChatLLM) ResetMessages()                                              {}
func (m *mockChatLLM) SeedMessages([]LLMMessage)                                   {}
func (m *mockChatLLM) SetSystemPrompt(string)                                      {}
func (m *mockChatLLM) GetMessages() []LLMMessage                                   { return nil }
func (m *mockChatLLM) LastToolTrace() []LLMToolCall                                { return nil }
func (m *mockChatLLM) Interrupt()                                                  {}
func (m *mockChatLLM) Hangup()                                                     {}

func TestHTTPToolFromCatalogRow_ParseHappyPath(t *testing.T) {
	row := CatalogToolRow{
		Name:           "order_lookup",
		DisplayName:    "Order Lookup",
		Description:    "Look up an order by id",
		Kind:           CatalogToolKindHTTP,
		Enabled:        true,
		Method:         "POST",
		URL:            "https://api.example.com/orders",
		HeadersJSON:    []byte(`{"Authorization":"Bearer test"}`),
		BodyTemplate:   `{"orderId":"{{orderId}}"}`,
		TimeoutMS:      5000,
		ParametersJSON: []byte(`{"type":"object","properties":{"orderId":{"type":"string"}},"required":["orderId"]}`),
	}
	cfg, err := HTTPToolFromCatalogRow(row)
	if err != nil {
		t.Fatalf("HTTPToolFromCatalogRow: %v", err)
	}
	if cfg.Name != "order_lookup" {
		t.Fatalf("name: got %q", cfg.Name)
	}
	if cfg.Method != "POST" {
		t.Fatalf("method: got %q", cfg.Method)
	}
	if cfg.Headers["Authorization"] != "Bearer test" {
		t.Fatalf("headers: %+v", cfg.Headers)
	}
	if string(cfg.Parameters) == "" {
		t.Fatal("expected parameters schema")
	}
}

func TestParseHTTPTools_SkipsDisabledAndNonHTTP(t *testing.T) {
	rows := []CatalogToolRow{
		{Name: "a", Kind: CatalogToolKindHTTP, Enabled: true, URL: "https://example.com/a"},
		{Name: "b", Kind: CatalogToolKindHTTP, Enabled: false, URL: "https://example.com/b"},
		{Name: "c", Kind: CatalogToolKindMCPStdio, Enabled: true, McpCommand: "echo"},
	}
	out := ParseHTTPTools(rows)
	if len(out) != 1 || out[0].Name != "a" {
		t.Fatalf("ParseHTTPTools: got %+v", out)
	}
}

func TestRegisterHTTPTools_HappyPath(t *testing.T) {
	t.Cleanup(common.ResetSSRFWhitelistForTest)
	common.SetSSRFWhitelistFromRaw(`127.0.0.1,localhost`)

	var gotMethod, gotBody string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		raw, _ := io.ReadAll(r.Body)
		gotBody = string(raw)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":"shipped","orderId":"42"}`))
	}))
	defer srv.Close()

	tool := HTTPToolConfig{
		Name:         "order_lookup",
		Description:  "lookup order",
		Method:       http.MethodPost,
		URL:          srv.URL + "/orders",
		BodyTemplate: `{"orderId":"{{orderId}}"}`,
		TimeoutMS:    5000,
		Parameters:   json.RawMessage(`{"type":"object","properties":{"orderId":{"type":"string"}}}`),
	}
	llm := &mockChatLLM{}
	RegisterHTTPTools(llm, []HTTPToolConfig{tool}, zap.NewNop())
	cb, ok := llm.tools["order_lookup"]
	if !ok {
		t.Fatal("tool not registered")
	}
	out, err := cb(map[string]interface{}{"orderId": "42"}, nil)
	if err != nil {
		t.Fatalf("invoke: %v", err)
	}
	if gotMethod != http.MethodPost {
		t.Fatalf("method: got %q", gotMethod)
	}
	if gotBody != `{"orderId":"42"}` {
		t.Fatalf("body: got %q", gotBody)
	}
	if out != `{"status":"shipped","orderId":"42"}` {
		t.Fatalf("response: got %q", out)
	}
}

func TestInvokeHTTPTool_GET(t *testing.T) {
	t.Cleanup(common.ResetSSRFWhitelistForTest)
	common.SetSSRFWhitelistFromRaw(`127.0.0.1,localhost`)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Fatalf("method: %s", r.Method)
		}
		if r.URL.Path != "/orders/42" {
			t.Fatalf("path: %s", r.URL.Path)
		}
		_, _ = w.Write([]byte("ok"))
	}))
	defer srv.Close()

	out, err := invokeHTTPTool(context.Background(), HTTPToolConfig{
		Name:      "get_order",
		Method:    http.MethodGet,
		URL:       srv.URL + "/orders/{{orderId}}",
		TimeoutMS: 3000,
	}, map[string]interface{}{"orderId": "42"}, nil)
	if err != nil {
		t.Fatalf("invoke: %v", err)
	}
	if out != "ok" {
		t.Fatalf("out: %q", out)
	}
}
