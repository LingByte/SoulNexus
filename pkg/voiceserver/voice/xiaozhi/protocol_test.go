// Copyright (c) 2026 LinByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

package xiaozhi

import (
	"encoding/json"
	"strings"
	"testing"
)

// roundTrip checks that the produced JSON has the expected `type` field and
// passes a basic shape contract. The xiaozhi-esp32 firmware is liberal in
// what it accepts beyond `type`, but it strictly rejects anything missing
// the type field, so that's the floor we guard.
func roundTrip(t *testing.T, raw []byte, wantType string) map[string]any {
	t.Helper()
	var out map[string]any
	if err := json.Unmarshal(raw, &out); err != nil {
		t.Fatalf("unmarshal %s: %v\nraw=%s", wantType, err, raw)
	}
	if out["type"] != wantType {
		t.Fatalf("type: got %v want %s\nraw=%s", out["type"], wantType, raw)
	}
	return out
}

func TestParseTextFrame_KnownTypes(t *testing.T) {
	cases := []struct {
		raw  string
		want string
	}{
		{`{"type":"hello","version":1}`, "hello"},
		{`{"type":"listen","state":"start","mode":"auto"}`, "listen"},
		{`{"type":"abort"}`, "abort"},
		{`{"type":"ping"}`, "ping"},
		{`{"type":"  hello  "}`, "hello"}, // trims whitespace
	}
	for _, c := range cases {
		got, err := ParseTextFrame([]byte(c.raw))
		if err != nil {
			t.Errorf("%q: %v", c.raw, err)
			continue
		}
		if got != c.want {
			t.Errorf("%q: got %q want %q", c.raw, got, c.want)
		}
	}
}

func TestParseTextFrame_BadJSON(t *testing.T) {
	if _, err := ParseTextFrame([]byte(`not-json`)); err == nil {
		t.Fatal("expected error for non-JSON payload")
	}
}

func TestMergeHelloAudio_FillsDefaults(t *testing.T) {
	ap := AudioParams{}
	MergeHelloAudio(&ap)
	d := DefaultHelloAudio()
	if ap != d {
		t.Fatalf("empty merge: got %+v want %+v", ap, d)
	}
	// Lowercase normalisation.
	ap = AudioParams{Format: "  OPUS  "}
	MergeHelloAudio(&ap)
	if ap.Format != "opus" {
		t.Fatalf("format normalise: got %q", ap.Format)
	}
}

func TestMakeWelcomeReply_ContainsSessionAndAudioParams(t *testing.T) {
	raw := MakeWelcomeReply("sess-1", DefaultHelloAudio())
	out := roundTrip(t, raw, RespHello)
	if out["session_id"] != "sess-1" {
		t.Fatalf("session_id: %v", out["session_id"])
	}
	ap, ok := out["audio_params"].(map[string]any)
	if !ok {
		t.Fatalf("audio_params missing or wrong type: %v", out["audio_params"])
	}
	if ap["sample_rate"].(float64) != 16000 {
		t.Fatalf("sample_rate: %v", ap["sample_rate"])
	}
	// Browser web clients negotiate format=pcm; verify we pass it through.
	rawWeb := MakeWelcomeReply("web-1", AudioParams{Format: "pcm", SampleRate: 16000, Channels: 1, FrameDuration: 60, BitDepth: 16})
	w := roundTrip(t, rawWeb, RespHello)
	wap := w["audio_params"].(map[string]any)
	if wap["format"] != "pcm" {
		t.Fatalf("web format: %v", wap["format"])
	}
}

func TestMakeSTTReply_ShapeMatchesFirmwareExpectation(t *testing.T) {
	raw := MakeSTTReply("s1", "你好")
	out := roundTrip(t, raw, RespSTT)
	if out["text"] != "你好" || out["session_id"] != "s1" {
		t.Fatalf("stt fields: %v", out)
	}
}

func TestMakeTTSStateReply_StartIncludesAudioParamsStopOmits(t *testing.T) {
	startRaw := MakeTTSStateReply("s1", "start", "opus")
	startOut := roundTrip(t, startRaw, RespTTS)
	if startOut["state"] != "start" {
		t.Fatalf("start state: %v", startOut["state"])
	}
	if _, ok := startOut["audio_params"]; !ok {
		t.Fatal("tts:start must carry audio_params for the firmware codec switch")
	}
	if ap, ok := startOut["audio_params"].(map[string]any); ok {
		if int(ap["frame_duration"].(float64)) != 60 {
			t.Fatalf("default frame_duration: %v", ap["frame_duration"])
		}
	}
	stopRaw := MakeTTSStateReply("s1", "stop", "opus")
	stopOut := roundTrip(t, stopRaw, RespTTS)
	if stopOut["state"] != "stop" {
		t.Fatalf("stop state: %v", stopOut["state"])
	}
	if _, ok := stopOut["audio_params"]; ok {
		t.Fatal("tts:stop must NOT carry audio_params (firmware re-uses last seen params)")
	}
	// Default codec when empty: opus.
	defaultRaw := MakeTTSStateReply("s1", "start", "")
	if !strings.Contains(string(defaultRaw), `"codec":"opus"`) {
		t.Fatalf("default codec should be opus, got %s", defaultRaw)
	}
}

func TestMakeTTSStateReplyFrames_CustomFrameMs(t *testing.T) {
	raw := MakeTTSStateReplyFrames("s1", "start", "pcm", 20)
	out := roundTrip(t, raw, RespTTS)
	ap := out["audio_params"].(map[string]any)
	if int(ap["frame_duration"].(float64)) != 20 {
		t.Fatalf("frame_duration: %v", ap["frame_duration"])
	}
}

func TestMakePongAndAbortConfirm(t *testing.T) {
	roundTrip(t, MakePongReply("s1"), RespPong)
	out := roundTrip(t, MakeAbortConfirm("s1"), RespAbortConfirm)
	if out["state"] != "confirmed" {
		t.Fatalf("abort state: %v", out["state"])
	}
}

func TestMakeError_FatalFlagPropagates(t *testing.T) {
	out := roundTrip(t, MakeError("boom", true), RespError)
	if out["fatal"] != true {
		t.Fatalf("fatal: %v", out["fatal"])
	}
	if out["message"] != "boom" {
		t.Fatalf("message: %v", out["message"])
	}
}
