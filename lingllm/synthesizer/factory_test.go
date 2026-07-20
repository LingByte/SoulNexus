package synthesizer

import (
	"testing"
)

func TestNewSynthesisFactory(t *testing.T) {
	factory := NewSynthesisFactory()
	if factory == nil {
		t.Error("NewSynthesisFactory should not return nil")
	}
}

func TestSynthesisFactoryGetSupportedProviders(t *testing.T) {
	factory := NewSynthesisFactory()
	providers := factory.GetSupportedProviders()
	if len(providers) == 0 {
		t.Error("GetSupportedProviders should return at least one provider")
	}
}

func TestSynthesisFactoryIsProviderSupported(t *testing.T) {
	factory := NewSynthesisFactory()

	// Test supported provider
	if !factory.IsProviderSupported(ProviderTencent) {
		t.Error("ProviderTencent should be supported")
	}

	// Test unsupported provider
	if factory.IsProviderSupported("unsupported-provider") {
		t.Error("Unsupported provider should return false")
	}
}

func TestSynthesisFactoryRegisterCreator(t *testing.T) {
	factory := NewSynthesisFactory()

	customProvider := TTSProvider("custom-provider")
	factory.RegisterCreator(customProvider, func(config SynthesisConfig) (AudioSynthesisEngine, error) {
		return nil, nil
	})

	if !factory.IsProviderSupported(customProvider) {
		t.Error("Custom provider should be supported after registration")
	}
}
