package synthesizer

// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/sirupsen/logrus"
)

// volcWSPool keeps pre-dialed WebSockets ready so Speak avoids a cold
// TLS+handshake on the hot path. Each utterance still uses a dedicated
// connection (Volcengine submit is one-shot per conn); after Take we dial
// replacements in the background.
//
// targetWarm=2 covers early-flush first segment + rest segment so the
// second Speak of a turn is also usually warm.
type volcWSPool struct {
	token      string
	targetWarm int

	mu      sync.Mutex
	warm    []*websocket.Conn
	warming int
	closed  bool
}

func newVolcWSPool(accessToken string) *volcWSPool {
	return &volcWSPool{token: accessToken, targetWarm: 2}
}

func (p *volcWSPool) dial(ctx context.Context) (*websocket.Conn, error) {
	if p == nil {
		return nil, fmt.Errorf("volcengine tts: nil ws pool")
	}
	dialer := websocket.Dialer{HandshakeTimeout: 10 * time.Second}
	conn, _, err := dialer.DialContext(ctx, volcengineWSURL, map[string][]string{
		"Authorization": {fmt.Sprintf("Bearer;%s", p.token)},
	})
	return conn, err
}

// Take returns a connected WebSocket. Prefer a warm conn; otherwise dial now.
// dialMs is the blocking dial latency (0 when a warm conn was used).
func (p *volcWSPool) Take(ctx context.Context) (conn *websocket.Conn, dialMs int64, fromWarm bool, err error) {
	if p == nil {
		return nil, 0, false, fmt.Errorf("volcengine tts: nil ws pool")
	}
	p.mu.Lock()
	if p.closed {
		p.mu.Unlock()
		return nil, 0, false, fmt.Errorf("volcengine tts: ws pool closed")
	}
	n := len(p.warm)
	if n > 0 {
		conn = p.warm[n-1]
		p.warm = p.warm[:n-1]
		p.mu.Unlock()
		go p.ensureWarm()
		return conn, 0, true, nil
	}
	p.mu.Unlock()
	start := time.Now()
	conn, err = p.dial(ctx)
	dialMs = time.Since(start).Milliseconds()
	if err == nil {
		go p.ensureWarm()
	}
	return conn, dialMs, false, err
}

// Prewarm starts background dials until targetWarm connections are ready.
func (p *volcWSPool) Prewarm() {
	if p == nil {
		return
	}
	go p.ensureWarm()
}

func (p *volcWSPool) ensureWarm() {
	if p == nil {
		return
	}
	for {
		p.mu.Lock()
		if p.closed {
			p.mu.Unlock()
			return
		}
		need := p.targetWarm - len(p.warm) - p.warming
		if need <= 0 {
			p.mu.Unlock()
			return
		}
		p.warming++
		p.mu.Unlock()

		ctx, cancel := context.WithTimeout(context.Background(), 12*time.Second)
		conn, err := p.dial(ctx)
		cancel()

		p.mu.Lock()
		p.warming--
		if p.closed {
			p.mu.Unlock()
			if conn != nil {
				_ = conn.Close()
			}
			return
		}
		if err != nil {
			p.mu.Unlock()
			logrus.WithError(err).Debug("volcengine tts ws: prewarm dial failed")
			return
		}
		p.warm = append(p.warm, conn)
		p.mu.Unlock()
	}
}

func (p *volcWSPool) Close() {
	if p == nil {
		return
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	p.closed = true
	for _, c := range p.warm {
		if c != nil {
			_ = c.Close()
		}
	}
	p.warm = nil
}
