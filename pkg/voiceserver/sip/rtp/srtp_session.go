package rtp

import (
	"fmt"

	"github.com/pion/srtp/v2"
)

// EnableSDESSRTP configures AES_CM_128_HMAC_SHA1_80 SRTP using RFC 4568 SDES material:
// decrypt inbound RTP using the peer's offered inline key (their RTP toward us),
// encrypt outbound RTP using our answer inline key.
func (s *Session) EnableSDESSRTP(peerOutboundKey, peerOutboundSalt, localOutboundKey, localOutboundSalt []byte) error {
	if s == nil {
		return fmt.Errorf("rtp: nil session")
	}
	prof := srtp.ProtectionProfileAes128CmHmacSha1_80
	rx, err := srtp.CreateContext(peerOutboundKey, peerOutboundSalt, prof)
	if err != nil {
		return fmt.Errorf("rtp: srtp rx context: %w", err)
	}
	tx, err := srtp.CreateContext(localOutboundKey, localOutboundSalt, prof)
	if err != nil {
		return fmt.Errorf("rtp: srtp tx context: %w", err)
	}
	s.srtpMu.Lock()
	s.srtpDecrypt = rx
	s.srtpEncrypt = tx
	s.srtpMu.Unlock()
	return nil
}
