package server

import (
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"

	svcmodels "github.com/LingByte/SoulNexus/internal/models/server"
	"github.com/LingByte/SoulNexus"
	"github.com/LingByte/SoulNexus/internal/models/auth"
	"github.com/LingByte/SoulNexus/pkg/constants"
	"github.com/LingByte/SoulNexus/pkg/petproject"
	"github.com/LingByte/SoulNexus/pkg/response"
	"github.com/LingByte/SoulNexus/pkg/stores"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

func (h *Handlers) loadMarketListingFiles(listing *svcmodels.PetMarketListing) (entry string, files map[string]string, err error) {
	if listing == nil || listing.PackageMeta == "" {
		return petproject.DefaultEntry, nil, fmt.Errorf("empty listing package")
	}
	meta, legacy := petproject.ParseMeta(listing.PackageMeta)
	if legacy || meta == nil {
		return petproject.DefaultEntry, nil, fmt.Errorf("invalid listing package meta")
	}
	store := stores.Default()
	files, err = petproject.ReadFiles(store, meta.Prefix, meta.ResolvePaths())
	if err != nil {
		return "", nil, err
	}
	return meta.Entry, files, nil
}

func (h *Handlers) persistMarketListingPackage(listing *svcmodels.PetMarketListing, entry string, files map[string]string, zipRaw []byte) error {
	prefix := petproject.MarketListingPrefix(listing.ID)
	store := stores.Default()
	if err := writePetProjectFiles(store, prefix, listing.ID, files, zipRaw); err != nil {
		return err
	}
	metaJSON, err := petproject.MarshalMeta(prefix, entry, petproject.PathsFromFiles(files))
	if err != nil {
		return err
	}
	listing.PackageMeta = metaJSON
	return h.db.Model(listing).Update("package_meta", metaJSON).Error
}

// ListPetMarketListings public marketplace browse.
func (h *Handlers) ListPetMarketListings(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	keyword := c.Query("keyword")
	rows, total, err := svcmodels.ListPublicPetMarketListings(h.db, page, limit, keyword)
	if err != nil {
		response.Fail(c, "Failed to list marketplace", err.Error())
		return
	}
	response.Success(c, "ok", gin.H{
		"data":  rows,
		"page":  page,
		"limit": limit,
		"total": total,
	})
}

// GetPetMarketListing returns listing metadata (no full files unless owner).
func (h *Handlers) GetPetMarketListing(c *gin.Context) {
	marketID := strings.TrimSpace(c.Param("marketId"))
	listing, err := svcmodels.GetPetMarketListingByMarketID(h.db, marketID)
	if err != nil {
		response.Fail(c, "Listing not found", nil)
		return
	}
	if listing.Visibility != "public" {
		user := auth.CurrentUser(c)
		if user == nil || user.ID != listing.AuthorID {
			response.Fail(c, "Listing not found", nil)
			return
		}
	}
	payload := gin.H{
		"listing": listing,
	}
	if user := auth.CurrentUser(c); user != nil {
		if score, ok := svcmodels.GetUserPetMarketRating(h.db, listing.ID, user.ID); ok {
			payload["userRating"] = score
		}
	}
	response.Success(c, "ok", payload)
}

// PublishPetMarketListing uploads .soulpet to public market (no jsSourceId).
func (h *Handlers) PublishPetMarketListing(c *gin.Context) {
	user := auth.CurrentUser(c)
	if user == nil {
		response.Fail(c, "User not logged in", nil)
		return
	}

	files, entry, zipRaw, err := h.readPetPackageInput(c)
	if err != nil {
		response.Fail(c, "Invalid package", err.Error())
		return
	}

	kind, issues := petproject.ValidateSoulpetPackage(files, validatePetEntryScript)
	if petproject.HasValidationErrors(issues) {
		response.Fail(c, "Package validation failed", gin.H{"issues": issues, "kind": kind})
		return
	}

	name := strings.TrimSpace(c.PostForm("name"))
	if name == "" {
		if meta, err := petproject.ParseSoulpetYAML(files[petproject.SoulpetYamlFile]); err == nil && meta.Name != "" {
			name = meta.Name
		}
	}
	if name == "" {
		if m, err := petproject.ParseManifestJSON(files[petproject.ManifestFile]); err == nil {
			name = m.Name
		}
	}
	if name == "" {
		name = "Soul Pet"
	}
	desc := strings.TrimSpace(c.PostForm("description"))
	if entry == "" {
		entry = petproject.DefaultEntry
	}
	if _, ok := files[petproject.SoulpetYamlFile]; !ok {
		files[petproject.SoulpetYamlFile] = petproject.EnsureSoulpetYAML(files, name)
	}

	gid, err := svcmodels.ResolveWriteGroupID(h.db, user.ID, nil)
	if err != nil {
		response.Fail(c, err.Error(), nil)
		return
	}

	tags := strings.TrimSpace(c.PostForm("tags"))
	previewEmoji := strings.TrimSpace(c.PostForm("previewEmoji"))
	if previewEmoji == "" {
		if meta, err := petproject.ParseSoulpetYAML(files[petproject.SoulpetYamlFile]); err == nil && meta.Market != nil {
			previewEmoji = meta.Market.PreviewEmoji
		}
	}

	listing := svcmodels.PetMarketListing{
		Name:         name,
		Description:  desc,
		Kind:         kind,
		AuthorID:     user.ID,
		GroupID:      gid,
		Tags:         tags,
		PreviewEmoji: previewEmoji,
		Visibility:   "public",
	}
	if err := svcmodels.CreatePetMarketListing(h.db, &listing); err != nil {
		response.Fail(c, "Failed to create listing", err.Error())
		return
	}
	if err := h.persistMarketListingPackage(&listing, entry, files, zipRaw); err != nil {
		_ = h.db.Delete(&listing).Error
		response.Fail(c, "Failed to upload listing files", err.Error())
		return
	}

	response.Success(c, "Published", gin.H{
		"marketId": listing.MarketID,
		"kind":     kind,
		"listing":  listing,
	})
}

// ForkPetMarketListing copies market package → 我的桌宠 (new jsSourceId).
func (h *Handlers) ForkPetMarketListing(c *gin.Context) {
	user := auth.CurrentUser(c)
	if user == nil {
		response.Fail(c, "User not logged in", nil)
		return
	}

	marketID := strings.TrimSpace(c.Param("marketId"))
	listing, err := svcmodels.GetPetMarketListingByMarketID(h.db, marketID)
	if err != nil {
		response.Fail(c, "Listing not found", nil)
		return
	}
	if listing.Visibility != "public" && listing.AuthorID != user.ID {
		response.Fail(c, "Listing not found", nil)
		return
	}

	entry, files, err := h.loadMarketListingFiles(listing)
	if err != nil {
		response.Fail(c, "Failed to load listing files", err.Error())
		return
	}

	var body struct {
		Name string `json:"name"`
	}
	_ = c.ShouldBindJSON(&body)
	name := strings.TrimSpace(body.Name)
	if name == "" {
		name = listing.Name + " (Fork)"
	}

	gid, err := svcmodels.ResolveWriteGroupID(h.db, user.ID, nil)
	if err != nil {
		response.Fail(c, err.Error(), nil)
		return
	}

	template := svcmodels.JSTemplate{
		Name:      name,
		Type:      "custom",
		Usage:     listing.Description,
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

	prefix, err := h.persistPetProject(&template, entry, files, name, listing.Description, nil)
	if err != nil {
		_ = svcmodels.DeleteJSTemplate(db, template.ID)
		response.Fail(c, "Failed to copy files", err.Error())
		return
	}

	_ = h.db.Model(listing).UpdateColumn("fork_count", gorm.Expr("fork_count + 1"))

	updated, _ := svcmodels.GetJSTemplateByID(db, template.ID)
	version := svcmodels.JSTemplateVersion{
		ID:         uuid.New().String(),
		TemplateID: template.ID,
		Version:    template.Version,
		Name:       template.Name,
		Content:    updated.Content,
		Status:     template.Status,
		Grayscale:  100,
		ChangeNote: "fork from market " + marketID,
		CreatedBy:  user.ID,
	}
	_ = svcmodels.CreateJSTemplateVersion(db, &version)

	response.Success(c, "Forked", gin.H{
		"template": updated,
		"storage":  petproject.StorageObject,
		"prefix":   prefix,
		"entry":    entry,
		"marketId": marketID,
	})
}

// DownloadPetMarketListingZip downloads public listing as .soulpet.zip.
func (h *Handlers) DownloadPetMarketListingZip(c *gin.Context) {
	marketID := strings.TrimSpace(c.Param("marketId"))
	listing, err := svcmodels.GetPetMarketListingByMarketID(h.db, marketID)
	if err != nil || listing.Visibility != "public" {
		response.Fail(c, "Listing not found", nil)
		return
	}
	_, files, err := h.loadMarketListingFiles(listing)
	if err != nil {
		response.Fail(c, "Failed to load files", err.Error())
		return
	}
	if _, ok := files[petproject.SoulpetYamlFile]; !ok {
		files[petproject.SoulpetYamlFile] = petproject.EnsureSoulpetYAML(files, listing.Name)
	}
	zipBytes, err := petproject.PackZip(files)
	if err != nil {
		response.Fail(c, "Failed to pack zip", err.Error())
		return
	}
	_ = h.db.Model(listing).UpdateColumn("download_count", gorm.Expr("download_count + 1"))
	safeName := strings.ReplaceAll(listing.Name, " ", "-")
	c.Header("Content-Type", "application/zip")
	c.Header("Content-Disposition", fmt.Sprintf(`attachment; filename="%s.soulpet.zip"`, safeName))
	c.Data(http.StatusOK, "application/zip", zipBytes)
}

// ServePetMarketPreviewLoaderJS public read-only preview (marketId, not jsSourceId).
func (h *Handlers) ServePetMarketPreviewLoaderJS(c *gin.Context) {
	marketID := strings.TrimSpace(c.Param("marketId"))
	listing, err := svcmodels.GetPetMarketListingByMarketID(h.db, marketID)
	if err != nil || listing.Visibility != "public" {
		c.Data(http.StatusNotFound, "application/javascript; charset=utf-8", []byte("console.error('[SoulPet] market listing not found');"))
		return
	}
	body := h.buildPetMarketPreviewLoaderJS(c, listing)
	if strings.TrimSpace(body) == "" {
		c.Data(http.StatusNotFound, "application/javascript; charset=utf-8", []byte("console.error('[SoulPet] empty market listing');"))
		return
	}
	c.Header("Content-Type", "application/javascript; charset=utf-8")
	c.Header("Cache-Control", "public, max-age=120")
	c.Data(http.StatusOK, "application/javascript; charset=utf-8", []byte(body))
}

// ServePetMarketPreviewFile serves static files for market preview.
func (h *Handlers) ServePetMarketPreviewFile(c *gin.Context) {
	marketID := strings.TrimSpace(c.Param("marketId"))
	filepath := strings.TrimPrefix(c.Param("filepath"), "/")
	listing, err := svcmodels.GetPetMarketListingByMarketID(h.db, marketID)
	if err != nil || listing.Visibility != "public" {
		c.Status(http.StatusNotFound)
		return
	}
	meta, legacy := petproject.ParseMeta(listing.PackageMeta)
	if legacy || meta == nil {
		c.Status(http.StatusNotFound)
		return
	}
	key, err := petproject.ObjectKey(meta.Prefix, filepath)
	if err != nil {
		c.Status(http.StatusBadRequest)
		return
	}
	store := stores.Default()
	rc, size, err := store.Read(key)
	if err != nil {
		c.Status(http.StatusNotFound)
		return
	}
	defer rc.Close()
	data, err := io.ReadAll(rc)
	if err != nil {
		c.Status(http.StatusInternalServerError)
		return
	}
	c.Header("Cache-Control", "public, max-age=300")
	c.Data(http.StatusOK, petMarketFileContentType(filepath, size), data)
}

func (h *Handlers) buildPetMarketPreviewLoaderJS(c *gin.Context, listing *svcmodels.PetMarketListing) string {
	entry, files, err := h.loadMarketListingFiles(listing)
	if err != nil {
		return ""
	}
	script := guardLegacyPetJS(petproject.EntryScriptFromFiles(files, entry))
	if strings.TrimSpace(script) == "" {
		return ""
	}
	if !strings.Contains(script, "soul-pet-desktop-capable") {
		script = strings.ReplaceAll(script, "getElementById('app')", "getElementById(window.__PET_MOUNT_ID__||'soul-pet-desktop-root')")
		script = strings.ReplaceAll(script, "getElementById(\"app\")", "getElementById(window.__PET_MOUNT_ID__||'soul-pet-desktop-root')")
	}
	script = patchEmbedSpriteRuntime(script)

	manifestJSON := petproject.ManifestJSONFromFiles(files)
	baseURL := h.apiBaseURL(c)
	fileBase := baseURL + "/pet-market/" + listing.MarketID + "/preview/file/"

	preamble := fmt.Sprintf(
		"window.SERVER_BASE=%q;window.ASSISTANT_NAME=%q;window.__PET_MARKET_ID__=%q;window.__PET_MANIFEST__=%s;window.__PET_PROJECT_BASE__=%q;\n",
		baseURL,
		listing.Name,
		listing.MarketID,
		manifestJSON,
		fileBase,
	)
	preamble += "(function(){var c=window.__AIPetConfig||{};if(c.mode!=='widget'&&!window.__PET_EMBED_MODE__)window.__PET_EMBED_MODE__='desktop';})();\n"
	preamble += embedPandaManifestPatchJS + "\n"
	preamble += embedDesktopBootstrapJS + "\n"
	preamble += fmt.Sprintf(
		"(function(){var el=document.getElementById('soul-pet-embed-style');if(el)return;var s=document.createElement('style');s.id='soul-pet-embed-style';s.textContent=%q;document.head.appendChild(s)})();\n",
		embedPetCSS,
	)
	sdkJS := strings.TrimRight(SoulNexus.SoulPetSDKJS, " \t\r\n")
	if sdkJS != "" && !strings.HasSuffix(sdkJS, ";") {
		sdkJS += ";"
	}
	if sdkJS != "" {
		preamble += sdkJS + "\n"
	}
	voiceBridge := strings.TrimRight(SoulNexus.PetVoiceBridgeJS, " \t\r\n")
	if voiceBridge != "" && !strings.HasSuffix(voiceBridge, ";") {
		voiceBridge += ";"
	}
	preamble += voiceBridge + "\n"
	return preamble + script
}

func petMarketFileContentType(name string, _ int64) string {
	switch strings.ToLower(pathExt(name)) {
	case ".png":
		return "image/png"
	case ".jpg", ".jpeg":
		return "image/jpeg"
	case ".webp":
		return "image/webp"
	case ".gif":
		return "image/gif"
	case ".json":
		return "application/json"
	case ".js":
		return "application/javascript; charset=utf-8"
	case ".css":
		return "text/css; charset=utf-8"
	case ".moc3":
		return "application/octet-stream"
	default:
		return "application/octet-stream"
	}
}

func pathExt(name string) string {
	if i := strings.LastIndex(name, "."); i >= 0 {
		return name[i:]
	}
	return ""
}

// PushPetPackage updates existing template from JSON files (CLI push).
func (h *Handlers) PushPetPackage(c *gin.Context) {
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

	kind, issues := petproject.ValidateSoulpetPackage(body.Files, validatePetEntryScript)
	if petproject.HasValidationErrors(issues) {
		response.Fail(c, "Package validation failed", gin.H{"issues": issues, "kind": kind})
		return
	}

	entry, violations := validatePetEntryScript(body.Entry, body.Files)
	if len(violations) > 0 {
		response.Fail(c, "代码不符合安全规范", gin.H{"violations": violations})
		return
	}

	files := body.Files
	var packageVersion string
	if shouldBumpPetVersion(body) {
		if err := h.archivePetTemplateVersion(template, user.ID, strings.TrimSpace(body.ChangeNote)); err != nil {
			response.Fail(c, "Failed to record version history", err.Error())
			return
		}
		template.Version++
		files, packageVersion = petproject.ApplyVersionBump(files, body.ChangeNote, true)
	}

	prefix, err := h.persistPetProject(template, entry, files, body.Name, body.Usage, nil)
	if err != nil {
		response.Fail(c, "Failed to upload project files", err.Error())
		return
	}

	updated, _ := svcmodels.GetJSTemplateByID(h.db, template.ID)
	out := gin.H{
		"kind":       kind,
		"template":   updated,
		"jsSourceId": updated.JsSourceID,
		"storage":    petproject.StorageObject,
		"prefix":     prefix,
		"entry":      entry,
		"version":    template.Version,
	}
	if packageVersion != "" {
		out["packageVersion"] = packageVersion
	}
	response.Success(c, "Pushed", out)
}

// RatePetMarketListing stores a 1–5 star rating for a public listing.
func (h *Handlers) RatePetMarketListing(c *gin.Context) {
	user := auth.CurrentUser(c)
	if user == nil {
		response.Fail(c, "User not logged in", nil)
		return
	}

	marketID := strings.TrimSpace(c.Param("marketId"))
	listing, err := svcmodels.GetPetMarketListingByMarketID(h.db, marketID)
	if err != nil || listing.Visibility != "public" {
		response.Fail(c, "Listing not found", nil)
		return
	}

	var input struct {
		Score int `json:"score" binding:"required"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		response.Fail(c, "Invalid score", err.Error())
		return
	}
	if input.Score < 1 || input.Score > 5 {
		response.Fail(c, "Invalid score", "Score must be 1–5")
		return
	}

	if err := svcmodels.UpsertPetMarketRating(h.db, listing.ID, user.ID, input.Score); err != nil {
		response.Fail(c, "Rate failed", err.Error())
		return
	}
	if err := svcmodels.RecomputeListingRating(h.db, listing); err != nil {
		response.Fail(c, "Rate failed", err.Error())
		return
	}

	response.Success(c, "Rated", gin.H{
		"rating":      listing.Rating,
		"ratingCount": listing.RatingCount,
		"userRating":  input.Score,
	})
}

// PullPetPackage returns full project files for CLI pull.
func (h *Handlers) PullPetPackage(c *gin.Context) {
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

	entry, files, err := h.loadTemplateProjectFiles(template)
	if err != nil {
		response.Fail(c, "Failed to load project files", err.Error())
		return
	}
	if _, ok := files[petproject.SoulpetYamlFile]; !ok {
		files[petproject.SoulpetYamlFile] = petproject.EnsureSoulpetYAML(files, template.Name)
	}

	response.Success(c, "ok", gin.H{
		"templateId": template.ID,
		"jsSourceId": template.JsSourceID,
		"name":       template.Name,
		"entry":      entry,
		"files":      files,
	})
}
