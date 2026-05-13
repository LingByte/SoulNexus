// Copyright (c) 2026 LinByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

package sfu

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"go.uber.org/zap"
)

// ServeWS is the http.HandlerFunc that upgrades to WebSocket and runs
// the signaling loop for one participant. The flow is:
//
//  1. Upgrade HTTP → WS.
//  2. Read the FIRST message; must be MsgJoin.
//  3. Verify the access token, admit to room (capacity / duplicate
//     identity handled inside Room.AddParticipant).
//  4. Start the heartbeat goroutine.
//  5. Pump messages until the read errors (client closed / net dead /
//     heartbeat timeout).
//
// Every exit path closes the participant via Participant.Close which
// is idempotent and handles the room-side cleanup. Errors during the
// pre-join phase just close the WS.
func (m *Manager) ServeWS(w http.ResponseWriter, r *http.Request) {
	if m.closed.Load() {
		http.Error(w, "sfu shutting down", http.StatusServiceUnavailable)
		return
	}
	ws, err := newWSConn(w, r, m.cfg.HeartbeatInterval, m.upgrader)
	if err != nil {
		// Upgrade failure is usually the Origin check rejecting the
		// request. gorilla writes the response code itself; we just
		// log the (likely benign) reason.
		m.logger.Warn("sfu: ws upgrade", zap.Error(err))
		return
	}

	// Wait for the client's join message. We give it a bounded window
	// so an idle TCP peer that upgraded but never sent JSON can't hold
	// a goroutine forever.
	env, err := ws.readEnvelope()
	if err != nil {
		m.logger.Debug("sfu: read join", zap.Error(err))
		ws.close()
		return
	}
	if env.Type != MsgJoin {
		ws.sendError("protocol", "first message must be join")
		ws.close()
		return
	}
	var join JoinData
	if len(env.Data) > 0 {
		if err := json.Unmarshal(env.Data, &join); err != nil {
			ws.sendError("protocol", "bad join payload")
			ws.close()
			return
		}
	}

	// Verify token (unless AllowUnauthenticated bypasses it; dev only).
	var claims AccessTokenClaims
	if m.cfg.AllowUnauthenticated {
		// Fallback claims: let anyone join as whatever they requested.
		// Only useful for the bundled demo; refuse to start the manager
		// this way in production by setting AuthSecret instead.
		claims = AccessTokenClaims{
			Room:      orDefault(join.Room, "dev"),
			Identity:  orDefault(join.Name, newParticipantID()),
			Name:      join.Name,
			ExpiresAt: time.Now().Add(time.Hour).Unix(),
		}
	} else {
		tok, err := ParseAccessToken(m.cfg.AuthSecret, join.Token)
		if err != nil {
			ws.sendError("bad_token", err.Error())
			ws.close()
			return
		}
		claims = tok.Claims
	}

	room, err := m.getOrCreateRoom(claims.Room)
	if err != nil {
		code := "join_failed"
		switch {
		case errors.Is(err, ErrTooManyRooms):
			code = "too_many_rooms"
		case errors.Is(err, ErrManagerClosed):
			code = "shutting_down"
		}
		ws.sendError(code, err.Error())
		ws.close()
		return
	}
	p, err := room.AddParticipant(claims, ws)
	if err != nil {
		code := "join_failed"
		switch {
		case errors.Is(err, ErrRoomFull):
			code = "room_full"
		case errors.Is(err, ErrDuplicateIdentity):
			code = "duplicate_identity"
		}
		ws.sendError(code, err.Error())
		ws.close()
		return
	}

	// Start heartbeat. One goroutine per participant; cheap because
	// ticker-driven and does one frame write per interval.
	stop := make(chan struct{})
	go heartbeat(ws, m.cfg.HeartbeatInterval, stop)

	// Main read loop. Any error here means the WS or the PC is gone;
	// close the participant and return.
	defer func() {
		close(stop)
		p.Close("read_loop_exit")
	}()

	for {
		env, err := ws.readEnvelope()
		if err != nil {
			return
		}
		if err := m.dispatch(p, env); err != nil {
			ws.sendError("dispatch", err.Error())
			// Dispatch errors are per-message; keep the socket open so
			// clients can recover (e.g. a stale ICE candidate during a
			// restart shouldn't kill the session).
		}
	}
}

// dispatch routes one inbound envelope to the appropriate participant
// method. Unknown message types are logged and ignored so old clients
// survive future server additions without drama.
func (m *Manager) dispatch(p *Participant, env Envelope) error {
	switch env.Type {
	case MsgOffer:
		var sdp SDPData
		if err := json.Unmarshal(env.Data, &sdp); err != nil {
			return err
		}
		answer, err := p.HandleOffer(sdp.SDP)
		if err != nil {
			return err
		}
		return p.ws.sendJSON(Envelope{Type: MsgAnswer, RequestID: env.RequestID}, SDPData{SDP: answer})

	case MsgAnswer:
		var sdp SDPData
		if err := json.Unmarshal(env.Data, &sdp); err != nil {
			return err
		}
		return p.HandleAnswer(sdp.SDP)

	case MsgICECandidate:
		var cand ICECandidateData
		if err := json.Unmarshal(env.Data, &cand); err != nil {
			return err
		}
		return p.HandleICECandidate(cand)

	case MsgSetMute:
		var mute SetMuteData
		if err := json.Unmarshal(env.Data, &mute); err != nil {
			return err
		}
		p.HandleSetMute(mute.TrackID, mute.Muted)
		return nil

	case MsgLeave:
		p.Close("client_leave")
		return nil

	case MsgPong, MsgPing:
		// Application-layer ping/pong is redundant with WS-level ping/
		// pong but some clients prefer it. Treat as a no-op; the WS
		// PongHandler already refreshed the read deadline.
		return nil

	default:
		m.logger.Debug("sfu: unknown message type", zap.String("type", string(env.Type)))
		return nil
	}
}

// heartbeat pings the client every interval. Two missed pongs cause
// the read deadline to fire (configured in newWSConn) and the read
// pump to exit. Returns when `stop` fires.
func heartbeat(ws *wsConn, interval time.Duration, stop <-chan struct{}) {
	if interval <= 0 {
		return
	}
	t := time.NewTicker(interval)
	defer t.Stop()
	for {
		select {
		case <-stop:
			return
		case <-t.C:
			if err := ws.sendPing(); err != nil {
				return
			}
		}
	}
}

func orDefault(v, def string) string {
	if v == "" {
		return def
	}
	return v
}
