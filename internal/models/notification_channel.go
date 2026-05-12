// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

package models

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/LingByte/SoulNexus/pkg/notification/mail"
	"gorm.io/gorm"
)

// 通知渠道类型常量
const (
	NotificationChannelTypeEmail = "email"
	NotificationChannelTypeSMS   = "sms"
)

// NotificationChannel 描述一种可配置的通知出口（邮件 / 短信 等）。
// 同一 OrgID 下 Code 唯一，Type+SortOrder 决定多渠道排序。
type NotificationChannel struct {
	BaseModel
	OrgID      uint   `json:"orgId" gorm:"uniqueIndex:idx_notify_org_code;not null;default:0;comment:tenant id (0=system)"`
	Type       string `json:"type" gorm:"size:32;not null;index:idx_notify_ch_type_sort,priority:1;comment:渠道类型"`
	Code       string `json:"code,omitempty" gorm:"size:64;uniqueIndex:idx_notify_org_code;comment:渠道编码"`
	Name       string `json:"name" gorm:"size:128;not null;comment:显示名称"`
	SortOrder  int    `json:"sortOrder" gorm:"not null;default:0;index:idx_notify_ch_type_sort,priority:2;comment:排序权重"`
	Enabled    bool   `json:"enabled" gorm:"not null;default:true;index;comment:是否启用"`
	Remark     string `json:"remark,omitempty" gorm:"size:255;comment:备注"`
	ConfigJSON string `json:"configJson,omitempty" gorm:"type:text;comment:渠道配置 JSON"`
}

// TableName GORM 表名
func (NotificationChannel) TableName() string { return "notification_channels" }

// EmailChannelFormView 用于前端表单回显（不回显 SMTP 密码 / SendCloud APIKey 明文）。
type EmailChannelFormView struct {
	Driver             string `json:"driver"`
	SMTPHost           string `json:"smtpHost"`
	SMTPPort           int64  `json:"smtpPort"`
	SMTPUsername       string `json:"smtpUsername"`
	SMTPFrom           string `json:"smtpFrom"`
	FromDisplayName    string `json:"fromDisplayName"`
	SMTPPasswordSet    bool   `json:"smtpPasswordSet"`
	SendcloudAPIUser   string `json:"sendcloudApiUser"`
	SendcloudAPIKeySet bool   `json:"sendcloudApiKeySet"`
	SendcloudFrom      string `json:"sendcloudFrom"`
}

// SMSChannelFormView 用于前端表单回显（敏感字段会被擦除）。
type SMSChannelFormView struct {
	Provider   string         `json:"provider"`
	Config     map[string]any `json:"config"`
	SecretKeys []string       `json:"secretKeys,omitempty"`
}

type smsChannelConfigEnvelope struct {
	Provider string         `json:"provider"`
	Config   map[string]any `json:"config"`
}

// NotificationChannelListResult 分页结果。
type NotificationChannelListResult struct {
	List      []NotificationChannel `json:"list"`
	Total     int64                 `json:"total"`
	Page      int                   `json:"page"`
	PageSize  int                   `json:"pageSize"`
	TotalPage int                   `json:"totalPage"`
}

// ListNotificationChannels 分页列出（可按 type 过滤）。
func ListNotificationChannels(db *gorm.DB, channelType string, page, pageSize int) (*NotificationChannelListResult, error) {
	if db == nil {
		return nil, errors.New("nil db")
	}
	if page < 1 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 20
	}
	offset := (page - 1) * pageSize

	q := db.Model(&NotificationChannel{})
	if t := strings.TrimSpace(channelType); t != "" {
		q = q.Where("type = ?", t)
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, err
	}
	var list []NotificationChannel
	listQ := db.Model(&NotificationChannel{})
	if t := strings.TrimSpace(channelType); t != "" {
		listQ = listQ.Where("type = ?", t)
	}
	if err := listQ.Order("type ASC, sort_order ASC, id ASC").Offset(offset).Limit(pageSize).Find(&list).Error; err != nil {
		return nil, err
	}
	totalPage := int(total) / pageSize
	if int(total)%pageSize != 0 {
		totalPage++
	}
	return &NotificationChannelListResult{
		List: list, Total: total, Page: page, PageSize: pageSize, TotalPage: totalPage,
	}, nil
}

// GetNotificationChannel 按主键读取。
func GetNotificationChannel(db *gorm.DB, id uint) (*NotificationChannel, error) {
	if db == nil {
		return nil, errors.New("nil db")
	}
	var row NotificationChannel
	if err := db.First(&row, id).Error; err != nil {
		return nil, err
	}
	return &row, nil
}

// BuildEmailChannelConfigJSON 把表单构造成 mail.MailConfig 兼容 JSON。
func BuildEmailChannelConfigJSON(driver, name string, smtpHost string, smtpPort int64, smtpUser, smtpPassword, smtpFrom, fromDisplayName string, scUser, scKey, scFrom string) (string, error) {
	driver = strings.ToLower(strings.TrimSpace(driver))
	cfg := mail.MailConfig{Name: strings.TrimSpace(name), FromName: strings.TrimSpace(fromDisplayName)}
	switch driver {
	case mail.ProviderSMTP:
		if strings.TrimSpace(smtpHost) == "" || smtpPort <= 0 || strings.TrimSpace(smtpFrom) == "" {
			return "", errors.New("SMTP 需要 host、port、发件地址")
		}
		cfg.Provider = mail.ProviderSMTP
		cfg.Host = strings.TrimSpace(smtpHost)
		cfg.Port = smtpPort
		cfg.Username = strings.TrimSpace(smtpUser)
		cfg.Password = smtpPassword
		cfg.From = strings.TrimSpace(smtpFrom)
	case mail.ProviderSendCloud:
		if strings.TrimSpace(scUser) == "" || strings.TrimSpace(scKey) == "" || strings.TrimSpace(scFrom) == "" {
			return "", errors.New("SendCloud 需要 api_user、api_key、发件地址")
		}
		cfg.Provider = mail.ProviderSendCloud
		cfg.APIUser = strings.TrimSpace(scUser)
		cfg.APIKey = strings.TrimSpace(scKey)
		cfg.From = strings.TrimSpace(scFrom)
	default:
		return "", fmt.Errorf("不支持的邮件驱动: %s", driver)
	}
	raw, err := json.Marshal(cfg)
	if err != nil {
		return "", err
	}
	return string(raw), nil
}

// DecodeEmailChannelForm 把 config_json 解析成扁平的表单视图（不回显密码）。
func DecodeEmailChannelForm(configJSON string) (*EmailChannelFormView, error) {
	var cfg mail.MailConfig
	if err := json.Unmarshal([]byte(configJSON), &cfg); err != nil {
		return nil, err
	}
	v := &EmailChannelFormView{}
	switch strings.ToLower(strings.TrimSpace(cfg.Provider)) {
	case mail.ProviderSendCloud:
		v.Driver = mail.ProviderSendCloud
		v.SendcloudAPIUser = cfg.APIUser
		v.SendcloudFrom = cfg.From
		v.SendcloudAPIKeySet = cfg.APIKey != ""
		v.FromDisplayName = cfg.FromName
	case mail.ProviderSMTP, "":
		v.Driver = mail.ProviderSMTP
		v.SMTPHost = cfg.Host
		v.SMTPPort = cfg.Port
		v.SMTPUsername = cfg.Username
		v.SMTPFrom = cfg.From
		v.SMTPPasswordSet = cfg.Password != ""
		v.FromDisplayName = cfg.FromName
	default:
		v.Driver = strings.ToLower(strings.TrimSpace(cfg.Provider))
	}
	return v, nil
}

// MergeEmailSecretsOnUpdate 在更新时保留客户端未填的敏感字段。
func MergeEmailSecretsOnUpdate(oldJSON, newJSON string) (string, error) {
	var oldC, newC mail.MailConfig
	if err := json.Unmarshal([]byte(oldJSON), &oldC); err != nil {
		return newJSON, err
	}
	if err := json.Unmarshal([]byte(newJSON), &newC); err != nil {
		return newJSON, err
	}
	if strings.ToLower(newC.Provider) == mail.ProviderSMTP && newC.Password == "" && oldC.Password != "" {
		newC.Password = oldC.Password
	}
	if strings.ToLower(newC.Provider) == mail.ProviderSendCloud && newC.APIKey == "" && oldC.APIKey != "" {
		newC.APIKey = oldC.APIKey
	}
	if strings.TrimSpace(newC.FromName) == "" && strings.TrimSpace(oldC.FromName) != "" {
		newC.FromName = oldC.FromName
	}
	out, err := json.Marshal(newC)
	if err != nil {
		return newJSON, err
	}
	return string(out), nil
}

// BuildSMSChannelConfigJSON 把短信表单构造成 envelope JSON。
func BuildSMSChannelConfigJSON(provider string, cfg any) (string, error) {
	p := strings.ToLower(strings.TrimSpace(provider))
	if p == "" {
		return "", errors.New("sms provider 不能为空")
	}
	var m map[string]any
	switch v := cfg.(type) {
	case map[string]any:
		m = v
	default:
		if cfg == nil {
			m = map[string]any{}
		} else {
			b, err := json.Marshal(cfg)
			if err != nil {
				return "", err
			}
			_ = json.Unmarshal(b, &m)
		}
	}
	env := smsChannelConfigEnvelope{Provider: p, Config: m}
	raw, err := json.Marshal(env)
	if err != nil {
		return "", err
	}
	if len(env.Config) == 0 {
		return "", fmt.Errorf("sms provider=%s 缺少配置", p)
	}
	return string(raw), nil
}

// DecodeSMSChannelForm 把 config_json 解析为表单视图（敏感值擦除）。
func DecodeSMSChannelForm(configJSON string) (*SMSChannelFormView, error) {
	var env smsChannelConfigEnvelope
	if err := json.Unmarshal([]byte(configJSON), &env); err != nil {
		return nil, err
	}
	out := &SMSChannelFormView{
		Provider: strings.ToLower(strings.TrimSpace(env.Provider)),
		Config:   env.Config,
	}
	switch out.Provider {
	case "yunpian", "luosimao", "juhe":
		out.SecretKeys = []string{"apiKey", "appKey"}
	case "twilio":
		out.SecretKeys = []string{"token"}
	case "huyi":
		out.SecretKeys = []string{"apiKey"}
	case "submail":
		out.SecretKeys = []string{"appKey"}
	case "chuanglan":
		out.SecretKeys = []string{"password"}
	case "tencent":
		out.SecretKeys = []string{"secretKey"}
	case "aliyun":
		out.SecretKeys = []string{"accessKeySecret"}
	case "huawei":
		out.SecretKeys = []string{"appSecret"}
	case "baidu":
		out.SecretKeys = []string{"secretKey"}
	case "ucloud":
		out.SecretKeys = []string{"privateKey"}
	case "netease":
		out.SecretKeys = []string{"appSecret"}
	case "rongcloud":
		out.SecretKeys = []string{"appSecret"}
	case "yuntongxun":
		out.SecretKeys = []string{"authToken"}
	case "tiniyo":
		out.SecretKeys = []string{"authToken"}
	case "yunpian2":
		out.SecretKeys = []string{"apiKey"}
	}
	for _, k := range out.SecretKeys {
		if _, ok := out.Config[k]; ok {
			out.Config[k] = ""
		}
	}
	return out, nil
}

// MergeSMSSecretsOnUpdate 在更新时保留旧的敏感字段（如客户端发送的是空串）。
func MergeSMSSecretsOnUpdate(oldJSON, newJSON string) (string, error) {
	var oldE, newE smsChannelConfigEnvelope
	if err := json.Unmarshal([]byte(oldJSON), &oldE); err != nil {
		return newJSON, err
	}
	if err := json.Unmarshal([]byte(newJSON), &newE); err != nil {
		return newJSON, err
	}
	if strings.ToLower(strings.TrimSpace(oldE.Provider)) != strings.ToLower(strings.TrimSpace(newE.Provider)) {
		return newJSON, nil
	}
	if newE.Config == nil {
		newE.Config = map[string]any{}
	}
	for k, ov := range oldE.Config {
		os, ok := ov.(string)
		if !ok || strings.TrimSpace(os) == "" {
			continue
		}
		if nv, ok := newE.Config[k]; ok {
			if ns, ok := nv.(string); ok && strings.TrimSpace(ns) == "" {
				newE.Config[k] = os
			}
		}
	}
	out, err := json.Marshal(newE)
	if err != nil {
		return newJSON, err
	}
	return string(out), nil
}
