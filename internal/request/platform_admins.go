package request

// Copyright (c) 2026 LingByte
// SPDX-License-Identifier: MIT

type PlatformAdminCreateReq struct {
	Email       string `json:"email" binding:"required,email"`
	Password    string `json:"password" binding:"required,min=8,max=128"`
	DisplayName string `json:"displayName"`
	Status      string `json:"status"`
}

type PlatformAdminUpdateReq struct {
	Email       string `json:"email"`
	DisplayName string `json:"displayName"`
}

type PlatformAdminStatusReq struct {
	Status string `json:"status" binding:"required"`
}

type PlatformAdminPasswordReq struct {
	Password string `json:"password" binding:"required,min=8,max=128"`
}
