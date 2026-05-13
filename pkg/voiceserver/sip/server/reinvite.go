package server

import (
	"context"
	"crypto/rand"
	"fmt"
	"net"
	"strconv"
	"strings"

	"github.com/LingByte/SoulNexus/pkg/voiceserver/logger"
	"github.com/LingByte/SoulNexus/pkg/voiceserver/sip/sdp"
	sipSession "github.com/LingByte/SoulNexus/pkg/voiceserver/sip/session"
	"github.com/LingByte/SoulNexus/pkg/voiceserver/sip/stack"
	"go.uber.org/zap"
)

// handleReInvite serves mid-dialog INVITE (hold / codec refresh / remote media refresh).
func (s *SIPServer) handleReInvite(msg *stack.Message, addr *net.UDPAddr, cs *sipSession.MediaLeg) *stack.Message {
	if s == nil || msg == nil || cs == nil || addr == nil {
		return nil
	}
	callID := strings.TrimSpace(msg.GetHeader("Call-ID"))
	toWithTag := ensureToTag(msg.GetHeader("To"))
	s.registerPendingInvite(msg, addr, toWithTag)

	rtpSess := cs.RTPSession()
	if rtpSess == nil {
		return s.makeResponse(msg, 488, "Not Acceptable Here", "", toWithTag)
	}

	if strings.TrimSpace(msg.Body) == "" {
		neg := cs.NegotiatedSDP()
		codecs := []sdp.Codec{neg}
		respSDP := sdp.GenerateWithProtoExtras(s.localIP, rtpSess.LocalAddr.Port, "RTP/AVP", codecs, nil)
		resp := s.makeResponse(msg, 200, "OK", respSDP, toWithTag)
		resp.SetHeader("Content-Type", "application/sdp")
		resp.SetHeader("To", toWithTag)
		resp.SetHeader("Contact", fmt.Sprintf("<sip:server@%s:%d>", s.localIP, s.listenPort))
		resp.SetHeader("Content-Length", strconv.Itoa(stack.BodyBytesLen(respSDP)))
		return resp
	}

	offer, err := sdp.Parse(msg.Body)
	if err != nil {
		return s.makeResponse(msg, 488, "Not Acceptable Here", "", toWithTag)
	}

	if strings.Contains(strings.ToUpper(offer.Proto), "SAVP") {
		if _, ok := sdp.PickAESCM128Offer(offer.CryptoOffers); !ok {
			return s.makeResponse(msg, 488, "Not Acceptable Here", "", toWithTag)
		}
	}

	_, negNew, err := sipSession.NegotiateOffer(offer.Codecs)
	if err != nil {
		return s.makeResponse(msg, 488, "Not Acceptable Here", "", toWithTag)
	}
	if strings.ToLower(strings.TrimSpace(negNew.Name)) != strings.ToLower(strings.TrimSpace(cs.NegotiatedSDP().Name)) {
		logger.Warn("sip re-INVITE rejected (codec change not supported)",
			zap.String("call_id", callID),
			zap.String("existing", cs.NegotiatedSDP().Name),
			zap.String("offered", negNew.Name),
		)
		return s.makeResponse(msg, 488, "Not Acceptable Here", "", toWithTag)
	}

	remoteIP := net.ParseIP(offer.IP)
	if remoteIP == nil || offer.Port <= 0 {
		return s.makeResponse(msg, 400, "Bad Request", "", toWithTag)
	}
	remoteAddr := &net.UDPAddr{IP: remoteIP, Port: offer.Port}
	if addr != nil && isPrivateIPv4(remoteIP) && addr.IP != nil && addr.IP.To4() != nil {
		if !isPrivateIPv4(addr.IP) {
			remoteAddr = &net.UDPAddr{IP: addr.IP, Port: offer.Port}
		}
	}
	rtpSess.SetRemoteAddr(remoteAddr)

	var sdpExtras []string
	if co, ok := sdp.PickAESCM128Offer(offer.CryptoOffers); ok && strings.Contains(strings.ToUpper(offer.Proto), "SAVP") {
		rk, rsalt, err := sdp.DecodeSDESInline(co.KeyParams)
		if err != nil {
			return s.makeResponse(msg, 488, "Not Acceptable Here", "", toWithTag)
		}
		lk := make([]byte, 16)
		lsalt := make([]byte, 14)
		if _, err := rand.Read(lk); err != nil {
			return s.makeResponse(msg, 500, "Internal Server Error", "", toWithTag)
		}
		if _, err := rand.Read(lsalt); err != nil {
			return s.makeResponse(msg, 500, "Internal Server Error", "", toWithTag)
		}
		cryptoLine, err := sdp.FormatCryptoLine(co.Tag, sdp.SuiteAESCM128HMACSHA180, lk, lsalt)
		if err != nil {
			return s.makeResponse(msg, 488, "Not Acceptable Here", "", toWithTag)
		}
		sdpExtras = append(sdpExtras, cryptoLine)
		if err := rtpSess.EnableSDESSRTP(rk, rsalt, lk, lsalt); err != nil {
			return s.makeResponse(msg, 500, "Internal Server Error", "", toWithTag)
		}
	}

	neg := cs.NegotiatedSDP()
	codecs := []sdp.Codec{neg}
	if te, ok := sdp.PickTelephoneEventFromOffer(offer.Codecs, neg.ClockRate); ok {
		codecs = append(codecs, te)
	}
	respSDP := sdp.GenerateWithProtoExtras(s.localIP, rtpSess.LocalAddr.Port, offer.Proto, codecs, sdpExtras)

	respMsg := s.makeResponse(msg, 200, "OK", respSDP, toWithTag)
	respMsg.SetHeader("Content-Type", "application/sdp")
	respMsg.SetHeader("To", toWithTag)
	respMsg.SetHeader("Contact", fmt.Sprintf("<sip:server@%s:%d>", s.localIP, s.listenPort))
	respMsg.SetHeader("Allow", strings.Join([]string{
		stack.MethodInvite,
		stack.MethodAck,
		stack.MethodBye,
		stack.MethodRegister,
		stack.MethodOptions,
		stack.MethodCancel,
		stack.MethodInfo,
		stack.MethodPrack,
		stack.MethodSubscribe,
		stack.MethodNotify,
		stack.MethodPublish,
		stack.MethodRefer,
		stack.MethodMessage,
		stack.MethodUpdate,
	}, ", "))
	respMsg.SetHeader("Content-Length", strconv.Itoa(stack.BodyBytesLen(respSDP)))

	if p := s.callPersistStore(); p != nil {
		bind := s.resolveInboundDIDBinding(msg)
		p.OnInvite(context.Background(), InvitePersistParams{
			TenantID:             bind.TenantID,
			InboundTrunkNumberID: bind.TrunkNumberID,
			CallID:               callID,
			From:                 msg.GetHeader("From"),
			To:                   msg.GetHeader("To"),
			RemoteSig:            addr.String(),
			RemoteRTP:            remoteAddr.String(),
			LocalRTP:             fmt.Sprintf("%s:%d", s.localIP, rtpSess.LocalAddr.Port),
			Codec:                neg.Name,
			PayloadType:          neg.PayloadType,
			ClockRate:            neg.ClockRate,
			CSeqInvite:           msg.GetHeader("CSeq"),
			Direction:            "inbound_reinvite",
		})
	}

	logger.Info("sip re-INVITE answered",
		zap.String("call_id", callID),
		zap.String("remote_rtp", remoteAddr.String()),
	)
	return respMsg
}
