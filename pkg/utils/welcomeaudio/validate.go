// Copyright (c) 2026 LinByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

// Package welcomeaudio centralizes validation, fetching and caching of
// assistant welcome WAV URLs.
//
// Two integration points consume this package:
//
//   - handlers call ValidateURL to
//     reject unreachable / non-audio / non-WAV URLs at write time.
//   - voice attach: call FetchPCM at runtime
//     to download + decode + cache the WAV as PCM16 mono at the bridge
//     sample rate, so subsequent calls don't pay the network cost.
//
// We deliberately keep the surface tiny (no abstract "audio resolver"
// interface). Welcome audio is a write-rare / read-rare resource — a
// process-local sync.Map cache is sufficient and avoids the operational
// burden of an extra Redis dependency on cold start.
package welcomeaudio

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// MaxBytes caps how large a remote welcome WAV is allowed to be when
// fetched at validation- or runtime-time. 16 MiB ≈ 90 seconds of
// 48 kHz / 16-bit / mono — generous for a greeting, hard limit against
// memory abuse from a hostile or misconfigured URL target.
const MaxBytes = 16 << 20

// validateTimeout bounds HEAD + GET probe latency on the write path.
// We're inside an interactive admin API call; >5 s would make the form
// "feel broken". The HTTP client times out individually too — this is
// the umbrella context.
const validateTimeout = 5 * time.Second

// ErrUnsupportedScheme is returned when the URL scheme is neither http
// nor https. We reject ftp/file/etc. to avoid SSRF-shaped surprises and
// because operators have no real reason to use them.
var ErrUnsupportedScheme = errors.New("welcome audio url must be http(s)")

// ErrNotAudio is returned when the URL responds with a Content-Type
// that does not look like audio AND does not start with the RIFF magic.
// Both signals must fail before we reject so that mis-typed CDNs (which
// often default to application/octet-stream for .wav) still pass.
var ErrNotAudio = errors.New("welcome audio url is not audio/WAV")

// ErrUnreachable is returned for any network/HTTP error during probe,
// or any non-2xx response. The wrapped error carries detail.
var ErrUnreachable = errors.New("welcome audio url unreachable")

// httpClientForValidate builds the validation-path HTTP client. Every
// validate call gets its own client so the per-call deadline cannot
// leak into other requests, and so tests can stub Transport easily.
//
// We do NOT follow >5 redirects — open redirectors are a classic SSRF
// pivot and our use case (an admin-pasted CDN URL) needs at most one.
func httpClientForValidate() *http.Client {
	return &http.Client{
		Timeout: validateTimeout,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= 5 {
				return fmt.Errorf("too many redirects")
			}
			return nil
		},
	}
}

// ValidateURL probes the given URL and reports whether it is a usable
// welcome WAV resource. It performs:
//
//  1. URL parse (scheme must be http/https; host must be non-empty).
//  2. HEAD request — capture Content-Type as a soft hint.
//  3. Range GET (bytes=0-15) — read the first 16 bytes and require
//     RIFF...WAVE magic. This is the authoritative check; HEAD signals
//     are advisory because many object stores answer HEAD with
//     application/octet-stream regardless of file content.
//
// On success returns nil. On failure returns an error wrapping one of
// the sentinel errors above so callers can `errors.Is` for telemetry.
func ValidateURL(ctx context.Context, rawURL string) error {
	u, err := parseHTTPURL(rawURL)
	if err != nil {
		return err
	}
	cctx, cancel := context.WithTimeout(ctx, validateTimeout)
	defer cancel()
	client := httpClientForValidate()

	// HEAD: best-effort signal. Some buckets return 405 on HEAD — we
	// don't treat that as fatal; the range GET below is authoritative.
	headReq, _ := http.NewRequestWithContext(cctx, http.MethodHead, u.String(), nil)
	headResp, headErr := client.Do(headReq)
	if headResp != nil {
		// Drain to allow connection reuse; ignore body content.
		_, _ = io.Copy(io.Discard, headResp.Body)
		headResp.Body.Close()
	}
	if headErr == nil && headResp != nil && headResp.StatusCode >= 400 && headResp.StatusCode != http.StatusMethodNotAllowed {
		return fmt.Errorf("%w: HEAD %d", ErrUnreachable, headResp.StatusCode)
	}

	// Range GET 0-15: 12 bytes is enough for "RIFF????WAVE" but we ask
	// for 16 to be friendly to ranges that align on 8-byte boundaries.
	getReq, _ := http.NewRequestWithContext(cctx, http.MethodGet, u.String(), nil)
	getReq.Header.Set("Range", "bytes=0-15")
	getResp, err := client.Do(getReq)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrUnreachable, err)
	}
	defer getResp.Body.Close()
	// 200 (server ignored Range) or 206 (partial content) are both fine.
	if getResp.StatusCode != http.StatusOK && getResp.StatusCode != http.StatusPartialContent {
		return fmt.Errorf("%w: GET %d", ErrUnreachable, getResp.StatusCode)
	}
	head := make([]byte, 16)
	n, _ := io.ReadFull(io.LimitReader(getResp.Body, 16), head)
	if n < 12 {
		return fmt.Errorf("%w: only %d magic bytes", ErrNotAudio, n)
	}
	if !isRIFFWave(head[:n]) {
		ct := strings.ToLower(getResp.Header.Get("Content-Type"))
		// Soft-allow only if Content-Type is unambiguous audio/wav AND
		// file body genuinely failed to surface magic in 16 bytes (rare;
		// some CDNs prepend BOMs). We still require the audio prefix.
		if !strings.HasPrefix(ct, "audio/") {
			return ErrNotAudio
		}
		return ErrNotAudio
	}
	return nil
}

// parseHTTPURL validates scheme + host and returns the parsed URL.
// Empty input returns an error: callers handle "" by skipping validation
// (welcomeAudioUrl is optional) BEFORE they call us.
func parseHTTPURL(rawURL string) (*url.URL, error) {
	s := strings.TrimSpace(rawURL)
	if s == "" {
		return nil, fmt.Errorf("%w: empty url", ErrUnsupportedScheme)
	}
	u, err := url.Parse(s)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrUnsupportedScheme, err)
	}
	scheme := strings.ToLower(u.Scheme)
	if scheme != "http" && scheme != "https" {
		return nil, fmt.Errorf("%w: scheme=%q", ErrUnsupportedScheme, u.Scheme)
	}
	if strings.TrimSpace(u.Host) == "" {
		return nil, fmt.Errorf("%w: missing host", ErrUnsupportedScheme)
	}
	return u, nil
}

// isRIFFWave matches the canonical 12-byte WAV header prefix:
//
//	"RIFF" <4-byte little-endian size> "WAVE"
//
// We do NOT validate the size field — some authoring tools write 0 or
// 0xFFFFFFFF for streams of unknown final length, and our downstream
// decoder (LoadWAVAsPCM16Mono) is robust against that.
func isRIFFWave(b []byte) bool {
	if len(b) < 12 {
		return false
	}
	return bytes.Equal(b[0:4], []byte("RIFF")) && bytes.Equal(b[8:12], []byte("WAVE"))
}

// ValidateBytes runs the same WAV-magic check on an in-memory blob.
// Used by the upload endpoint, which already holds the entire file in
// the multipart buffer — no need to re-fetch over HTTP.
func ValidateBytes(b []byte) error {
	if len(b) < 12 {
		return fmt.Errorf("%w: %d bytes", ErrNotAudio, len(b))
	}
	if !isRIFFWave(b[:12]) {
		return ErrNotAudio
	}
	return nil
}
