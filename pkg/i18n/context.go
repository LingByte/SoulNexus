package i18n

import (
	"context"
)

type contextKey string

const localeKey contextKey = "locale"

// WithLocale adds locale to context
func WithLocale(ctx context.Context, locale Locale) context.Context {
	return context.WithValue(ctx, localeKey, locale)
}

// GetLocaleFromContext gets locale from context
func GetLocaleFromContext(ctx context.Context) Locale {
	if locale, ok := ctx.Value(localeKey).(Locale); ok {
		return locale
	}
	return DefaultLocale
}

// SetLocale sets locale in context
func SetLocale(ctx context.Context, locale Locale) context.Context {
	return WithLocale(ctx, locale)
}
