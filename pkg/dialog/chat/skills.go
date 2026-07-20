// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

package chat

import (
	"context"
	"strings"

	"github.com/LingByte/SoulNexus/internal/models"
	"github.com/LingByte/SoulNexus/pkg/dialog/tenantcfg"
	"gorm.io/gorm"
)

// skillNamesFromEnv reads agentConfig.dialogSkills ([]string of skill codes).
func skillNamesFromEnv(env tenantcfg.VoiceEnv) []string {
	raw := env.AgentConfigRaw
	if raw == nil {
		return nil
	}
	v, ok := raw["dialogSkills"]
	if !ok {
		return nil
	}
	switch t := v.(type) {
	case []any:
		out := make([]string, 0, len(t))
		for _, item := range t {
			if s, ok := item.(string); ok && strings.TrimSpace(s) != "" {
				out = append(out, strings.TrimSpace(s))
			}
		}
		return out
	case []string:
		out := make([]string, 0, len(t))
		for _, s := range t {
			if strings.TrimSpace(s) != "" {
				out = append(out, strings.TrimSpace(s))
			}
		}
		return out
	default:
		return nil
	}
}

// LoadSkillsAppendix loads tenant-scoped skill bodies for the given codes into a system appendix.
// Missing / disabled skills are skipped. Order follows the request codes when possible.
func LoadSkillsAppendix(db *gorm.DB, tenantID uint, codes []string) string {
	if db == nil || tenantID == 0 || len(codes) == 0 {
		return ""
	}
	rows, err := models.GetTenantDialogSkillsByCodes(db, tenantID, codes)
	if err != nil || len(rows) == 0 {
		return ""
	}
	byCode := make(map[string]models.TenantDialogSkill, len(rows))
	for _, r := range rows {
		byCode[r.Code] = r
	}
	var b strings.Builder
	seen := map[string]struct{}{}
	appendSkill := func(code, name, body string) {
		code = models.NormalizeDialogSkillCode(code)
		if code == "" {
			return
		}
		if _, ok := seen[code]; ok {
			return
		}
		body = strings.TrimSpace(stripYAMLFrontmatter(body))
		if body == "" {
			return
		}
		seen[code] = struct{}{}
		if b.Len() > 0 {
			b.WriteString("\n\n")
		}
		label := strings.TrimSpace(name)
		if label == "" {
			label = code
		}
		b.WriteString("【技能: ")
		b.WriteString(label)
		b.WriteString("】\n")
		b.WriteString(body)
	}
	for _, code := range codes {
		code = models.NormalizeDialogSkillCode(code)
		if row, ok := byCode[code]; ok {
			appendSkill(row.Code, row.Name, row.Body)
		}
	}
	return b.String()
}

// LoadSkillsAppendixCtx is LoadSkillsAppendix with context (for future timeouts).
func LoadSkillsAppendixCtx(ctx context.Context, db *gorm.DB, tenantID uint, codes []string) string {
	if db != nil && ctx != nil {
		db = db.WithContext(ctx)
	}
	return LoadSkillsAppendix(db, tenantID, codes)
}

func stripYAMLFrontmatter(s string) string {
	s = strings.TrimSpace(s)
	if !strings.HasPrefix(s, "---") {
		return s
	}
	rest := s[3:]
	idx := strings.Index(rest, "\n---")
	if idx < 0 {
		return s
	}
	return strings.TrimSpace(rest[idx+4:])
}
