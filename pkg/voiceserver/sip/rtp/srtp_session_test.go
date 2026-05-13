package rtp

import (
	"bytes"
	"testing"

	"github.com/pion/srtp/v2"
)

func TestSRTP_AES128CM_EncryptDecrypt(t *testing.T) {
	key := bytes.Repeat([]byte{0xAB}, 16)
	salt := bytes.Repeat([]byte{0xCD}, 14)
	prof := srtp.ProtectionProfileAes128CmHmacSha1_80

	enc, err := srtp.CreateContext(key, salt, prof)
	if err != nil {
		t.Fatal(err)
	}
	dec, err := srtp.CreateContext(key, salt, prof)
	if err != nil {
		t.Fatal(err)
	}

	p := &RTPPacket{
		Header: RTPHeader{
			Version:        2,
			PayloadType:    0,
			SequenceNumber: 501,
			Timestamp:      3000,
			SSRC:           0x11223344,
		},
		Payload: []byte{0x01, 0x02, 0x03},
	}
	plain, err := p.Marshal()
	if err != nil {
		t.Fatal(err)
	}

	cipher, err := enc.EncryptRTP(nil, plain, nil)
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Equal(cipher, plain) {
		t.Fatal("expected ciphertext to differ from plaintext")
	}

	out, err := dec.DecryptRTP(nil, cipher, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(out, plain) {
		t.Fatalf("round-trip mismatch\nplain=%x\nout=%x", plain, out)
	}
}
