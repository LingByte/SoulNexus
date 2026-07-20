// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

package handlers

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/LingByte/SoulNexus/internal/listeners"
	"github.com/LingByte/SoulNexus/internal/models"
	"github.com/LingByte/SoulNexus/pkg/i18n"
	"github.com/LingByte/SoulNexus/pkg/notification/sms"
	"github.com/LingByte/SoulNexus/pkg/response"
	"github.com/LingByte/SoulNexus/pkg/utils"
	"github.com/LingByte/SoulNexus/pkg/utils/captcha"
	"github.com/gin-gonic/gin"
)

type sendSMSCodeReq struct {
	Phone string `json:"phone" binding:"required"`
	captcha.CaptchaFields
}

func smsCodeCacheKey(purpose emailCodePurpose, phone string) string {
	return "sms:" + string(purpose) + ":" + utils.NormalizePhone(phone)
}

func smsCodeCooldownCacheKey(purpose emailCodePurpose, phone string) string {
	return "sms:" + string(purpose) + ":cooldown:" + utils.NormalizePhone(phone)
}

func verifySMSCode(purpose emailCodePurpose, phone, code string) bool {
	phone = utils.NormalizePhone(phone)
	code = strings.TrimSpace(code)
	if phone == "" || code == "" {
		return false
	}
	stored, ok := emailCodeCacheGet(smsCodeCacheKey(purpose, phone))
	if !ok || stored != code {
		return false
	}
	emailCodeCacheDelete(smsCodeCacheKey(purpose, phone))
	return true
}

func (h *Handlers) dispatchSMSVerificationCode(c *gin.Context, purpose emailCodePurpose, phone string, userID uint) bool {
	phone = utils.NormalizePhone(phone)
	cooldownKey := smsCodeCooldownCacheKey(purpose, phone)
	if emailCodeCooldownActive(cooldownKey) {
		response.FailI18n(c, i18n.KeyAuthEmailCodeCooldown, nil)
		return false
	}

	chans, err := listeners.EnabledSMSChannels(h.db)
	if err != nil {
		response.FailI18n(c, i18n.KeyAuthSMSUnavailable, nil)
		return false
	}
	sender, err := sms.NewMultiSender(chans, h.db, c.ClientIP(), sms.WithSMSLogUserID(userID))
	if err != nil {
		response.FailI18n(c, i18n.KeyAuthSMSUnavailable, nil)
		return false
	}

	code := generateNumericEmailCode()
	emailCodeCacheSet(smsCodeCacheKey(purpose, phone), code, emailCodeTTL)
	emailCodeCooldownSet(cooldownKey)

	tmpl := strings.TrimSpace(utils.GetEnv("SMS_LOGIN_TEMPLATE"))
	sign := strings.TrimSpace(utils.GetEnv("SMS_LOGIN_SIGN_NAME"))
	content := fmt.Sprintf("您的验证码是 %s，%d 分钟内有效。", code, int(emailCodeTTL/time.Minute))
	if emailCodeTTL < time.Minute {
		content = fmt.Sprintf("您的验证码是 %s，请尽快使用。", code)
	}
	msg := sms.Message{
		Content:  content,
		Template: tmpl,
		SignName: sign,
		Data:     map[string]string{"code": code},
	}
	if tmpl != "" {
		// Template providers: prefer template; keep content for content-mode failover channels.
		msg.Content = content
	}
	sendReq := sms.SendRequest{
		To:      []sms.PhoneNumber{{Number: phone}},
		Message: msg,
	}
	if err := sender.Send(context.Background(), sendReq); err != nil {
		emailCodeCacheDelete(smsCodeCacheKey(purpose, phone))
		response.FailI18n(c, i18n.KeyAuthSMSSendFailed, nil)
		return false
	}
	return true
}

func (h *Handlers) sendLoginSMSCode(c *gin.Context) {
	var req sendSMSCodeReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailI18n(c, i18n.KeyInvalidParams, err.Error())
		return
	}
	if err := captcha.ValidatePayload(req.CaptchaID, req.CaptchaType, req.CaptchaValue); err != nil {
		if err == captcha.ErrPayloadRequired {
			response.FailI18n(c, i18n.KeyValidationCaptchaRequired, nil)
		} else {
			response.FailI18n(c, i18n.KeyValidationCaptchaInvalid, nil)
		}
		return
	}
	phone := utils.NormalizePhone(req.Phone)
	if phone == "" {
		response.FailI18n(c, i18n.KeyInvalidParams, "phone")
		return
	}
	user, err := models.GetActiveTenantUserByPhoneGlobal(h.db, phone)
	if err != nil {
		response.FailI18n(c, i18n.KeyAuthPhoneNotRegistered, nil)
		return
	}
	if !h.dispatchSMSVerificationCode(c, emailCodeLogin, phone, user.ID) {
		return
	}
	response.SuccessI18n(c, i18n.KeyAuthSMSCodeSent, nil)
}
