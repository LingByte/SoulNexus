package server

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	svcmodels "github.com/LingByte/SoulNexus/internal/models/server"
	"github.com/LingByte/SoulNexus"
	"github.com/LingByte/SoulNexus/internal/config"
	"github.com/LingByte/SoulNexus/internal/models/auth"
	"github.com/LingByte/SoulNexus/pkg/constants"
	jsPkg "github.com/LingByte/SoulNexus/pkg/js"
	"github.com/LingByte/SoulNexus/pkg/petproject"
	"github.com/LingByte/SoulNexus/pkg/response"
	"github.com/LingByte/SoulNexus/pkg/stores"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type petProjectSaveBody struct {
	Name  string            `json:"name"`
	Usage string            `json:"usage"`
	Entry string            `json:"entry"`
	Files map[string]string `json:"files"`
}

func validatePetEntryScript(entry string, files map[string]string) (string, []string) {
	entry = strings.TrimSpace(entry)
	if entry == "" {
		entry = petproject.DefaultEntry
	}
	script, ok := files[entry]
	if !ok || script == "" {
		return entry, nil
	}
	valid, violations := jsPkg.ValidateAST(script, jsPkg.PetScriptWhitelist)
	if !valid {
		return entry, violations
	}
	return entry, nil
}

func (h *Handlers) persistPetProject(template *svcmodels.JSTemplate, entry string, files map[string]string, name, usage string) (string, error) {
	prefix := petproject.DefaultPrefix(template.ID)
	store := stores.Default()
	if err := petproject.WriteFiles(store, prefix, files); err != nil {
		return "", err
	}
	metaJSON, err := petproject.MarshalMeta(prefix, entry, petproject.PathsFromFiles(files))
	if err != nil {
		return "", err
	}
	updates := map[string]interface{}{"content": metaJSON}
	if name := strings.TrimSpace(name); name != "" {
		updates["name"] = name
	}
	if usage != "" {
		updates["usage"] = usage
	}
	if err := h.db.Model(&svcmodels.JSTemplate{}).Where("id = ?", template.ID).Updates(updates).Error; err != nil {
		return "", err
	}
	_ = h.db.Model(&svcmodels.JSTemplateVersion{}).
		Where("template_id = ? AND version = ?", template.ID, template.Version).
		Update("content", metaJSON).Error
	return prefix, nil
}

// CreateJSTemplateWithProject atomically creates metadata + uploads files to object storage.
func (h *Handlers) CreateJSTemplateWithProject(c *gin.Context) {
	user := auth.CurrentUser(c)
	if user == nil {
		response.Fail(c, "User not logged in", nil)
		return
	}

	var body petProjectSaveBody
	if err := c.ShouldBindJSON(&body); err != nil {
		response.Fail(c, "Invalid parameters", err.Error())
		return
	}
	if strings.TrimSpace(body.Name) == "" {
		response.Fail(c, "Project name is required", nil)
		return
	}
	if err := petproject.ValidateFiles(body.Files); err != nil {
		response.Fail(c, "Invalid project files", err.Error())
		return
	}
	entry, violations := validatePetEntryScript(body.Entry, body.Files)
	if len(violations) > 0 {
		response.Fail(c, "代码不符合安全规范", gin.H{"violations": violations})
		return
	}

	gid, err := svcmodels.ResolveWriteGroupID(h.db, user.ID, nil)
	if err != nil {
		response.Fail(c, err.Error(), nil)
		return
	}

	template := svcmodels.JSTemplate{
		Name:      strings.TrimSpace(body.Name),
		Type:      "custom",
		Usage:     body.Usage,
		GroupID:   gid,
		CreatedBy: user.ID,
		Version:   1,
		Status:    "active",
	}
	db := c.MustGet(constants.DbField).(*gorm.DB)
	if err := svcmodels.CreateJSTemplate(db, &template); err != nil {
		response.Fail(c, "Failed to create template", err.Error())
		return
	}

	prefix, err := h.persistPetProject(&template, entry, body.Files, body.Name, body.Usage)
	if err != nil {
		_ = svcmodels.DeleteJSTemplate(db, template.ID)
		response.Fail(c, "Failed to upload project files", err.Error())
		return
	}

	updated, _ := svcmodels.GetJSTemplateByID(db, template.ID)
	versionContent := ""
	if updated != nil {
		versionContent = updated.Content
	}

	version := svcmodels.JSTemplateVersion{
		ID:         uuid.New().String(),
		TemplateID: template.ID,
		Version:    template.Version,
		Name:       template.Name,
		Content:    versionContent,
		Status:     template.Status,
		Grayscale:  100,
		ChangeNote: "初始版本",
		CreatedBy:  user.ID,
	}
	_ = svcmodels.CreateJSTemplateVersion(db, &version)

	response.Success(c, "Project created", gin.H{
		"template": updated,
		"storage":  petproject.StorageObject,
		"prefix":   prefix,
		"entry":    entry,
	})
}

// GetJSTemplateProject loads pet project files from object storage into API response.
func (h *Handlers) GetJSTemplateProject(c *gin.Context) {
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

	store := stores.Default()
	meta, _ := petproject.ParseMeta(template.Content)
	if meta != nil {
		paths := meta.ResolvePaths()
		files, err := petproject.ReadFiles(store, meta.Prefix, paths)
		if err != nil {
			response.Fail(c, "Failed to load project files", err.Error())
			return
		}
		response.Success(c, "ok", gin.H{
			"storage": petproject.StorageObject,
			"prefix":  meta.Prefix,
			"entry":   meta.Entry,
			"files":   files,
			"name":    template.Name,
			"usage":   template.Usage,
		})
		return
	}

	if inline, ok := petproject.ParseInlineProject(template.Content); ok {
		response.Success(c, "ok", gin.H{
			"storage": "inline",
			"entry":   inline.Entry,
			"files":   inline.Files,
			"name":    template.Name,
			"usage":   template.Usage,
		})
		return
	}

	response.Success(c, "ok", gin.H{
		"storage": "pending",
		"entry":   petproject.DefaultEntry,
		"files":   map[string]string{},
		"name":    template.Name,
		"usage":   template.Usage,
	})
}

// SaveJSTemplateProject persists project files to object storage; DB keeps metadata pointer only.
func (h *Handlers) SaveJSTemplateProject(c *gin.Context) {
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
	if template.Type == "default" {
		response.Fail(c, "Cannot update default template", nil)
		return
	}
	if !jsTemplateMutableByUser(h.db, user.ID, template) {
		response.Fail(c, "Insufficient permissions", nil)
		return
	}

	var body petProjectSaveBody
	if err := c.ShouldBindJSON(&body); err != nil {
		response.Fail(c, "Invalid parameters", err.Error())
		return
	}
	if err := petproject.ValidateFiles(body.Files); err != nil {
		response.Fail(c, "Invalid project files", err.Error())
		return
	}

	entry, violations := validatePetEntryScript(body.Entry, body.Files)
	if len(violations) > 0 {
		response.Fail(c, "代码不符合安全规范", gin.H{"violations": violations})
		return
	}

	prefix, err := h.persistPetProject(template, entry, body.Files, body.Name, body.Usage)
	if err != nil {
		response.Fail(c, "Failed to upload project files", err.Error())
		return
	}

	response.Success(c, "Project saved", gin.H{
		"storage": petproject.StorageObject,
		"prefix":  prefix,
		"entry":   entry,
	})
}

// resolveTemplateLoaderContent returns JS for loader.js (object storage or legacy inline).
func (h *Handlers) resolveTemplateLoaderContent(template *svcmodels.JSTemplate) string {
	if template == nil || template.Content == "" {
		return ""
	}
	store := stores.Default()
	script, err := petproject.EntryScript(store, template.ID, template.Content)
	if err != nil {
		return template.Content
	}
	return script
}

// resolveTemplateManifestJSON returns manifest object literal for embed preamble.
func (h *Handlers) resolveTemplateManifestJSON(template *svcmodels.JSTemplate) string {
	if template == nil || template.Content == "" {
		return "{}"
	}
	meta, legacy := petproject.ParseMeta(template.Content)
	if legacy {
		if inline, ok := petproject.ParseInlineProject(template.Content); ok {
			if raw, ok := inline.Files["manifest.json"]; ok && strings.TrimSpace(raw) != "" {
				return strings.TrimSpace(raw)
			}
		}
		return "{}"
	}
	if meta == nil {
		return "{}"
	}
	store := stores.Default()
	key, err := petproject.ObjectKey(meta.Prefix, "manifest.json")
	if err != nil {
		return "{}"
	}
	rc, _, err := store.Read(key)
	if err != nil {
		return "{}"
	}
	defer rc.Close()
	b, err := io.ReadAll(rc)
	if err != nil || len(b) == 0 {
		return "{}"
	}
	if !json.Valid(b) {
		return "{}"
	}
	return string(b)
}

// embedPetCSS is injected for JS embed — scoped only, must not touch host page layout.
const embedPetCSS = `.soul-pet-mount{position:relative;width:100%;height:100%;overflow:hidden;background:transparent}.soul-pet-mount .soul-pet-stage{position:absolute;inset:0;width:100%;height:100%;overflow:hidden}.soul-pet-mount .soul-pet-canvas{display:block;touch-action:none;width:100%;height:100%}`

// guardLegacyPetJS patches legacy pet.js for resize crashes and sprite frame timing.
func guardLegacyPetJS(script string) string {
	if strings.Contains(script, "liveDestroyed") {
		return script
	}
	script = strings.ReplaceAll(script, "app.renderer.resize(", "(app&&app.renderer)&&app.renderer.resize(")
	if strings.Contains(script, "drawFrameSheet") && !strings.Contains(script, "drawDt = lastTs") {
		script = strings.Replace(script, "var lastTs = 0", "var lastTs = 0\n  var drawDt = 0\n  var lastRenderedAnim = null", 1)
		script = strings.Replace(script,
			"function draw(ts) {\n    if (!ctx || destroyed) return\n    lastTs = ts",
			"function draw(ts) {\n    if (!ctx || destroyed) return\n    drawDt = lastTs ? (ts - lastTs) / 1000 : 0\n    lastTs = ts", 1)
		script = strings.Replace(script,
			"var dt = lastTs ? (performance.now() - lastTs) / 1000 : 0\n    frameAcc += dt * fps",
			"frameAcc += drawDt * fps", 1)
		if !strings.Contains(script, "lastRenderedAnim = animName") {
			script = strings.Replace(script,
				"var animName = pickAnim()\n    var entry = animName",
				"var animName = pickAnim()\n    if (animName !== lastRenderedAnim) {\n      frameIndex = 0\n      frameAcc = 0\n      lastRenderedAnim = animName\n    }\n    var entry = animName", 1)
		}
	}
	return script
}

func (h *Handlers) apiBaseURL(c *gin.Context) string {
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

// buildPetEmbedLoaderJS returns preamble (manifest, lip sync, voice) + pet.js for embed loaders.
func (h *Handlers) buildPetEmbedLoaderJS(c *gin.Context, template *svcmodels.JSTemplate, extraJS string) string {
	if template == nil {
		return ""
	}
	script := guardLegacyPetJS(h.resolveTemplateLoaderContent(template))
	if strings.TrimSpace(script) == "" {
		return ""
	}

	manifestJSON := h.resolveTemplateManifestJSON(template)
	baseURL := h.apiBaseURL(c)
	jsSourceID := template.JsSourceID

	preamble := fmt.Sprintf(
		"window.SERVER_BASE=%q;window.ASSISTANT_NAME=%q;window.__PET_TEMPLATE_ID__=%q;window.__PET_MANIFEST__=%s;window.__PET_PROJECT_BASE__=%q;\n",
		baseURL,
		template.Name,
		template.ID,
		manifestJSON,
		baseURL+"/js-templates/embed/"+jsSourceID+"/file/",
	)
	if strings.TrimSpace(extraJS) != "" {
		preamble += strings.TrimSpace(extraJS) + "\n"
	}
	preamble += fmt.Sprintf(
		"(function(){var el=document.getElementById('soul-pet-embed-style');if(el)return;var s=document.createElement('style');s.id='soul-pet-embed-style';s.textContent=%q;document.head.appendChild(s)})();\n",
		embedPetCSS,
	)
	voiceBridge := strings.TrimRight(SoulNexus.PetVoiceBridgeJS, " \t\r\n")
	if voiceBridge != "" && !strings.HasSuffix(voiceBridge, ";") {
		voiceBridge += ";"
	}
	preamble += voiceBridge + "\n"
	return preamble + script
}

func (h *Handlers) resolveTemplateStyleCSS(template *svcmodels.JSTemplate) string {
	if template == nil || template.Content == "" {
		return ""
	}
	meta, legacy := petproject.ParseMeta(template.Content)
	if legacy {
		if inline, ok := petproject.ParseInlineProject(template.Content); ok {
			return strings.TrimSpace(inline.Files["style.css"])
		}
		return ""
	}
	if meta == nil {
		return ""
	}
	store := stores.Default()
	key, err := petproject.ObjectKey(meta.Prefix, "style.css")
	if err != nil {
		return ""
	}
	rc, _, err := store.Read(key)
	if err != nil {
		return ""
	}
	defer rc.Close()
	b, err := io.ReadAll(rc)
	if err != nil {
		return ""
	}
	return string(b)
}

// ServeJSTemplatePetLoaderJS serves pet.js for third-party <script> embed (public, no auth).
func (h *Handlers) ServeJSTemplatePetLoaderJS(c *gin.Context) {
	jsSourceID := strings.TrimSpace(c.Param("jsSourceId"))
	if jsSourceID == "" {
		c.Data(http.StatusNotFound, "application/javascript; charset=utf-8", []byte("console.error('[SoulPet] missing jsSourceId');"))
		return
	}

	template, err := svcmodels.GetJSTemplateByJsSourceID(h.db, jsSourceID)
	if err != nil {
		c.Data(http.StatusNotFound, "application/javascript; charset=utf-8", []byte("console.error('[SoulPet] template not found');"))
		return
	}

	body := h.buildPetEmbedLoaderJS(c, template, "")
	if strings.TrimSpace(body) == "" {
		c.Data(http.StatusNotFound, "application/javascript; charset=utf-8", []byte("console.error('[SoulPet] empty project — save in Studio first');"))
		return
	}

	c.Header("Content-Type", "application/javascript; charset=utf-8")
	c.Header("Cache-Control", "public, max-age=60")
	c.String(http.StatusOK, body)
}
