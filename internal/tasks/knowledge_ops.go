package tasks

import (
	knowledge "github.com/LingByte/SoulNexus/pkg/knowledge/service"
	"gorm.io/gorm"
)

// StartKnowledgeOpsTasks previously wired voice session-ended knowledge collection.
func StartKnowledgeOpsTasks(_ *gorm.DB, _ *knowledge.Service) {}
