package synthesizer

// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

type testAudioSynthesisHandler struct {
	result []byte
}

func (h *testAudioSynthesisHandler) OnMessage(buf []byte) {
	h.result = append(h.result, buf...)
}

func (h *testAudioSynthesisHandler) OnTimestamp(timestamp SentenceTimestamp) {

}
