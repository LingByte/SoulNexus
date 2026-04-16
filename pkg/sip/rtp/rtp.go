package rtp

import (
	"encoding/binary"
	"fmt"
)

// RTPHeader RTP header fields (RFC 3550).
//
// Notes:
// - This implementation supports:
//   - CSRC list (CC)
//   - header extension (X)
//   - padding (P)
// - It does not attempt to interpret extension payloads beyond carrying raw bytes.
type RTPHeader struct {
	Version        uint8
	Padding        bool
	Extension      bool
	CSRCCount      uint8
	Marker         bool
	PayloadType    uint8
	SequenceNumber uint16
	Timestamp      uint32
	SSRC           uint32
}

// RTPPacket RTP packet containing an RTP header and payload.
type RTPPacket struct {
	Header RTPHeader

	// CSRC identifiers (0..15 entries).
	CSRC []uint32

	// If Header.Extension is true, these fields represent the extension header.
	ExtensionProfile uint16
	ExtensionPayload []byte // raw extension bytes

	// Payload is the RTP payload after removing CSRC/extension/padding.
	Payload []byte
}

// Marshal serializes RTPPacket into a byte slice.
func (p *RTPPacket) Marshal() ([]byte, error) {
	if p == nil {
		return nil, fmt.Errorf("rtp: nil packet")
	}

	if p.Header.Version == 0 {
		p.Header.Version = 2
	}

	cc := p.Header.CSRCCount & 0x0F
	if len(p.CSRC) > 0 {
		if len(p.CSRC) > 15 {
			return nil, fmt.Errorf("rtp: too many CSRC entries: %d", len(p.CSRC))
		}
		cc = uint8(len(p.CSRC))
	}

	extLenBytes := 0
	if p.Header.Extension {
		if len(p.ExtensionPayload)%4 != 0 {
			return nil, fmt.Errorf("rtp: extension payload must be multiple of 4 bytes, got=%d", len(p.ExtensionPayload))
		}
		extLenBytes = 4 + len(p.ExtensionPayload) // profile(2)+len(2) + payload
	}

	payloadLen := 0
	if p.Payload != nil {
		payloadLen = len(p.Payload)
	}
	headerLen := 12 + int(cc)*4 + extLenBytes
	buf := make([]byte, headerLen+payloadLen)

	// First byte:
	// V(2bit) + P(1bit) + X(1bit) + CC(4bit)
	buf[0] = (p.Header.Version&0x03)<<6 | (cc & 0x0F)
	if p.Header.Padding {
		buf[0] |= 0x20
	}
	if p.Header.Extension {
		buf[0] |= 0x10
	}

	// Second byte:
	// M(1bit) + PT(7bit)
	buf[1] = p.Header.PayloadType & 0x7F
	if p.Header.Marker {
		buf[1] |= 0x80
	}

	// Sequence number
	binary.BigEndian.PutUint16(buf[2:4], p.Header.SequenceNumber)
	// Timestamp
	binary.BigEndian.PutUint32(buf[4:8], p.Header.Timestamp)
	// SSRC
	binary.BigEndian.PutUint32(buf[8:12], p.Header.SSRC)

	offset := 12
	// CSRC list
	for i := 0; i < int(cc); i++ {
		var v uint32
		if i < len(p.CSRC) {
			v = p.CSRC[i]
		}
		binary.BigEndian.PutUint32(buf[offset:offset+4], v)
		offset += 4
	}

	// Header extension
	if p.Header.Extension {
		binary.BigEndian.PutUint16(buf[offset:offset+2], p.ExtensionProfile)
		// length in 32-bit words
		binary.BigEndian.PutUint16(buf[offset+2:offset+4], uint16(len(p.ExtensionPayload)/4))
		offset += 4
		copy(buf[offset:], p.ExtensionPayload)
		offset += len(p.ExtensionPayload)
	}

	// Payload
	copy(buf[offset:], p.Payload)

	return buf, nil
}

// Unmarshal parses RTP packet bytes into the RTPPacket.
func (p *RTPPacket) Unmarshal(data []byte) error {
	if p == nil {
		return fmt.Errorf("rtp: nil packet")
	}
	if len(data) < 12 {
		return fmt.Errorf("rtp: packet too short: %d bytes", len(data))
	}

	// First byte
	p.Header.Version = (data[0] >> 6) & 0x03
	p.Header.Padding = (data[0] & 0x20) != 0
	p.Header.Extension = (data[0] & 0x10) != 0
	p.Header.CSRCCount = data[0] & 0x0F

	// Second byte
	p.Header.Marker = (data[1] & 0x80) != 0
	p.Header.PayloadType = data[1] & 0x7F

	// Fixed header fields
	p.Header.SequenceNumber = binary.BigEndian.Uint16(data[2:4])
	p.Header.Timestamp = binary.BigEndian.Uint32(data[4:8])
	p.Header.SSRC = binary.BigEndian.Uint32(data[8:12])

	offset := 12

	// CSRC list
	cc := int(p.Header.CSRCCount & 0x0F)
	if len(data) < offset+cc*4 {
		return fmt.Errorf("rtp: packet too short for CSRC list: len=%d cc=%d", len(data), cc)
	}
	if cc > 0 {
		p.CSRC = make([]uint32, cc)
		for i := 0; i < cc; i++ {
			p.CSRC[i] = binary.BigEndian.Uint32(data[offset : offset+4])
			offset += 4
		}
	} else {
		p.CSRC = nil
	}

	// Header extension
	if p.Header.Extension {
		if len(data) < offset+4 {
			return fmt.Errorf("rtp: packet too short for extension header")
		}
		p.ExtensionProfile = binary.BigEndian.Uint16(data[offset : offset+2])
		extLenWords := binary.BigEndian.Uint16(data[offset+2 : offset+4])
		offset += 4
		extLenBytes := int(extLenWords) * 4
		if len(data) < offset+extLenBytes {
			return fmt.Errorf("rtp: packet too short for extension payload: need=%d have=%d", extLenBytes, len(data)-offset)
		}
		if extLenBytes > 0 {
			p.ExtensionPayload = make([]byte, extLenBytes)
			copy(p.ExtensionPayload, data[offset:offset+extLenBytes])
			offset += extLenBytes
		} else {
			p.ExtensionPayload = nil
		}
	} else {
		p.ExtensionProfile = 0
		p.ExtensionPayload = nil
	}

	// Payload (handle padding if present)
	end := len(data)
	if p.Header.Padding {
		if end == 0 {
			return fmt.Errorf("rtp: padding bit set on empty packet")
		}
		padLen := int(data[end-1])
		if padLen <= 0 || padLen > end-offset {
			return fmt.Errorf("rtp: invalid padding length: %d", padLen)
		}
		end -= padLen
	}
	if end < offset {
		return fmt.Errorf("rtp: invalid payload range: offset=%d end=%d", offset, end)
	}
	p.Payload = data[offset:end]
	return nil
}

