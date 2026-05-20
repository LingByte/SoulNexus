package svcmodels

// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

import (
	"errors"
	"fmt"
	"strings"
	"time"

	auth "github.com/LingByte/SoulNexus/internal/models/auth"
	"github.com/LingByte/SoulNexus/pkg/utils"
	"gorm.io/gorm"
)

// CreateUserCredential 创建用户凭证（绑定用户个人组织）
func CreateUserCredential(db *gorm.DB, userID uint, credential *auth.UserCredentialRequest) (*auth.UserCredential, error) {
	normalizedProvider, err := normalizeLLMProvider(credential.LLMProvider)
	if err != nil {
		return nil, err
	}

	credential.LLMProvider = normalizedProvider
	credential.LLMApiURL = strings.TrimSpace(credential.LLMApiURL)
	credential.LLMApiKey = strings.TrimSpace(credential.LLMApiKey)
	credential.Name = strings.TrimSpace(credential.Name)

	if credential.Name == "" {
		return nil, errors.New("credential name is required")
	}

	if credential.LLMProvider == "" {
		return nil, errors.New("llm provider is required")
	}

	if credential.LLMProvider != "ollama" && credential.LLMApiKey == "" {
		return nil, errors.New("llm api key is required")
	}

	if credential.LLMProvider == "coze" {
		if credential.LLMApiURL == "" {
			return nil, errors.New("coze bot id is required")
		}
	} else if credential.LLMApiURL == "" {
		switch credential.LLMProvider {
		case "openai", "anthropic", "ollama", "lmstudio":
			credential.LLMApiURL = defaultProviderURL(credential.LLMProvider)
		}
	}

	apiKey, err := utils.GenerateSecureToken(32)
	if err != nil {
		return nil, err
	}

	apiSecret, err := utils.GenerateSecureToken(64)
	if err != nil {
		return nil, err
	}

	asrConfig := credential.BuildASRConfig()
	ttsConfig := credential.BuildTTSConfig()

	var expiresAt *time.Time
	if credential.ExpiresAt != nil {
		raw := strings.TrimSpace(*credential.ExpiresAt)
		if raw != "" {
			var parsed time.Time
			var parseErr error
			if strings.Contains(raw, "T") {
				parsed, parseErr = time.Parse(time.RFC3339, raw)
			} else {
				parsed, parseErr = time.ParseInLocation("2006-01-02 15:04:05", raw, time.Local)
			}
			if parseErr != nil {
				return nil, fmt.Errorf("invalid expiresAt format")
			}
			expiresAt = &parsed
		}
	}
	tokenQuota := int64(0)
	if credential.TokenQuota != nil && *credential.TokenQuota >= 0 {
		tokenQuota = *credential.TokenQuota
	}
	requestQuota := int64(0)
	if credential.RequestQuota != nil && *credential.RequestQuota >= 0 {
		requestQuota = *credential.RequestQuota
	}
	useNativeQuota := false
	if credential.UseNativeQuota != nil {
		useNativeQuota = *credential.UseNativeQuota
	}
	unlimitedQuota := true
	if credential.UnlimitedQuota != nil {
		unlimitedQuota = *credential.UnlimitedQuota
	}

	pg, err := EnsurePersonalGroupForUser(db, userID)
	if err != nil {
		return nil, err
	}

	userCred := &auth.UserCredential{
		GroupID:        pg.ID,
		CreatedBy:      userID,
		APIKey:         apiKey,
		APISecret:      apiSecret,
		Name:           credential.Name,
		Status:         auth.CredentialStatusActive,
		ExpiresAt:      expiresAt,
		TokenQuota:     tokenQuota,
		RequestQuota:   requestQuota,
		UseNativeQuota: useNativeQuota,
		UnlimitedQuota: unlimitedQuota,
		LLMProvider:    credential.LLMProvider,
		LLMApiKey:      credential.LLMApiKey,
		LLMApiURL:      credential.LLMApiURL,
		AsrConfig:      asrConfig,
		TtsConfig:      ttsConfig,
		UsageCount:     0,
		LastUsedAt:     nil,
		BannedAt:       nil,
		BannedReason:   "",
		BannedBy:       nil,
	}

	if err := db.Create(userCred).Error; err != nil {
		return nil, err
	}
	return userCred, nil
}

func normalizeLLMProvider(provider string) (string, error) {
	p := strings.ToLower(strings.TrimSpace(provider))
	switch p {
	case "openi":
		return "openai", nil
	case "openai", "ollama", "coze", "anthropic", "lmstudio":
		return p, nil
	default:
		return "", fmt.Errorf("unsupported llm provider: %s", provider)
	}
}

func defaultProviderURL(provider string) string {
	switch provider {
	case "openai":
		return "https://api.openai.com/v1"
	case "anthropic":
		return "https://api.anthropic.com"
	case "ollama":
		return "http://localhost:11434/v1"
	case "lmstudio":
		return "http://localhost:1234/v1"
	default:
		return ""
	}
}

// GetUserCredentials returns credentials for every organization the user belongs to.
func GetUserCredentials(db *gorm.DB, userID uint) ([]*auth.UserCredential, error) {
	ids, err := MemberGroupIDs(db, userID)
	if err != nil {
		return nil, err
	}
	var credentials []*auth.UserCredential
	if len(ids) == 0 {
		return credentials, nil
	}
	err = db.Where("group_id IN ?", ids).Find(&credentials).Error
	if err != nil {
		return nil, err
	}
	return credentials, nil
}

// DeleteUserCredential 删除用户凭证（需在用户所属组织内）
func DeleteUserCredential(db *gorm.DB, userID uint, credentialID uint) error {
	ids, err := MemberGroupIDs(db, userID)
	if err != nil {
		return err
	}
	if len(ids) == 0 {
		return errors.New("credential not found or access denied")
	}
	result := db.Where("id = ? AND group_id IN ?", credentialID, ids).Delete(&auth.UserCredential{})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return errors.New("credential not found or access denied")
	}
	return nil
}
