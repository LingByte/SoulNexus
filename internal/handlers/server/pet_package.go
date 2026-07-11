package server

import (
	"fmt"
	"io"
	"net/http"
	"strings"

	svcmodels "github.com/LingByte/SoulNexus/internal/models/server"
	"github.com/LingByte/SoulNexus/internal/models/auth"
	"github.com/LingByte/SoulNexus/pkg/constants"
	"github.com/LingByte/SoulNexus/pkg/petproject"
	"github.com/LingByte/SoulNexus/pkg/response"
	"github.com/LingByte/SoulNexus/pkg/stores"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type petPackageValidateBody struct {
	Entry string            `json:"entry"`
	Files map[string]string `json:"files"`
}

func (h *Handlers) loadTemplateProjectFiles(template *svcmodels.JSTemplate) (entry string, files map[string]string, err error) {
	if template == nil {
		return "", nil, fmt.Errorf("template is nil")
	}
	store := stores.Default()
	meta, _ := petproject.ParseMeta(template.Content)
	if meta != nil {
		paths := meta.ResolvePaths()
		files, err = petproject.ReadFiles(store, meta.Prefix, paths)
		if err != nil {
			return "", nil, err
		}
		return meta.Entry, files, nil
	}
	if inline, ok := petproject.ParseInlineProject(template.Content); ok {
		return inline.Entry, inline.Files, nil
	}
	return petproject.DefaultEntry, map[string]string{}, nil
}

// ValidatePetPackage checks .soulpet package files before import/push.
func (h *Handlers) ValidatePetPackage(c *gin.Context) {
	user := auth.CurrentUser(c)
	if user == nil {
		response.Fail(c, "User not logged in", nil)
		return
	}

	files, entry, _, err := h.readPetPackageInput(c)
	if err != nil {
		response.Fail(c, "Invalid package", err.Error())
		return
	}

	kind, issues := petproject.ValidateSoulpetPackage(files, validatePetEntryScript)
	response.Success(c, "ok", gin.H{
		"kind":    kind,
		"valid":   !petproject.HasValidationErrors(issues),
		"issues":  issues,
		"entry":   entry,
		"files":   len(files),
	})
}

// ImportPetPackage creates a new「我的桌宠」from .soulpet files or zip.
func (h *Handlers) ImportPetPackage(c *gin.Context) {
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
		if raw, ok := files[petproject.SoulpetYamlFile]; ok {
			if meta, err := petproject.ParseSoulpetYAML(raw); err == nil && meta.Name != "" {
				name = meta.Name
			}
		}
		if name == "" {
			if m, err := petproject.ParseManifestJSON(files[petproject.ManifestFile]); err == nil {
				name = m.Name
			}
		}
	}
	if name == "" {
		name = "Imported Pet"
	}
	usage := strings.TrimSpace(c.PostForm("usage"))
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

	template := svcmodels.JSTemplate{
		Name:      name,
		Type:      "custom",
		Usage:     usage,
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

	prefix, err := h.persistPetProject(&template, entry, files, name, usage, zipRaw)
	if err != nil {
		_ = svcmodels.DeleteJSTemplate(db, template.ID)
		response.Fail(c, "Failed to upload project files", err.Error())
		return
	}

	updated, _ := svcmodels.GetJSTemplateByID(db, template.ID)
	versionContent := ""
	jsSourceID := template.JsSourceID
	if updated != nil {
		versionContent = updated.Content
		jsSourceID = updated.JsSourceID
	}
	version := svcmodels.JSTemplateVersion{
		ID:         uuid.New().String(),
		TemplateID: template.ID,
		Version:    template.Version,
		Name:       template.Name,
		Content:    versionContent,
		Status:     template.Status,
		Grayscale:  100,
		ChangeNote: "import .soulpet",
		CreatedBy:  user.ID,
	}
	_ = svcmodels.CreateJSTemplateVersion(db, &version)

	response.Success(c, "Imported", gin.H{
		"kind":       kind,
		"template":   updated,
		"jsSourceId": jsSourceID,
		"storage":    petproject.StorageObject,
		"prefix":     prefix,
		"entry":      entry,
	})
}

// ExportJSTemplateProjectZip downloads project as .soulpet.zip.
func (h *Handlers) ExportJSTemplateProjectZip(c *gin.Context) {
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
	if len(files) == 0 {
		response.Fail(c, "Project has no files", nil)
		return
	}
	if _, ok := files[petproject.SoulpetYamlFile]; !ok {
		files[petproject.SoulpetYamlFile] = petproject.EnsureSoulpetYAML(files, template.Name)
	}

	zipBytes, err := petproject.PackZip(files)
	if err != nil {
		response.Fail(c, "Failed to pack zip", err.Error())
		return
	}

	safeName := strings.ReplaceAll(template.Name, " ", "-")
	filename := fmt.Sprintf("%s.soulpet.zip", safeName)
	c.Header("Content-Type", "application/zip")
	c.Header("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, filename))
	c.Header("X-Soul-Pet-Entry", entry)
	c.Data(http.StatusOK, "application/zip", zipBytes)
}

func (h *Handlers) readPetPackageInput(c *gin.Context) (files map[string]string, entry string, zipRaw []byte, err error) {
	contentType := c.GetHeader("Content-Type")
	if strings.HasPrefix(contentType, "multipart/form-data") {
		file, ferr := c.FormFile("package")
		if ferr != nil {
			return nil, "", nil, ferr
		}
		f, ferr := file.Open()
		if ferr != nil {
			return nil, "", nil, ferr
		}
		defer f.Close()
		raw, ferr := io.ReadAll(f)
		if ferr != nil {
			return nil, "", nil, ferr
		}
		files, err = petproject.UnpackZip(raw)
		if err != nil {
			return nil, "", nil, err
		}
		entry = strings.TrimSpace(c.PostForm("entry"))
		return files, entry, raw, nil
	}

	var body petPackageValidateBody
	if err := c.ShouldBindJSON(&body); err != nil {
		return nil, "", nil, err
	}
	entry = strings.TrimSpace(body.Entry)
	return body.Files, entry, nil, nil
}
