package handlers

import (
	"context"
	"testing"

	llmcache "github.com/LingByte/lingllm/cache"
)

func TestEmailCodeNotOverwrittenByCooldown(t *testing.T) {
	email := "user@example.com"
	purpose := emailCodePassword
	codeKey := emailCodeCacheKey(purpose, email)
	cooldownKey := emailCodeCooldownCacheKey(purpose, email)

	emailCodeCacheSet(codeKey, "123456", emailCodeTTL)
	emailCodeCooldownSet(cooldownKey)

	stored, ok := emailCodeCacheGet(codeKey)
	if !ok {
		t.Fatal("verification code missing from cache")
	}
	if stored != "123456" {
		t.Fatalf("expected code 123456, got %q", stored)
	}
	if !emailCodeCooldownActive(cooldownKey) {
		t.Fatal("cooldown key should be active")
	}

	_ = llmcache.Delete(context.Background(), codeKey)
	_ = llmcache.Delete(context.Background(), cooldownKey)
}
