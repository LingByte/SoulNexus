// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

//go:build lingcodec

package encoder

/*
#cgo CFLAGS: -I${SRCDIR}/lecodec/include
#cgo darwin LDFLAGS: ${SRCDIR}/lecodec/target/release/liblecodec.a -framework Security -ldl -lm -lpthread
#cgo linux LDFLAGS: ${SRCDIR}/lecodec/target/release/liblecodec.a -ldl -lm -lpthread

#include "lecodec.h"
#include <stdlib.h>
*/
import "C"

import (
	"encoding/binary"
	"fmt"
	"runtime"
	"unsafe"
)

// LECodecEnabled reports whether the Rust lecodec backend is linked.
func LECodecEnabled() bool { return true }

// LECodecVersion returns the linked lecodec crate version.
func LECodecVersion() string { return C.GoString(C.le_version()) }

// RTP payload-type style IDs for the lecodec backend.
const (
	lePCMU           = uint8(C.LE_CODEC_PCMU)
	lePCMA           = uint8(C.LE_CODEC_PCMA)
	leG722           = uint8(C.LE_CODEC_G722)
	leG729           = uint8(C.LE_CODEC_G729)
	leTelephoneEvent = uint8(C.LE_CODEC_TELEPHONE_EVENT)
	leOpus           = uint8(C.LE_CODEC_OPUS)
)

const (
	opusAppVoIP     = int(C.LE_OPUS_VOIP)
	opusAppAudio    = int(C.LE_OPUS_AUDIO)
	opusAppLowDelay = int(C.LE_OPUS_LOWDELAY)
)

type leEnc struct {
	h *C.LeEncoder
}

type leDec struct {
	h *C.LeDecoder
}

func openLEEnc(codec uint8) (*leEnc, error) {
	h := C.le_encoder_open(C.uint8_t(codec))
	if h == nil {
		return nil, fmt.Errorf("%w: codec=%d", errLECreate, codec)
	}
	e := &leEnc{h: h}
	runtime.SetFinalizer(e, (*leEnc).Close)
	return e, nil
}

func openOpusEnc(rate uint32, ch uint16, app int) (*leEnc, error) {
	h := C.le_opus_encoder_open(C.uint32_t(rate), C.uint16_t(ch), C.int(app))
	if h == nil {
		return nil, errLECreate
	}
	e := &leEnc{h: h}
	runtime.SetFinalizer(e, (*leEnc).Close)
	return e, nil
}

func (e *leEnc) Close() {
	if e == nil || e.h == nil {
		return
	}
	runtime.SetFinalizer(e, nil)
	C.le_encoder_close(e.h)
	e.h = nil
}

func (e *leEnc) Encode(samples []int16) ([]byte, error) {
	if e == nil || e.h == nil {
		return nil, errLENil
	}
	var out *C.uint8_t
	var outLen C.size_t
	var sp *C.int16_t
	if n := len(samples); n > 0 {
		sp = (*C.int16_t)(unsafe.Pointer(&samples[0]))
	}
	ret := C.le_encoder_encode(e.h, sp, C.size_t(len(samples)), &out, &outLen)
	if ret != C.LE_OK {
		return nil, fmt.Errorf("%w: code=%d", errLEEncode, int(ret))
	}
	if out == nil || outLen == 0 {
		return []byte{}, nil
	}
	defer C.le_free(out, outLen)
	return C.GoBytes(unsafe.Pointer(out), C.int(outLen)), nil
}

func (e *leEnc) EncodeInto(samples []int16, buf []byte) (int, error) {
	if e == nil || e.h == nil {
		return 0, errLENil
	}
	if len(buf) == 0 {
		return 0, errLEBuf
	}
	var sp *C.int16_t
	if len(samples) > 0 {
		sp = (*C.int16_t)(unsafe.Pointer(&samples[0]))
	}
	ret := C.le_encoder_encode_into(e.h, sp, C.size_t(len(samples)),
		(*C.uint8_t)(unsafe.Pointer(&buf[0])), C.size_t(len(buf)))
	if ret == C.LE_ERR_BUF {
		return 0, errLEBuf
	}
	if ret < 0 {
		return 0, fmt.Errorf("%w: code=%d", errLEEncode, int(ret))
	}
	return int(ret), nil
}

func openLEDec(codec uint8) (*leDec, error) {
	h := C.le_decoder_open(C.uint8_t(codec))
	if h == nil {
		return nil, fmt.Errorf("%w: codec=%d", errLECreate, codec)
	}
	d := &leDec{h: h}
	runtime.SetFinalizer(d, (*leDec).Close)
	return d, nil
}

func openOpusDec(rate uint32, ch uint16) (*leDec, error) {
	h := C.le_opus_decoder_open(C.uint32_t(rate), C.uint16_t(ch))
	if h == nil {
		return nil, errLECreate
	}
	d := &leDec{h: h}
	runtime.SetFinalizer(d, (*leDec).Close)
	return d, nil
}

func (d *leDec) Close() {
	if d == nil || d.h == nil {
		return
	}
	runtime.SetFinalizer(d, nil)
	C.le_decoder_close(d.h)
	d.h = nil
}

func (d *leDec) Decode(data []byte) ([]int16, error) {
	if d == nil || d.h == nil {
		return nil, errLENil
	}
	var out *C.int16_t
	var outLen C.size_t
	var dp *C.uint8_t
	if n := len(data); n > 0 {
		dp = (*C.uint8_t)(unsafe.Pointer(&data[0]))
	}
	ret := C.le_decoder_decode(d.h, dp, C.size_t(len(data)), &out, &outLen)
	if ret != C.LE_OK {
		return nil, fmt.Errorf("%w: code=%d", errLEDecode, int(ret))
	}
	if out == nil || outLen == 0 {
		return []int16{}, nil
	}
	defer C.le_free_samples(out, outLen)
	samples := make([]int16, int(outLen))
	copy(samples, unsafe.Slice((*int16)(unsafe.Pointer(out)), int(outLen)))
	return samples, nil
}

func samplesToLE(samples []int16) []byte {
	out := make([]byte, len(samples)*2)
	for i, s := range samples {
		binary.LittleEndian.PutUint16(out[i*2:], uint16(s))
	}
	return out
}

func leToSamples(data []byte) []int16 {
	if len(data)%2 != 0 {
		data = data[:len(data)-1]
	}
	out := make([]int16, len(data)/2)
	for i := range out {
		out[i] = int16(binary.LittleEndian.Uint16(data[i*2:]))
	}
	return out
}
