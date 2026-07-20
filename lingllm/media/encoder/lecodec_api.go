// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

//go:build lingcodec

package encoder

import "sync"

// EncodePCMA encodes little-endian PCM16 mono to G.711 A-law.
func EncodePCMA(pcmData []byte) ([]byte, error) {
	return Pcm2pcma(pcmData)
}

type pooledEnc struct{ enc *leEnc }
type pooledDec struct{ dec *leDec }

var (
	poolPCMAEnc = sync.Pool{New: func() any {
		e, err := openLEEnc(lePCMA)
		if err != nil {
			return (*pooledEnc)(nil)
		}
		return &pooledEnc{enc: e}
	}}
	poolPCMUDec = sync.Pool{New: func() any {
		d, err := openLEDec(lePCMU)
		if err != nil {
			return (*pooledDec)(nil)
		}
		return &pooledDec{dec: d}
	}}
	poolPCMADec = sync.Pool{New: func() any {
		d, err := openLEDec(lePCMA)
		if err != nil {
			return (*pooledDec)(nil)
		}
		return &pooledDec{dec: d}
	}}
)

// Pcm2pcma encodes little-endian PCM16 mono to G.711 A-law.
func Pcm2pcma(pcmData []byte) ([]byte, error) {
	p, _ := poolPCMAEnc.Get().(*pooledEnc)
	if p == nil || p.enc == nil {
		enc, err := openLEEnc(lePCMA)
		if err != nil {
			return nil, err
		}
		defer enc.Close()
		return enc.Encode(leToSamples(pcmData))
	}
	defer poolPCMAEnc.Put(p)
	return p.enc.Encode(leToSamples(pcmData))
}

// DecodePCMU decodes G.711 μ-law to little-endian PCM16 mono.
func DecodePCMU(ulawData []byte) ([]byte, error) {
	p, _ := poolPCMUDec.Get().(*pooledDec)
	if p == nil || p.dec == nil {
		dec, err := openLEDec(lePCMU)
		if err != nil {
			return nil, err
		}
		defer dec.Close()
		samples, err := dec.Decode(ulawData)
		if err != nil {
			return nil, err
		}
		return samplesToLE(samples), nil
	}
	defer poolPCMUDec.Put(p)
	samples, err := p.dec.Decode(ulawData)
	if err != nil {
		return nil, err
	}
	return samplesToLE(samples), nil
}

// DecodePCMA decodes G.711 A-law to little-endian PCM16 mono.
func DecodePCMA(alawData []byte) ([]byte, error) {
	p, _ := poolPCMADec.Get().(*pooledDec)
	if p == nil || p.dec == nil {
		dec, err := openLEDec(lePCMA)
		if err != nil {
			return nil, err
		}
		defer dec.Close()
		samples, err := dec.Decode(alawData)
		if err != nil {
			return nil, err
		}
		return samplesToLE(samples), nil
	}
	defer poolPCMADec.Put(p)
	samples, err := p.dec.Decode(alawData)
	if err != nil {
		return nil, err
	}
	return samplesToLE(samples), nil
}

const (
	G722_RATE_DEFAULT = 64000
	G722_DEFAULT      = 0
)

// G722Encoder wraps the native G.722 encoder.
type G722Encoder struct {
	enc *leEnc
	mu  sync.Mutex
}

// G722Decoder wraps the native G.722 decoder.
type G722Decoder struct {
	dec *leDec
	mu  sync.Mutex
}

// NewG722Encoder creates a G.722 encoder (rate/mode kept for API compat).
func NewG722Encoder(_, _ int) *G722Encoder {
	enc, err := openLEEnc(leG722)
	if err != nil {
		panic(err)
	}
	return &G722Encoder{enc: enc}
}

// Encode compresses little-endian PCM16 @ 16 kHz to G.722.
func (e *G722Encoder) Encode(pcmData []byte) []byte {
	if e == nil || e.enc == nil || len(pcmData) == 0 {
		return nil
	}
	e.mu.Lock()
	defer e.mu.Unlock()
	out, err := e.enc.Encode(leToSamples(pcmData))
	if err != nil {
		return nil
	}
	return out
}

// NewG722Decoder creates a G.722 decoder.
func NewG722Decoder(_, _ int) *G722Decoder {
	dec, err := openLEDec(leG722)
	if err != nil {
		panic(err)
	}
	return &G722Decoder{dec: dec}
}

// Decode expands G.722 to little-endian PCM16 @ 16 kHz.
func (d *G722Decoder) Decode(g722Data []byte) []byte {
	if d == nil || d.dec == nil || len(g722Data) == 0 {
		return nil
	}
	d.mu.Lock()
	defer d.mu.Unlock()
	samples, err := d.dec.Decode(g722Data)
	if err != nil {
		return nil
	}
	return samplesToLE(samples)
}
