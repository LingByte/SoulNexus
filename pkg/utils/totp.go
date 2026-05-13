package utils

// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

import (
	"encoding/base64"
	"errors"
	"strings"

	"github.com/pquerna/otp"
	"github.com/pquerna/otp/totp"
	qrcode "github.com/skip2/go-qrcode"
)

const (
	// DefaultTOTPIssuer authenticator apps 显示的发行方名称（otpauth URL）。
	// DefaultTOTPIssuer: shown by authenticator apps. Changing this affects new
	// enrollments only — existing TOTP entries on user devices keep their old
	// label until the user re-enrolls.
	DefaultTOTPIssuer = "SoulNexus"
	// DefaultTOTPSecretSize TOTP 密钥字节长度。
	DefaultTOTPSecretSize = 32
	// DefaultTOTPQRPNGSize 二维码 PNG 边长（像素）。
	DefaultTOTPQRPNGSize = 256
)

// TOTPSetup 注册双因素认证时返回给客户端的材料。
type TOTPSetup struct {
	Secret    string // Base32 密钥，可手动录入
	URL       string // otpauth:// 链接
	QRDataURL string // data:image/png;base64,... 便于前端直接展示
}

// ValidateTOTP 校验当前时间窗口内的 TOTP 码（前后 trim，空码/空密钥为 false）。
func ValidateTOTP(code, secret string) bool {
	code = strings.TrimSpace(code)
	secret = strings.TrimSpace(secret)
	if code == "" || secret == "" {
		return false
	}
	return totp.Validate(code, secret)
}

// GenerateTOTPSetup 生成新的 TOTP 密钥、otpauth URL 及二维码 PNG（Data URL）。
// issuer 为空时使用 DefaultTOTPIssuer；secretSize <= 0 时使用 DefaultTOTPSecretSize。
func GenerateTOTPSetup(issuer, accountName string, secretSize int) (*TOTPSetup, error) {
	if issuer == "" {
		issuer = DefaultTOTPIssuer
	}
	if secretSize <= 0 {
		secretSize = DefaultTOTPSecretSize
	}
	key, err := totp.Generate(totp.GenerateOpts{
		Issuer:      issuer,
		AccountName: accountName,
		SecretSize:  uint(secretSize),
	})
	if err != nil {
		return nil, err
	}
	qr, err := TOTPKeyToPNGDataURL(key, DefaultTOTPQRPNGSize)
	if err != nil {
		return nil, err
	}
	return &TOTPSetup{
		Secret:    key.Secret(),
		URL:       key.URL(),
		QRDataURL: qr,
	}, nil
}

// TOTPKeyToPNGDataURL 将已生成的 otp.Key 转为二维码 PNG 的 Data URL（便于 img src 直接使用）。
func TOTPKeyToPNGDataURL(key *otp.Key, pngSize int) (string, error) {
	if key == nil {
		return "", errors.New("totp: key is nil")
	}
	if pngSize <= 0 {
		pngSize = DefaultTOTPQRPNGSize
	}
	qrCode, err := qrcode.New(key.URL(), qrcode.Medium)
	if err != nil {
		return "", err
	}
	png, err := qrCode.PNG(pngSize)
	if err != nil {
		return "", err
	}
	return "data:image/png;base64," + base64.StdEncoding.EncodeToString(png), nil
}
