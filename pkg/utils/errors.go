package utils

import (
	"errors"
	"fmt"
	"net/http"
)

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

var ErrQuotaExceeded = errors.New("quota exceeded") // User quota has been exhausted

var ErrLLMCallFailed = errors.New("failed to call language model") // Failed to call language model

var ErrEmptyPassword = errors.New("empty password") // Password is empty, typically used for registration or login validation failure

var ErrEmptyEmail = errors.New("empty email") // Email is empty, typically used for registration, login, password recovery and other operations

var ErrSameEmail = errors.New("same email") // Old and new email are the same, triggered when user tries to change email

var ErrEmailExists = errors.New("email exists, please use another email") // Email already exists, trying to register or update to an already registered email

var ErrUserNotExists = errors.New("user not exists") // User does not exist, commonly used for login, query or operations on non-existent users

var ErrForbidden = errors.New("forbidden access") // Access denied, user is logged in but has no permission to access the target resource

var ErrUserNotAllowLogin = errors.New("user not allow login") // User is prohibited from logging in, possibly banned by administrator

var ErrUserNotAllowSignup = errors.New("user not allow signup") // User is prohibited from registering, system configuration or policy restricts registration behavior

var ErrNotActivated = errors.New("user not activated") // User account is not activated, usually used when email activation is not completed

var ErrTokenRequired = errors.New("token required") // Missing required token, for example when accessing protected resources

var ErrInvalidToken = errors.New("invalid token") // Token format is illegal or does not conform to specifications

var ErrBadToken = errors.New("bad token") // Token has been tampered with, forged or is invalid

var ErrTokenExpired = errors.New("token expired") // Token has expired

var ErrEmailRequired = errors.New("email required") // Email field must be provided but was not provided

// General Resource/Data Processing Related Errors

var ErrNotFound = errors.New("not found") // Requested data or resource not found

var ErrNotChanged = errors.New("not changed") // Data has not changed, for example no actual field changes in update request

var ErrInvalidView = errors.New("with invalid view") // Request used an invalid view identifier or parameter

// Permission and Logic Control Related Errors

var ErrOnlySuperUser = errors.New("only super user can do this") // Operations limited to super users only

var ErrInvalidPrimaryKey = errors.New("invalid primary key") // Primary key is invalid, possibly due to format error or missing

// Common errors
var (
	// Tools related errors
	ErrInvalidToolListFormat = errors.New("invalid tool list response format")
	ErrInvalidToolFormat     = errors.New("invalid tool format")
	ErrToolNotFound          = errors.New("tool not found")
	ErrInvalidToolParams     = errors.New("invalid tool parameters")

	// JSON-RPC related errors
	ErrParseJSONRPC           = errors.New("failed to parse JSON-RPC message")
	ErrInvalidJSONRPCFormat   = errors.New("invalid JSON-RPC format")
	ErrInvalidJSONRPCResponse = errors.New("invalid JSON-RPC response")
	ErrInvalidJSONRPCRequest  = errors.New("invalid JSON-RPC request")
	ErrInvalidJSONRPCParams   = errors.New("invalid JSON-RPC parameters")

	// Resource related errors
	ErrInvalidResourceFormat = errors.New("invalid resource format")
	ErrResourceNotFound      = errors.New("resource not found")

	// Prompt related errors
	ErrInvalidPromptFormat = errors.New("invalid prompt format")
	ErrPromptNotFound      = errors.New("prompt not found")

	// Tool manager errors
	ErrEmptyToolName         = errors.New("tool name cannot be empty")
	ErrToolAlreadyRegistered = errors.New("tool already registered")
	ErrToolExecutionFailed   = errors.New("tool execution failed")

	// Resource manager errors
	ErrEmptyResourceURI = errors.New("resource URI cannot be empty")

	// Prompt manager errors
	ErrEmptyPromptName = errors.New("prompt name cannot be empty")

	// Lifecycle manager errors
	ErrSessionAlreadyInitialized = errors.New("session already initialized")
	ErrSessionNotInitialized     = errors.New("session not initialized")

	// Parameter errors
	ErrInvalidParams = errors.New("invalid parameters")
	ErrMissingParams = errors.New("missing required parameters")

	// Client errors
	ErrAlreadyInitialized = errors.New("client already initialized")
	ErrNotInitialized     = errors.New("client not initialized")
	ErrInvalidServerURL   = errors.New("invalid server URL")
)
