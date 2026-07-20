package constants

// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

// Auth mail signals — handlers emit; internal/listeners deliver asynchronously.
const (
	SigMailWelcome            = "auth:mail:welcome"
	SigMailVerificationCode   = "auth:mail:verification_code"
	SigMailDeviceVerifyCode   = "auth:mail:device_verify_code"
	SigMailNewDeviceLogin     = "auth:mail:new_device_login"
)

// MailWelcomePayload is emitted after first successful login / registration.
type MailWelcomePayload struct {
	PrincipalType string
	UserID        uint
	Email         string
	DisplayName   string
	ReceiveEmail  bool
	ClientIP      string
}

// MailVerificationCodePayload carries a one-time email verification code.
type MailVerificationCodePayload struct {
	UserID   uint
	Email    string
	Code     string
	Purpose  string
	ClientIP string
}

// MailDeviceVerifyCodePayload carries a device-login verification code.
type MailDeviceVerifyCodePayload struct {
	UserID    uint
	Email     string
	Username  string
	Code      string
	DeviceKey string
	ClientIP  string
}

// MailNewDeviceLoginPayload notifies the user of a login from a new or unverified device.
type MailNewDeviceLoginPayload struct {
	UserID       uint
	Email        string
	Username     string
	LoginTime    string
	ClientIP     string
	Location     string
	DeviceType   string
	IsSuspicious bool
}
