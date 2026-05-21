package authgrpc

// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

import (
	"encoding/json"

	authv1 "github.com/LingByte/SoulNexus/internal/grpc/auth/pb/auth/v1"
	authmodel "github.com/LingByte/SoulNexus/internal/models/auth"
)

// UserToProto maps a DB user to the internal RPC message (no password / 2FA secrets).
func UserToProto(u *authmodel.User) *authv1.User {
	if u == nil {
		return nil
	}
	out := &authv1.User{
		Id:               uint64(u.ID),
		Email:            u.Email,
		Status:           u.Status,
		Source:           u.Source,
		EmailVerified:    u.EmailVerified,
		PhoneVerified:    u.PhoneVerified,
		TwoFactorEnabled: u.TwoFactorEnabled,
		Phone:            u.Phone,
		Locale:           u.Locale,
		Timezone:         u.Timezone,
		ThemeMode:        u.ThemeMode,
		RoleSlugs:        append([]string(nil), u.RoleSlugs...),
		Profile: &authv1.UserProfile{
			DisplayName: u.Profile.DisplayName,
			Avatar:      u.Profile.Avatar,
		},
	}
	return out
}

// CredentialToProto maps a DB credential including provider secrets for service-to-service RPC.
func CredentialToProto(c *authmodel.UserCredential) (*authv1.UserCredential, error) {
	if c == nil {
		return nil, nil
	}
	asrJSON, err := json.Marshal(c.AsrConfig)
	if err != nil {
		return nil, err
	}
	ttsJSON, err := json.Marshal(c.TtsConfig)
	if err != nil {
		return nil, err
	}
	return &authv1.UserCredential{
		Id:             uint64(c.ID),
		GroupId:        uint64(c.GroupID),
		CreatedBy:      uint64(c.CreatedBy),
		Name:           c.Name,
		ApiKey:         c.APIKey,
		ApiSecret:      c.APISecret,
		Status:         string(c.Status),
		LlmProvider:    c.LLMProvider,
		LlmApiKey:      c.LLMApiKey,
		LlmApiUrl:      c.LLMApiURL,
		AsrConfigJson:  string(asrJSON),
		TtsConfigJson:  string(ttsJSON),
		TokenQuota:     c.TokenQuota,
		TokenUsed:      c.TokenUsed,
		RequestQuota:   c.RequestQuota,
		UseNativeQuota: c.UseNativeQuota,
		UnlimitedQuota: c.UnlimitedQuota,
		UsageCount:     c.UsageCount,
	}, nil
}

func UserFromProto(p *authv1.User) *authmodel.User {
	if p == nil {
		return nil
	}
	u := &authmodel.User{
		Email:            p.Email,
		Status:           p.Status,
		Source:           p.Source,
		EmailVerified:    p.EmailVerified,
		PhoneVerified:    p.PhoneVerified,
		TwoFactorEnabled: p.TwoFactorEnabled,
		Phone:            p.Phone,
		Locale:           p.Locale,
		Timezone:         p.Timezone,
		ThemeMode:        p.ThemeMode,
		RoleSlugs:        append([]string(nil), p.RoleSlugs...),
	}
	u.ID = uint(p.Id)
	if p.Profile != nil {
		u.Profile = authmodel.UserProfile{
			DisplayName: p.Profile.DisplayName,
			Avatar:      p.Profile.Avatar,
		}
	}
	return u
}

func CredentialFromProto(p *authv1.UserCredential) (*authmodel.UserCredential, error) {
	if p == nil {
		return nil, nil
	}
	c := &authmodel.UserCredential{
		Name:           p.Name,
		APIKey:         p.ApiKey,
		APISecret:      p.ApiSecret,
		Status:         authmodel.CredentialStatus(p.Status),
		LLMProvider:    p.LlmProvider,
		LLMApiKey:      p.LlmApiKey,
		LLMApiURL:      p.LlmApiUrl,
		TokenQuota:     p.TokenQuota,
		TokenUsed:      p.TokenUsed,
		RequestQuota:   p.RequestQuota,
		UseNativeQuota: p.UseNativeQuota,
		UnlimitedQuota: p.UnlimitedQuota,
		UsageCount:     p.UsageCount,
	}
	c.ID = uint(p.Id)
	c.GroupID = uint(p.GroupId)
	c.CreatedBy = uint(p.CreatedBy)
	if p.AsrConfigJson != "" {
		var asr authmodel.ProviderConfig
		if err := json.Unmarshal([]byte(p.AsrConfigJson), &asr); err != nil {
			return nil, err
		}
		c.AsrConfig = asr
	}
	if p.TtsConfigJson != "" {
		var tts authmodel.ProviderConfig
		if err := json.Unmarshal([]byte(p.TtsConfigJson), &tts); err != nil {
			return nil, err
		}
		c.TtsConfig = tts
	}
	return c, nil
}
