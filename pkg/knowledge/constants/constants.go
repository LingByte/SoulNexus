package constants

// Knowledge document indexing lifecycle (stored on knowledge_documents.status).
const (
	KnowledgeDocStatusQueued     = "queued"
	KnowledgeDocStatusParsing    = "parsing"
	KnowledgeDocStatusPreview    = "preview"
	KnowledgeDocStatusIndexing   = "indexing"
	KnowledgeDocStatusProcessing = "processing" // legacy alias; treated as in-progress
	KnowledgeDocStatusActive     = "active"
	KnowledgeDocStatusFailed     = "failed"
)

// KnowledgeDocStatusInProgress reports whether indexing is still running.
func KnowledgeDocStatusInProgress(status string) bool {
	switch status {
	case KnowledgeDocStatusQueued, KnowledgeDocStatusParsing, KnowledgeDocStatusPreview,
		KnowledgeDocStatusIndexing, KnowledgeDocStatusProcessing:
		return true
	default:
		return false
	}
}

// Knowledge index chunk modes.
const (
	KnowledgeIndexModeFlat        = "flat"
	KnowledgeIndexModeParentChild = "parent_child"
)

// Knowledge chunk source types.
const (
	KnowledgeChunkSourceIngest   = "ingest"
	KnowledgeChunkSourceManual   = "manual"
	KnowledgeChunkSourceResolved = "resolved"
	KnowledgeChunkSourceTable    = "table"
)

// Knowledge chunk status.
const (
	KnowledgeChunkStatusActive  = "active"
	KnowledgeChunkStatusDeleted = "deleted"
)

// Knowledge sync source types.
const (
	KnowledgeSyncTypeURL   = "url"
	KnowledgeSyncTypeAPI   = "api"
	KnowledgeSyncTypeTable = "table"
)

// Knowledge sync source status.
const (
	KnowledgeSyncStatusActive   = "active"
	KnowledgeSyncStatusPaused   = "paused"
	KnowledgeSyncStatusDisabled = "disabled"
)

// Knowledge unanswered question status.
const (
	KnowledgeUnansweredStatusOpen     = "open"
	KnowledgeUnansweredStatusResolved = "resolved"
	KnowledgeUnansweredStatusIgnored  = "ignored"
)
