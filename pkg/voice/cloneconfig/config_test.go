package cloneconfig

import (
	"testing"

	"github.com/LingByte/SoulNexus/pkg/utils"
)

func TestResolveXunfeiTrainCredentialsPrefersMainWhenTrainAppIDInvalid(t *testing.T) {
	t.Cleanup(utils.PurgeEnvCacheForTest)
	t.Setenv("XUNFEI_APP_ID", "abc12345")
	t.Setenv("XUNFEI_API_KEY", "main-api-key")
	t.Setenv("XUNFEI_TRAIN_APP_ID", "01234567890123456789012345678901") // 32 chars, likely APIKey
	t.Setenv("XUNFEI_TRAIN_API_KEY", "wrong-train-key")

	appID, apiKey := resolveXunfeiTrainCredentials()
	if appID != "abc12345" || apiKey != "main-api-key" {
		t.Fatalf("got appID=%q apiKey=%q, want main credentials", appID, apiKey)
	}
}

func TestResolveXunfeiTrainCredentialsUsesTrainWhenValid(t *testing.T) {
	t.Cleanup(utils.PurgeEnvCacheForTest)
	t.Setenv("XUNFEI_APP_ID", "mainapp1")
	t.Setenv("XUNFEI_API_KEY", "main-api-key")
	t.Setenv("XUNFEI_TRAIN_APP_ID", "trainapp")
	t.Setenv("XUNFEI_TRAIN_API_KEY", "train-api-key")

	appID, apiKey := resolveXunfeiTrainCredentials()
	if appID != "trainapp" || apiKey != "train-api-key" {
		t.Fatalf("got appID=%q apiKey=%q, want train credentials", appID, apiKey)
	}
}

func TestResolveXunfeiTrainCredentialsIgnoresPartialTrainOverride(t *testing.T) {
	t.Cleanup(utils.PurgeEnvCacheForTest)
	t.Setenv("XUNFEI_APP_ID", "mainapp1")
	t.Setenv("XUNFEI_API_KEY", "main-api-key")
	t.Setenv("XUNFEI_TRAIN_APP_ID", "trainapp")
	t.Setenv("XUNFEI_TRAIN_API_KEY", "")

	appID, apiKey := resolveXunfeiTrainCredentials()
	if appID != "mainapp1" || apiKey != "main-api-key" {
		t.Fatalf("got appID=%q apiKey=%q, want main credentials", appID, apiKey)
	}
}
