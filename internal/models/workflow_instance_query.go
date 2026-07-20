package models

import (
	"strings"
	"time"

	"github.com/LingByte/SoulNexus/pkg/utils"
	"gorm.io/gorm"
)

// WorkflowInstanceListFilter narrows workflow invocation log queries.
type WorkflowInstanceListFilter struct {
	DefinitionID  uint
	GroupID       uint
	TriggerSource string
	Keyword       string
	From          *time.Time
	To            *time.Time
}

// ListWorkflowInstancesPage returns paginated workflow run records.
func ListWorkflowInstancesPage(db *gorm.DB, filter WorkflowInstanceListFilter, page, size int) ([]WorkflowInstance, int64, error) {
	q := db.Model(&WorkflowInstance{})
	if filter.DefinitionID > 0 {
		q = q.Where("definition_id = ?", filter.DefinitionID)
	}
	if filter.GroupID > 0 {
		q = q.Where("group_id = ?", filter.GroupID)
	}
	if src := strings.TrimSpace(filter.TriggerSource); src != "" && src != "all" {
		q = q.Where("trigger_source = ?", src)
	}
	if filter.From != nil {
		q = q.Where("created_at >= ?", *filter.From)
	}
	if filter.To != nil {
		q = q.Where("created_at <= ?", *filter.To)
	}
	if kw := strings.TrimSpace(filter.Keyword); kw != "" {
		like := "%" + kw + "%"
		q = q.Where("(definition_name LIKE ? OR trigger_user LIKE ? OR CAST(id AS CHAR) LIKE ?)", like, like, like)
	}
	return utils.FindPage[WorkflowInstance](q, page, size, "id DESC", utils.DefaultMaxPageSize)
}
