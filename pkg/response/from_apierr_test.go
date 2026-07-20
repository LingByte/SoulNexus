// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

package response

import (
	stderrors "errors"
	"net/http"
	"testing"

	apperr "github.com/LingByte/SoulNexus/pkg/errors"
	utilnet "github.com/LingByte/SoulNexus/pkg/utils/common"
	"github.com/LingByte/SoulNexus/pkg/utils/validate"
)

func TestFrom_APIErrSentinels(t *testing.T) {
	for _, sentinel := range apperr.SentinelErrors() {
		t.Run(sentinel.Error(), func(t *testing.T) {
			ae := From(sentinel)
			if ae == nil {
				t.Fatal("From returned nil")
			}
			if ae.Code == CodeInternal {
				t.Fatalf("unexpected CodeInternal for %v", sentinel)
			}
			if HTTPStatusOf(ae) == http.StatusInternalServerError {
				t.Fatalf("unexpected HTTP 500 for %v", ae)
			}
		})
	}
}

func TestFrom_PkgErrors(t *testing.T) {
	cases := []error{
		apperr.ErrUnauthorized,
		apperr.ErrAttachmentNotExist,
		apperr.ErrNotAttachmentOwner,
		apperr.ErrTokenRequired,
		apperr.ErrInvalidToken,
		apperr.ErrQuotaExceeded,
		apperr.ErrInvalidPrimaryKey,
		utilnet.ErrSSRFRedirectBlocked,
	}
	for _, err := range cases {
		t.Run(err.Error(), func(t *testing.T) {
			ae := From(err)
			if ae == nil || ae.Code == CodeInternal {
				t.Fatalf("From(%v) = %#v", err, ae)
			}
		})
	}
}

func TestFrom_TypedAPIError(t *testing.T) {
	ae := From(apperr.ErrAttachmentNotExist)
	if ae == nil {
		t.Fatal("nil AppError")
	}
	if HTTPStatusOf(ae) != http.StatusNotFound {
		t.Fatalf("status=%d want 404", HTTPStatusOf(ae))
	}
}

func TestFrom_SSRFBlocked(t *testing.T) {
	wrapped := stderrors.Join(utilnet.ErrSSRFRedirectBlocked, stderrors.New("detail"))
	ae := From(wrapped)
	if ae == nil || ae.Code != CodeBadRequest {
		t.Fatalf("unexpected mapping: %#v", ae)
	}
}

func TestFrom_ParseIDWrapped(t *testing.T) {
	_, err := validate.ParseID("abc")
	if err == nil {
		t.Fatal("expected error")
	}
	ae := From(err)
	if ae == nil || ae.Code != CodeValidation {
		t.Fatalf("unexpected mapping: %#v", ae)
	}
}
