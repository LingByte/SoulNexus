package redislock

import (
	"context"
	"testing"
	"time"
)

func TestTryLockWithoutRedisAllows(t *testing.T) {
	unlock, ok, err := TryLock(context.Background(), "test", time.Second)
	if err != nil {
		t.Fatal(err)
	}
	if !ok {
		t.Fatal("expected ok without redis")
	}
	unlock()
}

func TestEnabledDefaultFalse(t *testing.T) {
	if Enabled() {
		t.Fatal("expected disabled without InitFromEnv")
	}
}
