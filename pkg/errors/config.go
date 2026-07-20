package errors

import "errors"

// Copyright (c) 2026 LingByte
// SPDX-License-Identifier: MIT

var (
	ErrConfigNotFound      = errors.New("config not found")
	ErrConfigKeyExists     = errors.New("config key already exists")
	ErrConfigKeyRequired   = errors.New("config key is required")
	ErrConfigFormatInvalid = errors.New("invalid config format")
)
