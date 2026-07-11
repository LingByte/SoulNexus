package server

import (
	"fmt"
	"io"
	"net/http"
	"path/filepath"
	"strings"

	svcmodels "github.com/LingByte/SoulNexus/internal/models/server"
	"github.com/LingByte/SoulNexus/pkg/petproject"
	"github.com/LingByte/SoulNexus/pkg/stores"
	"github.com/gin-gonic/gin"
)

func mimeForProjectPath(path string) string {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".json":
		return "application/json; charset=utf-8"
	case ".png":
		return "image/png"
	case ".jpg", ".jpeg":
		return "image/jpeg"
	case ".webp":
		return "image/webp"
	case ".gif":
		return "image/gif"
	case ".css":
		return "text/css; charset=utf-8"
	case ".js":
		return "application/javascript; charset=utf-8"
	case ".md":
		return "text/markdown; charset=utf-8"
	case ".txt":
		return "text/plain; charset=utf-8"
	case ".moc3":
		return "application/octet-stream"
	default:
		return "application/octet-stream"
	}
}

func (h *Handlers) readProjectFileBytes(template *svcmodels.JSTemplate, rel string) ([]byte, error) {
	rel = strings.TrimPrefix(strings.TrimSpace(rel), "/")
	if rel == "" || strings.Contains(rel, "..") {
		return nil, fmt.Errorf("invalid path")
	}
	meta, legacy := petproject.ParseMeta(template.Content)
	if legacy {
		inline, ok := petproject.ParseInlineProject(template.Content)
		if !ok {
			return nil, fmt.Errorf("project not found")
		}
		raw, ok := inline.Files[rel]
		if !ok {
			return nil, fmt.Errorf("file not found")
		}
		return petproject.DecodeFileContent(raw)
	}
	if meta == nil {
		return nil, fmt.Errorf("project not found")
	}
	store := stores.Default()
	key, err := petproject.ObjectKey(meta.Prefix, rel)
	if err != nil {
		return nil, err
	}
	rc, _, err := store.Read(key)
	if err != nil {
		return nil, err
	}
	defer rc.Close()
	return io.ReadAll(rc)
}

// ServeJSTemplateEmbedFile serves project assets for public embed (no auth).
func (h *Handlers) ServeJSTemplateEmbedFile(c *gin.Context) {
	jsSourceID := strings.TrimSpace(c.Param("jsSourceId"))
	rel := strings.TrimPrefix(c.Param("filepath"), "/")
	if jsSourceID == "" || rel == "" {
		c.Status(http.StatusNotFound)
		return
	}
	template, err := svcmodels.GetJSTemplateByJsSourceID(h.db, jsSourceID)
	if err != nil {
		c.Status(http.StatusNotFound)
		return
	}
	data, err := h.readProjectFileBytes(template, rel)
	if err != nil {
		c.Status(http.StatusNotFound)
		return
	}
	c.Header("Cache-Control", "public, max-age=300")
	c.Data(http.StatusOK, mimeForProjectPath(rel), data)
}
