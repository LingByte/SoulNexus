package i18n

import (
	"embed"
	"strings"
	"sync"

	"github.com/gin-gonic/gin"
	"github.com/nicksnyder/go-i18n/v2/i18n"

	"golang.org/x/text/language"
	"gopkg.in/yaml.v3"
)

const (
	LangZhCN    = "zh-CN"
	LangZhTW    = "zh-TW"
	LangEn      = "en"
	DefaultLang = LangEn // Fallback to English if language not supported
)

//go:embed locales/*.yaml
var localeFS embed.FS

var (
	bundle     *i18n.Bundle
	localizers = make(map[string]*i18n.Localizer)
	mu         sync.RWMutex
	initOnce   sync.Once
)

type UserSetting struct {
	NotifyType                       string  `json:"notify_type,omitempty"`                          // QuotaWarningType 额度预警类型
	QuotaWarningThreshold            float64 `json:"quota_warning_threshold,omitempty"`              // QuotaWarningThreshold 额度预警阈值
	WebhookUrl                       string  `json:"webhook_url,omitempty"`                          // WebhookUrl webhook地址
	WebhookSecret                    string  `json:"webhook_secret,omitempty"`                       // WebhookSecret webhook密钥
	NotificationEmail                string  `json:"notification_email,omitempty"`                   // NotificationEmail 通知邮箱地址
	BarkUrl                          string  `json:"bark_url,omitempty"`                             // BarkUrl Bark推送URL
	GotifyUrl                        string  `json:"gotify_url,omitempty"`                           // GotifyUrl Gotify服务器地址
	GotifyToken                      string  `json:"gotify_token,omitempty"`                         // GotifyToken Gotify应用令牌
	GotifyPriority                   int     `json:"gotify_priority"`                                // GotifyPriority Gotify消息优先级
	UpstreamModelUpdateNotifyEnabled bool    `json:"upstream_model_update_notify_enabled,omitempty"` // 是否接收上游模型更新定时检测通知（仅管理员）
	AcceptUnsetRatioModel            bool    `json:"accept_unset_model_ratio_model,omitempty"`       // AcceptUnsetRatioModel 是否接受未设置价格的模型
	RecordIpLog                      bool    `json:"record_ip_log,omitempty"`                        // 是否记录请求和错误日志IP
	SidebarModules                   string  `json:"sidebar_modules,omitempty"`                      // SidebarModules 左侧边栏模块配置
	BillingPreference                string  `json:"billing_preference,omitempty"`                   // BillingPreference 扣费策略（订阅/钱包）
	Language                         string  `json:"language,omitempty"`                             // Language 用户语言偏好 (zh, en)
}

var TranslateMessage func(c *gin.Context, key string, args ...map[string]any) string

func GetContextKeyType[T any](c *gin.Context, key string) (T, bool) {
	if value, ok := c.Get(string(key)); ok {
		if v, ok := value.(T); ok {
			return v, true
		}
	}
	var t T
	return t, false
}

// Init initializes the i18n bundle and loads all translation files
func Init() error {
	var initErr error
	initOnce.Do(func() {
		bundle = i18n.NewBundle(language.Chinese)
		bundle.RegisterUnmarshalFunc("yaml", yaml.Unmarshal)

		// Load embedded translation files
		files := []string{"locales/zh-CN.yaml", "locales/zh-TW.yaml", "locales/en.yaml"}
		for _, file := range files {
			_, err := bundle.LoadMessageFileFS(localeFS, file)
			if err != nil {
				initErr = err
				return
			}
		}

		// Pre-create localizers for supported languages
		localizers[LangZhCN] = i18n.NewLocalizer(bundle, LangZhCN)
		localizers[LangZhTW] = i18n.NewLocalizer(bundle, LangZhTW)
		localizers[LangEn] = i18n.NewLocalizer(bundle, LangEn)

		// Set the TranslateMessage function in common package
		TranslateMessage = T
	})
	return initErr
}

// GetLocalizer returns a localizer for the specified language
func GetLocalizer(lang string) *i18n.Localizer {
	lang = normalizeLang(lang)

	mu.RLock()
	loc, ok := localizers[lang]
	mu.RUnlock()

	if ok {
		return loc
	}

	// Create new localizer for unknown language (fallback to default)
	mu.Lock()
	defer mu.Unlock()

	// Double-check after acquiring write lock
	if loc, ok = localizers[lang]; ok {
		return loc
	}

	loc = i18n.NewLocalizer(bundle, lang, DefaultLang)
	localizers[lang] = loc
	return loc
}

// T translates a message key using the language from gin context
func T(c *gin.Context, key string, args ...map[string]any) string {
	lang := GetLangFromContext(c)
	return Translate(lang, key, args...)
}

// Translate translates a message key for the specified language
func Translate(lang, key string, args ...map[string]any) string {
	loc := GetLocalizer(lang)

	config := &i18n.LocalizeConfig{
		MessageID: key,
	}

	if len(args) > 0 && args[0] != nil {
		config.TemplateData = args[0]
	}

	msg, err := loc.Localize(config)
	if err != nil {
		// Return key as fallback if translation not found
		return key
	}
	return msg
}

// userLangLoaderFunc is a function that loads user language from database/cache
// It's set by the model package to avoid circular imports
var userLangLoaderFunc func(userId int) string

// SetUserLangLoader sets the function to load user language (called from model package)
func SetUserLangLoader(loader func(userId int) string) {
	userLangLoaderFunc = loader
}

// GetLangFromContext extracts the language setting from gin context
// It checks multiple sources in priority order:
// 1. User settings (ContextKeyUserSetting) - if already loaded (e.g., by TokenAuth)
// 2. Lazy load user language from cache/DB using user ID
// 3. Language set by middleware (ContextKeyLanguage) - from Accept-Language header
// 4. Default language (English)
func GetLangFromContext(c *gin.Context) string {
	if c == nil {
		return DefaultLang
	}

	// 1. Try to get language from user settings (if already loaded by TokenAuth or other middleware)
	if userSetting, ok := GetContextKeyType[UserSetting](c, "user_setting"); ok {
		if userSetting.Language != "" {
			normalized := normalizeLang(userSetting.Language)
			if IsSupported(normalized) {
				return normalized
			}
		}
	}

	// 2. Lazy load user language using user ID (for session-based auth where full settings aren't loaded)
	if userLangLoaderFunc != nil {
		if userId, exists := c.Get("id"); exists {
			if uid, ok := userId.(int); ok && uid > 0 {
				lang := userLangLoaderFunc(uid)
				if lang != "" {
					normalized := normalizeLang(lang)
					if IsSupported(normalized) {
						return normalized
					}
				}
			}
		}
	}

	// 3. Try to get language from context (set by I18n middleware from Accept-Language)
	if lang := c.GetString(string("language")); lang != "" {
		normalized := normalizeLang(lang)
		if IsSupported(normalized) {
			return normalized
		}
	}

	// 4. Try Accept-Language header directly (fallback if middleware didn't run)
	if acceptLang := c.GetHeader("Accept-Language"); acceptLang != "" {
		lang := ParseAcceptLanguage(acceptLang)
		if IsSupported(lang) {
			return lang
		}
	}

	return DefaultLang
}

// ParseAcceptLanguage parses the Accept-Language header and returns the preferred language
func ParseAcceptLanguage(header string) string {
	if header == "" {
		return DefaultLang
	}

	// Simple parsing: take the first language tag
	parts := strings.Split(header, ",")
	if len(parts) == 0 {
		return DefaultLang
	}

	// Get the first language and remove quality value
	firstLang := strings.TrimSpace(parts[0])
	if idx := strings.Index(firstLang, ";"); idx > 0 {
		firstLang = firstLang[:idx]
	}

	return normalizeLang(firstLang)
}

// normalizeLang normalizes language code to supported format
func normalizeLang(lang string) string {
	lang = strings.ToLower(strings.TrimSpace(lang))

	// Handle common variations
	switch {
	case strings.HasPrefix(lang, "zh-tw"):
		return LangZhTW
	case strings.HasPrefix(lang, "zh"):
		return LangZhCN
	case strings.HasPrefix(lang, "en"):
		return LangEn
	default:
		return DefaultLang
	}
}

// SupportedLanguages returns a list of supported language codes
func SupportedLanguages() []string {
	return []string{LangZhCN, LangZhTW, LangEn}
}

// IsSupported checks if a language code is supported
func IsSupported(lang string) bool {
	lang = normalizeLang(lang)
	for _, supported := range SupportedLanguages() {
		if lang == supported {
			return true
		}
	}
	return false
}
