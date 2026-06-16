// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

package tts

import (
	"fmt"
	"strings"

	"github.com/LingByte/SoulNexus/pkg/utils"
	"github.com/LingByte/lingllm/synthesizer"
)

// VolcengineCloneEnvOption builds clone TTS options from process env (utils.GetEnv).
// Used when a dialog session selects a trained voice clone — credentials come from
// VOLCENGINE_CLONE_* rather than user API keys.
func VolcengineCloneEnvOption(assetID string) (synthesizer.VolcengineCloneOption, error) {
	assetID = strings.TrimSpace(assetID)
	if assetID == "" {
		return synthesizer.VolcengineCloneOption{}, fmt.Errorf("volcengine clone: empty assetID")
	}

	appID := strings.TrimSpace(utils.GetEnv("VOLCENGINE_CLONE_APP_ID"))
	token := strings.TrimSpace(utils.GetEnv("VOLCENGINE_CLONE_TOKEN"))
	if appID == "" || token == "" {
		return synthesizer.VolcengineCloneOption{}, fmt.Errorf("missing VOLCENGINE_CLONE_APP_ID or VOLCENGINE_CLONE_TOKEN")
	}

	cluster := strings.TrimSpace(utils.GetEnv("VOLCENGINE_CLONE_CLUSTER"))
	if cluster == "" {
		cluster = "volcano_icl"
	}

	rate := 16000
	if v := utils.GetIntEnv("VOLCENGINE_CLONE_SAMPLE_RATE"); v > 0 {
		rate = int(v)
	}
	sourceRate := 24000
	if rate >= 24000 {
		sourceRate = rate
	}

	encoding := strings.TrimSpace(utils.GetEnv("VOLCENGINE_CLONE_ENCODING"))
	if encoding == "" {
		encoding = "pcm"
	}

	speedRatio := 1.0
	if v := utils.GetFloatEnv("VOLCENGINE_CLONE_SPEED_RATIO"); v > 0 {
		speedRatio = v
	}

	opt := synthesizer.VolcengineCloneOption{
		AppID:       appID,
		AccessToken: token,
		Cluster:     cluster,
		AssetID:     assetID,
		Encoding:    encoding,
		Rate:        rate,
		SourceRate:  sourceRate,
		SpeedRatio:  speedRatio,
		Streaming:   true,
	}

	if v := strings.TrimSpace(utils.GetEnv("VOLCENGINE_CLONE_VOICE_TYPE")); v != "" {
		opt.ResourceID = v
	}

	return opt, nil
}

// NewVolcengineCloneEngineFromEnv creates a Volcengine voice-clone synthesizer using env credentials.
func NewVolcengineCloneEngineFromEnv(assetID string) (synthesizer.AudioSynthesisEngine, error) {
	opt, err := VolcengineCloneEnvOption(assetID)
	if err != nil {
		return nil, err
	}
	return synthesizer.NewVolcengineCloneEngine(opt)
}
