// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

package response

import (
	stderrors "errors"
	"net/http"
	"strings"

	"github.com/LingByte/SoulNexus/pkg/i18n"
	"github.com/LingByte/SoulNexus/pkg/logger"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// Response is the standard API JSON envelope.
type Response struct {
	Code    int         `json:"code"`
	Message string      `json:"msg"`
	Data    interface{} `json:"data"`
}

// Success responds with the legacy SoulNexus-compatible success envelope.
func Success(c *gin.Context, msg string, data interface{}) {
	c.JSON(http.StatusOK, gin.H{
		"code": ErrCodeSuccess,
		"msg":  msg,
		"data": data,
	})
}

// SuccessI18n responds with a localized success message for the request locale.
func SuccessI18n(c *gin.Context, key string, data interface{}, args ...any) {
	c.JSON(http.StatusOK, gin.H{
		"code": ErrCodeSuccess,
		"msg":  i18n.TGin(c, key, args...),
		"data": data,
	})
}

// WriteError renders any error using the unified AppError envelope.
func WriteError(c *gin.Context, err error) {
	Render(c, err)
}

// HandleError is an alias for WriteError.
func HandleError(c *gin.Context, err error) {
	WriteError(c, err)
}

// Fail responds with an internal AppError. Prefer WriteError with a typed code.
func Fail(c *gin.Context, msg string, data interface{}) {
	ae := New(CodeInternal, msg)
	if details := asDetails(data); details != nil {
		ae = ae.WithDetails(details)
	}
	Render(c, ae)
}

// FailWithCode responds with an AppError, optionally overriding HTTP status when
// errCode is an HTTP status (400-599). Legacy callers passing HTTP status as code
// are mapped to the closest AppError code.
func FailWithCode(c *gin.Context, errCode int, msg string, data interface{}) {
	ae := mapLegacyErrCode(errCode, msg)
	if details := asDetails(data); details != nil {
		ae = ae.WithDetails(details)
	}
	Render(c, ae)
}

// FailI18n responds with a localized AppError message.
func FailI18n(c *gin.Context, key string, data interface{}, args ...any) {
	ae := NewI18n(codeForI18nKey(key), key, args...)
	if details := asDetails(data); details != nil {
		ae = ae.WithDetails(details)
	}
	Render(c, ae)
}

// Result writes a custom HTTP response with explicit code/message/data.
func Result(context *gin.Context, httpStatus int, code int, msg string, data gin.H) {
	context.JSON(httpStatus, gin.H{
		"code": code,
		"msg":  msg,
		"data": data,
	})
}

// AbortWithStatus aborts the request with a bare HTTP status (no body).
func AbortWithStatus(c *gin.Context, httpStatus int) {
	c.AbortWithStatus(httpStatus)
}

type knownError struct {
	substr string
	code   Code
	msgKey string
}

var knownErrors = []knownError{
	{"username must be at least 2 characters long", CodeValidation, i18n.KeyValidationUsernameShort},
	{"username can only contain", CodeValidation, i18n.KeyValidationUsernameFormat},
	{"email has exists", CodeConflict, i18n.KeyTenantEmailExists},
	{"password must be at least 8 characters long", CodeValidation, i18n.KeyValidationPasswordShort},
	{"captcha is required", CodeValidation, i18n.KeyValidationCaptchaRequired},
	{"invalid captcha code", CodeValidation, i18n.KeyValidationCaptchaInvalid},
}

// AbortWithStatusJSON converts an arbitrary error into a proper JSON error response.
func AbortWithStatusJSON(c *gin.Context, httpStatus int, err error) {
	if err == nil {
		return
	}
	errorMsg := err.Error()

	for _, ke := range knownErrors {
		if strings.Contains(errorMsg, ke.substr) {
			Render(c, NewI18n(ke.code, ke.msgKey))
			return
		}
	}

	logger.ErrorCtx(c.Request.Context(), "internal server error",
		zap.String("path", c.Request.URL.Path),
		zap.String("method", c.Request.Method),
		zap.Error(err),
	)

	if httpStatus >= 500 {
		Render(c, New(CodeInternal, errorMsg).WithStatus(httpStatus))
		return
	}

	Render(c, NewI18n(CodeInternal, i18n.KeyInternalError).WithStatus(httpStatus))
}

func asDetails(data interface{}) map[string]any {
	if data == nil {
		return nil
	}
	if m, ok := data.(gin.H); ok && len(m) > 0 {
		return m
	}
	if m, ok := data.(map[string]any); ok && len(m) > 0 {
		return m
	}
	return nil
}

func mapLegacyErrCode(errCode int, msg string) *AppError {
	switch {
	case errCode == http.StatusServiceUnavailable:
		return New(CodeServiceUnavail, msg)
	case errCode == http.StatusBadRequest:
		return New(CodeBadRequest, msg)
	case errCode == http.StatusUnauthorized:
		return New(CodeUnauthorized, msg)
	case errCode == http.StatusForbidden:
		return New(CodeForbidden, msg)
	case errCode == http.StatusNotFound:
		return New(CodeNotFound, msg)
	case errCode >= 400 && errCode < 600:
		return New(CodeInternal, msg).WithStatus(errCode)
	default:
		return New(CodeInternal, msg)
	}
}

func appErrorFrom(err error) (*AppError, bool) {
	if err == nil {
		return nil, false
	}
	var ae *AppError
	if stderrors.As(err, &ae) {
		return ae, true
	}
	return nil, false
}

// WriteAppError renders a typed AppError without wrapping through From.
func WriteAppError(c *gin.Context, ae *AppError) {
	if ae == nil {
		return
	}
	Render(c, ae)
}

// IsAppError reports whether err is an *AppError.
func IsAppError(err error) bool {
	_, ok := appErrorFrom(err)
	return ok
}

func codeForI18nKey(key string) Code {
	switch key {
	case i18n.KeyAuthInvalidCredentials, i18n.KeyAuthInvalidEmailCode, i18n.KeyAuthInvalidTotp,
		i18n.KeyAuthNeedsTotp, i18n.KeyAuthNeedsDeviceVerify, i18n.KeyAuthInvalidVoiceprint, i18n.KeyAuthEmailNotRegistered,
		i18n.KeyAuthEmailAlreadyRegistered, i18n.KeyAuthEmailSameAsCurrent, i18n.KeyAuthMissingToken,
		i18n.KeyAuthInvalidToken, i18n.KeyPasswordWrong:
		return CodeAuthFailed
	case i18n.KeyAuthEmailCodeCooldown, i18n.KeyRateLimited:
		return CodeRateLimited
	case i18n.KeyInvalidParams, i18n.KeyInvalidBody, i18n.KeyValidationUsernameShort,
		i18n.KeyValidationUsernameFormat, i18n.KeyValidationPasswordShort,
		i18n.KeyValidationCaptchaRequired, i18n.KeyValidationCaptchaInvalid:
		return CodeValidation
	case i18n.KeyUnauthorized:
		return CodeUnauthorized
	case i18n.KeyForbidden, i18n.KeyPermInsufficient, i18n.KeyPermInsufficientCode,
		i18n.KeyPermInsufficientCredential, i18n.KeyPermNeedTenantUser,
		i18n.KeyPermNeedTenantContext, i18n.KeyPermPlatformNoTenantRBAC,
		i18n.KeyPermInvalidCredential:
		return CodeForbidden
	case i18n.KeyNotFound, i18n.KeyTenantNotFound:
		return CodeNotFound
	case i18n.KeyConflict, i18n.KeyTenantEmailExists, i18n.KeyTotpAlreadyOn:
		return CodeConflict
	case i18n.KeyTenantRegisterDisabled, i18n.KeyTenantSuspended, i18n.KeyTenantUserUnavailable,
		i18n.KeyAccountDeletionPending, i18n.KeyAccountDeletionAlreadyPending,
		i18n.KeyAccountDeletionNotPending:
		return CodeBadRequest
	case i18n.KeyAuthJWTNotReady, i18n.KeyAuthEmailCodeSendFailed, i18n.KeyTenantSignTokenFailed,
		i18n.KeyServiceUnavailable, i18n.KeyUpstreamTimeout, i18n.KeyDatabaseUnavailable:
		return CodeServiceUnavail
	default:
		return CodeInternal
	}
}
