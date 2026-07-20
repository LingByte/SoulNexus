package models

import (
	"testing"

	"github.com/LingByte/SoulNexus/internal/constants"
)

func TestNewAssistantForCreate_defaultScene(t *testing.T) {
	ast := NewAssistantForCreate(1, "Bot", "", "", "", "", "", "", "", "", "", "", true, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	if ast.Scene != constants.AssistantSceneGeneral {
		t.Fatalf("scene=%q", ast.Scene)
	}
}

func TestBuildAssistantUpdates_requiresName(t *testing.T) {
	_, err := BuildAssistantUpdates(Assistant{Scene: "sales"}, "", "", "", "", nil,
		"", "", "", "", "", "", "",
		"", "", "", "", "", "", "", "", "", "", "", "", "", "")
	if err == nil {
		t.Fatal("expected name required")
	}
}

func TestBuildAssistantUpdates_inheritsScene(t *testing.T) {
	updates, err := BuildAssistantUpdates(Assistant{Scene: "sales"}, "Bot", "", "", "", nil,
		"", "", "", "", "", "", "",
		"", "", "", "", "", "", "", "", "", "", "", "", "", "")
	if err != nil {
		t.Fatal(err)
	}
	if updates["scene"] != "sales" {
		t.Fatalf("scene=%v", updates["scene"])
	}
}
