// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

package models

import (
	"errors"
	"regexp"
	"sort"
	"strings"

	"gorm.io/gorm"
)

// MailTemplate 可持久化的邮件模板，供发信侧渲染主题/正文（占位符 {{Name}} 由业务层注入）。
type MailTemplate struct {
	BaseModel
	OrgID       uint   `json:"orgId" gorm:"uniqueIndex:idx_mail_tpl_org_code_locale;not null;default:0;comment:tenant id (0=system)"`
	Code        string `json:"code" gorm:"uniqueIndex:idx_mail_tpl_org_code_locale;size:64;not null;comment:模板编码"`
	Name        string `json:"name" gorm:"size:128;not null;comment:模板名称"`
	Subject     string `json:"subject" gorm:"size:255;comment:邮件标题模板"`
	HTMLBody    string `json:"htmlBody" gorm:"type:longtext;comment:HTML 正文"`
	TextBody    string `json:"textBody,omitempty" gorm:"type:longtext;comment:纯文本正文"`
	Description string `json:"description,omitempty" gorm:"size:512;comment:说明"`
	Variables   string `json:"variables,omitempty" gorm:"type:text;comment:占位符说明 JSON"`
	Locale      string `json:"locale,omitempty" gorm:"uniqueIndex:idx_mail_tpl_org_code_locale;size:32;default:'';comment:语言如 zh-CN"`
	Enabled     bool   `json:"enabled" gorm:"default:true;index;comment:是否启用"`
}

// TableName GORM 表名
func (MailTemplate) TableName() string { return "mail_templates" }

var (
	htmlTagStripper = regexp.MustCompile(`(?is)<[^>]+>`)
	whitespaceRE    = regexp.MustCompile(`\s+`)
	placeholderRE   = regexp.MustCompile(`\{\{\s*([A-Za-z_][A-Za-z0-9_]*)\s*\}\}`)
)

// HTMLToPlainText 将 HTML 转纯文本（去标签，压缩空白）。
func HTMLToPlainText(htmlBody string) string {
	if htmlBody == "" {
		return ""
	}
	s := htmlTagStripper.ReplaceAllString(htmlBody, " ")
	s = strings.ReplaceAll(s, "&nbsp;", " ")
	s = strings.ReplaceAll(s, "&amp;", "&")
	s = strings.ReplaceAll(s, "&lt;", "<")
	s = strings.ReplaceAll(s, "&gt;", ">")
	s = strings.ReplaceAll(s, "&quot;", "\"")
	s = whitespaceRE.ReplaceAllString(s, " ")
	return strings.TrimSpace(s)
}

// DeriveTemplateVariables 从 HTML/纯文本中扫描占位符，输出 JSON 数组。
func DeriveTemplateVariables(htmlBody, textBody string) string {
	src := htmlBody + "\n" + textBody
	matches := placeholderRE.FindAllStringSubmatch(src, -1)
	if len(matches) == 0 {
		return ""
	}
	seen := map[string]struct{}{}
	for _, m := range matches {
		if len(m) >= 2 {
			seen[m[1]] = struct{}{}
		}
	}
	if len(seen) == 0 {
		return ""
	}
	keys := make([]string, 0, len(seen))
	for k := range seen {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	var b strings.Builder
	b.WriteString("[")
	for i, k := range keys {
		if i > 0 {
			b.WriteString(",")
		}
		b.WriteString(`{"name":"`)
		b.WriteString(k)
		b.WriteString(`"}`)
	}
	b.WriteString("]")
	return b.String()
}

// MailTemplateDerivedTextBody 由 HTML 派生纯文本。
func MailTemplateDerivedTextBody(htmlBody string) string { return HTMLToPlainText(htmlBody) }

// MailTemplateNormalizeVariables 若 variables 为空则从 HTML/text 自动推导。
func MailTemplateNormalizeVariables(htmlBody, textBody, variables string) string {
	v := strings.TrimSpace(variables)
	if v != "" {
		return v
	}
	return DeriveTemplateVariables(htmlBody, textBody)
}

// ApplyMailTemplateHTMLDerivedFields 写入 HTML 时同步派生 TextBody/Variables。
func ApplyMailTemplateHTMLDerivedFields(tpl *MailTemplate, htmlBody, variables string) {
	if tpl == nil {
		return
	}
	plain := MailTemplateDerivedTextBody(htmlBody)
	tpl.HTMLBody = htmlBody
	tpl.TextBody = plain
	tpl.Variables = MailTemplateNormalizeVariables(htmlBody, plain, variables)
}

// CountMailTemplatesByOrg 租户下模板数量。
func CountMailTemplatesByOrg(db *gorm.DB, orgID uint) (int64, error) {
	var n int64
	err := db.Model(&MailTemplate{}).Where("org_id = ?", orgID).Count(&n).Error
	return n, err
}

// ListMailTemplatesByOrg 分页列出。
func ListMailTemplatesByOrg(db *gorm.DB, orgID uint, offset, limit int) ([]MailTemplate, error) {
	var list []MailTemplate
	err := db.Where("org_id = ?", orgID).Order("id DESC").Offset(offset).Limit(limit).Find(&list).Error
	return list, err
}

// GetMailTemplateByOrgAndID 按 org+id 读取。
func GetMailTemplateByOrgAndID(db *gorm.DB, orgID, id uint) (*MailTemplate, error) {
	var tpl MailTemplate
	if err := db.Where("org_id = ?", orgID).First(&tpl, id).Error; err != nil {
		return nil, err
	}
	return &tpl, nil
}

// GetMailTemplateByCode 按 code+locale 读取（用于运行时渲染）。
func GetMailTemplateByCode(db *gorm.DB, orgID uint, code, locale string) (*MailTemplate, error) {
	var tpl MailTemplate
	q := db.Where("org_id = ? AND code = ? AND enabled = ?", orgID, code, true)
	if strings.TrimSpace(locale) != "" {
		q = q.Where("locale = ?", locale)
	}
	if err := q.First(&tpl).Error; err != nil {
		return nil, err
	}
	return &tpl, nil
}

// CreateMailTemplate 插入。
func CreateMailTemplate(db *gorm.DB, tpl *MailTemplate) error {
	if tpl == nil {
		return errors.New("nil template")
	}
	return db.Create(tpl).Error
}

// SaveMailTemplate 全量保存。
func SaveMailTemplate(db *gorm.DB, tpl *MailTemplate) error {
	if tpl == nil {
		return errors.New("nil template")
	}
	return db.Save(tpl).Error
}

// DeleteMailTemplateByOrgAndID 按 org+id 删除。
func DeleteMailTemplateByOrgAndID(db *gorm.DB, orgID, id uint) (int64, error) {
	res := db.Where("org_id = ?", orgID).Delete(&MailTemplate{}, id)
	return res.RowsAffected, res.Error
}
