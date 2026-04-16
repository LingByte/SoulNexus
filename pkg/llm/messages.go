package llm

import "strings"

func buildShortTermMessages(text string, options *QueryOptions) []ChatMessage {
	msgs := make([]ChatMessage, 0, 8)
	if options != nil && len(options.Messages) > 0 {
		for _, m := range options.Messages {
			role := strings.ToLower(strings.TrimSpace(m.Role))
			content := strings.TrimSpace(m.Content)
			if content == "" {
				continue
			}
			switch role {
			case "system", "user", "assistant":
				msgs = append(msgs, ChatMessage{Role: role, Content: content})
			}
		}
	}
	userText := strings.TrimSpace(text)
	if userText == "" {
		return msgs
	}
	if len(msgs) > 0 {
		last := msgs[len(msgs)-1]
		if last.Role == "user" && last.Content == userText {
			return msgs
		}
	}
	msgs = append(msgs, ChatMessage{Role: "user", Content: userText})
	return msgs
}

func chatMessagesToMap(messages []ChatMessage) []map[string]string {
	out := make([]map[string]string, 0, len(messages))
	for _, m := range messages {
		out = append(out, map[string]string{
			"role":    m.Role,
			"content": m.Content,
		})
	}
	return out
}

func chatMessagesToAnthropic(messages []ChatMessage) []map[string]any {
	out := make([]map[string]any, 0, len(messages))
	for _, m := range messages {
		if m.Role == "system" {
			continue
		}
		out = append(out, map[string]any{
			"role": m.Role,
			"content": []map[string]string{
				{"type": "text", "text": m.Content},
			},
		})
	}
	return out
}

func mergedSystemPrompt(base string, messages []ChatMessage) string {
	parts := make([]string, 0, 2)
	if strings.TrimSpace(base) != "" {
		parts = append(parts, strings.TrimSpace(base))
	}
	if len(messages) > 0 {
		var extra strings.Builder
		for _, m := range messages {
			if m.Role == "system" && strings.TrimSpace(m.Content) != "" {
				if extra.Len() > 0 {
					extra.WriteString("\n")
				}
				extra.WriteString(strings.TrimSpace(m.Content))
			}
		}
		if extra.Len() > 0 {
			parts = append(parts, extra.String())
		}
	}
	return strings.Join(parts, "\n")
}

func chatMessagesToPrompt(messages []ChatMessage) string {
	if len(messages) == 0 {
		return ""
	}
	var b strings.Builder
	for _, m := range messages {
		role := strings.ToUpper(strings.TrimSpace(m.Role))
		if role == "" {
			role = "USER"
		}
		b.WriteString(role)
		b.WriteString(": ")
		b.WriteString(strings.TrimSpace(m.Content))
		b.WriteString("\n")
	}
	return strings.TrimSpace(b.String())
}
