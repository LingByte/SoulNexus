package server

// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

import (
	"context"
	"errors"

	"github.com/LingByte/SoulNexus/internal/models/auth"
	"gorm.io/gorm"
)

var errInvalidAPICredential = errors.New("invalid api credentials")

// resolveCredential loads API key credentials from the local database.
func (h *Handlers) resolveCredential(_ context.Context, apiKey, apiSecret string) (*auth.UserCredential, error) {
	if apiKey == "" || apiSecret == "" {
		return nil, errInvalidAPICredential
	}
	cred, err := auth.GetUserCredentialByApiSecretAndApiKey(h.db, apiKey, apiSecret)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errInvalidAPICredential
		}
		return nil, err
	}
	if cred == nil {
		return nil, errInvalidAPICredential
	}
	return cred, nil
}

// resolveCredentialOwner resolves credentials and the owning active user.
func (h *Handlers) resolveCredentialOwner(ctx context.Context, apiKey, apiSecret string) (*auth.User, *auth.UserCredential, error) {
	cred, err := h.resolveCredential(ctx, apiKey, apiSecret)
	if err != nil {
		return nil, nil, err
	}
	user, err := auth.GetUserByID(h.db, cred.CreatedBy)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil, errInvalidAPICredential
		}
		return nil, nil, err
	}
	if user == nil || user.Status != auth.UserStatusActive {
		return nil, nil, errInvalidAPICredential
	}
	return user, cred, nil
}
