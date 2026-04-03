package outbound

import (
	"context"
)

// ScriptRunner drives IVR-style steps after a campaign/callback leg is up (prompts, DTMF, API side-effects).
// Implementations live outside this package (e.g. internal/campaign) to keep SIP core dependency-free.
type ScriptRunner interface {
	// Run blocks until the script completes or ctx is cancelled.
	Run(ctx context.Context, leg EstablishedLeg) error
}

// CampaignQueue is a future abstraction for durable job processing (DB, Redis, etc.).
// Not wired in v1 — documented for extension.
type CampaignQueue interface {
	Enqueue(ctx context.Context, job CampaignJob) error
}

// CampaignJob identifies one scripted outbound attempt.
type CampaignJob struct {
	Scenario      Scenario
	Target        DialTarget
	ScriptID      string
	CorrelationID string
	MediaProfile  MediaProfile
}
