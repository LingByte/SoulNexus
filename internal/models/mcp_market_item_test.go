package models

import (
	"testing"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func TestActivateMcpMarketItemForTenant(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:mcp_market_test?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&McpMarketItem{}, &TenantAssistantTool{}); err != nil {
		t.Fatal(err)
	}
	item := &McpMarketItem{
		Slug:        "demo",
		Name:        "demo",
		DisplayName: "Demo",
		Status:      McpMarketStatusPublished,
		Kind:        AssistantToolKindMCPSSE,
		McpSSEURL:   "http://127.0.0.1:3920/sse",
		TimeoutMS:   10000,
	}
	if err := CreateMcpMarketItem(db, item); err != nil {
		t.Fatal(err)
	}
	tool, err := ActivateMcpMarketItemForTenant(db, 7, item.ID)
	if err != nil {
		t.Fatal(err)
	}
	if tool.Source != AssistantToolSourceMarket || tool.MarketItemID == nil || *tool.MarketItemID != item.ID {
		t.Fatalf("tool=%+v", tool)
	}
	again, err := ActivateMcpMarketItemForTenant(db, 7, item.ID)
	if err != nil {
		t.Fatal(err)
	}
	if again.ID != tool.ID {
		t.Fatalf("expected idempotent activate")
	}
}
