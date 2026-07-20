// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

package response

import (
	stderrors "errors"
	"net/http"

	apperr "github.com/LingByte/SoulNexus/pkg/errors"
	"github.com/LingByte/SoulNexus/pkg/i18n"
	utilnet "github.com/LingByte/SoulNexus/pkg/utils/common"
)

type sentinelMapping struct {
	err    error
	code   Code
	msgKey string
	status int
}

var apiErrMappings = []sentinelMapping{
	{apperr.ErrQuotaExceeded, CodeQuotaExceeded, i18n.KeyQuotaExceeded, 0},
	{apperr.ErrLLMCallFailed, CodeProviderError, i18n.KeyServiceUnavailable, 0},
	{apperr.ErrEmptyPassword, CodeValidation, i18n.KeyValidationPasswordShort, 0},
	{apperr.ErrEmptyEmail, CodeValidation, i18n.KeyInvalidParams, 0},
	{apperr.ErrSameEmail, CodeAuthFailed, i18n.KeyAuthEmailSameAsCurrent, 0},
	{apperr.ErrEmailExists, CodeConflict, i18n.KeyTenantEmailExists, 0},
	{apperr.ErrUserNotExists, CodeNotFound, i18n.KeyAuthEmailNotRegistered, 0},
	{apperr.ErrForbidden, CodeForbidden, i18n.KeyForbidden, 0},
	{apperr.ErrUserNotAllowLogin, CodeForbidden, i18n.KeyTenantUserUnavailable, 0},
	{apperr.ErrUserNotAllowSignup, CodeBadRequest, i18n.KeyTenantRegisterDisabled, 0},
	{apperr.ErrNotActivated, CodeAuthFailed, i18n.KeyAuthInvalidCredentials, 0},
	{apperr.ErrTokenRequired, CodeAuthFailed, i18n.KeyAuthMissingToken, 0},
	{apperr.ErrInvalidToken, CodeAuthFailed, i18n.KeyAuthInvalidToken, 0},
	{apperr.ErrBadToken, CodeAuthFailed, i18n.KeyAuthInvalidToken, 0},
	{apperr.ErrTokenExpired, CodeAuthFailed, i18n.KeyAuthInvalidToken, 0},
	{apperr.ErrEmailRequired, CodeValidation, i18n.KeyInvalidParams, 0},
	{apperr.ErrNotFound, CodeNotFound, i18n.KeyNotFound, 0},
	{apperr.ErrNotChanged, CodeBadRequest, i18n.KeyConflict, 0},
	{apperr.ErrInvalidView, CodeBadRequest, i18n.KeyInvalidParams, 0},
	{apperr.ErrOnlySuperUser, CodeForbidden, i18n.KeyForbidden, 0},
	{apperr.ErrInvalidPrimaryKey, CodeValidation, i18n.KeyInvalidParams, 0},
	{apperr.ErrInvalidToolListFormat, CodeBadRequest, i18n.KeyInvalidParams, 0},
	{apperr.ErrInvalidToolFormat, CodeBadRequest, i18n.KeyInvalidParams, 0},
	{apperr.ErrToolNotFound, CodeNotFound, i18n.KeyNotFound, 0},
	{apperr.ErrInvalidToolParams, CodeValidation, i18n.KeyInvalidParams, 0},
	{apperr.ErrParseJSONRPC, CodeBadRequest, i18n.KeyInvalidBody, 0},
	{apperr.ErrInvalidJSONRPCFormat, CodeBadRequest, i18n.KeyInvalidBody, 0},
	{apperr.ErrInvalidJSONRPCResponse, CodeBadRequest, i18n.KeyInvalidBody, 0},
	{apperr.ErrInvalidJSONRPCRequest, CodeBadRequest, i18n.KeyInvalidBody, 0},
	{apperr.ErrInvalidJSONRPCParams, CodeValidation, i18n.KeyInvalidParams, 0},
	{apperr.ErrInvalidResourceFormat, CodeBadRequest, i18n.KeyInvalidParams, 0},
	{apperr.ErrResourceNotFound, CodeNotFound, i18n.KeyNotFound, 0},
	{apperr.ErrInvalidPromptFormat, CodeBadRequest, i18n.KeyInvalidParams, 0},
	{apperr.ErrPromptNotFound, CodeNotFound, i18n.KeyNotFound, 0},
	{apperr.ErrEmptyToolName, CodeValidation, i18n.KeyInvalidParams, 0},
	{apperr.ErrToolAlreadyRegistered, CodeConflict, i18n.KeyConflict, 0},
	{apperr.ErrToolExecutionFailed, CodeProviderError, i18n.KeyServiceUnavailable, 0},
	{apperr.ErrEmptyResourceURI, CodeValidation, i18n.KeyInvalidParams, 0},
	{apperr.ErrEmptyPromptName, CodeValidation, i18n.KeyInvalidParams, 0},
	{apperr.ErrSessionAlreadyInitialized, CodeConflict, i18n.KeyConflict, 0},
	{apperr.ErrSessionNotInitialized, CodeBadRequest, i18n.KeyServiceUnavailable, 0},
	{apperr.ErrInvalidParams, CodeValidation, i18n.KeyInvalidParams, 0},
	{apperr.ErrMissingParams, CodeValidation, i18n.KeyInvalidParams, 0},
	{apperr.ErrAlreadyInitialized, CodeConflict, i18n.KeyConflict, 0},
	{apperr.ErrNotInitialized, CodeBadRequest, i18n.KeyServiceUnavailable, 0},
	{apperr.ErrInvalidServerURL, CodeValidation, i18n.KeyInvalidParams, 0},
}

func fromAPIErr(err error) (*AppError, bool) {
	if err == nil {
		return nil, false
	}

	var ue *apperr.Error
	if stderrors.As(err, &ue) {
		code := httpStatusToCode(ue.Code)
		return New(code, ue.Message).WithStatus(ue.Code), true
	}

	for _, m := range apiErrMappings {
		if stderrors.Is(err, m.err) {
			ae := NewI18n(m.code, m.msgKey)
			if m.status != 0 {
				ae = ae.WithStatus(m.status)
			}
			ae = ae.WithCause(err)
			return ae, true
		}
	}

	if stderrors.Is(err, utilnet.ErrSSRFRedirectBlocked) {
		return WrapErr(CodeBadRequest, err).WithStatus(http.StatusBadRequest), true
	}

	return nil, false
}

func httpStatusToCode(status int) Code {
	switch status {
	case http.StatusBadRequest:
		return CodeBadRequest
	case http.StatusUnauthorized:
		return CodeUnauthorized
	case http.StatusForbidden:
		return CodeForbidden
	case http.StatusNotFound:
		return CodeNotFound
	case http.StatusConflict:
		return CodeConflict
	case http.StatusTooManyRequests:
		return CodeRateLimited
	case http.StatusPaymentRequired:
		return CodeQuotaExceeded
	case http.StatusGatewayTimeout:
		return CodeUpstreamTimeout
	case http.StatusServiceUnavailable:
		return CodeServiceUnavail
	default:
		return CodeInternal
	}
}
