package clients

// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

import (
	"context"
	"errors"

	authmodel "github.com/LingByte/SoulNexus/internal/models/auth"
	grpcpkg "github.com/LingByte/SoulNexus/internal/grpc"
)

// NewAPIKeyUserResolver returns an auth.APIKeyUserResolver backed by the auth gRPC service.
func (b *Bundle) NewAPIKeyUserResolver() authmodel.APIKeyUserResolver {
	if b == nil || b.Auth == nil {
		return nil
	}
	auth := b.Auth
	return func(ctx context.Context, apiKey, apiSecret string) (*authmodel.User, error) {
		cred, err := auth.ResolveCredential(ctx, apiKey, apiSecret)
		if err != nil {
			if grpcpkg.IsNotFound(err) {
				return nil, errors.New("invalid api credentials")
			}
			return nil, err
		}
		if cred == nil || cred.CreatedBy == 0 {
			return nil, errors.New("invalid api credentials")
		}
		user, err := auth.GetUser(ctx, cred.CreatedBy)
		if err != nil {
			if grpcpkg.IsNotFound(err) {
				return nil, errors.New("invalid api credentials")
			}
			return nil, err
		}
		if user == nil || user.Status != authmodel.UserStatusActive {
			return nil, errors.New("invalid api credentials")
		}
		return user, nil
	}
}

// InstallAPIKeyUserResolver registers the gRPC resolver on the auth package (cmd/server).
func (b *Bundle) InstallAPIKeyUserResolver() {
	authmodel.SetAPIKeyUserResolver(b.NewAPIKeyUserResolver())
}
