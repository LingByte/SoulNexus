// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

package handlers

import (
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

const userActivityRetentionDays = 180

// inferAuditServiceName maps request path to a short product/service label (audit UI).
func inferAuditServiceName(target string) string {
	t := strings.ToLower(target)
	switch {
	case strings.Contains(t, "/auth"):
		return "Account"
	case strings.Contains(t, "/credential"):
		return "Credential"
	case strings.Contains(t, "/assistant"):
		return "Assistant"
	case strings.Contains(t, "/group"):
		return "Group"
	case strings.Contains(t, "/chat"):
		return "Chat"
	case strings.Contains(t, "/voice"):
		return "Voice"
	case strings.Contains(t, "/notification"):
		return "Notification"
	case strings.Contains(t, "/upload"):
		return "Upload"
	case strings.Contains(t, "/billing") || strings.Contains(t, "/quota"):
		return "Billing"
	case strings.Contains(t, "/knowledge"):
		return "Knowledge"
	case strings.Contains(t, "/workflow"):
		return "Workflow"
	default:
		return "Other"
	}
}

func inferAuditEventCode(method, target string) string {
	path := strings.TrimSpace(target)
	if idx := strings.Index(path, "?"); idx >= 0 {
		path = path[:idx]
	}
	segs := strings.FieldsFunc(strings.Trim(path, "/"), func(c rune) bool { return c == '/' })
	skip := map[string]bool{"api": true, "v1": true, "v2": true, "v3": true}
	var parts []string
	for _, s := range segs {
		if s == "" || skip[strings.ToLower(s)] {
			continue
		}
		part := strings.ReplaceAll(s, "-", "_")
		if len(part) == 0 {
			continue
		}
		if len(part) == 1 {
			parts = append(parts, strings.ToUpper(part))
			continue
		}
		parts = append(parts, strings.ToUpper(part[:1])+part[1:])
	}
	if len(parts) == 0 {
		if method != "" {
			return strings.ToUpper(method)
		}
		return "Request"
	}
	return strings.Join(parts, "")
}

func auditResourceHint(target, details string) string {
	path := strings.TrimSpace(target)
	if idx := strings.Index(path, "?"); idx >= 0 {
		path = path[:idx]
	}
	if len(path) > 120 {
		return path[:117] + "..."
	}
	if path != "" {
		return path
	}
	d := strings.TrimSpace(details)
	if len(d) > 120 {
		return d[:117] + "..."
	}
	return d
}

func applyUserActivityFilters(c *gin.Context, query *gorm.DB, userID uint) *gorm.DB {
	query = query.Where("user_id = ?", userID)

	cutoff := time.Now().AddDate(0, 0, -userActivityRetentionDays)
	query = query.Where("created_at >= ?", cutoff)

	if start := strings.TrimSpace(c.Query("start")); start != "" {
		if t, err := time.ParseInLocation("2006-01-02", start, time.Local); err == nil {
			if t.After(cutoff) {
				query = query.Where("created_at >= ?", t)
			}
		} else if t, err := time.Parse(time.RFC3339, start); err == nil {
			if t.After(cutoff) {
				query = query.Where("created_at >= ?", t)
			}
		}
	}
	if end := strings.TrimSpace(c.Query("end")); end != "" {
		if t, err := time.ParseInLocation("2006-01-02", end, time.Local); err == nil {
			query = query.Where("created_at < ?", t.AddDate(0, 0, 1))
		} else if t, err := time.Parse(time.RFC3339, end); err == nil {
			query = query.Where("created_at <= ?", t)
		}
	}

	if svc := strings.TrimSpace(c.Query("service")); svc != "" && svc != "all" {
		switch svc {
		case "account":
			query = query.Where("LOWER(target) LIKE ?", "%/auth%")
		case "credential":
			query = query.Where("LOWER(target) LIKE ?", "%credential%")
		case "assistant":
			query = query.Where("LOWER(target) LIKE ?", "%/assistant%")
		case "group":
			query = query.Where("LOWER(target) LIKE ?", "%/group%")
		case "chat":
			query = query.Where("LOWER(target) LIKE ?", "%/chat%")
		case "voice":
			query = query.Where("LOWER(target) LIKE ?", "%/voice%")
		case "notification":
			query = query.Where("LOWER(target) LIKE ?", "%/notification%")
		case "upload":
			query = query.Where("LOWER(target) LIKE ?", "%/upload%")
		case "billing":
			query = query.Where("(LOWER(target) LIKE ? OR LOWER(target) LIKE ?)", "%/billing%", "%/quota%")
		case "knowledge":
			query = query.Where("LOWER(target) LIKE ?", "%/knowledge%")
		case "workflow":
			query = query.Where("LOWER(target) LIKE ?", "%/workflow%")
		case "other":
			query = query.Where(
				"NOT ("+
					"LOWER(target) LIKE ? OR LOWER(target) LIKE ? OR LOWER(target) LIKE ? OR "+
					"LOWER(target) LIKE ? OR LOWER(target) LIKE ? OR LOWER(target) LIKE ? OR "+
					"LOWER(target) LIKE ? OR LOWER(target) LIKE ? OR LOWER(target) LIKE ? OR "+
					"LOWER(target) LIKE ? OR LOWER(target) LIKE ? OR LOWER(target) LIKE ?"+
					")",
				"%/auth%", "%credential%", "%/assistant%", "%/group%", "%/chat%", "%/voice%",
				"%/notification%", "%/upload%", "%/billing%", "%/quota%", "%/knowledge%", "%/workflow%",
			)
		}
	}

	if ev := strings.TrimSpace(c.Query("event")); ev != "" {
		pat := "%" + ev + "%"
		query = query.Where("(target LIKE ? OR details LIKE ? OR request_method LIKE ?)", pat, pat, pat)
	}

	if ip := strings.TrimSpace(c.Query("ip")); ip != "" {
		query = query.Where("ip_address LIKE ?", "%"+ip+"%")
	}

	if kid := strings.TrimSpace(c.Query("credentialId")); kid != "" {
		if strings.Contains(kid, "%") || strings.Contains(kid, "_") {
			// avoid wildcard injection in LIKE
			kid = strings.ReplaceAll(kid, "%", "")
			kid = strings.ReplaceAll(kid, "_", "")
		}
		if kid != "" {
			pat := "%" + kid + "%"
			query = query.Where("(target LIKE ? OR details LIKE ?)", pat, pat)
		}
	}

	if res := strings.TrimSpace(c.Query("resource")); res != "" {
		pat := "%" + res + "%"
		query = query.Where("(target LIKE ? OR details LIKE ?)", pat, pat)
	}

	if action := strings.TrimSpace(c.Query("action")); action != "" {
		query = query.Where("action = ?", strings.ToUpper(action))
	}

	return query
}
