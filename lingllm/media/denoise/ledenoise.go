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
	"encoding/binary"
	"fmt"
	"runtime"
	"sync"
	"unsafe"
)

func LedenoiseEnabled() bool { return true }

func LedenoiseVersion() string { return C.GoString(C.ld_version()) }

// Ledenoise wraps native RNNoise (nnnoiseless) with resample to 48 kHz.
type Ledenoise struct {
	h          *C.LdDenoise
	sampleRate int
	mu         sync.Mutex
	outScratch []int16
}

// NewLedenoise opens a streaming denoise instance for mono PCM16 at sampleRate.
func NewLedenoise(sampleRate int) (*Ledenoise, error) {
	if sampleRate <= 0 {
		sampleRate = 16000
	}
	h := C.ld_denoise_open(C.uint32_t(sampleRate))
	if h == nil {
		return nil, fmt.Errorf("ledenoise: open rate=%d failed", sampleRate)
	}
	d := &Ledenoise{h: h, sampleRate: sampleRate}
	runtime.SetFinalizer(d, (*Ledenoise).Close)
	return d, nil
}

// Process denoises mono PCM16LE bytes. Returns a new slice (may differ in length).
func (d *Ledenoise) Process(pcm []byte) []byte {
	if d == nil || d.h == nil || len(pcm) < 2 {
		return pcm
	}
	n := len(pcm) / 2
	samples := make([]int16, n)
	for i := 0; i < n; i++ {
		samples[i] = int16(binary.LittleEndian.Uint16(pcm[i*2:]))
	}
	d.mu.Lock()
	defer d.mu.Unlock()
	// Resample may expand slightly; give headroom.
	capN := n*3 + 64
	if len(d.outScratch) < capN {
		d.outScratch = make([]int16, capN)
	}
	rc := C.ld_denoise_process(
		d.h,
		(*C.int16_t)(unsafe.Pointer(&samples[0])),
		C.size_t(n),
		(*C.int16_t)(unsafe.Pointer(&d.outScratch[0])),
		C.size_t(capN),
	)
	if rc < 0 {
		return pcm
	}
	outN := int(rc)
	out := make([]byte, outN*2)
	for i := 0; i < outN; i++ {
		binary.LittleEndian.PutUint16(out[i*2:], uint16(d.outScratch[i]))
	}
	return out
}

// Close releases native resources.
func (d *Ledenoise) Close() error {
	if d == nil || d.h == nil {
		return nil
	}
	d.mu.Lock()
	defer d.mu.Unlock()
	runtime.SetFinalizer(d, nil)
	C.ld_denoise_close(d.h)
	d.h = nil
	return nil
}
