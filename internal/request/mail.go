package request

// Copyright (c) 2026 LingByte
// SPDX-License-Identifier: MIT

type MailTemplateCreateReq struct {
	Code        string `json:"code" binding:"required,max=64"`
	Name        string `json:"name" binding:"required,max=128"`
	Subject     string `json:"subject" binding:"max=255"`
	HTMLBody    string `json:"htmlBody" binding:"required"`
	Description string `json:"description" binding:"max=512"`
	Variables   string `json:"variables"`
	Locale      string `json:"locale" binding:"max=32"`
	Enabled     *bool  `json:"enabled"`
}

type MailTemplateUpdateReq struct {
	Name        string `json:"name" binding:"required,max=128"`
	Subject     string `json:"subject" binding:"max=255"`
	HTMLBody    string `json:"htmlBody" binding:"required"`
	Description string `json:"description" binding:"max=512"`
	Variables   string `json:"variables"`
	Locale      string `json:"locale" binding:"max=32"`
	Enabled     *bool  `json:"enabled"`
}
