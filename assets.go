package LingEcho

import (
	"bytes"
	"embed"
	_ "embed"
	"errors"
	"html/template"
	"io"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strconv"

	"github.com/LingByte/SoulNexus/pkg/constants"
	"github.com/LingByte/SoulNexus/pkg/utils"
	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/render"
	"gorm.io/gorm"
)

//go:embed static/admin/monitor.html
var MonitorUIHTML string

//go:embed static
var EmbedStaticAssets embed.FS

//go:embed templates
var EmbedTemplates embed.FS

//go:embed templates/email/welcome.html
var WelcomeHTML string

//go:embed templates/email/verification.html
var VerificationHTML string

//go:embed templates/email/group_invitation.html
var GroupInvitationHTML string

//go:embed templates/email/email_verification.html
var EmailVerificationHTML string

//go:embed templates/email/password_reset.html
var PasswordResetHTML string

//go:embed templates/email/device_verification.html
var DeviceVerificationHTML string

//go:embed templates/email/new_device_login.html
var NewDeviceLoginHTML string

//go:embed static/js/client.js
var AssistantJsModule string

//go:embed static/js/lingecho-sdk.js
var LingEchoSDKModule string

type CombineEmbedFS struct {
	embeds    []EmbedFS
	assertDir string
}
type EmbedFS struct {
	EmbedRoot string
	Embedfs   embed.FS
}

func NewCombineEmbedFS(assertDir string, es ...EmbedFS) *CombineEmbedFS {
	return &CombineEmbedFS{
		embeds:    es,
		assertDir: assertDir,
	}
}

func (c *CombineEmbedFS) Open(name string) (fs.File, error) {
	if c.assertDir != "" {
		f, err := os.Open(filepath.Join(c.assertDir, name))
		if err == nil {
			return f, nil
		}
	}

	var err error
	var ef fs.File

	for _, efs := range c.embeds {
		ef, err = efs.Embedfs.Open(filepath.Join(efs.EmbedRoot, name))
		if err == nil {
			return EmbedFile{ef}, nil
		}
	}

	return EmbedFile{ef}, err
}

func (c *CombineEmbedFS) ReadDir(name string) ([]fs.DirEntry, error) {
	if c.assertDir != "" {
		f, err := os.ReadDir(filepath.Join(c.assertDir, name))
		if err == nil {
			return f, nil
		}
	}
	return nil, errors.New("not found")
}

type EmbedFile struct {
	f fs.File
}

// Close implements http.File
func (ef EmbedFile) Close() error {
	return ef.f.Close()
}

// Read implements http.File
func (ef EmbedFile) Read(p []byte) (n int, err error) {
	return ef.f.Read(p)
}

// Seek implements http.File
func (ef EmbedFile) Seek(offset int64, whence int) (int64, error) {
	return offset, nil
}

// Readdir implements http.File
func (ef EmbedFile) Readdir(count int) ([]fs.FileInfo, error) {
	return nil, nil
}

// Stat implements http.File
func (ef EmbedFile) Stat() (fs.FileInfo, error) {
	return ef.f.Stat()
}

func GetRenderPageContext(c *gin.Context) map[string]any {
	db := c.MustGet(constants.DbField).(*gorm.DB)
	utils.LoadAutoloads(db)
	loginNext := c.Query("next")
	if loginNext == "" {
		loginNext = utils.GetValue(db, constants.KEY_SITE_LOGIN_NEXT)
	}

	return map[string]any{
		"LoginNext":    loginNext,
		"RegisterNext": utils.GetValue(db, constants.KEY_SITE_SIGNIN_URL),
		"Site": map[string]any{
			"Url":                  utils.GetValue(db, constants.KEY_SITE_URL),
			"Name":                 utils.GetValue(db, constants.KEY_SITE_NAME),
			"Admin":                utils.GetValue(db, constants.KEY_SITE_ADMIN),
			"Keywords":             utils.GetValue(db, constants.KEY_SITE_KEYWORDS),
			"Description":          utils.GetValue(db, constants.KEY_SITE_DESCRIPTION),
			"GA":                   utils.GetValue(db, constants.KEY_SITE_GA),
			"LogoUrl":              utils.GetValue(db, constants.KEY_SITE_LOGO_URL),
			"FaviconUrl":           utils.GetValue(db, constants.KEY_SITE_FAVICON_URL),
			"TermsUrl":             utils.GetValue(db, constants.KEY_SITE_TERMS_URL),
			"PrivacyUrl":           utils.GetValue(db, constants.KEY_SITE_PRIVACY_URL),
			"SigninUrl":            utils.GetValue(db, constants.KEY_SITE_SIGNIN_URL),
			"SignupUrl":            utils.GetValue(db, constants.KEY_SITE_SIGNUP_URL),
			"LogoutUrl":            utils.GetValue(db, constants.KEY_SITE_LOGOUT_URL),
			"ResetPasswordUrl":     utils.GetValue(db, constants.KEY_SITE_RESET_PASSWORD_URL),
			"SigninApi":            utils.GetValue(db, constants.KEY_SITE_SIGNIN_API),
			"SignupApi":            utils.GetValue(db, constants.KEY_SITE_SIGNUP_API),
			"ResetPasswordDoneApi": utils.GetValue(db, constants.KEY_SITE_RESET_PASSWORD_DONE_API),
			"UserIdType":           utils.GetValue(db, constants.KEY_SITE_USER_ID_TYPE),
		},
	}
}

func HintAssetsRoot(dirName string) string {
	for _, dir := range []string{".", ".."} {
		testDirName := filepath.Join(os.ExpandEnv(dir), dirName)
		st, err := os.Stat(testDirName)

		if err == nil && st.IsDir() {
			return testDirName
		}
	}
	return ""
}

type CombineTemplates struct {
	CombineFS *CombineEmbedFS
	Template  *template.Template
	Delims    render.Delims
	FuncMap   template.FuncMap
}

func NewCombineTemplates(combineFS *CombineEmbedFS) *CombineTemplates {
	return &CombineTemplates{
		CombineFS: combineFS,
		Delims:    render.Delims{Left: "{{", Right: "}}"},
		FuncMap:   NewTemplateFuncs(),
	}
}

func NewTemplateFuncs() template.FuncMap {
	return template.FuncMap{
		"default": func(value string) string {
			return value
		},
	}
}

func SanitizeSensitiveValues(prefix string, data any) map[string]any {
	if data == nil {
		return nil
	}

	vals, ok := data.(map[string]any)
	if !ok {
		return nil
	}
	outVals := make(map[string]any)
	lowKeyRe := regexp.MustCompile(`(?i)(password|salt|secret|key)$`)
	for k, v := range vals {
		if len(prefix) > 0 {
			k = prefix + "." + k
		}
		if subVals, ok := v.(map[string]any); ok {
			subVals = SanitizeSensitiveValues(k, subVals)
			for sk, sv := range subVals {
				outVals[sk] = sv
			}
			continue
		}
		if lowKeyRe.MatchString(k) {
			outVals[k] = "********"
		} else {
			outVals[k] = v
		}
	}
	return outVals
}

func formatSources(source string) []any {
	var sources []any
	lines := regexp.MustCompile(`\r?\n`).Split(source, -1)
	for i, line := range lines {
		sources = append(sources, map[string]any{
			"Num":  i + 1,
			"Text": line,
		})
	}
	return sources
}

func (c *CombineTemplates) RenderError(name, source string, data any, err error) render.Render {
	return render.Data{
		ContentType: "text/html; charset=utf-8",
		Data:        c.renderTemplateError(name, source, data, err),
	}
}

func (c *CombineTemplates) renderTemplateError(name, source string, data any, err error) []byte {
	lineAt := int64(0)
	if err != nil {
		re := regexp.MustCompile(`:(\d+):`)
		matches := re.FindStringSubmatch(err.Error())
		if len(matches) > 1 {
			lineAt, _ = strconv.ParseInt(matches[1], 10, 64)
		}
	}

	ctx := map[string]any{
		"Name":    name,
		"Error":   err,
		"Context": SanitizeSensitiveValues("", data),
		"Message": err.Error(),
		"Sources": formatSources(source),
		"LineAt":  lineAt,
	}
	tmpl := `<!DOCTYPE html> 
<html>
<head>
	<title>Error</title>
</head>
<body>
	<h1>Error</h1>
	<p>{{ .Message }}</p>
</body>
</html>`

	if gin.Mode() == gin.DebugMode {
		debugTmplFile, err := c.CombineFS.Open(".error.html")
		if err == nil {
			if tmplData, err := io.ReadAll(debugTmplFile); err == nil {
				tmpl = string(tmplData)
			}
		}
	}

	t := template.Must(template.New(name).Funcs(c.FuncMap).Delims(c.Delims.Left, c.Delims.Right).Parse(tmpl))
	var wr = bytes.NewBuffer(make([]byte, 0, 1024))
	err = t.Execute(wr, ctx)
	if err != nil {
		return []byte(`<html><head><title>Error</title></head><body><h1>Error</h1><p>` + err.Error() + `</p></body></html>`)
	}
	return wr.Bytes()
}

// gin.render.Render
func (c *CombineTemplates) Instance(name string, ctx any) render.Render {
	tmplFile, err := c.CombineFS.Open(name)
	if err != nil {
		return c.RenderError(name, "", ctx, err)
	}

	tmplData, err := io.ReadAll(tmplFile)
	if err != nil {
		return c.RenderError(name, "", ctx, err)
	}
	tmpl := string(tmplData)
	t, err := template.New(name).Funcs(c.FuncMap).Delims(c.Delims.Left, c.Delims.Right).Parse(tmpl)
	if err != nil {
		return c.RenderError(name, tmpl, ctx, err)
	}

	if c.FuncMap == nil {
		c.FuncMap = NewTemplateFuncs()
	}

	r := &DebugTempalte{
		c:        c,
		Template: t,
		Name:     name,
		Data:     ctx,
	}
	return r
}

// HTML contains template reference and its name with given interface object.
type DebugTempalte struct {
	c        *CombineTemplates
	Template *template.Template
	Name     string
	Data     any
}

// Render (HTML) executes template and writes its result with custom ContentType for response.
func (r DebugTempalte) Render(w http.ResponseWriter) error {
	var wr = bytes.NewBuffer(make([]byte, 0, 1024))
	var err error
	if r.Name == "" {
		err = r.Template.Execute(wr, r.Data)
	} else {
		err = r.Template.ExecuteTemplate(wr, r.Name, r.Data)
	}

	r.WriteContentType(w)

	if err != nil {
		r.c.renderTemplateError(r.Name, "", r.Data, err)
		return err
	}
	_, err = w.Write(wr.Bytes())
	return err
}

// WriteContentType (HTML) writes HTML ContentType.
func (r DebugTempalte) WriteContentType(w http.ResponseWriter) {
	header := w.Header()
	if val := header["Content-Type"]; len(val) == 0 {
		header["Content-Type"] = []string{"text/html; charset=utf-8"}
	}
}

func WithStaticAssets(r *gin.Engine, staticPrefix, staticRootDir string) gin.HandlerFunc {
	if staticRootDir == "" {
		staticRootDir = "static"
	}
	staticAssets := NewCombineEmbedFS(HintAssetsRoot(staticRootDir), EmbedFS{"static", EmbedStaticAssets})
	if staticPrefix == "" {
		staticPrefix = "/static"
	}
	r.StaticFS(staticPrefix, http.FS(staticAssets))
	return func(ctx *gin.Context) {
		ctx.Set(constants.AssetsField, staticAssets)
		ctx.Next()
	}
}

func WithTemplates(r *gin.Engine, templateRootDir string) gin.HandlerFunc {
	if templateRootDir == "" {
		templateRootDir = "templates"
	}
	templatesAssets := NewCombineEmbedFS(HintAssetsRoot(templateRootDir), EmbedFS{"templates", EmbedTemplates})
	r.HTMLRender = NewCombineTemplates(templatesAssets)

	return func(ctx *gin.Context) {
		ctx.Set(constants.TemplatesField, templatesAssets)
		ctx.Next()
	}
}
