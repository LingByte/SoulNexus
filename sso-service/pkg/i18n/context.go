package i18n

// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

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
