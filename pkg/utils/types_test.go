// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

package utils_test

import (
	"errors"
	"testing"

	apperr "github.com/LingByte/SoulNexus/pkg/errors"
	"github.com/LingByte/SoulNexus/pkg/utils"
	utilnet "github.com/LingByte/SoulNexus/pkg/utils/common"
)

func TestErrReexportsMatchSource(t *testing.T) {
	pairs := []struct {
		name string
		got  error
		want error
	}{
		{"ErrUnauthorized", utils.ErrUnauthorized, apperr.ErrUnauthorized},
		{"ErrAttachmentNotExist", utils.ErrAttachmentNotExist, apperr.ErrAttachmentNotExist},
		{"ErrNotAttachmentOwner", utils.ErrNotAttachmentOwner, apperr.ErrNotAttachmentOwner},
		{"ErrQuotaExceeded", utils.ErrQuotaExceeded, apperr.ErrQuotaExceeded},
		{"ErrLLMCallFailed", utils.ErrLLMCallFailed, apperr.ErrLLMCallFailed},
		{"ErrEmptyPassword", utils.ErrEmptyPassword, apperr.ErrEmptyPassword},
		{"ErrEmptyEmail", utils.ErrEmptyEmail, apperr.ErrEmptyEmail},
		{"ErrSameEmail", utils.ErrSameEmail, apperr.ErrSameEmail},
		{"ErrEmailExists", utils.ErrEmailExists, apperr.ErrEmailExists},
		{"ErrUserNotExists", utils.ErrUserNotExists, apperr.ErrUserNotExists},
		{"ErrForbidden", utils.ErrForbidden, apperr.ErrForbidden},
		{"ErrUserNotAllowLogin", utils.ErrUserNotAllowLogin, apperr.ErrUserNotAllowLogin},
		{"ErrUserNotAllowSignup", utils.ErrUserNotAllowSignup, apperr.ErrUserNotAllowSignup},
		{"ErrNotActivated", utils.ErrNotActivated, apperr.ErrNotActivated},
		{"ErrTokenRequired", utils.ErrTokenRequired, apperr.ErrTokenRequired},
		{"ErrInvalidToken", utils.ErrInvalidToken, apperr.ErrInvalidToken},
		{"ErrBadToken", utils.ErrBadToken, apperr.ErrBadToken},
		{"ErrTokenExpired", utils.ErrTokenExpired, apperr.ErrTokenExpired},
		{"ErrEmailRequired", utils.ErrEmailRequired, apperr.ErrEmailRequired},
		{"ErrNotFound", utils.ErrNotFound, apperr.ErrNotFound},
		{"ErrNotChanged", utils.ErrNotChanged, apperr.ErrNotChanged},
		{"ErrInvalidView", utils.ErrInvalidView, apperr.ErrInvalidView},
		{"ErrOnlySuperUser", utils.ErrOnlySuperUser, apperr.ErrOnlySuperUser},
		{"ErrInvalidPrimaryKey", utils.ErrInvalidPrimaryKey, apperr.ErrInvalidPrimaryKey},
		{"ErrInvalidToolListFormat", utils.ErrInvalidToolListFormat, apperr.ErrInvalidToolListFormat},
		{"ErrInvalidToolFormat", utils.ErrInvalidToolFormat, apperr.ErrInvalidToolFormat},
		{"ErrToolNotFound", utils.ErrToolNotFound, apperr.ErrToolNotFound},
		{"ErrInvalidToolParams", utils.ErrInvalidToolParams, apperr.ErrInvalidToolParams},
		{"ErrParseJSONRPC", utils.ErrParseJSONRPC, apperr.ErrParseJSONRPC},
		{"ErrInvalidJSONRPCFormat", utils.ErrInvalidJSONRPCFormat, apperr.ErrInvalidJSONRPCFormat},
		{"ErrInvalidJSONRPCResponse", utils.ErrInvalidJSONRPCResponse, apperr.ErrInvalidJSONRPCResponse},
		{"ErrInvalidJSONRPCRequest", utils.ErrInvalidJSONRPCRequest, apperr.ErrInvalidJSONRPCRequest},
		{"ErrInvalidJSONRPCParams", utils.ErrInvalidJSONRPCParams, apperr.ErrInvalidJSONRPCParams},
		{"ErrInvalidResourceFormat", utils.ErrInvalidResourceFormat, apperr.ErrInvalidResourceFormat},
		{"ErrResourceNotFound", utils.ErrResourceNotFound, apperr.ErrResourceNotFound},
		{"ErrInvalidPromptFormat", utils.ErrInvalidPromptFormat, apperr.ErrInvalidPromptFormat},
		{"ErrPromptNotFound", utils.ErrPromptNotFound, apperr.ErrPromptNotFound},
		{"ErrEmptyToolName", utils.ErrEmptyToolName, apperr.ErrEmptyToolName},
		{"ErrToolAlreadyRegistered", utils.ErrToolAlreadyRegistered, apperr.ErrToolAlreadyRegistered},
		{"ErrToolExecutionFailed", utils.ErrToolExecutionFailed, apperr.ErrToolExecutionFailed},
		{"ErrEmptyResourceURI", utils.ErrEmptyResourceURI, apperr.ErrEmptyResourceURI},
		{"ErrEmptyPromptName", utils.ErrEmptyPromptName, apperr.ErrEmptyPromptName},
		{"ErrSessionAlreadyInitialized", utils.ErrSessionAlreadyInitialized, apperr.ErrSessionAlreadyInitialized},
		{"ErrSessionNotInitialized", utils.ErrSessionNotInitialized, apperr.ErrSessionNotInitialized},
		{"ErrInvalidParams", utils.ErrInvalidParams, apperr.ErrInvalidParams},
		{"ErrMissingParams", utils.ErrMissingParams, apperr.ErrMissingParams},
		{"ErrAlreadyInitialized", utils.ErrAlreadyInitialized, apperr.ErrAlreadyInitialized},
		{"ErrNotInitialized", utils.ErrNotInitialized, apperr.ErrNotInitialized},
		{"ErrInvalidServerURL", utils.ErrInvalidServerURL, apperr.ErrInvalidServerURL},
		{"ErrSSRFRedirectBlocked", utils.ErrSSRFRedirectBlocked, utilnet.ErrSSRFRedirectBlocked},
	}
	for _, tc := range pairs {
		t.Run(tc.name, func(t *testing.T) {
			if !errors.Is(tc.got, tc.want) {
				t.Fatalf("re-export mismatch: got=%v want=%v", tc.got, tc.want)
			}
		})
	}
}

func TestParseIDUsesInvalidPrimaryKey(t *testing.T) {
	_, err := utils.ParseID("0")
	if !errors.Is(err, utils.ErrInvalidPrimaryKey) {
		t.Fatalf("expected ErrInvalidPrimaryKey, got %v", err)
	}
}
