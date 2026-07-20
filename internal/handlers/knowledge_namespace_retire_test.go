package handlers

import (
	"testing"

	knmodels "github.com/LingByte/SoulNexus/pkg/knowledge/models"
	"github.com/LingByte/SoulNexus/pkg/utils/common"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func TestRetireKnowledgeNamespaceSlugFreesUnique(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:kb_retire?mode=memory&cache=shared"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&knmodels.KnowledgeNamespace{}, &knmodels.KnowledgeDocument{}, &knmodels.KnowledgeChunk{}); err != nil {
		t.Fatal(err)
	}

	groupID := uint(42)
	row := knmodels.KnowledgeNamespace{
		BaseModel: common.BaseModel{ID: 1001},
		GroupID:   groupID,
		Namespace: "kb-42-n59c21d57",
		Name:      "云阶知识库",
		Status:    "active",
	}
	if err := db.Create(&row).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&knmodels.KnowledgeChunk{
		BaseModel: common.BaseModel{ID: 2001},
		GroupID:   groupID,
		Namespace: row.Namespace,
		RecordID:  "r1",
		Content:   "hello",
		Status:    "active",
	}).Error; err != nil {
		t.Fatal(err)
	}

	if err := retireKnowledgeNamespaceSlug(db, row); err != nil {
		t.Fatalf("retire: %v", err)
	}

	slug := ensureUniqueKnowledgeNamespace(db, groupID, "kb-42-n59c21d57")
	if slug != "kb-42-n59c21d57" {
		t.Fatalf("expected original slug freed, got %q", slug)
	}

	recreate := knmodels.KnowledgeNamespace{
		BaseModel: common.BaseModel{ID: 1002},
		GroupID:   groupID,
		Namespace: slug,
		Name:      "云阶知识库",
		Status:    "active",
	}
	if err := db.Create(&recreate).Error; err != nil {
		t.Fatalf("recreate same slug: %v", err)
	}

	var chunks int64
	_ = db.Model(&knmodels.KnowledgeChunk{}).Where("namespace = ?", slug).Count(&chunks)
	if chunks != 0 {
		t.Fatalf("old chunks should not stay on freed slug, got %d", chunks)
	}
}
