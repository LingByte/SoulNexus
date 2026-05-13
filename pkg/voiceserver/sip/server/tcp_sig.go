package server

import (
	"bufio"
	"context"
	"crypto/tls"
	"net"
	"strings"
	"time"

	"github.com/LingByte/SoulNexus/pkg/voiceserver/logger"
	"github.com/LingByte/SoulNexus/pkg/voiceserver/sip/stack"
	"go.uber.org/zap"
)

// startSigTransportListeners starts optional SIP-over-TCP / SIP-over-TLS
// listeners using SIPServer.cfg (no env reads).
func (s *SIPServer) startSigTransportListeners() {
	if s == nil || s.sigCtx == nil {
		return
	}
	ctx := s.sigCtx
	if tcpAddr := strings.TrimSpace(s.cfg.SIPTCPListen); tcpAddr != "" {
		go s.listenTCP(ctx, tcpAddr)
		if logger.Lg != nil {
			logger.Lg.Info("sip tcp listener", zap.String("addr", tcpAddr))
		}
	}
	tlsAddr := strings.TrimSpace(s.cfg.SIPTLSListen)
	cert := strings.TrimSpace(s.cfg.SIPTLSCertFile)
	key := strings.TrimSpace(s.cfg.SIPTLSKeyFile)
	if tlsAddr != "" && cert != "" && key != "" {
		go s.listenTLS(ctx, tlsAddr, cert, key)
		if logger.Lg != nil {
			logger.Lg.Info("sip tls listener", zap.String("addr", tlsAddr))
		}
	}
}

func (s *SIPServer) listenTCP(ctx context.Context, addr string) {
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		if logger.Lg != nil {
			logger.Lg.Warn("sip tcp listen failed", zap.String("addr", addr), zap.Error(err))
		}
		return
	}
	go func() {
		<-ctx.Done()
		_ = ln.Close()
	}()
	tcpLn, _ := ln.(*net.TCPListener)
	for {
		select {
		case <-ctx.Done():
			return
		default:
		}
		if tcpLn != nil {
			_ = tcpLn.SetDeadline(time.Now().Add(2 * time.Second))
		}
		conn, err := ln.Accept()
		if err != nil {
			if ne, ok := err.(net.Error); ok && ne.Timeout() {
				continue
			}
			if ctx.Err() != nil {
				return
			}
			continue
		}
		go s.runOneTCPConn(ctx, conn)
	}
}

func (s *SIPServer) listenTLS(ctx context.Context, addr, certFile, keyFile string) {
	cert, err := tls.LoadX509KeyPair(certFile, keyFile)
	if err != nil {
		if logger.Lg != nil {
			logger.Lg.Warn("sip tls cert load failed", zap.Error(err))
		}
		return
	}
	plain, err := net.Listen("tcp", addr)
	if err != nil {
		if logger.Lg != nil {
			logger.Lg.Warn("sip tls listen failed", zap.String("addr", addr), zap.Error(err))
		}
		return
	}
	go func() {
		<-ctx.Done()
		_ = plain.Close()
	}()
	tcpLn, _ := plain.(*net.TCPListener)
	tlsCfg := &tls.Config{Certificates: []tls.Certificate{cert}}
	ln := tls.NewListener(plain, tlsCfg)
	for {
		select {
		case <-ctx.Done():
			return
		default:
		}
		if tcpLn != nil {
			_ = tcpLn.SetDeadline(time.Now().Add(2 * time.Second))
		}
		conn, err := ln.Accept()
		if err != nil {
			if ne, ok := err.(net.Error); ok && ne.Timeout() {
				continue
			}
			if ctx.Err() != nil {
				return
			}
			continue
		}
		go s.runOneTCPConn(ctx, conn)
	}
}

func (s *SIPServer) runOneTCPConn(ctx context.Context, conn net.Conn) {
	defer func() { _ = conn.Close() }()
	ra, ok := conn.RemoteAddr().(*net.TCPAddr)
	if !ok {
		return
	}
	udpAddr := &net.UDPAddr{IP: ra.IP, Port: ra.Port}
	br := bufio.NewReader(conn)
	for {
		select {
		case <-ctx.Done():
			return
		default:
		}
		_ = conn.SetReadDeadline(time.Now().Add(90 * time.Second))
		msg, err := stack.ReadMessage(br)
		if err != nil {
			return
		}
		if !msg.IsRequest {
			if s.ep != nil {
				s.ep.InvokeOnSIPResponse(msg, udpAddr)
			}
			continue
		}
		s.dispatchSignalingRequestTCP(msg, udpAddr, func(resp *stack.Message) error {
			_, err := conn.Write([]byte(resp.String()))
			return err
		})
	}
}

func (s *SIPServer) dispatchSignalingRequestTCP(req *stack.Message, addr *net.UDPAddr, send func(*stack.Message) error) {
	if s == nil || s.ep == nil || req == nil || send == nil {
		return
	}
	resp := s.ep.DispatchRequest(req, addr)
	if resp != nil {
		_ = send(resp)
	}
}
