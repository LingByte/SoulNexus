package server

import (
	"fmt"
	"io"
	"net/http"
	"path/filepath"
	"strings"
	"sync"
	"time"

	svcmodels "github.com/LingByte/SoulNexus/internal/models/server"
	"github.com/LingByte/SoulNexus/internal/config"
	"github.com/LingByte/SoulNexus/internal/models/auth"
	"github.com/LingByte/SoulNexus/pkg/petproject"
	"github.com/LingByte/SoulNexus/pkg/response"
	"github.com/LingByte/SoulNexus/pkg/stores"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type petPreviewSession struct {
	files   map[string][]byte
	expires time.Time
}

type petStudioPreviewBody struct {
	Files map[string]string `json:"files"`
}

var (
	petPreviewSessions sync.Map
	petPreviewTTL      = 2 * time.Hour
)

func init() {
	go func() {
		ticker := time.NewTicker(10 * time.Minute)
		defer ticker.Stop()
		for range ticker.C {
			now := time.Now()
			petPreviewSessions.Range(func(key, value any) bool {
				s := value.(*petPreviewSession)
				if now.After(s.expires) {
					petPreviewSessions.Delete(key)
				}
				return true
			})
		}
	}()
}

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

func apiBaseURL(c *gin.Context) string {
	scheme := "http"
	if c.Request.TLS != nil {
		scheme = "https"
	}
	apiPrefix := config.GlobalConfig.Server.APIPrefix
	if apiPrefix == "" {
		apiPrefix = "/api"
	}
	return fmt.Sprintf("%s://%s%s", scheme, c.Request.Host, apiPrefix)
}

// RegisterPetStudioPreview stores in-memory project files for unsaved Studio preview.
func (h *Handlers) RegisterPetStudioPreview(c *gin.Context) {
	user := auth.CurrentUser(c)
	if user == nil {
		response.Fail(c, "User not logged in", nil)
		return
	}
	var body petStudioPreviewBody
	if err := c.ShouldBindJSON(&body); err != nil || len(body.Files) == 0 {
		response.Fail(c, "Invalid preview payload", nil)
		return
	}
	decoded := make(map[string][]byte, len(body.Files))
	for path, content := range body.Files {
		path = strings.TrimPrefix(strings.TrimSpace(path), "/")
		if path == "" || strings.Contains(path, "..") {
			response.Fail(c, "Invalid file path in preview", path)
			return
		}
		b, err := petproject.DecodeFileContent(content)
		if err != nil {
			response.Fail(c, "Invalid file content", err.Error())
			return
		}
		decoded[path] = b
	}
	token := uuid.New().String()
	petPreviewSessions.Store(token, &petPreviewSession{
		files:   decoded,
		expires: time.Now().Add(petPreviewTTL),
	})
	base := apiBaseURL(c) + "/pet/studio-preview/" + token + "/"
	response.Success(c, "ok", gin.H{
		"token":   token,
		"baseUrl": base,
	})
}

// ServePetStudioPreviewFile serves a file from an in-memory preview session.
func (h *Handlers) ServePetStudioPreviewFile(c *gin.Context) {
	token := strings.TrimSpace(c.Param("token"))
	rel := strings.TrimPrefix(c.Param("filepath"), "/")
	if token == "" || rel == "" || strings.Contains(rel, "..") {
		c.Status(http.StatusNotFound)
		return
	}
	raw, ok := petPreviewSessions.Load(token)
	if !ok {
		c.Status(http.StatusNotFound)
		return
	}
	session := raw.(*petPreviewSession)
	if time.Now().After(session.expires) {
		petPreviewSessions.Delete(token)
		c.Status(http.StatusNotFound)
		return
	}
	data, ok := session.files[rel]
	if !ok {
		c.Status(http.StatusNotFound)
		return
	}
	c.Header("Cache-Control", "no-store")
	c.Data(http.StatusOK, mimeForProjectPath(rel), data)
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

// ServeJSTemplateProjectFile serves one project file (auth required).
func (h *Handlers) ServeJSTemplateProjectFile(c *gin.Context) {
	id := c.Param("id")
	user := auth.CurrentUser(c)
	if user == nil {
		response.Fail(c, "User not logged in", nil)
		return
	}
	template, err := svcmodels.GetJSTemplateByID(h.db, id)
	if err != nil {
		response.Fail(c, "Template not found", nil)
		return
	}
	if !jsTemplateVisibleToUser(h.db, user.ID, template) {
		response.Fail(c, "Insufficient permissions", nil)
		return
	}
	rel := strings.TrimPrefix(c.Param("filepath"), "/")
	data, err := h.readProjectFileBytes(template, rel)
	if err != nil {
		c.Status(http.StatusNotFound)
		return
	}
	c.Header("Cache-Control", "public, max-age=60")
	c.Data(http.StatusOK, mimeForProjectPath(rel), data)
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
