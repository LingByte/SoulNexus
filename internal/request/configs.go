package request

// Copyright (c) 2026 LingByte
// SPDX-License-Identifier: MIT

type SystemConfigCreateReq struct {
	Key      string `json:"key" binding:"required,max=128"`
	Desc     string `json:"desc" binding:"max=200"`
	Value    string `json:"value"`
	Format   string `json:"format"`
	Autoload bool   `json:"autoload"`
	Public   bool   `json:"public"`
}

type SystemConfigUpdateReq struct {
	Desc     *string `json:"desc"`
	Value    *string `json:"value"`
	Format   *string `json:"format"`
	Autoload *bool   `json:"autoload"`
	Public   *bool   `json:"public"`
}

type RoutePolicyReq struct {
	Enabled  bool     `json:"enabled"`
	RouteIDs []string `json:"routeIds"`
}
