// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

package models

import (
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"
	"unicode"

	"github.com/LingByte/SoulNexus/pkg/constants"
	"github.com/LingByte/SoulNexus/pkg/utils"
	"github.com/LingByte/SoulNexus/pkg/utils/common"
	"github.com/mozillazg/go-pinyin"
	"gorm.io/gorm"
)

const (
	DialogChannelAPI      = "api"
	DialogChannelDebug    = "debug"
	DialogChannelWeCom    = "wecom"
	DialogChannelWeChatOA = "wechat_oa"

	DialogConvStatusOpen   = "open"
	DialogConvStatusClosed = "closed"

	DialogMsgRoleUser      = "user"
	DialogMsgRoleAssistant = "assistant"
	DialogMsgRoleSystem    = "system"

	DialogProviderWeComApp = "wecom_app"
	DialogProviderWeChatOA = "wechat_oa"
)

// DialogConversation is a channel-agnostic text dialog session.
type DialogConversation struct {
	common.BaseModel

	TenantID         uint   `json:"tenantId" gorm:"index;not null"`
	AssistantID      uint   `json:"assistantId" gorm:"index;not null"`
	Channel          string `json:"channel" gorm:"size:32;not null;index;comment:api|debug|wecom|wechat_oa"`
	ChannelAccountID string `json:"channelAccountId" gorm:"size:128;index;comment:corp/agent/app id"`
	ExternalUserID   string `json:"externalUserId" gorm:"size:128;index;comment:channel user id"`
	Status           string `json:"status" gorm:"size:16;not null;default:open;index"`
	MetadataJSON     string `json:"-" gorm:"type:text"`
}

func (DialogConversation) TableName() string { return constants.DIALOG_CONVERSATION_TABLE_NAME }

// CallKey is the stable callbinding / LLM session id for this conversation.
func (c DialogConversation) CallKey() string {
	if c.ID == 0 {
		return ""
	}
	return fmt.Sprintf("dialog-%d", c.ID)
}

// DialogMessage is one turn message in a dialog conversation.
type DialogMessage struct {
	common.BaseModel

	ConversationID uint    `json:"conversationId" gorm:"index;not null"`
	Role           string  `json:"role" gorm:"size:16;not null;index"`
	Content        string  `json:"content" gorm:"type:text;not null"`
	KnowledgeJSON  string  `json:"-" gorm:"type:text"`
	NLUJSON        string  `json:"-" gorm:"type:text"`
	ToolsJSON      string  `json:"-" gorm:"type:text"`
	Confidence     *float64 `json:"confidence,omitempty" gorm:"comment:0-1 LLM-judge score"`
	ConfidenceJSON string  `json:"-" gorm:"type:text"`
	LatencyMs      int64   `json:"latencyMs" gorm:"default:0"`
}

func (DialogMessage) TableName() string { return constants.DIALOG_MESSAGE_TABLE_NAME }

// TenantDialogChannel is inbound dialog channel config (WeCom app / WeChat OA).
// Separate from TenantIMChannel (outbound notification webhooks).
type TenantDialogChannel struct {
	common.BaseModel

	TenantID    uint   `json:"tenantId" gorm:"index;not null;uniqueIndex:idx_tenant_dialog_ch_code"`
	Provider    string `json:"provider" gorm:"size:32;not null;index;comment:wecom_app|wechat_oa"`
	Code        string `json:"code" gorm:"size:64;not null;uniqueIndex:idx_tenant_dialog_ch_code"`
	Name        string `json:"name" gorm:"size:128;not null"`
	AssistantID uint   `json:"assistantId" gorm:"index;not null"`
	Enabled     bool   `json:"enabled" gorm:"not null;default:true;index"`
	Remark      string `json:"remark,omitempty" gorm:"size:255"`
	ConfigJSON  string `json:"-" gorm:"type:text;comment:inbound credentials JSON"`
}

func (TenantDialogChannel) TableName() string { return constants.TENANT_DIALOG_CHANNEL_TABLE_NAME }

// TenantDialogChannelPublic is the API-safe view (secrets masked).
type TenantDialogChannelPublic struct {
	ID          uint           `json:"id"`
	TenantID    uint           `json:"tenantId"`
	Provider    string         `json:"provider"`
	Code        string         `json:"code"`
	Name        string         `json:"name"`
	AssistantID string         `json:"assistantId"`
	Enabled     bool           `json:"enabled"`
	Remark      string         `json:"remark,omitempty"`
	Config      map[string]any `json:"config"`
}

func (row TenantDialogChannel) ToPublic() TenantDialogChannelPublic {
	cfg := map[string]any{}
	_ = json.Unmarshal([]byte(strings.TrimSpace(row.ConfigJSON)), &cfg)
	for _, k := range []string{"secret", "appSecret", "encodingAESKey", "token"} {
		if v, ok := cfg[k].(string); ok && strings.TrimSpace(v) != "" {
			cfg[k+"Set"] = true
			cfg[k] = ""
		}
	}
	return TenantDialogChannelPublic{
		ID: row.ID, TenantID: row.TenantID, Provider: row.Provider,
		Code: row.Code, Name: row.Name,
		AssistantID: strconv.FormatUint(uint64(row.AssistantID), 10),
		Enabled: row.Enabled, Remark: row.Remark, Config: cfg,
	}
}

func NormalizeDialogProvider(p string) string {
	switch strings.ToLower(strings.TrimSpace(p)) {
	case DialogProviderWeComApp, "wecom":
		return DialogProviderWeComApp
	case DialogProviderWeChatOA, "wechat", "official_account", "oa":
		return DialogProviderWeChatOA
	default:
		return ""
	}
}

func ListTenantDialogChannelsPage(db *gorm.DB, tenantID uint, page, size int) ([]TenantDialogChannel, int64, error) {
	q := db.Model(&TenantDialogChannel{}).Where("tenant_id = ?", tenantID)
	return utils.FindPage[TenantDialogChannel](q, page, size, "id DESC", utils.DefaultMaxPageSize)
}

func GetTenantDialogChannel(db *gorm.DB, id, tenantID uint) (TenantDialogChannel, error) {
	var row TenantDialogChannel
	err := db.Where("id = ? AND tenant_id = ?", id, tenantID).First(&row).Error
	return row, err
}

func GetTenantDialogChannelByCode(db *gorm.DB, tenantID uint, code string) (TenantDialogChannel, error) {
	var row TenantDialogChannel
	err := db.Where("tenant_id = ? AND code = ?", tenantID, strings.TrimSpace(code)).First(&row).Error
	return row, err
}

func BuildDialogChannelConfigJSON(provider string, cfg map[string]any) (string, error) {
	kind := NormalizeDialogProvider(provider)
	if kind == "" {
		return "", fmt.Errorf("unsupported dialog provider: %s", provider)
	}
	if cfg == nil {
		cfg = map[string]any{}
	}
	clean := map[string]any{}
	keys := []string{"corpId", "agentId", "secret", "token", "encodingAESKey", "appId", "appSecret"}
	for _, k := range keys {
		if v, ok := cfg[k].(string); ok && strings.TrimSpace(v) != "" {
			clean[k] = strings.TrimSpace(v)
		}
	}
	switch kind {
	case DialogProviderWeComApp:
		for _, k := range []string{"corpId", "agentId", "secret", "token", "encodingAESKey"} {
			if strings.TrimSpace(fmt.Sprint(clean[k])) == "" {
				return "", fmt.Errorf("%s required", k)
			}
		}
	case DialogProviderWeChatOA:
		for _, k := range []string{"appId", "appSecret", "token", "encodingAESKey"} {
			if strings.TrimSpace(fmt.Sprint(clean[k])) == "" {
				return "", fmt.Errorf("%s required", k)
			}
		}
	}
	raw, err := json.Marshal(clean)
	if err != nil {
		return "", err
	}
	return string(raw), nil
}

func MergeDialogChannelSecretsOnUpdate(oldJSON, newJSON string) (string, error) {
	var oldC, newC map[string]any
	if err := json.Unmarshal([]byte(oldJSON), &oldC); err != nil {
		return newJSON, err
	}
	if err := json.Unmarshal([]byte(newJSON), &newC); err != nil {
		return newJSON, err
	}
	if newC == nil {
		newC = map[string]any{}
	}
	for _, k := range []string{"secret", "appSecret", "encodingAESKey", "token"} {
		ns, _ := newC[k].(string)
		os, _ := oldC[k].(string)
		if strings.TrimSpace(ns) == "" && strings.TrimSpace(os) != "" {
			newC[k] = os
		}
	}
	out, err := json.Marshal(newC)
	if err != nil {
		return newJSON, err
	}
	return string(out), nil
}

func ParseDialogChannelConfig(row TenantDialogChannel) (map[string]string, error) {
	cfg := map[string]any{}
	if strings.TrimSpace(row.ConfigJSON) == "" {
		return nil, errors.New("empty dialog channel config")
	}
	if err := json.Unmarshal([]byte(row.ConfigJSON), &cfg); err != nil {
		return nil, err
	}
	out := map[string]string{}
	for k, v := range cfg {
		if s, ok := v.(string); ok {
			out[k] = strings.TrimSpace(s)
		}
	}
	return out, nil
}

// TenantDialogSkill is a tenant-scoped skill (prompt and/or sandbox script) bindable to assistants.
type TenantDialogSkill struct {
	common.BaseModel

	TenantID    uint   `json:"tenantId" gorm:"index;not null;uniqueIndex:idx_tenant_dialog_skill_code"`
	Code        string `json:"code" gorm:"size:64;not null;uniqueIndex:idx_tenant_dialog_skill_code;comment:stable bind key"`
	Name        string `json:"name" gorm:"size:128;not null"`
	Description string `json:"description,omitempty" gorm:"size:512"`
	// Kind: prompt | python | node
	Kind string `json:"kind" gorm:"size:16;not null;default:prompt;index"`
	// Body is markdown prompt appendix (prompt skills) or usage hint (script skills).
	Body string `json:"body" gorm:"type:text;comment:markdown prompt / usage hint"`
	// ScriptContent is inline script for python/node when no uploaded bundle.
	ScriptContent string `json:"scriptContent,omitempty" gorm:"type:longtext"`
	// EntryFile is relative entry under the skill assets dir (default main.py / index.js).
	EntryFile string `json:"entryFile,omitempty" gorm:"size:255"`
	// HasAssets is true after a zip/folder upload was extracted for this skill.
	HasAssets bool `json:"hasAssets" gorm:"not null;default:false"`
	Enabled   bool `json:"enabled" gorm:"not null;default:true;index"`
}

func (TenantDialogSkill) TableName() string { return constants.TENANT_DIALOG_SKILL_TABLE_NAME }

const (
	DialogSkillKindPrompt = "prompt"
	DialogSkillKindPython = "python"
	DialogSkillKindNode   = "node"
)

func NormalizeDialogSkillKind(kind string) string {
	switch strings.ToLower(strings.TrimSpace(kind)) {
	case DialogSkillKindPython, "py", "python3":
		return DialogSkillKindPython
	case DialogSkillKindNode, "js", "javascript", "nodejs":
		return DialogSkillKindNode
	default:
		return DialogSkillKindPrompt
	}
}

// NormalizeDialogSkillCode keeps letters (incl. CJK), digits, hyphen and underscore.
// Path separators and dots are stripped. Empty input stays empty.
func NormalizeDialogSkillCode(code string) string {
	code = strings.TrimSpace(code)
	code = strings.ReplaceAll(code, " ", "-")
	var b strings.Builder
	for _, r := range code {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9', r == '-', r == '_':
			if r >= 'A' && r <= 'Z' {
				r = r - 'A' + 'a'
			}
			b.WriteRune(r)
		case r >= 0x4e00 && r <= 0x9fff: // CJK Unified Ideographs (common)
			b.WriteRune(r)
		case unicode.IsLetter(r) || unicode.IsNumber(r):
			b.WriteRune(r)
		}
	}
	out := b.String()
	if len([]rune(out)) > 64 {
		out = string([]rune(out)[:64])
	}
	return out
}

// EnsureDialogSkillCode returns a usable code: prefer code, then name, then generated.
func EnsureDialogSkillCode(code, name string) string {
	if c := NormalizeDialogSkillCode(code); c != "" {
		return c
	}
	if c := NormalizeDialogSkillCode(name); c != "" {
		return c
	}
	// ASCII fallback via pinyin when name is pure CJK that somehow failed — keep deterministic-ish.
	if slug := skillCodePinyinSlug(name); slug != "" {
		return slug
	}
	return fmt.Sprintf("skill-%d", time.Now().UnixNano()%1_000_000_000)
}

func skillCodePinyinSlug(name string) string {
	name = strings.TrimSpace(name)
	if name == "" {
		return ""
	}
	args := pinyin.NewArgs()
	args.Style = pinyin.Normal
	parts := pinyin.Pinyin(name, args)
	var b strings.Builder
	for _, syl := range parts {
		if len(syl) == 0 || syl[0] == "" {
			continue
		}
		if b.Len() > 0 {
			b.WriteByte('-')
		}
		b.WriteString(syl[0])
	}
	return NormalizeDialogSkillCode(b.String())
}

func DefaultDialogSkillEntry(kind string) string {
	switch NormalizeDialogSkillKind(kind) {
	case DialogSkillKindPython:
		return "main.py"
	case DialogSkillKindNode:
		return "index.js"
	default:
		return ""
	}
}

func ValidateTenantDialogSkill(row *TenantDialogSkill) error {
	if row == nil {
		return gorm.ErrInvalidData
	}
	row.Kind = NormalizeDialogSkillKind(row.Kind)
	row.Code = EnsureDialogSkillCode(row.Code, row.Name)
	row.Name = strings.TrimSpace(row.Name)
	row.Description = strings.TrimSpace(row.Description)
	row.Body = strings.TrimSpace(row.Body)
	row.ScriptContent = strings.TrimSpace(row.ScriptContent)
	row.EntryFile = strings.TrimSpace(row.EntryFile)
	if row.TenantID == 0 {
		return fmt.Errorf("tenantId is required")
	}
	if row.Code == "" {
		return fmt.Errorf("code is required")
	}
	if row.Name == "" {
		return fmt.Errorf("name is required")
	}
	switch row.Kind {
	case DialogSkillKindPrompt:
		if row.Body == "" {
			return fmt.Errorf("body is required for prompt skills")
		}
	case DialogSkillKindPython, DialogSkillKindNode:
		if row.EntryFile == "" {
			row.EntryFile = DefaultDialogSkillEntry(row.Kind)
		}
		if row.ScriptContent == "" && !row.HasAssets {
			return fmt.Errorf("scriptContent or uploaded assets required for %s skills", row.Kind)
		}
		if row.Body == "" {
			row.Body = fmt.Sprintf("需要时可调用工具 run_skill_%s 执行本技能脚本。", SkillToolSuffix(row.Code))
		}
	}
	return nil
}

// SkillToolSuffix is the ASCII tool-name suffix for a skill code (pinyin if CJK).
func SkillToolSuffix(code string) string {
	code = NormalizeDialogSkillCode(code)
	if code == "" {
		return "skill"
	}
	onlyASCII := true
	for _, r := range code {
		if r > 127 {
			onlyASCII = false
			break
		}
	}
	if onlyASCII {
		s := strings.ReplaceAll(code, "-", "_")
		if len(s) > 40 {
			s = s[:40]
		}
		return s
	}
	if slug := skillCodePinyinSlug(code); slug != "" {
		s := strings.ReplaceAll(slug, "-", "_")
		if len(s) > 40 {
			s = s[:40]
		}
		return s
	}
	return "skill"
}

func ListTenantDialogSkills(db *gorm.DB, tenantID uint, enabledOnly bool) ([]TenantDialogSkill, error) {
	if db == nil || tenantID == 0 {
		return nil, nil
	}
	q := db.Where("tenant_id = ?", tenantID)
	if enabledOnly {
		q = q.Where("enabled = ?", true)
	}
	var rows []TenantDialogSkill
	err := q.Order("code ASC").Find(&rows).Error
	return rows, err
}

func ListTenantDialogSkillsPage(db *gorm.DB, tenantID uint, page, size int) ([]TenantDialogSkill, int64, error) {
	q := db.Model(&TenantDialogSkill{}).Where("tenant_id = ?", tenantID)
	return utils.FindPage[TenantDialogSkill](q, page, size, "id DESC", utils.DefaultMaxPageSize)
}

func GetTenantDialogSkill(db *gorm.DB, tenantID, id uint) (TenantDialogSkill, error) {
	var row TenantDialogSkill
	err := db.Where("id = ? AND tenant_id = ?", id, tenantID).First(&row).Error
	return row, err
}

func GetTenantDialogSkillsByCodes(db *gorm.DB, tenantID uint, codes []string) ([]TenantDialogSkill, error) {
	if db == nil || tenantID == 0 || len(codes) == 0 {
		return nil, nil
	}
	norm := make([]string, 0, len(codes))
	seen := map[string]struct{}{}
	for _, c := range codes {
		c = NormalizeDialogSkillCode(c)
		if c == "" {
			continue
		}
		if _, ok := seen[c]; ok {
			continue
		}
		seen[c] = struct{}{}
		norm = append(norm, c)
	}
	if len(norm) == 0 {
		return nil, nil
	}
	var rows []TenantDialogSkill
	err := db.Where("tenant_id = ? AND enabled = ? AND code IN ?", tenantID, true, norm).
		Order("code ASC").
		Find(&rows).Error
	return rows, err
}

func CreateTenantDialogSkill(db *gorm.DB, row *TenantDialogSkill) error {
	if err := ValidateTenantDialogSkill(row); err != nil {
		return err
	}
	return db.Create(row).Error
}

func UpdateTenantDialogSkill(db *gorm.DB, tenantID, id uint, updates map[string]any) error {
	if len(updates) == 0 {
		return nil
	}
	if code, ok := updates["code"].(string); ok {
		updates["code"] = NormalizeDialogSkillCode(code)
	}
	return db.Model(&TenantDialogSkill{}).Where("tenant_id = ? AND id = ?", tenantID, id).Updates(updates).Error
}

func DeleteTenantDialogSkill(db *gorm.DB, tenantID, id uint) error {
	return db.Where("tenant_id = ? AND id = ?", tenantID, id).Delete(&TenantDialogSkill{}).Error
}
