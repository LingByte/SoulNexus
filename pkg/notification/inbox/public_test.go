package inbox

import (
	"testing"
	"time"
)

func TestMessageToPublic(t *testing.T) {
	t.Parallel()
	ts := time.Date(2026, 1, 15, 10, 30, 0, 0, time.UTC)
	m := Message{
		ID:        42,
		UserID:    7,
		Title:     "Hello",
		Content:   "World",
		Read:      true,
		CreatedAt: ts,
	}
	pub := MessageToPublic(m)
	if pub.ID != "42" || pub.UserID != "7" {
		t.Errorf("ids: id=%q user_id=%q", pub.ID, pub.UserID)
	}
	if pub.Title != "Hello" || pub.Content != "World" || !pub.Read {
		t.Errorf("fields mismatch: %+v", pub)
	}
	if pub.CreatedAt != "2026-01-15T10:30:00Z" {
		t.Errorf("CreatedAt = %q", pub.CreatedAt)
	}
}

func TestMessagesToPublic(t *testing.T) {
	t.Parallel()
	rows := []Message{
		{ID: 1, UserID: 2, Title: "a", Content: "b", CreatedAt: time.Unix(0, 0).UTC()},
		{ID: 3, UserID: 4, Title: "c", Content: "d", CreatedAt: time.Unix(0, 0).UTC()},
	}
	out := MessagesToPublic(rows)
	if len(out) != 2 {
		t.Fatalf("len = %d, want 2", len(out))
	}
	if out[0].ID != "1" || out[1].ID != "3" {
		t.Errorf("ids = %q, %q", out[0].ID, out[1].ID)
	}
}

func TestMessagesToPublic_empty(t *testing.T) {
	t.Parallel()
	out := MessagesToPublic(nil)
	if len(out) != 0 {
		t.Errorf("len = %d, want 0", len(out))
	}
}
