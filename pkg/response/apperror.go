// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

package response

import (
	"errors"
	stderrors "errors"
	"fmt"
	"net/http"

	apperr "github.com/LingByte/SoulNexus/pkg/errors"
	"github.com/LingByte/SoulNexus/pkg/i18n"
	"github.com/gin-gonic/gin"
)

// Code is a stable business error identifier. Clients branch on it; do not reuse semantics.
type Code string

// Numeric error codes for API responses.
// 1000-1999: Business errors (client errors)
// 2000-2999: System errors (server errors)
const (
	ErrCodeSuccess = 200

	// Business errors (1000-1999)
	ErrCodeInvalidParams    = 1000
	ErrCodeNotFound         = 1001
	ErrCodeUnauthorized     = 1002
	ErrCodeForbidden        = 1003
	ErrCodeConflict         = 1004
	ErrCodeRateLimited      = 1005
	ErrCodeTenantMismatch   = 1006
	ErrCodeQuotaExceeded    = 1007
	ErrCodeUpstreamTimeout  = 1008
	ErrCodeServiceUnavail   = 1009
	ErrCodeDuplicate        = 1010
	ErrCodeValidationFailed = 1011

	// Auth errors (1100-1199)
	ErrCodeInvalidCredentials = 1100
	ErrCodeNeedsTotp          = 1101
	ErrCodeInvalidTotp        = 1102
	ErrCodeJWTNotReady        = 1103
	ErrCodeMissingToken       = 1104
	ErrCodeInvalidToken       = 1105

	// Tenant errors (1200-1299)
	ErrCodeRegisterDisabled = 1200
	ErrCodeEmailExists      = 1201
	ErrCodeInvalidEmail     = 1202
	ErrCodeTenantNotFound   = 1203
	ErrCodeTenantSuspended  = 1204
	ErrCodeUserUnavailable  = 1205
	ErrCodeSignTokenFailed  = 1206

	// Permission errors (1300-1399)
	ErrCodePermInsufficient      = 1300
	ErrCodePermInsufficientCred  = 1301
	ErrCodePermNeedTenantUser    = 1302
	ErrCodePermNeedTenantContext = 1303
	ErrCodePermPlatformNoRBAC    = 1304
	ErrCodePermInvalidCred       = 1305

	// Credential errors (1400-1499)
	ErrCodeCredPermRequired = 1400
	ErrCodeCredPermEmpty    = 1401
	ErrCodeCredAllowIPReq   = 1402
	ErrCodeCredAllowIPEmpty = 1403
	ErrCodeCredInvalidPerm  = 1404
	ErrCodeCredNameEmpty    = 1405
	ErrCodeCredNoticeSecret = 1406

	// Organization errors (1500-1599)
	ErrCodeOrgInvalidPermID  = 1500
	ErrCodeOrgInvalidRoleID  = 1501
	ErrCodeOrgAdminRoleFixed = 1502

	// Account errors (1600-1699)
	ErrCodePasswordWrong   = 1600
	ErrCodePasswordChanged = 1601
	ErrCodeTotpAlreadyOn   = 1602
	ErrCodeTotpNotOn       = 1603
	ErrCodeTotpInvalidCode = 1604
	ErrCodeTotpEnabled     = 1605
	ErrCodeTotpDisabled    = 1606
	ErrCodeTotpSetupFirst  = 1607

	// System errors (2000-2999)
	ErrCodeInternal        = 2000
	ErrCodeDatabaseUnavail = 2001
	ErrCodeProviderError   = 2002
)

const (
	CodeBadRequest        Code = "BAD_REQUEST"
	CodeUnauthorized      Code = "UNAUTHORIZED"
	CodeForbidden         Code = "FORBIDDEN"
	CodeNotFound          Code = "NOT_FOUND"
	CodeConflict          Code = "CONFLICT"
	CodeRateLimited       Code = "RATE_LIMITED"
	CodeInternal          Code = "INTERNAL"
	CodeServiceUnavail    Code = "SERVICE_UNAVAILABLE"
	CodeValidation        Code = "VALIDATION_FAILED"
	CodeTenantMismatch    Code = "TENANT_MISMATCH"
	CodeAuthFailed        Code = "AUTH_FAILED"
	CodeCredentialInvalid Code = "CREDENTIAL_INVALID"
	CodeQuotaExceeded     Code = "QUOTA_EXCEEDED"
	CodeProviderError     Code = "PROVIDER_ERROR"
	CodeUpstreamTimeout   Code = "UPSTREAM_TIMEOUT"
	CodeDuplicate         Code = "DUPLICATE"
)

// AppError is the unified business error shape.
type AppError struct {
	Code       Code
	MsgKey     string
	MsgArgs    []any
	Message    string
	HTTPStatus int
	Cause      error
	Details    map[string]any
}

func (e *AppError) Error() string {
	if e == nil {
		return ""
	}
	msg := e.Message
	if msg == "" && e.MsgKey != "" {
		msg = i18n.T(i18n.LocaleZhCN, e.MsgKey, e.MsgArgs...)
	}
	if e.Cause != nil {
		return fmt.Sprintf("[%s] %s: %v", e.Code, msg, e.Cause)
	}
	return fmt.Sprintf("[%s] %s", e.Code, msg)
}

func (e *AppError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Cause
}

func (e *AppError) Is(target error) bool {
	if e == nil {
		return target == nil
	}
	var t *AppError
	if errors.As(target, &t) {
		return e.Code == t.Code
	}
	return false
}

// Err creates an AppError from a Code only.
// User-facing text is resolved at render time via I18nKeyFor(code).
// Use this for standard error codes where the default i18n message suffices.
func Err(code Code) *AppError {
	return &AppError{Code: code, HTTPStatus: HTTPStatusFor(code)}
}

// New constructs an AppError with an explicit message (non-i18n path).
// Prefer Err(code) or NewI18n(code, msgKey) when possible.
func New(code Code, message string) *AppError {
	return &AppError{Code: code, Message: message, HTTPStatus: HTTPStatusFor(code)}
}

// Newf is New with a formatted message.
func Newf(code Code, format string, args ...any) *AppError {
	return New(code, fmt.Sprintf(format, args...))
}

// NewI18n constructs an AppError whose user-facing text is resolved at render time.
func NewI18n(code Code, msgKey string, args ...any) *AppError {
	return &AppError{
		Code:       code,
		MsgKey:     msgKey,
		MsgArgs:    args,
		HTTPStatus: HTTPStatusFor(code),
	}
}

// Wrap attaches a cause to an AppError.
func Wrap(code Code, message string, cause error) *AppError {
	return &AppError{
		Code:       code,
		Message:    message,
		HTTPStatus: HTTPStatusFor(code),
		Cause:      cause,
	}
}

// WrapErr creates an AppError from a code and underlying error.
// User-facing text comes from I18nKeyFor(code); the cause is preserved.
func WrapErr(code Code, cause error) *AppError {
	return &AppError{
		Code:       code,
		HTTPStatus: HTTPStatusFor(code),
		Cause:      cause,
	}
}

// WrapI18n attaches a cause; user text comes from msgKey at render time.
func WrapI18n(code Code, msgKey string, cause error, args ...any) *AppError {
	return &AppError{
		Code:       code,
		MsgKey:     msgKey,
		MsgArgs:    args,
		HTTPStatus: HTTPStatusFor(code),
		Cause:      cause,
	}
}

func (e *AppError) WithStatus(status int) *AppError {
	if e == nil {
		return nil
	}
	e.HTTPStatus = status
	return e
}

func (e *AppError) WithDetails(d map[string]any) *AppError {
	if e == nil {
		return nil
	}
	e.Details = d
	return e
}

func (e *AppError) WithCause(cause error) *AppError {
	if e == nil {
		return nil
	}
	e.Cause = cause
	return e
}

// From converts any error to *AppError; unknown errors become CodeInternal.
func From(err error) *AppError {
	if err == nil {
		return nil
	}
	var ae *AppError
	if errors.As(err, &ae) {
		return ae
	}
	if mapped, ok := fromAPIErr(err); ok {
		return mapped
	}
	return Wrap(CodeInternal, "internal error", err)
}

// I18nKeyFor returns the default i18n key for a Code.
func I18nKeyFor(code Code) string {
	switch code {
	case CodeBadRequest, CodeValidation:
		return i18n.KeyInvalidParams
	case CodeUnauthorized, CodeAuthFailed, CodeCredentialInvalid:
		return i18n.KeyUnauthorized
	case CodeForbidden:
		return i18n.KeyForbidden
	case CodeNotFound:
		return i18n.KeyNotFound
	case CodeConflict, CodeDuplicate:
		return i18n.KeyConflict
	case CodeRateLimited:
		return i18n.KeyRateLimited
	case CodeTenantMismatch:
		return i18n.KeyTenantMismatch
	case CodeQuotaExceeded:
		return i18n.KeyQuotaExceeded
	case CodeUpstreamTimeout:
		return i18n.KeyUpstreamTimeout
	case CodeServiceUnavail, CodeProviderError:
		return i18n.KeyServiceUnavailable
	case CodeInternal:
		return i18n.KeyInternalError
	default:
		return i18n.KeyInternalError
	}
}

func MessageForGin(c *gin.Context, ae *AppError) string {
	if ae == nil {
		return ""
	}
	if ae.MsgKey != "" {
		return i18n.TGin(c, ae.MsgKey, ae.MsgArgs...)
	}
	if ae.Message != "" {
		return ae.Message
	}
	return i18n.TGin(c, I18nKeyFor(ae.Code))
}

// Envelope builds the standard JSON body for API error responses.
func Envelope(c *gin.Context, ae *AppError) gin.H {
	if ae == nil {
		return gin.H{"code": ErrCodeInternal, "msg": "", "error": string(CodeInternal), "data": nil}
	}
	return gin.H{
		"code":  ErrCodeFor(ae.Code),
		"msg":   MessageForGin(c, ae),
		"error": string(ae.Code),
		"data":  ae.Details,
	}
}

// HTTPStatusOf returns the HTTP status for an AppError, with fallback mapping.
func HTTPStatusOf(ae *AppError) int {
	if ae == nil {
		return http.StatusInternalServerError
	}
	if ae.HTTPStatus != 0 {
		return ae.HTTPStatus
	}
	return HTTPStatusFor(ae.Code)
}

func HTTPStatusFor(code Code) int {
	switch code {
	case CodeBadRequest, CodeValidation:
		return http.StatusBadRequest
	case CodeUnauthorized, CodeAuthFailed, CodeCredentialInvalid:
		return http.StatusUnauthorized
	case CodeForbidden, CodeTenantMismatch:
		return http.StatusForbidden
	case CodeNotFound:
		return http.StatusNotFound
	case CodeConflict, CodeDuplicate:
		return http.StatusConflict
	case CodeRateLimited:
		return http.StatusTooManyRequests
	case CodeQuotaExceeded:
		return http.StatusPaymentRequired
	case CodeUpstreamTimeout:
		return http.StatusGatewayTimeout
	case CodeServiceUnavail, CodeProviderError:
		return http.StatusServiceUnavailable
	case CodeInternal:
		fallthrough
	default:
		return http.StatusInternalServerError
	}
}

// ErrCodeFor returns the numeric error code for a string Code.
func ErrCodeFor(code Code) int {
	switch code {
	case CodeBadRequest, CodeValidation:
		return ErrCodeInvalidParams
	case CodeUnauthorized, CodeAuthFailed, CodeCredentialInvalid:
		return ErrCodeUnauthorized
	case CodeForbidden, CodeTenantMismatch:
		return ErrCodeForbidden
	case CodeNotFound:
		return ErrCodeNotFound
	case CodeConflict, CodeDuplicate:
		return ErrCodeConflict
	case CodeRateLimited:
		return ErrCodeRateLimited
	case CodeQuotaExceeded:
		return ErrCodeQuotaExceeded
	case CodeUpstreamTimeout:
		return ErrCodeUpstreamTimeout
	case CodeServiceUnavail, CodeProviderError:
		return ErrCodeServiceUnavail
	case CodeInternal:
		return ErrCodeInternal
	default:
		return ErrCodeInternal
	}
}

// Render writes the AppError JSON envelope and aborts the request.
func Render(c *gin.Context, err error) {
	if err == nil {
		return
	}
	ae := From(err)
	if ae == nil {
		return
	}
	c.AbortWithStatusJSON(HTTPStatusOf(ae), Envelope(c, ae))
}

// CredentialRouteAppError maps AK/SK route validation errors to API errors.
func CredentialRouteAppError(err error) *AppError {
	if err == nil {
		return nil
	}
	switch {
	case stderrors.Is(err, apperr.ErrAKSKRouteIDsRequired):
		return NewI18n(CodeBadRequest, i18n.KeyAllowedRouteIDsRequired)
	case stderrors.Is(err, apperr.ErrAKSKSystemRoutesClosed):
		return NewI18n(CodeBadRequest, i18n.KeyRouteNotOpenPlatform)
	case stderrors.Is(err, apperr.ErrAKSKRouteIDNotOpen):
		return NewI18n(CodeBadRequest, i18n.KeyRouteIDNotOpenPlatform)
	default:
		return From(err)
	}
}
