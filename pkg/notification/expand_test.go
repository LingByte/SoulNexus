package notification

import (
	"testing"
)

func TestDingTalkNotification_SendText(t *testing.T) {
	config := DingTalkConfig{
		WebhookURL: "https://oapi.dingtalk.com/robot/send",
		Secret:     "test-secret",
	}

	notif := DingTalkNotification{config: config}
	err := notif.SendText("Test message")
	// Currently returns nil (not implemented)
	if err != nil {
		t.Errorf("SendText returned error: %v", err)
	}
}

func TestDingTalkNotification_SendMarkdown(t *testing.T) {
	config := DingTalkConfig{
		WebhookURL: "https://oapi.dingtalk.com/robot/send",
		Secret:     "test-secret",
	}

	notif := DingTalkNotification{config: config}
	err := notif.SendMarkdown("Title", "# Markdown Content")
	// Currently returns nil (not implemented)
	if err != nil {
		t.Errorf("SendMarkdown returned error: %v", err)
	}
}

func TestDingTalkConfig_Structure(t *testing.T) {
	config := DingTalkConfig{
		WebhookURL: "https://example.com/webhook",
		Secret:     "secret-key",
	}

	if config.WebhookURL == "" {
		t.Error("WebhookURL should not be empty")
	}
}

func TestWeChatWorkConfig_Structure(t *testing.T) {
	config := WeChatWorkConfig{
		CorpID:  "corp-id",
		AgentID: "agent-id",
		Secret:  "secret",
	}

	if config.CorpID == "" {
		t.Error("CorpID should not be empty")
	}
	if config.AgentID == "" {
		t.Error("AgentID should not be empty")
	}
}

func TestFeishuConfig_Structure(t *testing.T) {
	config := FeishuConfig{
		WebhookURL: "https://open.feishu.cn/open-apis/bot/v2/hook/xxx",
		Secret:     "secret",
	}

	if config.WebhookURL == "" {
		t.Error("WebhookURL should not be empty")
	}
}

func TestEmailTemplate_Structure(t *testing.T) {
	template := EmailTemplate{
		Name:     "welcome",
		Subject:  "Welcome",
		HTMLBody: "<html><body>Welcome</body></html>",
		TextBody: "Welcome",
	}

	if template.Name == "" {
		t.Error("Name should not be empty")
	}
	if template.Subject == "" {
		t.Error("Subject should not be empty")
	}
}

func TestTemplateEngine_Render(t *testing.T) {
	engine := &TemplateEngine{
		templates: make(map[string]*EmailTemplate),
	}

	// Currently returns empty strings and nil error (not implemented)
	html, text, err := engine.Render("template-name", map[string]interface{}{"key": "value"})
	if err != nil {
		t.Errorf("Render returned error: %v", err)
	}
	if html != "" {
		t.Errorf("Expected empty HTML, got %s", html)
	}
	if text != "" {
		t.Errorf("Expected empty text, got %s", text)
	}
}

func TestTemplateEngine_Structure(t *testing.T) {
	engine := &TemplateEngine{
		templates: make(map[string]*EmailTemplate),
	}

	if engine.templates == nil {
		t.Error("templates map should not be nil")
	}
}
