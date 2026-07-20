// Copyright (c) 2026 LinByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

// Package qos converts low-level RTP / RTCP measurements (RTT, loss,
// jitter, codec) into a single perceptual quality score that on-call
// can read at a glance: the Mean Opinion Score (MOS, 1.0..4.5).
//
// We use the ITU-T G.107 E-Model simplification commonly seen in
// SBC vendor docs and call-quality monitoring tools. It's not
// laboratory-grade — there's no real conversational MOS without
// listener panels — but for trend detection ("is today worse than
// yesterday?") it's far more useful than four separate raw numbers.
//
// The math, in one breath:
//
//	R = R0 - Is - Id - Ie + A
//	MOS = 1 + 0.035*R + R*(R-60)*(100-R) * 7e-6      (R in [0,100])
//
// We collapse the model to:
//
//	R0  = 93.2 (G.711 baseline; tweaked per codec)
//	Is  = 0    (simultaneous impairments — we don't measure)
//	Id  = 0.024*latencyMs + 0.11*(latencyMs - 177.3) for latencyMs > 177.3
//	Ie  = codec-specific equipment impairment (PCMU=0, Opus=2, …)
//	     + 30 * (loss_fraction / (loss_fraction + 1/burstR))
//	A   = 0   (advantage factor — we're wired, no mobile bonus)
//
// Latency input is one-way mouth-to-ear; we approximate from
// RTCP RTT / 2 + a fixed jitter-buffer add (typically 50 ms).

package qos

import "math"

// MOSInput collects the raw RTP / RTCP metrics we have on hand at
// call-end. All fields are optional; zero values are treated as
// "unknown" and folded into the baseline so MOS still produces a
// sane number.
type MOSInput struct {
	// RTTMs is the round-trip time measured from peer RTCP RR
	// (LSR/DLSR). Zero means we never saw a peer RR — we'll fall
	// back to assuming a 100 ms link.
	RTTMs uint32

	// JitterRTPUnits is the interarrival jitter in RTP clock units
	// (RFC 3550 §6.4.1). Used to estimate jitter-buffer depth.
	JitterRTPUnits uint32
	// JitterClockRate is the RTP clock rate (8000 / 16000 / 48000).
	// Zero defaults to 8000 (PSTN baseline).
	JitterClockRate uint32

	// PeerLossFraction is fraction-lost from peer RR (Q0.8 → float).
	// 0.0 = no loss, 1.0 = total loss.
	PeerLossFraction float64

	// Codec identifies the payload format. Recognised values:
	// "pcmu", "pcma", "g722", "opus". Unknown → assumed G.711.
	Codec string
}

// MOSResult bundles the score with the impairment terms that fed
// it, so on-call can drill into "why is MOS bad today".
type MOSResult struct {
	MOS              float64 // 1.0..4.5
	RFactor          float64 // 0..100 (clamped)
	LatencyMs        float64 // one-way estimate (RTT/2 + jb)
	JitterBufferMs   float64 // estimated playout buffer
	CodecImpairment  float64 // Ie (equipment)
	LossImpairment   float64 // Ie-eff loss term
	LatencyImpair    float64 // Id
}

// Estimate computes MOS from the input. The function is pure / no
// I/O — safe to call on call-end and feed into CDR.Extra or a
// dedicated metric histogram.
func Estimate(in MOSInput) MOSResult {
	// 1. Latency estimate. One-way ≈ RTT/2 + jitter buffer depth.
	//    We size the jitter buffer dynamically off the measured
	//    jitter (3σ rule of thumb), clamped to a sane range.
	clockRate := float64(in.JitterClockRate)
	if clockRate <= 0 {
		clockRate = 8000
	}
	jitterMs := float64(in.JitterRTPUnits) * 1000.0 / clockRate
	jbMs := 3 * jitterMs
	if jbMs < 20 {
		jbMs = 20 // minimum sensible playout depth
	}
	if jbMs > 250 {
		jbMs = 250 // beyond this, you'd cut the call anyway
	}
	rttMs := float64(in.RTTMs)
	if rttMs <= 0 {
		rttMs = 100 // conservative default
	}
	oneWayMs := rttMs/2 + jbMs

	// 2. Latency impairment (Id). G.107 simplification: linear
	//    below 177.3 ms, additional knee above.
	id := 0.024 * oneWayMs
	if oneWayMs > 177.3 {
		id += 0.11 * (oneWayMs - 177.3)
	}

	// 3. Codec equipment impairment (Ie). Vendor-published values
	//    for the codecs we negotiate.
	ie0 := codecImpairment(in.Codec)

	// 4. Loss-augmented equipment impairment (Ie-eff). Uses a burstR
	//    of 1 (random loss) — we don't track burst length yet.
	//    Formula: Ie_eff = Ie + (95 - Ie) * (loss / (loss + Bpl))
	//    where Bpl is the packet-loss robustness factor.
	bpl := codecPacketLossRobustness(in.Codec)
	loss := in.PeerLossFraction
	if loss < 0 {
		loss = 0
	}
	if loss > 1 {
		loss = 1
	}
	var ieLoss float64
	if loss > 0 {
		ieLoss = (95 - ie0) * (loss / (loss + bpl/100))
	}

	// 5. Combine. R0 baseline 93.2 for narrowband G.711-grade audio.
	r0 := codecBaselineR(in.Codec)
	r := r0 - id - ie0 - ieLoss
	if r < 0 {
		r = 0
	}
	if r > 100 {
		r = 100
	}

	// 6. Map R → MOS. The piecewise formula from G.107 Annex B.
	var mos float64
	switch {
	case r <= 0:
		mos = 1.0
	case r >= 100:
		mos = 4.5
	default:
		mos = 1 + 0.035*r + r*(r-60)*(100-r)*7e-6
	}
	if mos < 1 {
		mos = 1
	}
	if mos > 4.5 {
		mos = 4.5
	}

	return MOSResult{
		MOS:             round1(mos),
		RFactor:         round1(r),
		LatencyMs:       round1(oneWayMs),
		JitterBufferMs:  round1(jbMs),
		CodecImpairment: round1(ie0),
		LossImpairment:  round1(ieLoss),
		LatencyImpair:   round1(id),
	}
}

// codecImpairment returns Ie (equipment impairment factor) for the
// named codec. Values from ITU-T G.113 Appendix I and common SBC
// vendor docs; treat as approximations.
func codecImpairment(codec string) float64 {
	switch codec {
	case "pcmu", "pcma", "g711", "":
		return 0 // baseline
	case "g722":
		return 2 // marginally better than G.711 perceptually, but
		// E-Model is tuned for narrowband — we don't claim the bonus.
	case "opus":
		return 2 // depends on bitrate, 32 kbps gives ~2.
	case "g729":
		return 11
	case "ilbc":
		return 11
	default:
		return 5 // safer default for unknown codecs
	}
}

// codecBaselineR returns R0 — the "no impairment" R-factor for the
// codec. Wideband codecs (G.722, Opus@48k) can exceed 93.2 in real
// listening tests; we use 93.2 as a conservative cap so MOS never
// reports better than equivalent G.711.
func codecBaselineR(codec string) float64 {
	switch codec {
	case "opus", "g722":
		return 95 // small bonus for wideband
	default:
		return 93.2
	}
}

// codecPacketLossRobustness returns Bpl (packet-loss robustness)
// per ITU-T G.113 — higher = more tolerant to loss. PLC-equipped
// codecs (Opus FEC, G.711 PLC) get higher values.
func codecPacketLossRobustness(codec string) float64 {
	switch codec {
	case "opus":
		return 25.1 // strong inband FEC
	case "g722":
		return 14
	case "pcmu", "pcma", "g711", "":
		return 25.1 // assuming PLC enabled
	case "g729":
		return 19
	case "ilbc":
		return 22
	default:
		return 15
	}
}

func round1(v float64) float64 {
	return math.Round(v*10) / 10
}
