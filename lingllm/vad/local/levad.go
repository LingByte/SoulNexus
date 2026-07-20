//go:build levad

package local

/*
#cgo CFLAGS: -I${SRCDIR}/../native/levad/include
#cgo darwin LDFLAGS: ${SRCDIR}/../native/levad/target/release/liblevad.a -ldl -lm -lpthread
#cgo linux LDFLAGS: ${SRCDIR}/../native/levad/target/release/liblevad.a -ldl -lm -lpthread

#include "levad.h"
*/
import "C"

import (
	"encoding/binary"
	"fmt"
	"runtime"
	"sync"
	"unsafe"
)

func LevadEnabled() bool { return true }

func LevadVersion() string { return C.GoString(C.lv_version()) }

const (
	LevadKindSilero = uint8(C.LV_KIND_SILERO)
	LevadKindTen    = uint8(C.LV_KIND_TEN)
)

// LevadVAD wraps TinySilero or TinyTen from liblevad.
type LevadVAD struct {
	h          *C.LvVad
	sampleRate int
	kind       uint8
	mu         sync.Mutex
}

// NewLevadVAD opens a native VAD instance (requires -tags levad and built liblevad).
func NewLevadVAD(kind uint8, sampleRate int, threshold float32) (*LevadVAD, error) {
	if sampleRate <= 0 {
		sampleRate = 16000
	}
	h := C.lv_vad_open(C.uint8_t(kind), C.uint32_t(sampleRate), C.float(threshold))
	if h == nil {
		return nil, fmt.Errorf("levad: open kind=%d rate=%d failed (need 16kHz for TinyTen)", kind, sampleRate)
	}
	v := &LevadVAD{h: h, sampleRate: sampleRate, kind: kind}
	runtime.SetFinalizer(v, (*LevadVAD).Close)
	return v, nil
}

// NewTinySileroVAD creates TinySilero (512 samples @ 16kHz per decision).
func NewTinySileroVAD(sampleRate int) (*LevadVAD, error) {
	return NewLevadVAD(LevadKindSilero, sampleRate, 0.5)
}

// NewTinyTenVAD creates TinyTen (hop 256 @ 16kHz; requires 16kHz).
func NewTinyTenVAD(sampleRate int) (*LevadVAD, error) {
	if sampleRate > 0 && sampleRate != 16000 {
		return nil, fmt.Errorf("levad tinyten: only 16000 Hz supported, got %d", sampleRate)
	}
	return NewLevadVAD(LevadKindTen, 16000, 0.5)
}

// IsSpeech returns true when any classified chunk in pcm is speech.
func (v *LevadVAD) IsSpeech(pcm []byte) (bool, error) {
	if v == nil || v.h == nil || len(pcm) < 2 {
		return false, nil
	}
	n := len(pcm) / 2
	samples := make([]int16, n)
	for i := 0; i < n; i++ {
		samples[i] = int16(binary.LittleEndian.Uint16(pcm[i*2:]))
	}
	v.mu.Lock()
	defer v.mu.Unlock()
	var speech C.int
	rc := C.lv_vad_process(v.h, (*C.int16_t)(unsafe.Pointer(&samples[0])), C.size_t(n), &speech)
	switch rc {
	case C.LV_OK:
		return speech != 0, nil
	case C.LV_NEED_MORE:
		return false, nil
	case C.LV_ERR_PANIC:
		return false, fmt.Errorf("levad: panic in process")
	default:
		return false, fmt.Errorf("levad: process rc=%d", int(rc))
	}
}

// Close releases native resources.
func (v *LevadVAD) Close() error {
	if v == nil || v.h == nil {
		return nil
	}
	v.mu.Lock()
	defer v.mu.Unlock()
	runtime.SetFinalizer(v, nil)
	C.lv_vad_close(v.h)
	v.h = nil
	return nil
}
