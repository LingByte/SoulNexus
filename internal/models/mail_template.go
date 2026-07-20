// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

package models

import (
	"errors"
	"regexp"
	"sort"
	"strings"

	"github.com/LingByte/SoulNexus/pkg/utils"
	"github.com/LingByte/SoulNexus/pkg/utils/common"
	"gorm.io/gorm"
)

type MailTemplate struct {
	common.BaseModel
	Code        string `json:"code" gorm:"uniqueIndex:idx_mail_tpl_code_locale;size:64;not null;comment:模板编码"`
	Name        string `json:"name" gorm:"size:128;not null;comment:模板名称"`
	Subject     string `json:"subject" gorm:"size:255;comment:邮件标题模板"`
	HTMLBody    string `json:"htmlBody" gorm:"type:longtext;comment:HTML 正文"`
	TextBody    string `json:"textBody,omitempty" gorm:"type:longtext;comment:纯文本正文"`
	Description string `json:"description,omitempty" gorm:"size:512;comment:说明"`
	Variables   string `json:"variables,omitempty" gorm:"type:text;comment:占位符说明 JSON"`
	Locale      string `json:"locale,omitempty" gorm:"uniqueIndex:idx_mail_tpl_code_locale;size:32;default:'';comment:语言如 zh-CN"`
	Enabled     bool   `json:"enabled" gorm:"default:true;index;comment:是否启用"`
}

func (MailTemplate) TableName() string { return "mail_templates" }

var (
	htmlTagStripper = regexp.MustCompile(`(?is)<[^>]+>`)
	whitespaceRE    = regexp.MustCompile(`\s+`)
	placeholderRE   = regexp.MustCompile(`\{\{\s*([A-Za-z_][A-Za-z0-9_]*)\s*\}\}`)
)

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

func MailTemplateDerivedTextBody(htmlBody string) string { return HTMLToPlainText(htmlBody) }

func resolveMailTemplateVariables(htmlBody, textBody, variables string) string {
	v := strings.TrimSpace(variables)
	if v != "" {
		return v
	}
	return DeriveTemplateVariables(htmlBody, textBody)
}

func ApplyMailTemplateHTMLDerivedFields(tpl *MailTemplate, htmlBody, variables string) {
	if tpl == nil {
		return
	}
	plain := MailTemplateDerivedTextBody(htmlBody)
	tpl.HTMLBody = htmlBody
	tpl.TextBody = plain
	tpl.Variables = resolveMailTemplateVariables(htmlBody, plain, variables)
}

func CountMailTemplates(db *gorm.DB) (int64, error) {
	var n int64
	err := db.Model(&MailTemplate{}).Count(&n).Error
	return n, err
}

func ListMailTemplates(db *gorm.DB, offset, limit int) ([]MailTemplate, error) {
	var list []MailTemplate
	err := db.Order("id DESC").Offset(offset).Limit(limit).Find(&list).Error
	return list, err
}

// ListMailTemplatesPage returns paginated mail templates.
func ListMailTemplatesPage(db *gorm.DB, page, size int) ([]MailTemplate, int64, error) {
	return utils.FindPage[MailTemplate](db.Model(&MailTemplate{}), page, size, "id DESC", utils.MaxPageSize200)
}

func GetMailTemplateByID(db *gorm.DB, id uint) (*MailTemplate, error) {
	var tpl MailTemplate
	if err := db.First(&tpl, id).Error; err != nil {
		return nil, err
	}
	return &tpl, nil
}

func GetMailTemplateByCode(db *gorm.DB, code, locale string) (*MailTemplate, error) {
	var tpl MailTemplate
	q := db.Where("code = ? AND enabled = ?", code, true)
	if strings.TrimSpace(locale) != "" {
		q = q.Where("locale = ?", locale)
	}
	if err := q.First(&tpl).Error; err != nil {
		return nil, err
	}
	return &tpl, nil
}

func CreateMailTemplate(db *gorm.DB, tpl *MailTemplate) error {
	if tpl == nil {
		return errors.New("nil template")
	}
	return db.Create(tpl).Error
}

func SaveMailTemplate(db *gorm.DB, tpl *MailTemplate) error {
	if tpl == nil {
		return errors.New("nil template")
	}
	return db.Save(tpl).Error
}

func DeleteMailTemplateByID(db *gorm.DB, id uint) (int64, error) {
	res := db.Delete(&MailTemplate{}, id)
	return res.RowsAffected, res.Error
}
