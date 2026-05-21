package client

// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	grpcpkg "github.com/LingByte/SoulNexus/internal/grpc"
	authgrpc "github.com/LingByte/SoulNexus/internal/grpc/auth"
	authv1 "github.com/LingByte/SoulNexus/internal/grpc/auth/pb/auth/v1"
	authmodel "github.com/LingByte/SoulNexus/internal/models/auth"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// API is the server-side facade for auth-domain data over gRPC.
type API interface {
	GetUser(ctx context.Context, id uint) (*authmodel.User, error)
	GetUserByEmail(ctx context.Context, email string) (*authmodel.User, error)
	ResolveCredential(ctx context.Context, apiKey, apiSecret string) (*authmodel.UserCredential, error)
	ListUserCredentials(ctx context.Context, userID uint) ([]*authmodel.UserCredential, error)
	GetCredential(ctx context.Context, id uint) (*authmodel.UserCredential, error)
	DeleteCredential(ctx context.Context, id uint) error
	DeleteUserCredentialForUser(ctx context.Context, userID, credentialID uint) error
	MarkCredentialUsed(ctx context.Context, credentialID uint) error
	AdminListCredentials(ctx context.Context, page, pageSize int, search, status string, userID uint) ([]*authmodel.UserCredential, int64, error)
	UpdateCredentialStatus(ctx context.Context, req *authv1.UpdateCredentialStatusRequest) (*authmodel.UserCredential, error)
	CreateUserCredential(ctx context.Context, userID uint, body *authmodel.UserCredentialRequest) (*authmodel.UserCredential, error)
	Close() error
}

type grpcClient struct {
	conn   *grpc.ClientConn
	stub   authv1.UserAuthServiceClient
	dialTO time.Duration
}

// Dial connects to the auth user gRPC service (e.g. localhost:7075).
func Dial(ctx context.Context, target string) (API, error) {
	if target == "" {
		return nil, fmt.Errorf("grpc auth: empty target")
	}
	dctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	conn, err := grpc.DialContext(dctx, target,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithBlock(),
	)
	if err != nil {
		return nil, fmt.Errorf("grpc auth dial %s: %w", target, err)
	}
	return &grpcClient{
		conn:   conn,
		stub:   authv1.NewUserAuthServiceClient(conn),
		dialTO: 15 * time.Second,
	}, nil
}

func (c *grpcClient) rpcCtx(ctx context.Context) (context.Context, context.CancelFunc) {
	if ctx == nil {
		ctx = context.Background()
	}
	if _, ok := ctx.Deadline(); ok {
		return ctx, func() {}
	}
	return context.WithTimeout(ctx, c.dialTO)
}

func (c *grpcClient) Close() error {
	if c.conn != nil {
		return c.conn.Close()
	}
	return nil
}

func (c *grpcClient) GetUser(ctx context.Context, id uint) (*authmodel.User, error) {
	rpcCtx, cancel := c.rpcCtx(ctx)
	defer cancel()
	resp, err := c.stub.GetUser(rpcCtx, &authv1.GetUserRequest{Id: uint64(id)})
	if err != nil {
		return nil, unwrap(err)
	}
	return authgrpc.UserFromProto(resp), nil
}

func (c *grpcClient) GetUserByEmail(ctx context.Context, email string) (*authmodel.User, error) {
	rpcCtx, cancel := c.rpcCtx(ctx)
	defer cancel()
	resp, err := c.stub.GetUserByEmail(rpcCtx, &authv1.GetUserByEmailRequest{Email: email})
	if err != nil {
		return nil, unwrap(err)
	}
	return authgrpc.UserFromProto(resp), nil
}

func (c *grpcClient) ResolveCredential(ctx context.Context, apiKey, apiSecret string) (*authmodel.UserCredential, error) {
	rpcCtx, cancel := c.rpcCtx(ctx)
	defer cancel()
	resp, err := c.stub.ResolveCredential(rpcCtx, &authv1.ResolveCredentialRequest{
		ApiKey:    apiKey,
		ApiSecret: apiSecret,
	})
	if err != nil {
		return nil, unwrap(err)
	}
	return authgrpc.CredentialFromProto(resp)
}

func (c *grpcClient) ListUserCredentials(ctx context.Context, userID uint) ([]*authmodel.UserCredential, error) {
	rpcCtx, cancel := c.rpcCtx(ctx)
	defer cancel()
	resp, err := c.stub.ListUserCredentials(rpcCtx, &authv1.ListUserCredentialsRequest{UserId: uint64(userID)})
	if err != nil {
		return nil, unwrap(err)
	}
	out := make([]*authmodel.UserCredential, 0, len(resp.Credentials))
	for _, p := range resp.Credentials {
		cred, err := authgrpc.CredentialFromProto(p)
		if err != nil {
			return nil, err
		}
		out = append(out, cred)
	}
	return out, nil
}

func (c *grpcClient) GetCredential(ctx context.Context, id uint) (*authmodel.UserCredential, error) {
	rpcCtx, cancel := c.rpcCtx(ctx)
	defer cancel()
	resp, err := c.stub.GetCredential(rpcCtx, &authv1.GetCredentialRequest{Id: uint64(id)})
	if err != nil {
		return nil, unwrap(err)
	}
	return authgrpc.CredentialFromProto(resp)
}

func (c *grpcClient) DeleteCredential(ctx context.Context, id uint) error {
	rpcCtx, cancel := c.rpcCtx(ctx)
	defer cancel()
	_, err := c.stub.DeleteCredential(rpcCtx, &authv1.DeleteCredentialRequest{Id: uint64(id)})
	return unwrap(err)
}

func (c *grpcClient) MarkCredentialUsed(ctx context.Context, credentialID uint) error {
	rpcCtx, cancel := c.rpcCtx(ctx)
	defer cancel()
	_, err := c.stub.MarkCredentialUsed(rpcCtx, &authv1.MarkCredentialUsedRequest{CredentialId: uint64(credentialID)})
	return unwrap(err)
}

func (c *grpcClient) DeleteUserCredentialForUser(ctx context.Context, userID, credentialID uint) error {
	rpcCtx, cancel := c.rpcCtx(ctx)
	defer cancel()
	_, err := c.stub.DeleteUserCredentialForUser(rpcCtx, &authv1.DeleteUserCredentialForUserRequest{
		UserId:       uint64(userID),
		CredentialId: uint64(credentialID),
	})
	return unwrap(err)
}

func (c *grpcClient) AdminListCredentials(ctx context.Context, page, pageSize int, search, status string, userID uint) ([]*authmodel.UserCredential, int64, error) {
	rpcCtx, cancel := c.rpcCtx(ctx)
	defer cancel()
	resp, err := c.stub.AdminListCredentials(rpcCtx, &authv1.AdminListCredentialsRequest{
		Page:     int32(page),
		PageSize: int32(pageSize),
		Search:   search,
		Status:   status,
		UserId:   uint64(userID),
	})
	if err != nil {
		return nil, 0, unwrap(err)
	}
	out := make([]*authmodel.UserCredential, 0, len(resp.Credentials))
	for _, p := range resp.Credentials {
		cred, err := authgrpc.CredentialFromProto(p)
		if err != nil {
			return nil, 0, err
		}
		out = append(out, cred)
	}
	return out, resp.Total, nil
}

func (c *grpcClient) UpdateCredentialStatus(ctx context.Context, req *authv1.UpdateCredentialStatusRequest) (*authmodel.UserCredential, error) {
	rpcCtx, cancel := c.rpcCtx(ctx)
	defer cancel()
	resp, err := c.stub.UpdateCredentialStatus(rpcCtx, req)
	if err != nil {
		return nil, unwrap(err)
	}
	return authgrpc.CredentialFromProto(resp)
}

func (c *grpcClient) CreateUserCredential(ctx context.Context, userID uint, body *authmodel.UserCredentialRequest) (*authmodel.UserCredential, error) {
	raw, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}
	rpcCtx, cancel := c.rpcCtx(ctx)
	defer cancel()
	resp, err := c.stub.CreateUserCredential(rpcCtx, &authv1.CreateUserCredentialRequest{
		UserId:      uint64(userID),
		RequestJson: string(raw),
	})
	if err != nil {
		return nil, unwrap(err)
	}
	return authgrpc.CredentialFromProto(resp)
}

func unwrap(err error) error {
	return grpcpkg.WrapClientError(err)
}
