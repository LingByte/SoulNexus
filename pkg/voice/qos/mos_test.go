// Copyright (c) 2026 LinByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

package qos

import "testing"

// MOS sanity expectations. These aren't lab-grade ground truth (no
// such thing without listener panels) — they're "common sense"
// boundaries that any reasonable implementation must meet.

func TestEstimate_PristineG711_HighMOS(t *testing.T) {
	// Local-network G.711: ~30 ms RTT, no loss, ~5 ms jitter →
	// expected MOS close to the codec ceiling (~4.3-4.4).
	r := Estimate(MOSInput{
		RTTMs:           30,
		JitterRTPUnits:  40, // ~5 ms at 8 kHz
		JitterClockRate: 8000,
		Codec:           "pcmu",
	})
	if r.MOS < 4.2 || r.MOS > 4.5 {
		t.Errorf("clean G.711 MOS=%.2f; expected ~4.3", r.MOS)
	}
}

func TestEstimate_HighLatencyDegradesMOS(t *testing.T) {
	clean := Estimate(MOSInput{RTTMs: 30, Codec: "pcmu"})
	farAway := Estimate(MOSInput{RTTMs: 600, Codec: "pcmu"})
	if farAway.MOS >= clean.MOS {
		t.Errorf("600 ms RTT must degrade MOS; clean=%.2f far=%.2f", clean.MOS, farAway.MOS)
	}
}

func TestEstimate_PacketLossDegradesMOS(t *testing.T) {
	clean := Estimate(MOSInput{RTTMs: 30, Codec: "pcmu"})
	lossy := Estimate(MOSInput{RTTMs: 30, PeerLossFraction: 0.05, Codec: "pcmu"})
	if lossy.MOS >= clean.MOS-0.1 {
		t.Errorf("5%% loss must visibly degrade MOS; clean=%.2f lossy=%.2f",
			clean.MOS, lossy.MOS)
	}
}

func TestEstimate_OpusBeatsG711AtSameImpairments(t *testing.T) {
	// Opus with its wideband baseline and FEC should score at least
	// as high as G.711 under identical loss conditions.
	g711 := Estimate(MOSInput{RTTMs: 80, PeerLossFraction: 0.02, Codec: "pcmu"})
	opus := Estimate(MOSInput{RTTMs: 80, PeerLossFraction: 0.02, Codec: "opus", JitterClockRate: 48000})
	if opus.MOS < g711.MOS-0.05 {
		t.Errorf("Opus should not be measurably worse than G.711 under loss; "+
			"opus=%.2f g711=%.2f", opus.MOS, g711.MOS)
	}
}

func TestEstimate_ClampsToValidRange(t *testing.T) {
	// Absurd inputs must still produce a number in 1.0..4.5.
	cases := []MOSInput{
		{RTTMs: 100000, PeerLossFraction: 1.0, Codec: "pcmu"},
		{RTTMs: 0, PeerLossFraction: -1, Codec: "unknown"},
		{},
	}
	for i, in := range cases {
		r := Estimate(in)
		if r.MOS < 1.0 || r.MOS > 4.5 {
			t.Errorf("case %d: MOS out of range: %.2f", i, r.MOS)
		}
		if r.RFactor < 0 || r.RFactor > 100 {
			t.Errorf("case %d: R out of range: %.2f", i, r.RFactor)
		}
	}
}

func TestEstimate_ImpairmentTermsAreReported(t *testing.T) {
	// Sanity: with high latency, LatencyImpair > 0. With loss,
	// LossImpairment > 0. Lets dashboards drill into the cause.
	r := Estimate(MOSInput{RTTMs: 500, PeerLossFraction: 0.10, Codec: "pcmu"})
	if r.LatencyImpair <= 0 {
		t.Errorf("LatencyImpair must be positive for high latency: %v", r)
	}
	if r.LossImpairment <= 0 {
		t.Errorf("LossImpairment must be positive for 10%% loss: %v", r)
	}
}

func TestEstimate_UnknownCodecDoesntPanic(t *testing.T) {
	// Should fall through to safe defaults.
	r := Estimate(MOSInput{Codec: "unknown-codec-xyz"})
	if r.MOS < 1 || r.MOS > 4.5 {
		t.Errorf("unknown codec: MOS out of range: %v", r.MOS)
	}
}
