//go:build ledenoise

package denoise

/*
#cgo CFLAGS: -I${SRCDIR}/../native/ledenoise/include
#cgo darwin LDFLAGS: ${SRCDIR}/../native/ledenoise/target/release/libledenoise.a -ldl -lm -lpthread
#cgo linux LDFLAGS: ${SRCDIR}/../native/ledenoise/target/release/libledenoise.a -ldl -lm -lpthread

#include "ledenoise.h"
*/
import "C"

import (
	"fmt"
	"runtime"
	"sync"
	"unsafe"
)

// SNREnabled reports whether the native SNR estimator is linked.
func SNREnabled() bool { return true }

// LedenoiseSNR wraps Rust time-domain noise-floor SNR (ld_snr_*).
type LedenoiseSNR struct {
	h          *C.LdSnr
	sampleRate int
	mu         sync.Mutex
}

// NewLedenoiseSNR opens a native SNR estimator for mono PCM16.
func NewLedenoiseSNR(sampleRate int) (*LedenoiseSNR, error) {
	if sampleRate <= 0 {
		sampleRate = 8000
	}
	h := C.ld_snr_open(C.uint32_t(sampleRate))
	if h == nil {
		return nil, fmt.Errorf("ledenoise snr: open rate=%d failed", sampleRate)
	}
	s := &LedenoiseSNR{h: h, sampleRate: sampleRate}
	runtime.SetFinalizer(s, (*LedenoiseSNR).Close)
	return s, nil
}

// Process feeds one PCM16 frame; returns smoothed SNR dB and readiness.
func (s *LedenoiseSNR) Process(samples []int16) (snrDB float64, ready bool) {
	if s == nil || s.h == nil || len(samples) == 0 {
		return 0, false
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	var out C.float
	rc := C.ld_snr_process(
		s.h,
		(*C.int16_t)(unsafe.Pointer(&samples[0])),
		C.size_t(len(samples)),
		&out,
	)
	if rc < 0 {
		return 0, false
	}
	return float64(out), rc == 1
}

// Close releases native resources.
func (s *LedenoiseSNR) Close() error {
	if s == nil || s.h == nil {
		return nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	runtime.SetFinalizer(s, nil)
	C.ld_snr_close(s.h)
	s.h = nil
	return nil
}
