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
	jsPkg "github.com/LingByte/SoulNexus/pkg/js"
	"github.com/LingByte/SoulNexus/pkg/petproject"
	"github.com/LingByte/SoulNexus/pkg/stores"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type petProjectSaveBody struct {
	Name        string            `json:"name"`
	Usage       string            `json:"usage"`
	Entry       string            `json:"entry"`
	Files       map[string]string `json:"files"`
	ChangeNote  string            `json:"changeNote"`
	BumpVersion *bool             `json:"bumpVersion"`
}

func (h *Handlers) archivePetTemplateVersion(template *svcmodels.JSTemplate, userID uint, changeNote string) error {
	if changeNote == "" {
		changeNote = "项目更新"
	}
	snapshot := svcmodels.JSTemplateVersion{
		ID:         uuid.New().String(),
		TemplateID: template.ID,
		Version:    template.Version,
		Name:       template.Name,
		Content:    template.Content,
		Status:     template.Status,
		Grayscale:  100,
		ChangeNote: changeNote,
		CreatedBy:  userID,
	}
	if err := svcmodels.CreateJSTemplateVersion(h.db, &snapshot); err != nil {
		return err
	}
	return h.db.Model(template).Update("version", template.Version+1).Error
}

func shouldBumpPetVersion(body petProjectSaveBody) bool {
	if body.BumpVersion != nil {
		return *body.BumpVersion
	}
	return strings.TrimSpace(body.ChangeNote) != ""
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

func (h *Handlers) persistPetProject(template *svcmodels.JSTemplate, entry string, files map[string]string, name, usage string, zipRaw []byte) (string, error) {
	prefix := petproject.DefaultPrefix(template.ID)
	store := stores.Default()
	if err := writePetProjectFiles(store, prefix, template.ID, files, zipRaw); err != nil {
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

func writePetProjectFiles(store petproject.FullStore, prefix, packageID string, files map[string]string, zipRaw []byte) error {
	if len(zipRaw) > 0 {
		_, err := petproject.PublishZip(store, packageID, zipRaw, prefix)
		return err
	}
	return petproject.WriteFiles(store, prefix, files)
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
const embedPetCSS = `.soul-pet-mount{position:relative;width:100%;height:100%;overflow:hidden;background:transparent}.soul-pet-mount.soul-pet-desktop{position:fixed!important;inset:0!important;width:100vw!important;height:100vh!important;max-width:none!important;max-height:none!important;pointer-events:none!important;z-index:2147483645!important;background:transparent!important}.soul-pet-mount.soul-pet-desktop .soul-pet-stage{pointer-events:auto;width:100%!important;height:100%!important;max-width:none!important;max-height:none!important}.soul-pet-mount .soul-pet-stage{position:absolute;inset:0;width:100%;height:100%;overflow:hidden}.soul-pet-mount .soul-pet-canvas{display:block;touch-action:none;width:100%;height:100%}`

// embedDesktopBootstrapJS forces full-viewport mount for third-party embed (no host div required).
const embedDesktopBootstrapJS = `(function soulPetDesktopEmbedBootstrap(){
if(window.__SOUL_PET_DESKTOP_BOOTSTRAP__)return;
window.__SOUL_PET_DESKTOP_BOOTSTRAP__=true;
var cfg=window.__AIPetConfig||{};
if(cfg.mode==='widget'||window.__PET_EMBED_MODE__==='widget')return;
window.__PET_EMBED_MODE__='desktop';
window.__PET_MOUNT_ID__='soul-pet-desktop-root';
var DESKTOP_STYLE='position:fixed;inset:0;width:100vw;height:100vh;overflow:hidden;pointer-events:none;z-index:2147483645;background:transparent;';
function ensureRoot(){
  var r=document.getElementById('soul-pet-desktop-root');
  if(!r){r=document.createElement('div');r.id='soul-pet-desktop-root';document.body.appendChild(r);}
  if(r.dataset.soulDesktopReady==='1'&&r.style.position==='fixed')return;
  r.dataset.soulDesktopReady='1';
  r.className='soul-pet-mount soul-pet-desktop';
  r.style.cssText=DESKTOP_STYLE;
  var st=r.querySelector('.soul-pet-stage');
  if(st)st.style.pointerEvents='auto';
}
ensureRoot();
if(document.readyState==='loading')document.addEventListener('DOMContentLoaded',ensureRoot);
setTimeout(ensureRoot,0);
setTimeout(ensureRoot,120);
setTimeout(ensureRoot,500);
})();`

// embedPandaManifestPatchJS points panda frame sequences at bundled static sprites.
const embedPandaManifestPatchJS = `(function(){
try{
  var m=window.__PET_MANIFEST__;
  if(!m||!m.assets||!m.assets.sprite)return;
  if((m.assets.sprite.baseUrl||'').indexOf('http')===0)return;
  var anims=m.assets.sprite.animations||{};
  var idle=anims.idle||{};
  var f=(idle.files&&idle.files[0])||'';
  if(f.indexOf('panda/action_')!==0)return;
  var base=(window.SERVER_BASE||'').replace(/\/+$/, '');
  m.assets.sprite.baseUrl=base+'/static/pet/examples/sprites/';
}catch(e){}
})();`

const embedSpriteLazyHelpersJS = `
  function ensureLazyFrame(entry, idx) {
    if (!entry.lazyFiles) return entry.imgs && entry.imgs[idx]
    if (!entry.imgs) entry.imgs = []
    if (entry.imgs[idx]) return entry.imgs[idx]
    if (!entry._loading) entry._loading = {}
    if (entry._loading[idx]) return null
    var path = entry.lazyFiles[idx]
    if (path == null) return null
    entry._loading[idx] = true
    var img = new Image()
    img.crossOrigin = 'anonymous'
    img.onload = function () { entry.imgs[idx] = img; delete entry._loading[idx] }
    img.onerror = function () { loadErrors.push(path); delete entry._loading[idx] }
    img.src = assetUrl(path)
    return null
  }
  function prefetchLazyFrames(entry, idx, ahead) {
    var max = Math.max(1, (entry.def && entry.def.frames) || (entry.lazyFiles && entry.lazyFiles.length) || 1)
    for (var d = 0; d < ahead; d++) {
      var i = idx + d
      if (i >= 0 && i < max) ensureLazyFrame(entry, i)
    }
  }
`

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

func patchEmbedSpriteRuntime(script string) string {
	if strings.Contains(script, "LAZY_FRAME_THRESHOLD") || !strings.Contains(script, "function loadSheet") {
		return script
	}
	script = strings.Replace(script, "var loadErrors = []", "var loadErrors = []\n  var LAZY_FRAME_THRESHOLD = 16", 1)
	if idx := strings.Index(script, "function loadSheet"); idx > 0 {
		script = script[:idx] + embedSpriteLazyHelpersJS + "\n" + script[idx:]
	}
	script = strings.Replace(script,
		"if (def.files && def.files.length) {\n        Promise.all(def.files.map(function (path) {",
		"if (def.files && def.files.length) {\n        if (def.files.length > LAZY_FRAME_THRESHOLD) {\n          resolve({ name: name, def: def, lazyFiles: def.files, imgs: [] })\n          return\n        }\n        Promise.all(def.files.map(function (path) {",
		1)
	script = strings.Replace(script,
		"var count = Math.max(1, def.frames || (entry.imgs && entry.imgs.length) || 1)",
		"var count = Math.max(1, def.frames || (entry.lazyFiles && entry.lazyFiles.length) || (entry.imgs && entry.imgs.length) || 1)",
		1)
	script = strings.Replace(script,
		"if (layout.offsetY) dy += layout.offsetY\n    if (entry.imgs && entry.imgs.length) {",
		"if (layout.offsetY) dy += layout.offsetY\n    if (entry.lazyFiles && entry.lazyFiles.length) {\n      prefetchLazyFrames(entry, idx, 4)\n      var lazyImg = ensureLazyFrame(entry, idx)\n      if (lazyImg) ctx.drawImage(lazyImg, 0, 0, fw, fh, dx, dy, dw, dh)\n      return\n    }\n    if (entry.imgs && entry.imgs.length) {",
		1)
	script = strings.Replace(script,
		"if (entry && (entry.img || (entry.imgs && entry.imgs.length))) drawFrameSheet(entry, w, h)",
		"if (entry && (entry.img || (entry.imgs && entry.imgs.length) || (entry.lazyFiles && entry.lazyFiles.length))) drawFrameSheet(entry, w, h)",
		1)
	script = strings.Replace(script,
		"if (root.style.position !== 'fixed' && root.style.position !== 'absolute') {",
		"if (window.__PET_EMBED_MODE__ !== 'desktop' && root.style.position !== 'fixed' && root.style.position !== 'absolute') {",
		-1)
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
	if !strings.Contains(script, "soul-pet-desktop-capable") {
		script = strings.ReplaceAll(script, "getElementById('app')", "getElementById(window.__PET_MOUNT_ID__||'soul-pet-desktop-root')")
		script = strings.ReplaceAll(script, "getElementById(\"app\")", "getElementById(window.__PET_MOUNT_ID__||'soul-pet-desktop-root')")
	}
	script = patchEmbedSpriteRuntime(script)

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
	preamble += "(function(){var c=window.__AIPetConfig||{};if(c.mode!=='widget'&&!window.__PET_EMBED_MODE__)window.__PET_EMBED_MODE__='desktop';})();\n"
	preamble += embedPandaManifestPatchJS + "\n"
	preamble += embedDesktopBootstrapJS + "\n"
	if strings.TrimSpace(extraJS) != "" {
		preamble += strings.TrimSpace(extraJS) + "\n"
	}
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
		c.Data(http.StatusNotFound, "application/javascript; charset=utf-8", []byte("console.error('[SoulPet] empty project — import a .soulpet.zip first');"))
		return
	}

	c.Header("Content-Type", "application/javascript; charset=utf-8")
	c.Header("Cache-Control", "public, max-age=60")
	c.String(http.StatusOK, body)
}
