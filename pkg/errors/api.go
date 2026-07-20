package errors

// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

import (
	stderrors "errors"
	"fmt"
	"net/http"
)

// Error is an HTTP-status-coded API error (legacy shape for storage/clients).
type Error struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

func (e Error) StatusCode() int {
	return e.Code
}

func (e Error) Error() string {
	return fmt.Sprintf("[%d] %s", e.Code, e.Message)
}

var ErrUnauthorized = &Error{Code: http.StatusUnauthorized, Message: "unauthorized"}

var ErrAttachmentNotExist = &Error{Code: http.StatusNotFound, Message: "attachment not exist"}

var ErrNotAttachmentOwner = &Error{Code: http.StatusForbidden, Message: "not attachment owner"}

// Authentication & Registration Related Errors

var ErrQuotaExceeded = stderrors.New("quota exceeded")

var ErrLLMCallFailed = stderrors.New("failed to call language model")

var ErrEmptyPassword = stderrors.New("empty password")

var ErrEmptyEmail = stderrors.New("empty email")

var ErrSameEmail = stderrors.New("same email")

var ErrEmailExists = stderrors.New("email exists, please use another email")

var ErrUserNotExists = stderrors.New("user not exists")

var ErrForbidden = stderrors.New("forbidden access")

var ErrUserNotAllowLogin = stderrors.New("user not allow login")

var ErrUserNotAllowSignup = stderrors.New("user not allow signup")

var ErrNotActivated = stderrors.New("user not activated")

var ErrTokenRequired = stderrors.New("token required")

var ErrInvalidToken = stderrors.New("invalid token")

var ErrBadToken = stderrors.New("bad token")

var ErrTokenExpired = stderrors.New("token expired")

var ErrEmailRequired = stderrors.New("email required")

// General Resource/Data Processing Related Errors

var ErrNotFound = stderrors.New("not found")

var ErrNotChanged = stderrors.New("not changed")

var ErrInvalidView = stderrors.New("with invalid view")

// Permission and Logic Control Related Errors

var ErrOnlySuperUser = stderrors.New("only super user can do this")

var ErrInvalidPrimaryKey = stderrors.New("invalid primary key")

// Common errors
var (
	// Tools related errors
	ErrInvalidToolListFormat = stderrors.New("invalid tool list response format")
	ErrInvalidToolFormat     = stderrors.New("invalid tool format")
	ErrToolNotFound          = stderrors.New("tool not found")
	ErrInvalidToolParams     = stderrors.New("invalid tool parameters")

	// JSON-RPC related errors
	ErrParseJSONRPC           = stderrors.New("failed to parse JSON-RPC message")
	ErrInvalidJSONRPCFormat   = stderrors.New("invalid JSON-RPC format")
	ErrInvalidJSONRPCResponse = stderrors.New("invalid JSON-RPC response")
	ErrInvalidJSONRPCRequest  = stderrors.New("invalid JSON-RPC request")
	ErrInvalidJSONRPCParams   = stderrors.New("invalid JSON-RPC parameters")

	// Resource related errors
	ErrInvalidResourceFormat = stderrors.New("invalid resource format")
	ErrResourceNotFound      = stderrors.New("resource not found")

	// Prompt related errors
	ErrInvalidPromptFormat = stderrors.New("invalid prompt format")
	ErrPromptNotFound      = stderrors.New("prompt not found")

	// Tool manager errors
	ErrEmptyToolName         = stderrors.New("tool name cannot be empty")
	ErrToolAlreadyRegistered = stderrors.New("tool already registered")
	ErrToolExecutionFailed   = stderrors.New("tool execution failed")

	// Resource manager errors
	ErrEmptyResourceURI = stderrors.New("resource URI cannot be empty")

	// Prompt manager errors
	ErrEmptyPromptName = stderrors.New("prompt name cannot be empty")

	// Lifecycle manager errors
	ErrSessionAlreadyInitialized = stderrors.New("session already initialized")
	ErrSessionNotInitialized     = stderrors.New("session not initialized")

	// Parameter errors
	ErrInvalidParams = stderrors.New("invalid parameters")
	ErrMissingParams = stderrors.New("missing required parameters")

	// Client errors
	ErrAlreadyInitialized = stderrors.New("client already initialized")
	ErrNotInitialized     = stderrors.New("client not initialized")
	ErrInvalidServerURL   = stderrors.New("invalid server URL")
)
