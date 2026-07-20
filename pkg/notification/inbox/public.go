package inbox

import "strconv"

// MessagePublic is the JSON-safe inbox row (snowflake ids as strings).
type MessagePublic struct {
	ID          string `json:"id"`
	UserID      string `json:"user_id"`
	Title       string `json:"title"`
	Content     string `json:"content"`
	ActionURL   string `json:"action_url,omitempty"`
	ActionLabel string `json:"action_label,omitempty"`
	Read        bool   `json:"read"`
	CreatedAt   string `json:"created_at"`
}

func MessageToPublic(m Message) MessagePublic {
	return MessagePublic{
		ID:          strconv.FormatUint(uint64(m.ID), 10),
		UserID:      strconv.FormatUint(uint64(m.UserID), 10),
		Title:       m.Title,
		Content:     m.Content,
		ActionURL:   m.ActionURL,
		ActionLabel: m.ActionLabel,
		Read:        m.Read,
		CreatedAt:   m.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}
}

func MessagesToPublic(rows []Message) []MessagePublic {
	out := make([]MessagePublic, 0, len(rows))
	for _, row := range rows {
		out = append(out, MessageToPublic(row))
	}
	return out
}
