package clients

// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

import (
	"context"
	"fmt"

	authclient "github.com/LingByte/SoulNexus/internal/grpc/auth/client"
)

// Bundle holds outbound gRPC clients for internal microservices.
// Add new fields here as more services are split out (e.g. Billing, Notify).
type Bundle struct {
	Auth authclient.API
}

// Close releases all client connections.
func (b *Bundle) Close() error {
	if b == nil || b.Auth == nil {
		return nil
	}
	return b.Auth.Close()
}

// DialAuth connects to the auth user/credential gRPC service.
func DialAuth(ctx context.Context, addr string) (authclient.API, error) {
	if addr == "" {
		return nil, fmt.Errorf("grpc clients: auth service address is empty")
	}
	return authclient.Dial(ctx, addr)
}
