package grpc

// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

import (
	"errors"
	"fmt"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// RPCError wraps a gRPC status for use on the client side after unwrap.
type RPCError struct {
	Code    codes.Code
	Message string
}

func (e *RPCError) Error() string {
	if e == nil {
		return ""
	}
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

// WrapClientError converts a gRPC client error into RPCError when possible.
func WrapClientError(err error) error {
	if err == nil {
		return nil
	}
	if st, ok := status.FromError(err); ok {
		return &RPCError{Code: st.Code(), Message: st.Message()}
	}
	return err
}

// IsNotFound reports whether err is a gRPC NotFound (or wrapped RPCError).
func IsNotFound(err error) bool {
	if err == nil {
		return false
	}
	var re *RPCError
	if errors.As(err, &re) {
		return re.Code == codes.NotFound
	}
	if st, ok := status.FromError(err); ok {
		return st.Code() == codes.NotFound
	}
	return false
}
