// Package rnnoise wraps the Xiph RNNoise C library (librnnoise) for 48 kHz
// PCM16 noise suppression. It does not use third-party Go bindings; link
// against the system or vendored rnnoise library via CGO.
//
// By default the package builds as a stub which passes audio through
// unchanged — that keeps the project go-buildable on machines without
// librnnoise installed. To link the real library, build with:
//
//	go build -tags rnnoise
//
// with CGO_ENABLED=1 and rnnoise headers/libs installed:
//
//	# macOS
//	brew install rnnoise
//	# Linux (Debian/Ubuntu — package name varies; otherwise build from source)
//	apt-get install librnnoise-dev   # or: see https://github.com/xiph/rnnoise
//
// Sample rate / frame size: librnnoise expects exactly 480 samples per
// frame at 48 kHz, mono. Callers feeding pipeline PCM at a different
// rate must resample first; helpers in pkg/voice/asr handle that for
// the ASR feed path.
package rnnoise
