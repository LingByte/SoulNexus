package knowledge

import (
	"context"
	"fmt"
	"strings"
	"time"

	knconst "github.com/LingByte/SoulNexus/pkg/knowledge/constants"
	knmodels "github.com/LingByte/SoulNexus/pkg/knowledge/models"
	"gorm.io/gorm"
)

// QuestionCollector post-call knowledge ops (call-ended pipeline removed).
type QuestionCollector struct {
	db *gorm.DB
	kb *Service
}

// NewQuestionCollector builds a post-call knowledge ops collector.
func NewQuestionCollector(db *gorm.DB, kb *Service) *QuestionCollector {
	return &QuestionCollector{db: db, kb: kb}
}

// ProcessCallEnded is a no-op without telephony call records.
func (c *QuestionCollector) ProcessCallEnded(ctx context.Context, call any) error {
	_ = ctx
	_ = call
	return nil
}

var resolveAssistantIDHook func(callID string) uint

// SetResolveAssistantIDHook wires call-scoped assistant lookup (optional).
func SetResolveAssistantIDHook(fn func(callID string) uint) {
	resolveAssistantIDHook = fn
}

// ResolveUnansweredToChunk promotes an unanswered question into a manual KB chunk
// and marks the row resolved.
func ResolveUnansweredToChunk(ctx context.Context, db *gorm.DB, kb *Service, groupID, questionID uint, title, content string) error {
	if db == nil || kb == nil || questionID == 0 {
		return gorm.ErrInvalidDB
	}
	content = strings.TrimSpace(content)
	if content == "" {
		return fmt.Errorf("content is required")
	}
	var row knmodels.KnowledgeUnansweredQuestion
	if err := db.Where("id = ? AND group_id = ?", questionID, groupID).First(&row).Error; err != nil {
		return err
	}
	if title = strings.TrimSpace(title); title == "" {
		title = strings.TrimSpace(row.Question)
	}
	if title == "" {
		title = fmt.Sprintf("unanswered-%d", questionID)
	}
	collection := strings.TrimSpace(row.Namespace)
	if collection == "" {
		var ns knmodels.KnowledgeNamespace
		if err := db.Where("id = ? AND group_id = ?", row.NamespaceID, groupID).First(&ns).Error; err != nil {
			return err
		}
		collection = ns.Name
	}
	recordID := fmt.Sprintf("unanswered-%d", questionID)
	if _, err := kb.UpsertManualChunk(ctx, collection, recordID, recordID, title, content); err != nil {
		return err
	}
	now := time.Now()
	return db.Model(&row).Updates(map[string]any{
		"status":     knconst.KnowledgeUnansweredStatusResolved,
		"resolved_at": now,
	}).Error
}

// UnansweredDraft is the prefill payload for the resolve UI.
type UnansweredDraft struct {
	Title   string `json:"title"`
	Content string `json:"content"`
}

// DraftUnansweredAnswer returns a simple title/content draft from the stored question.
// Ops-LLM drafting against call transcripts was removed.
func DraftUnansweredAnswer(ctx context.Context, db *gorm.DB, _ *Service, groupID, questionID uint) (UnansweredDraft, error) {
	_ = ctx
	var out UnansweredDraft
	if db == nil || questionID == 0 {
		return out, gorm.ErrInvalidDB
	}
	var row knmodels.KnowledgeUnansweredQuestion
	if err := db.Where("id = ? AND group_id = ?", questionID, groupID).First(&row).Error; err != nil {
		return out, err
	}
	out.Title = strings.TrimSpace(row.Question)
	out.Content = strings.TrimSpace(row.Question)
	return out, nil
}
