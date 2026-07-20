package logger

import (
	"sync"
	"testing"
	"time"

	"go.uber.org/zap"
)

func TestSafeGo_NormalExecution(t *testing.T) {
	var called bool
	var wg sync.WaitGroup
	wg.Add(1)

	SafeGo("test-normal", func() {
		defer wg.Done()
		called = true
	})

	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("SafeGo goroutine did not complete")
	}

	if !called {
		t.Fatal("SafeGo should have called the function")
	}
}

func TestSafeGo_NilFunction(t *testing.T) {
	// Should not panic
	SafeGo("test-nil", nil)
}

func TestSafeGo_EmptyName(t *testing.T) {
	var called bool
	var wg sync.WaitGroup
	wg.Add(1)

	SafeGo("", func() {
		defer wg.Done()
		called = true
	})

	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("SafeGo with empty name did not complete")
	}

	if !called {
		t.Fatal("SafeGo should have called the function even with empty name")
	}
}

func TestRecoverGoroutine_WithLogger(t *testing.T) {
	oldLg := Lg
	Lg = zap.NewNop()
	defer func() { Lg = oldLg }()

	// Test recoverGoroutine directly (not via SafeGo) to avoid
	// defer-ordering races in tests.
	done := make(chan struct{})
	go func() {
		defer close(done)
		defer recoverGoroutine("direct-test")
		panic("intentional test panic")
	}()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("recoverGoroutine should have recovered the panic")
	}
}

func TestRecoverGoroutine_NilLogger(t *testing.T) {
	oldLg := Lg
	Lg = nil
	defer func() { Lg = oldLg }()

	done := make(chan struct{})
	go func() {
		defer close(done)
		defer recoverGoroutine("direct-nil-test")
		panic("intentional panic when Lg is nil")
	}()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("recoverGoroutine should recover panic even when Lg is nil")
	}
}

func TestRecoverGoroutine_NoPanic(t *testing.T) {
	// recoverGoroutine should be a no-op when there's no panic
	// This is covered by TestSafeGo_NormalExecution, but test directly too
	recoverGoroutine("no-panic")
	// No assertion needed - if it panics, the test fails
}

func TestPrintStderrPanic(t *testing.T) {
	// printStderrPanic writes to stderr - just ensure it doesn't panic
	printStderrPanic("test-name", "test-panic-value")
}
