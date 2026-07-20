package errors

import "errors"

// Copyright (c) 2026 LingByte
// SPDX-License-Identifier: MIT

var (
	ErrTenantSlugAllocate          = errors.New("could not allocate unique tenant slug")
	ErrInvalidContactEmail         = errors.New("invalid contact email")
	ErrJWTKeyManagerNotInitialized = errors.New("jwt key manager not initialized")
)
