package logger

// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0
//
// SafeGo wraps a goroutine with panic recovery + structured logging.
//
// Why this exists:
//   The codebase has ~80 `go func()` sites, most uncaught. A single panic
//   in any of them (nil map write, slice index OOB, send on closed chan,
//   third-party SDK bug, …) brings down the entire API server.
//   Wrapping at every call site is invasive; this helper makes the safe
//   path one extra line:
//
//       logger.SafeGo("transfer-dial", func() {
//           outboundMgr.Dial(req)
//       })
//
// The recovered panic is logged at Error with:
//   - name (caller-supplied tag, makes the log line greppable)
//   - panic value (zap.Any)
//   - stack trace via zap.Stack (full goroutine stack at the panic site)
//
// If logger.Lg is nil (init order edge case, tests), we still recover —
// silently — to preserve the "don't crash on panic" invariant. The
// per-call name lets you still find this goroutine in any logging that
// happens AFTER Lg is initialised (e.g. via the trailing log below).

import (
	"fmt"
	"os"
	"runtime/debug"

	"go.uber.org/zap"
)

// printStderrPanic is the early-init fallback when Lg is not yet ready.
// It mirrors what zap would have produced so panics aren't silently
// dropped between process start and Init().
func printStderrPanic(name string, r interface{}) {
	fmt.Fprintf(os.Stderr, "goroutine panic recovered (logger not ready): name=%s panic=%v\n%s\n",
		name, r, debug.Stack())
}

// SafeGo runs fn in a new goroutine with panic recovery and structured
// logging. Returns immediately. name is a short tag identifying the
// goroutine in logs (e.g. "transfer-dial", "campaign-worker"). Empty
// name is tolerated but discouraged.
func SafeGo(name string, fn func()) {
	if fn == nil {
		return
	}
	go func() {
		defer recoverGoroutine(name)
		fn()
	}()
}

// recoverGoroutine is the shared panic handler. Kept un-inlined so the
// reported caller in logs / panics points at the user goroutine site,
// not at this helper.
func recoverGoroutine(name string) {
	r := recover()
	if r == nil {
		return
	}
	if Lg == nil {
		// Last-ditch: at least surface to stderr so we don't lose the
		// panic entirely during early-init.
		printStderrPanic(name, r)
		return
	}
	Lg.Error("goroutine panic recovered",
		zap.String("goroutine", name),
		zap.Any("panic", r),
		zap.Stack("stack"),
	)
}
