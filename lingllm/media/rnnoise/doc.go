// Package rnnoise wraps the Xiph RNNoise C library (librnnoise) for 48 kHz PCM16
// noise suppression. It does not use third-party Go bindings; link against the
// system or vendored rnnoise library via CGO.

// By default the package builds as a stub. To link Xiph librnnoise, use
//
//	go build -tags rnnoise
//
// with CGO enabled and rnnoise headers/libs installed (see README.md).
package rnnoise
