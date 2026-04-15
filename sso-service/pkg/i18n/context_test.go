package i18n

// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

import (
	"context"
	"testing"
)

func TestWithLocale(t *testing.T) {
	ctx := context.Background()
	ctx = WithLocale(ctx, "zh-CN")

	locale := GetLocaleFromContext(ctx)
	if locale != "zh-CN" {
		t.Errorf("expected zh-CN, got %s", locale)
	}
}

func TestGetLocaleFromContext_Default(t *testing.T) {
	ctx := context.Background()
	locale := GetLocaleFromContext(ctx)
	if locale != DefaultLocale {
		t.Errorf("expected %s, got %s", DefaultLocale, locale)
	}
}

func TestSetLocale(t *testing.T) {
	ctx := context.Background()
	ctx = SetLocale(ctx, "en")

	locale := GetLocaleFromContext(ctx)
	if locale != "en" {
		t.Errorf("expected en, got %s", locale)
	}
}
