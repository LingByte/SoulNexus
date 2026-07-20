// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

//go:build !lingcodec

package encoder

// LECodecEnabled is false without -tags lingcodec.
func LECodecEnabled() bool { return false }

// LECodecVersion is empty without the lecodec backend.
func LECodecVersion() string { return "" }

const (
	lePCMU           uint8 = 0
	lePCMA           uint8 = 8
	leG722           uint8 = 9
	leG729           uint8 = 18
	leTelephoneEvent uint8 = 101
	leOpus           uint8 = 111
)

const (
	opusAppVoIP     = 0
	opusAppAudio    = 1
	opusAppLowDelay = 2
)

type leEnc struct{}
type leDec struct{}

func openLEEnc(_ uint8) (*leEnc, error) { return nil, ErrLECodecUnavailable }
func openOpusEnc(_ uint32, _ uint16, _ int) (*leEnc, error) {
	return nil, ErrLECodecUnavailable
}
func (e *leEnc) Close()                                      {}
func (e *leEnc) Encode(_ []int16) ([]byte, error)            { return nil, ErrLECodecUnavailable }
func (e *leEnc) EncodeInto(_ []int16, _ []byte) (int, error) { return 0, ErrLECodecUnavailable }

func openLEDec(_ uint8) (*leDec, error)             { return nil, ErrLECodecUnavailable }
func openOpusDec(_ uint32, _ uint16) (*leDec, error)    { return nil, ErrLECodecUnavailable }
func (d *leDec) Close()                                 {}
func (d *leDec) Decode(_ []byte) ([]int16, error)       { return nil, ErrLECodecUnavailable }

func samplesToLE(_ []int16) []byte { return nil }
func leToSamples(_ []byte) []int16 { return nil }
