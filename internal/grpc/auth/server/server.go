package server

// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

import (
	"context"
	"encoding/json"
	"net"
	"strings"
	"time"

	grpcpkg "github.com/LingByte/SoulNexus/internal/grpc"
	authgrpc "github.com/LingByte/SoulNexus/internal/grpc/auth"
	authv1 "github.com/LingByte/SoulNexus/internal/grpc/auth/pb/auth/v1"
	authmodel "github.com/LingByte/SoulNexus/internal/models/auth"
	svcmodels "github.com/LingByte/SoulNexus/internal/models/server"
	"github.com/LingByte/SoulNexus/pkg/logger"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gorm.io/gorm"
)

// UserAuthServer implements auth.v1.UserAuthService against the auth database.
type UserAuthServer struct {
	authv1.UnimplementedUserAuthServiceServer
	db *gorm.DB
}

func New(db *gorm.DB) *UserAuthServer {
	return &UserAuthServer{db: db}
}

func (s *UserAuthServer) GetUser(ctx context.Context, req *authv1.GetUserRequest) (*authv1.User, error) {
	if req == nil || req.Id == 0 {
		return nil, status.Error(codes.InvalidArgument, "id required")
	}
	u, err := authmodel.GetUserByID(s.db, uint(req.Id))
	if err != nil {
		return nil, grpcpkg.MapError(err)
	}
	if u == nil {
		return nil, status.Error(codes.NotFound, "user not found")
	}
	return authgrpc.UserToProto(u), nil
}

func (s *UserAuthServer) GetUserByEmail(ctx context.Context, req *authv1.GetUserByEmailRequest) (*authv1.User, error) {
	if req == nil || req.Email == "" {
		return nil, status.Error(codes.InvalidArgument, "email required")
	}
	u, err := authmodel.GetUserByEmail(s.db, req.Email)
	if err != nil {
		return nil, grpcpkg.MapError(err)
	}
	if u == nil {
		return nil, status.Error(codes.NotFound, "user not found")
	}
	return authgrpc.UserToProto(u), nil
}

func (s *UserAuthServer) ResolveCredential(ctx context.Context, req *authv1.ResolveCredentialRequest) (*authv1.UserCredential, error) {
	if req == nil || req.ApiKey == "" || req.ApiSecret == "" {
		return nil, status.Error(codes.InvalidArgument, "api_key and api_secret required")
	}
	c, err := authmodel.GetUserCredentialByApiSecretAndApiKey(s.db, req.ApiKey, req.ApiSecret)
	if err != nil {
		return nil, grpcpkg.MapError(err)
	}
	if c == nil {
		return nil, status.Error(codes.NotFound, "credential not found")
	}
	return authgrpc.CredentialToProto(c)
}

func (s *UserAuthServer) ListUserCredentials(ctx context.Context, req *authv1.ListUserCredentialsRequest) (*authv1.ListUserCredentialsResponse, error) {
	if req == nil || req.UserId == 0 {
		return nil, status.Error(codes.InvalidArgument, "user_id required")
	}
	list, err := svcmodels.GetUserCredentials(s.db, uint(req.UserId))
	if err != nil {
		return nil, grpcpkg.MapError(err)
	}
	out := &authv1.ListUserCredentialsResponse{}
	for _, c := range list {
		p, err := authgrpc.CredentialToProto(c)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "credential encode: %v", err)
		}
		out.Credentials = append(out.Credentials, p)
	}
	return out, nil
}

func (s *UserAuthServer) GetCredential(ctx context.Context, req *authv1.GetCredentialRequest) (*authv1.UserCredential, error) {
	if req == nil || req.Id == 0 {
		return nil, status.Error(codes.InvalidArgument, "id required")
	}
	var c authmodel.UserCredential
	if err := s.db.First(&c, req.Id).Error; err != nil {
		return nil, grpcpkg.MapError(err)
	}
	return authgrpc.CredentialToProto(&c)
}

func (s *UserAuthServer) DeleteCredential(ctx context.Context, req *authv1.DeleteCredentialRequest) (*authv1.DeleteCredentialResponse, error) {
	if req == nil || req.Id == 0 {
		return nil, status.Error(codes.InvalidArgument, "id required")
	}
	if err := s.db.Delete(&authmodel.UserCredential{}, req.Id).Error; err != nil {
		return nil, grpcpkg.MapError(err)
	}
	return &authv1.DeleteCredentialResponse{}, nil
}

func (s *UserAuthServer) MarkCredentialUsed(ctx context.Context, req *authv1.MarkCredentialUsedRequest) (*authv1.MarkCredentialUsedResponse, error) {
	if req == nil || req.CredentialId == 0 {
		return nil, status.Error(codes.InvalidArgument, "credential_id required")
	}
	if err := authmodel.MarkCredentialUsed(s.db, uint(req.CredentialId)); err != nil {
		return nil, grpcpkg.MapError(err)
	}
	return &authv1.MarkCredentialUsedResponse{}, nil
}

func (s *UserAuthServer) AdminListCredentials(ctx context.Context, req *authv1.AdminListCredentialsRequest) (*authv1.AdminListCredentialsResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "request required")
	}
	page := int(req.Page)
	if page < 1 {
		page = 1
	}
	pageSize := int(req.PageSize)
	if pageSize < 1 {
		pageSize = 20
	}
	query := s.db.Model(&authmodel.UserCredential{})
	if search := strings.TrimSpace(req.Search); search != "" {
		like := "%" + search + "%"
		query = query.Where("name LIKE ? OR api_key LIKE ? OR llm_provider LIKE ?", like, like, like)
	}
	if status := strings.TrimSpace(req.Status); status != "" {
		query = query.Where("status = ?", status)
	}
	if req.UserId > 0 {
		query = query.Where("created_by = ?", req.UserId)
	}
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, grpcpkg.MapError(err)
	}
	var creds []authmodel.UserCredential
	if err := query.Order("id DESC").Offset((page - 1) * pageSize).Limit(pageSize).Find(&creds).Error; err != nil {
		return nil, grpcpkg.MapError(err)
	}
	out := &authv1.AdminListCredentialsResponse{Total: total}
	for i := range creds {
		p, err := authgrpc.CredentialToProto(&creds[i])
		if err != nil {
			return nil, status.Errorf(codes.Internal, "credential encode: %v", err)
		}
		out.Credentials = append(out.Credentials, p)
	}
	return out, nil
}

func (s *UserAuthServer) UpdateCredentialStatus(ctx context.Context, req *authv1.UpdateCredentialStatusRequest) (*authv1.UserCredential, error) {
	if req == nil || req.Id == 0 {
		return nil, status.Error(codes.InvalidArgument, "id required")
	}
	var cred authmodel.UserCredential
	if err := s.db.First(&cred, req.Id).Error; err != nil {
		return nil, grpcpkg.MapError(err)
	}
	statusVal := authmodel.CredentialStatus(strings.TrimSpace(req.Status))
	switch statusVal {
	case authmodel.CredentialStatusActive, authmodel.CredentialStatusBanned, authmodel.CredentialStatusSuspended:
	default:
		return nil, status.Error(codes.InvalidArgument, "invalid credential status")
	}
	updateVals := map[string]any{"status": statusVal}
	switch statusVal {
	case authmodel.CredentialStatusActive:
		updateVals["banned_at"] = nil
		updateVals["banned_reason"] = ""
		updateVals["banned_by"] = nil
	case authmodel.CredentialStatusBanned:
		now := time.Now()
		updateVals["banned_at"] = &now
		updateVals["banned_reason"] = req.BannedReason
	case authmodel.CredentialStatusSuspended:
	}
	if req.ExpiresAt != nil {
		raw := strings.TrimSpace(*req.ExpiresAt)
		if raw == "" {
			updateVals["expires_at"] = nil
		} else {
			var parsed time.Time
			var parseErr error
			if strings.Contains(raw, "T") {
				parsed, parseErr = time.Parse(time.RFC3339, raw)
			} else {
				parsed, parseErr = time.ParseInLocation("2006-01-02 15:04:05", raw, time.Local)
			}
			if parseErr != nil {
				return nil, status.Error(codes.InvalidArgument, "invalid expires_at format")
			}
			updateVals["expires_at"] = &parsed
		}
	}
	if req.TokenQuota != nil {
		if *req.TokenQuota < 0 {
			return nil, status.Error(codes.InvalidArgument, "token_quota must be >= 0")
		}
		updateVals["token_quota"] = *req.TokenQuota
	}
	if req.RequestQuota != nil {
		if *req.RequestQuota < 0 {
			return nil, status.Error(codes.InvalidArgument, "request_quota must be >= 0")
		}
		updateVals["request_quota"] = *req.RequestQuota
	}
	if req.UseNativeQuota != nil {
		updateVals["use_native_quota"] = *req.UseNativeQuota
	}
	if req.UnlimitedQuota != nil {
		updateVals["unlimited_quota"] = *req.UnlimitedQuota
	}
	if err := s.db.Model(&cred).Updates(updateVals).Error; err != nil {
		return nil, grpcpkg.MapError(err)
	}
	_ = s.db.First(&cred, cred.ID).Error
	return authgrpc.CredentialToProto(&cred)
}

func (s *UserAuthServer) CreateUserCredential(ctx context.Context, req *authv1.CreateUserCredentialRequest) (*authv1.UserCredential, error) {
	if req == nil || req.UserId == 0 || req.RequestJson == "" {
		return nil, status.Error(codes.InvalidArgument, "user_id and request_json required")
	}
	var body authmodel.UserCredentialRequest
	if err := json.Unmarshal([]byte(req.RequestJson), &body); err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request_json")
	}
	cred, err := svcmodels.CreateUserCredential(s.db, uint(req.UserId), &body)
	if err != nil {
		return nil, grpcpkg.MapError(err)
	}
	return authgrpc.CredentialToProto(cred)
}

func (s *UserAuthServer) DeleteUserCredentialForUser(ctx context.Context, req *authv1.DeleteUserCredentialForUserRequest) (*authv1.DeleteCredentialResponse, error) {
	if req == nil || req.UserId == 0 || req.CredentialId == 0 {
		return nil, status.Error(codes.InvalidArgument, "user_id and credential_id required")
	}
	if err := svcmodels.DeleteUserCredential(s.db, uint(req.UserId), uint(req.CredentialId)); err != nil {
		return nil, grpcpkg.MapError(err)
	}
	return &authv1.DeleteCredentialResponse{}, nil
}

// Listen starts the gRPC server on addr (e.g. ":7075"). Blocks until ctx is cancelled.
func Listen(ctx context.Context, addr string, db *gorm.DB) error {
	lis, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}
	srv := grpc.NewServer()
	authv1.RegisterUserAuthServiceServer(srv, New(db))
	logger.Info("auth gRPC listen", zap.String("addr", addr))
	done := make(chan struct{})
	go func() {
		<-ctx.Done()
		srv.GracefulStop()
		close(done)
	}()
	err = srv.Serve(lis)
	<-done
	return err
}
