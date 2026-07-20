package voiceprintconfig

import (
	"context"
	"fmt"
	"time"
)

// SelfTestCheck is one validation step in a voiceprint self-test run.
type SelfTestCheck struct {
	Name   string `json:"name"`
	OK     bool   `json:"ok"`
	Detail string `json:"detail,omitempty"`
}

// SelfTestReport aggregates env validation and optional live health probe.
type SelfTestReport struct {
	Enabled  bool            `json:"enabled"`
	Provider string          `json:"provider,omitempty"`
	Label    string          `json:"label,omitempty"`
	OK       bool            `json:"ok"`
	Checks   []SelfTestCheck `json:"checks"`
}

// RunSelfTest validates configuration and optionally probes the provider.
// When probeLive is false, only env/credential checks run (safe for CI).
func RunSelfTest(ctx context.Context, probeLive bool) SelfTestReport {
	report := SelfTestReport{Checks: make([]SelfTestCheck, 0, 4)}
	slug, provider, ok := ResolveEnabled()
	if !ok {
		report.Checks = append(report.Checks, SelfTestCheck{
			Name:   "resolve_enabled",
			OK:     false,
			Detail: fmt.Sprintf("%s is unset or credentials are incomplete", envProvider),
		})
		return report
	}
	report.Enabled = true
	report.Provider = slug
	report.Label = ProviderLabel(provider)
	report.Checks = append(report.Checks, SelfTestCheck{
		Name:   "resolve_enabled",
		OK:     true,
		Detail: slug,
	})

	if err := ValidateEnv(provider); err != nil {
		report.Checks = append(report.Checks, SelfTestCheck{
			Name:   "validate_env",
			OK:     false,
			Detail: err.Error(),
		})
		return report
	}
	report.Checks = append(report.Checks, SelfTestCheck{
		Name:   "validate_env",
		OK:     true,
		Detail: "credentials present",
	})

	if !probeLive {
		report.OK = true
		return report
	}
	if ctx == nil {
		ctx = context.Background()
	}
	probeCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	bridge, err := NewBridge()
	if err != nil {
		report.Checks = append(report.Checks, SelfTestCheck{
			Name:   "create_bridge",
			OK:     false,
			Detail: err.Error(),
		})
		return report
	}
	defer bridge.Close()

	if err := bridge.HealthCheck(probeCtx); err != nil {
		report.Checks = append(report.Checks, SelfTestCheck{
			Name:   "health_check",
			OK:     false,
			Detail: err.Error(),
		})
		return report
	}
	report.Checks = append(report.Checks, SelfTestCheck{
		Name:   "health_check",
		OK:     true,
		Detail: "provider reachable",
	})
	report.OK = true
	return report
}
