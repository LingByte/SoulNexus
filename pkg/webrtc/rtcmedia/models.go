package rtcmedia

// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	media2 "github.com/LingByte/SoulNexus/pkg/media"
	"github.com/LingByte/SoulNexus/pkg/webrtc/constants"
	"github.com/pion/webrtc/v3"
	"github.com/pion/webrtc/v3/pkg/media"
	"github.com/sirupsen/logrus"
)

// WebRTCOption WebRTC 配置选项
type WebRTCOption struct {
	ICEServers []webrtc.ICEServer `json:"iceServers"` // ICE 服务器
	StreamID   string             `json:"streamId"`   // 流 ID
	ICETimeout time.Duration      `json:"iceTimeout"` // ICE 超时时间
	Codec      string             `json:"codec"`      // 编解码器名称
}

func (wts *WebRTCOption) GetICETimeout() time.Duration {
	if wts.ICETimeout == 0 {
		return constants.DefaultICETimeout
	}
	return wts.ICETimeout
}

func (wts WebRTCOption) String() string {
	return fmt.Sprintf("WebRTCOption{ICEServers: %d, StreamID: %s,ICETimeout: %v}",
		len(wts.ICEServers), wts.StreamID, wts.ICETimeout)
}

type WebRTCTransport struct {
	opt             WebRTCOption                   // WebRTC 配置
	config          webrtc.Configuration           // WebRTC配置
	peerConnection  *webrtc.PeerConnection         // WebRTC连接
	txTrack         *webrtc.TrackLocalStaticSample // 发送音频数据
	rxTrack         *webrtc.TrackRemote            // 接收音频数据
	connectionState webrtc.PeerConnectionState     // 连接状态
	codec           media2.CodecConfig
	Candidates      []webrtc.ICECandidateInit `json:"candidates"`       // ICE 候选者
	OfferSDP        string                    `json:"offer,omitempty"`  // Offer SDP
	AnswerSDP       string                    `json:"answer,omitempty"` // Answer SDP
	mu              sync.RWMutex              // 读写锁
	playAudioStop   chan struct{}             // 用于停止播放音频
}

// NewWebRTCTransport 创建新的 WebRTC 传输
func NewWebRTCTransport(opt WebRTCOption) *WebRTCTransport {
	if opt.StreamID == "" {
		opt.StreamID = constants.DefaultStreamID
	}
	if opt.ICETimeout == 0 {
		opt.ICETimeout = constants.DefaultICETimeout
	}
	if opt.Codec == "" {
		opt.Codec = constants.CodecOPUS
	}

	return &WebRTCTransport{
		opt: opt,
		config: webrtc.Configuration{
			ICEServers: opt.ICEServers,
		},
		connectionState: webrtc.PeerConnectionStateNew,
		codec: media2.CodecConfig{
			Codec:         strings.ToLower(opt.Codec),
			SampleRate:    8000,
			Channels:      1,
			BitDepth:      8,
			FrameDuration: "20ms",
		},
	}
}

// getCodecParameters 根据编解码器名称获取参数
func (wts *WebRTCTransport) getCodecParameters() webrtc.RTPCodecParameters {
	switch wts.opt.Codec {
	case constants.CodecPCMA:
		return webrtc.RTPCodecParameters{
			RTPCodecCapability: webrtc.RTPCodecCapability{MimeType: webrtc.MimeTypePCMA, ClockRate: 8000},
			PayloadType:        8,
		}
	case constants.CodecG722:
		return webrtc.RTPCodecParameters{
			RTPCodecCapability: webrtc.RTPCodecCapability{MimeType: webrtc.MimeTypeG722, ClockRate: 8000},
			PayloadType:        9,
		}
	case constants.CodecOPUS:
		return webrtc.RTPCodecParameters{
			RTPCodecCapability: webrtc.RTPCodecCapability{MimeType: webrtc.MimeTypeOpus, ClockRate: 48000},
			PayloadType:        111,
		}
	default: // pcmu
		return webrtc.RTPCodecParameters{
			RTPCodecCapability: webrtc.RTPCodecCapability{MimeType: webrtc.MimeTypePCMU, ClockRate: 8000},
			PayloadType:        0,
		}
	}
}

// GetMediaEngine 获取媒体引擎配置
func GetMediaEngine() *webrtc.MediaEngine {
	m := &webrtc.MediaEngine{}

	// 注册 G.722
	m.RegisterCodec(webrtc.RTPCodecParameters{
		RTPCodecCapability: webrtc.RTPCodecCapability{MimeType: webrtc.MimeTypeG722, ClockRate: 8000},
		PayloadType:        9,
	}, webrtc.RTPCodecTypeAudio)

	// 注册 Opus
	m.RegisterCodec(webrtc.RTPCodecParameters{
		RTPCodecCapability: webrtc.RTPCodecCapability{MimeType: webrtc.MimeTypeOpus, ClockRate: 48000},
		PayloadType:        111,
	}, webrtc.RTPCodecTypeAudio)

	// 注册 PCMU (G.711 μ-law)
	m.RegisterCodec(webrtc.RTPCodecParameters{
		RTPCodecCapability: webrtc.RTPCodecCapability{MimeType: webrtc.MimeTypePCMU, ClockRate: 8000},
		PayloadType:        0,
	}, webrtc.RTPCodecTypeAudio)

	// 注册 PCMA (G.711 A-law)
	m.RegisterCodec(webrtc.RTPCodecParameters{
		RTPCodecCapability: webrtc.RTPCodecCapability{MimeType: webrtc.MimeTypePCMA, ClockRate: 8000},
		PayloadType:        8,
	}, webrtc.RTPCodecTypeAudio)

	//telephone-event
	m.RegisterCodec(webrtc.RTPCodecParameters{
		RTPCodecCapability: webrtc.RTPCodecCapability{MimeType: "audio/telephone-event", ClockRate: 8000},
		PayloadType:        101,
	}, webrtc.RTPCodecTypeAudio)
	return m
}

// NewPeerConnection builds the PeerConnection, local send track, and default OnTrack handler.
func (wts *WebRTCTransport) NewPeerConnection() error {
	wts.mu.Lock()
	defer wts.mu.Unlock()

	api := webrtc.NewAPI(webrtc.WithMediaEngine(GetMediaEngine()))
	connection, err := api.NewPeerConnection(wts.config)
	if err != nil {
		return fmt.Errorf("webrtc: NewPeerConnection: %w", err)
	}
	wts.peerConnection = connection

	wts.peerConnection.OnICECandidate(func(i *webrtc.ICECandidate) {
		if i == nil {
			return
		}
		wts.mu.Lock()
		wts.Candidates = append(wts.Candidates, i.ToJSON())
		wts.mu.Unlock()
		if VerboseSDP() {
			logrus.WithField("candidate", i.ToJSON().Candidate).Debug("ICE candidate")
		}
	})

	wts.peerConnection.OnConnectionStateChange(func(state webrtc.PeerConnectionState) {
		wts.mu.Lock()
		wts.connectionState = state
		wts.mu.Unlock()
		logrus.WithField("state", state.String()).Info("webrtc: connection state")
		switch state {
		case webrtc.PeerConnectionStateDisconnected,
			webrtc.PeerConnectionStateFailed,
			webrtc.PeerConnectionStateClosed:
			if wts.playAudioStop != nil {
				close(wts.playAudioStop)
				wts.playAudioStop = nil
			}
		}
	})

	wts.peerConnection.OnTrack(func(remoteTrack *webrtc.TrackRemote, _ *webrtc.RTPReceiver) {
		wts.mu.Lock()
		wts.rxTrack = remoteTrack
		wts.mu.Unlock()
		logrus.WithFields(logrus.Fields{
			"codec":    remoteTrack.Codec().MimeType,
			"ssrc":     remoteTrack.SSRC(),
			"streamID": remoteTrack.StreamID(),
		}).Info("webrtc: remote track")
	})

	wts.txTrack, err = webrtc.NewTrackLocalStaticSample(
		wts.getCodecParameters().RTPCodecCapability,
		"audio",
		wts.opt.StreamID,
	)
	if err != nil {
		return fmt.Errorf("webrtc: NewTrackLocalStaticSample: %w", err)
	}
	if _, err = wts.peerConnection.AddTrack(wts.txTrack); err != nil {
		return fmt.Errorf("webrtc: AddTrack: %w", err)
	}
	return nil
}

func (wts *WebRTCTransport) Codec() media2.CodecConfig {
	return wts.codec
}

// SetRemoteDescription 设置远程描述
// 支持两种格式：
// 1. JSON 格式的 SessionDescription: {"type":"offer","sdp":"v=0\r\n..."}
// 2. 纯 SDP 字符串: "v=0\r\n..."
func (wts *WebRTCTransport) SetRemoteDescription(sdp string) error {
	var sessionDescription webrtc.SessionDescription

	err := json.Unmarshal([]byte(sdp), &sessionDescription)
	if err != nil {
		sessionDescription = webrtc.SessionDescription{
			Type: webrtc.SDPTypeOffer,
			SDP:  sdp,
		}
	}

	if VerboseSDP() && sessionDescription.SDP != "" {
		preview := sessionDescription.SDP
		if len(preview) > 300 {
			preview = preview[:300] + "..."
		}
		logrus.WithField("has_audio", strings.Contains(sessionDescription.SDP, "m=audio")).Debug(preview)
	}

	if wts.peerConnection == nil {
		return errors.New("webrtc: peer connection is nil")
	}
	return wts.peerConnection.SetRemoteDescription(sessionDescription)
}

func (wts *WebRTCTransport) CreateOffer() (offer string, candidates []string, err error) {
	if wts.peerConnection == nil {
		return "", nil, errors.New("peer connection is nil")
	}
	wts.mu.Lock()
	defer wts.mu.Unlock()

	// 创建 offer
	offerSDP, err := wts.peerConnection.CreateOffer(nil)
	if err != nil {
		logrus.WithError(err).Error("Failed to create offer")
		return
	}

	// 设置本地描述
	err = wts.peerConnection.SetLocalDescription(offerSDP)
	if err != nil {
		logrus.WithError(err).Error("Failed to set local description")
		return
	}

	// 等待 ICE gathering（短超时即可，后续由 trickle 补候选，避免阻塞数十秒才返回 SDP）
	gatherWait := wts.opt.ICETimeout
	if gatherWait > constants.ICEGatherSignalingMaxWait {
		gatherWait = constants.ICEGatherSignalingMaxWait
	}
	if gatherWait <= 0 {
		gatherWait = constants.ICEGatherSignalingMaxWait
	}
	gatherComplete := webrtc.GatheringCompletePromise(wts.peerConnection)
	select {
	case <-time.After(gatherWait):
		logrus.WithField("timeout", gatherWait).Warn("ICE gathering timeout, continue with partial candidates")
	case <-gatherComplete:
	}

	if len(wts.Candidates) == 0 {
		logrus.Warn("no ICE candidates generated yet, continue with SDP only")
	}

	// 提取 candidate 字符串
	for _, c := range wts.Candidates {
		candidates = append(candidates, c.Candidate)
	}

	// 获取 offer SDP 字符串（不需要 JSON 序列化，直接返回 SDP 字符串）
	localOfferSDP := wts.peerConnection.LocalDescription()
	offer = localOfferSDP.SDP
	wts.OfferSDP = offer

	// 安全地截取 offer 用于日志（避免越界）
	offerPreview := offer
	if len(offer) > 50 {
		offerPreview = offer[:50] + "..."
	}
	logrus.WithFields(logrus.Fields{
		constants.WebRTCOffer:     offerPreview,
		constants.WebRTCCandidate: len(candidates),
	}).Info("Offer generated")

	return offer, candidates, nil
}

func (wts *WebRTCTransport) CreateAnswer(clientCandidates []string) (serverAnswer string, serverCandidates []string, err error) {
	if wts.peerConnection == nil {
		return "", nil, errors.New("peer connection is nil")
	}

	wts.mu.Lock()
	defer wts.mu.Unlock()

	// 创建 answer
	answerSDP, err := wts.peerConnection.CreateAnswer(nil)
	if err != nil {
		logrus.WithError(err).Error("Failed to create answer")
		return
	}

	// 设置本地描述
	err = wts.peerConnection.SetLocalDescription(answerSDP)
	if err != nil {
		logrus.WithError(err).Error("Failed to set local description")
		return
	}

	gatherWait := wts.opt.ICETimeout
	if gatherWait > constants.ICEGatherSignalingMaxWait {
		gatherWait = constants.ICEGatherSignalingMaxWait
	}
	if gatherWait <= 0 {
		gatherWait = constants.ICEGatherSignalingMaxWait
	}
	gatherComplete := webrtc.GatheringCompletePromise(wts.peerConnection)
	select {
	case <-time.After(gatherWait):
		logrus.WithField("timeout", gatherWait).Warn("ICE gathering timeout while creating answer, continue with partial candidates")
	case <-gatherComplete:
	}

	if len(wts.Candidates) == 0 {
		logrus.Warn("no server ICE candidates generated yet, continue with SDP only")
	}

	// 添加客户端的 ICE candidates
	for _, c := range clientCandidates {
		err = wts.peerConnection.AddICECandidate(webrtc.ICECandidateInit{Candidate: c})
		if err != nil {
			logrus.WithError(err).WithField("candidate", c).Warn("Failed to add ICE candidate")
		}
	}

	// 提取服务器的 candidates
	for _, c := range wts.Candidates {
		serverCandidates = append(serverCandidates, c.Candidate)
	}

	// 获取 answer SDP 字符串（不需要 JSON 序列化，直接返回 SDP 字符串）
	localSDP := wts.peerConnection.LocalDescription()
	serverAnswer = localSDP.SDP
	wts.AnswerSDP = serverAnswer

	// 安全地截取 answer 用于日志（避免越界）
	answerPreview := serverAnswer
	if len(serverAnswer) > 50 {
		answerPreview = serverAnswer[:50] + "..."
	}
	logrus.WithFields(logrus.Fields{
		constants.WebRTCAnswer:    answerPreview,
		constants.WebRTCCandidate: len(serverCandidates),
	}).Info("Answer generated")

	return serverAnswer, serverCandidates, nil
}

func (wts *WebRTCTransport) SelectPreferredCodec() (*media2.CodecConfig, error) {
	sdp, err := wts.peerConnection.LocalDescription().Unmarshal()
	if err != nil {
		logrus.WithField("transport", wts).WithError(err).Error("webrtc: Failed to unmarshal Local description")
		return nil, err
	}

	codec := media2.CodecConfig{}
	for _, m := range sdp.MediaDescriptions {
		if m.MediaName.Media == string(webrtc.MediaKindAudio) {
			for _, attr := range m.Attributes {
				if attr.Key == "rtpmap" {
					if strings.HasPrefix(attr.Value, m.MediaName.Formats[0]) {
						vals := strings.Split(attr.Value, " ")[1]
						codec.Codec = strings.ToLower(strings.Split(vals, "/")[0])
						codec.SampleRate, _ = strconv.Atoi(strings.Split(vals, "/")[1])
						codec.Channels = 1
						codec.BitDepth = 8
						codec.FrameDuration = "20ms"
						return &codec, nil
					}
				}
			}
		}
	}
	return nil, fmt.Errorf("webrtc: did not find codec in SDP")
}

// AddICECandidate 添加 ICE 候选者
func (wts *WebRTCTransport) AddICECandidate(candidate string) error {
	wts.mu.Lock()
	defer wts.mu.Unlock()

	return wts.peerConnection.AddICECandidate(webrtc.ICECandidateInit{Candidate: candidate})
}

// GetConnectionState 获取连接状态
func (wts *WebRTCTransport) GetConnectionState() webrtc.PeerConnectionState {
	wts.mu.RLock()
	defer wts.mu.RUnlock()

	if wts.peerConnection == nil {
		return webrtc.PeerConnectionStateNew
	}

	return wts.peerConnection.ConnectionState()
}

// GetRxTrack 获取接收轨道 (线程安全)
func (wts *WebRTCTransport) GetRxTrack() *webrtc.TrackRemote {
	wts.mu.RLock()
	defer wts.mu.RUnlock()
	return wts.rxTrack
}

// GetTxTrack 获取发送轨道 (线程安全)
func (wts *WebRTCTransport) GetTxTrack() *webrtc.TrackLocalStaticSample {
	wts.mu.RLock()
	defer wts.mu.RUnlock()
	return wts.txTrack
}

func (wts *WebRTCTransport) Next(ctx context.Context) (media2.MediaPacket, error) {
	wts.mu.RLock()
	pc := wts.peerConnection
	rx := wts.rxTrack
	wts.mu.RUnlock()

	if rx == nil {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(10 * time.Millisecond):
		}
		return nil, nil
	}
	if pc != nil && pc.ConnectionState() != webrtc.PeerConnectionStateConnected {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(5 * time.Millisecond):
		}
		return nil, nil
	}

	rtpPacket, _, err := rx.ReadRTP()
	if err != nil {
		logrus.WithField("transport", wts).WithError(err).Error("webrtc: Error reading RTP packet")
		return nil, err
	}
	return &media2.AudioPacket{
		Payload: rtpPacket.Payload,
	}, nil
}

func (wts *WebRTCTransport) Send(frame media2.MediaPacket) (int, error) {
	if wts.peerConnection == nil || wts.txTrack == nil {
		return 0, nil
	}
	switch frame.(type) {
	case *media2.AudioPacket:
	default:
		return 0, nil
	}

	audioFrame := frame.(*media2.AudioPacket)
	duration := len(audioFrame.Body()) / GetSampleSize(wts.codec.SampleRate, wts.codec.BitDepth, wts.codec.Channels)
	sample := media.Sample{
		Data:     audioFrame.Body(),
		Duration: time.Duration(duration) * time.Millisecond,
	}
	wts.txTrack.WriteSample(sample)
	return len(frame.Body()), nil
}

func (wts *WebRTCTransport) Close() error {
	if wts.txTrack != nil {
		wts.txTrack = nil
	}
	if wts.rxTrack != nil {
		wts.rxTrack = nil
	}
	if wts.peerConnection != nil {
		wts.peerConnection.Close()
		wts.peerConnection = nil
	}
	return nil
}

// GetSampleSize returns the size of an audio sample in bytes.
func GetSampleSize(sampleRate, bitDepth, channels int) int {
	return sampleRate * bitDepth / 1000 / 8
}

// OnTrack replaces the default OnTrack handler. Call before SetRemoteDescription / signaling if you need custom behavior.
func (wts *WebRTCTransport) OnTrack(f func(*webrtc.TrackRemote, *webrtc.RTPReceiver)) {
	wts.mu.Lock()
	defer wts.mu.Unlock()
	if wts.peerConnection == nil || f == nil {
		return
	}
	wts.peerConnection.OnTrack(func(remoteTrack *webrtc.TrackRemote, receiver *webrtc.RTPReceiver) {
		wts.mu.Lock()
		wts.rxTrack = remoteTrack
		wts.mu.Unlock()
		f(remoteTrack, receiver)
	})
}

// VerboseSDP enables SDP preview and ICE debug logs (set WEBRTC_DEBUG=1).
func VerboseSDP() bool {
	return strings.TrimSpace(os.Getenv("WEBRTC_DEBUG")) == "1" ||
		strings.EqualFold(strings.TrimSpace(os.Getenv("WEBRTC_DEBUG")), "true")
}
