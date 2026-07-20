package request

import "github.com/LingByte/SoulNexus/pkg/utils/captcha"

// Copyright (c) 2026 LingByte
// SPDX-License-Identifier: MIT

type TenantLoginReq struct {
	Email       string `json:"email"`
	Phone       string `json:"phone"`
	Password    string `json:"password"`
	EmailCode   string `json:"emailCode"`
	SMSCode     string `json:"smsCode"`
	TotpCode    string `json:"totpCode"`
	LoginMethod string `json:"loginMethod"` // "password" (default) | "email_code" | "phone_code"
	DeviceID    string `json:"deviceId"`
	DeviceVerifyCode      string `json:"deviceVerifyCode"`
	VoiceprintAudioBase64 string `json:"voiceprintAudioBase64"`
	TrustDeviceFor7Days   bool   `json:"trustDeviceFor7Days"`
	captcha.CaptchaFields
}

type ChangeEmailReq struct {
	NewEmail  string `json:"newEmail" binding:"required,email"`
	EmailCode string `json:"emailCode" binding:"required"`
}

type AccountDeletionReq struct {
	Password  string `json:"password"`
	EmailCode string `json:"emailCode"`
	Method    string `json:"method"` // password | email_code
}
