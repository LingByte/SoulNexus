package handlers

import (
	"fmt"

	knconst "github.com/LingByte/SoulNexus/pkg/knowledge/constants"
	knowledge "github.com/LingByte/SoulNexus/pkg/knowledge/service"
	knworker "github.com/LingByte/SoulNexus/pkg/knowledge/worker"
	"github.com/LingByte/SoulNexus/pkg/i18n"
	"github.com/LingByte/SoulNexus/pkg/response"
	"github.com/gin-gonic/gin"
)

// getKnowledgeDocumentPreview returns parse + chunk preview before indexing.
//
//	Endpoint: GET /knowledge/namespaces/:id/documents/:docId/preview
func (h *Handlers) getKnowledgeDocumentPreview(c *gin.Context) {
	_, doc, ok := h.loadKnowledgeDocument(c)
	if !ok {
		return
	}
	preview := knowledge.DecodeDocumentPreview(doc.PreviewJSON)
	if preview == nil {
		response.SuccessI18n(c, i18n.KeySuccess, map[string]any{
			"docId":  fmt.Sprintf("%d", doc.ID),
			"status": doc.Status,
		})
		return
	}
	response.SuccessI18n(c, i18n.KeySuccess, map[string]any{
		"docId":         fmt.Sprintf("%d", doc.ID),
		"status":        doc.Status,
		"indexMode":     doc.IndexMode,
		"summary":       doc.SummaryText,
		"preview":       preview,
		"childCount":    preview.ChildCount,
		"parentCount":   preview.ParentCount,
		"chunkStrategy": preview.Strategy,
	})
}

// confirmKnowledgeDocumentIndex enqueues vector indexing after user confirms preview.
//
//	Endpoint: POST /knowledge/namespaces/:id/documents/:docId/confirm-index
func (h *Handlers) confirmKnowledgeDocumentIndex(c *gin.Context) {
	ns, doc, ok := h.loadKnowledgeDocument(c)
	if !ok {
		return
	}
	if h.kbWorker == nil {
		response.Render(c, response.NewI18n(response.CodeInternal, i18n.KeyKnowledgeWorkerUnavail))
		return
	}
	if doc.Status != knconst.KnowledgeDocStatusPreview {
		response.Render(c, response.Wrap(response.CodeBadRequest, "document is not awaiting preview confirmation", nil))
		return
	}

	oldIDs := knowledge.DecodeRecordIDs(doc.RecordIDs)
	h.enqueueKnowledgeDocumentIngest(knworker.IngestJob{
		DocID:        doc.ID,
		Namespace:    ns.Namespace,
		Title:        doc.Title,
		FileName:     doc.SourceFileName,
		RawFileURL:   doc.RawFileURL,
		TextURL:      doc.TextURL,
		OldRecordIDs: oldIDs,
		ConfirmIndex: true,
	})

	response.SuccessI18n(c, i18n.KeySuccess, knowledgeDocumentDTO(ns.ID, doc))
}
