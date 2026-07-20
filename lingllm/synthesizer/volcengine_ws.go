package synthesizer

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/sirupsen/logrus"
)

const volcengineWSURL = "wss://openspeech.bytedance.com/api/v1/tts/ws_binary"

type volcengineWSResponse struct {
	MessageType              int
	MessageTypeSpecificFlags int
	MessageCompression       int
	SequenceNumber           int
	Audio                    []byte
	IsLast                   bool
	ErrorCode                int
	ErrorMessage             string
	Timestamp                SentenceTimestamp
}

func (v *volcengineSpeechSynthesisListener) sendStreamRequest(ctx context.Context, opt VolcengineTTSOption, text string) (SentenceTimestamp, error) {
	if text == "" {
		return SentenceTimestamp{}, nil
	}

	payload := buildVolcengineWSRequest(opt, text, optSubmit)
	compressed := gzipCompress(payload)
	req := make([]byte, len(defaultHeader))
	copy(req, defaultHeader)
	sizeBuf := make([]byte, 4)
	binary.BigEndian.PutUint32(sizeBuf, uint32(len(compressed)))
	req = append(req, sizeBuf...)
	req = append(req, compressed...)

	var (
		conn     *websocket.Conn
		dialMs   int64
		fromWarm bool
		err      error
	)
	if v.pool != nil {
		conn, dialMs, fromWarm, err = v.pool.Take(ctx)
	} else {
		startDial := time.Now()
		dialer := websocket.Dialer{HandshakeTimeout: 10 * time.Second}
		conn, _, err = dialer.DialContext(ctx, volcengineWSURL, map[string][]string{
			"Authorization": {fmt.Sprintf("Bearer;%s", opt.AccessToken)},
		})
		dialMs = time.Since(startDial).Milliseconds()
	}
	if err != nil {
		return SentenceTimestamp{}, err
	}
	defer conn.Close()

	if err := conn.WriteMessage(websocket.BinaryMessage, req); err != nil {
		return SentenceTimestamp{}, err
	}

	var ts SentenceTimestamp
	start := time.Now()
	var ttfbLogged bool

	for {
		select {
		case <-ctx.Done():
			return ts, ctx.Err()
		default:
		}

		if err := conn.SetReadDeadline(time.Now().Add(30 * time.Second)); err != nil {
			return ts, err
		}
		_, message, err := conn.ReadMessage()
		if err != nil {
			if websocket.IsCloseError(err, websocket.CloseNormalClosure, websocket.CloseGoingAway) {
				break
			}
			return ts, err
		}

		resp, err := parseVolcengineWSMessage(message)
		if err != nil {
			logrus.WithError(err).Debug("volcengine tts ws: skip frame")
			continue
		}
		if resp.MessageType == 15 {
			return ts, fmt.Errorf("volcengine tts ws error: code=%d msg=%s", resp.ErrorCode, resp.ErrorMessage)
		}
		if resp.MessageType == 12 && len(resp.Timestamp.Words) > 0 {
			ts = resp.Timestamp
		}
		if len(resp.Audio) > 0 {
			if !ttfbLogged {
				ttfbLogged = true
				logrus.WithFields(logrus.Fields{
					"text":      text,
					"ttfb_ms":   time.Since(start).Milliseconds(),
					"dial_ms":   dialMs,
					"from_warm": fromWarm,
					"voice":     opt.VoiceType,
				}).Info("volcengine tts ws: first audio chunk")
			}
			v.handler.OnMessage(resp.Audio)
		}
		if resp.IsLast {
			break
		}
	}
	if !ttfbLogged {
		return ts, fmt.Errorf("volcengine tts ws: no audio for text=%q voice=%s", text, opt.VoiceType)
	}
	return ts, nil
}

func buildVolcengineWSRequest(opt VolcengineTTSOption, text, operation string) []byte {
	reqID := uuid.NewString()
	params := map[string]map[string]interface{}{
		"app": {
			"appid":   opt.AppID,
			"token":   "access_token",
			"cluster": opt.Cluster,
		},
		"user": {"uid": "uid"},
		"audio": {
			"voice_type":   opt.VoiceType,
			"encoding":     opt.Encoding,
			"speed_ratio":  opt.SpeedRatio,
			"volume_ratio": opt.VolumeRatio,
			"pitch_ratio":  opt.PitchRatio,
		},
		"request": {
			"reqid":          reqID,
			"text":           text,
			"text_type":      "plain",
			"operation":      operation,
			"with_timestamp": "1",
		},
	}
	if strings.HasPrefix(text, SsmlSpeak) {
		params["request"]["text_type"] = "ssml"
	}
	if opt.Rate > 0 {
		params["audio"]["rate"] = opt.Rate
	}
	out, _ := json.Marshal(params)
	return out
}

func parseVolcengineWSMessage(message []byte) (*volcengineWSResponse, error) {
	if len(message) < 4 {
		return nil, errors.New("message too short")
	}
	r := &volcengineWSResponse{}
	headerSize := int(message[0] & 0x0f)
	r.MessageType = int((message[1] & 0xf0) >> 4)
	r.MessageTypeSpecificFlags = int(message[1] & 0x0f)
	r.MessageCompression = int(message[2] & 0x0f)
	payload := message[headerSize*4:]

	switch r.MessageType {
	case 11:
		if r.MessageTypeSpecificFlags == 0 {
			return r, nil
		}
		if len(payload) < 8 {
			return nil, errors.New("audio payload too short")
		}
		seq := int32(binary.BigEndian.Uint32(payload[:4]))
		size := int(binary.BigEndian.Uint32(payload[4:8]))
		r.SequenceNumber = int(seq)
		payload = payload[8:]
		if size > 0 && len(payload) >= size {
			r.Audio = payload[:size]
		}
		if seq < 0 {
			r.IsLast = true
		}
		return r, nil
	case 15:
		if len(payload) < 8 {
			return nil, errors.New("error payload too short")
		}
		code := int(binary.BigEndian.Uint32(payload[:4]))
		msgSize := int(binary.BigEndian.Uint32(payload[4:8]))
		msg := payload[8:]
		if len(msg) < msgSize {
			return nil, errors.New("error message size mismatch")
		}
		if r.MessageCompression == 1 {
			gr, err := gzip.NewReader(bytes.NewReader(msg[:msgSize]))
			if err != nil {
				return nil, err
			}
			unzipped, err := io.ReadAll(gr)
			_ = gr.Close()
			if err != nil {
				return nil, err
			}
			msg = unzipped
		} else {
			msg = msg[:msgSize]
		}
		r.ErrorCode = code
		r.ErrorMessage = string(msg)
		return r, nil
	case 12:
		if len(payload) < 4 {
			return nil, errors.New("frontend payload too short")
		}
		msgSize := int(binary.BigEndian.Uint32(payload[:4]))
		msg := payload[4:]
		if len(msg) < msgSize {
			return nil, errors.New("frontend size mismatch")
		}
		if r.MessageCompression == 1 {
			gr, err := gzip.NewReader(bytes.NewReader(msg[:msgSize]))
			if err != nil {
				return nil, err
			}
			unzipped, err := io.ReadAll(gr)
			_ = gr.Close()
			if err != nil {
				return nil, err
			}
			msg = unzipped
		} else {
			msg = msg[:msgSize]
		}
		var addition VolcAddition
		if err := json.Unmarshal(msg, &addition); err == nil && addition.Frontend != "" {
			_ = json.Unmarshal([]byte(addition.Frontend), &r.Timestamp)
		}
		return r, nil
	default:
		return r, nil
	}
}
