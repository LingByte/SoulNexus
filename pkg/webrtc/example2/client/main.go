package main

// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

import (
	"encoding/json"
	"fmt"
	"log"
	"math"
	"net/url"
	"os"
	"os/signal"
	"sync"
	"time"

	"github.com/LingByte/SoulNexus/pkg/devices"
	media2 "github.com/LingByte/SoulNexus/pkg/media"
	"github.com/LingByte/SoulNexus/pkg/media/encoder"
	"github.com/LingByte/SoulNexus/pkg/webrtc/constants"
	"github.com/LingByte/SoulNexus/pkg/webrtc/rtcmedia"
	"github.com/gen2brain/malgo"
	"github.com/gorilla/websocket"
	"github.com/pion/rtp"
	"github.com/pion/webrtc/v3"
	"github.com/pion/webrtc/v3/pkg/media"
)

// Constants
const (
	// WebSocket configuration
	wsScheme = "ws"
	wsHost   = "localhost:8080"
	wsPath   = "/websocket"

	// Connection retry configuration
	maxConnectionRetries       = 100
	connectionRetryDelay       = 100 * time.Millisecond
	connectionStateLogInterval = 10

	// Audio configuration
	// Use 16kHz for microphone capture and ASR processing.
	// Most microphones support 16kHz well (better than 8kHz).
	// The PCMA codec will handle the encoding internally.
	// QCloud ASR uses 16k_zh model which expects 16kHz signal.
	targetSampleRate = 16000
	audioChannels    = 1
	audioBitDepth    = 16

	// Audio gain (amplification factor, 1.0 = no gain, 2.0 = double volume)
	// Increase this if microphone volume is too low for ASR
	audioGain = 2.0 // 增加增益以提高 ASR 识别率

	// Logging intervals
	packetLogInterval = 100
)

// SignalMessage represents a WebSocket signaling message
type SignalMessage struct {
	Type      string      `json:"type"`
	SessionID string      `json:"session_id,omitempty"`
	Data      interface{} `json:"data,omitempty"`
}

// Client represents a WebRTC client for AI voice conversation
type Client struct {
	wsConn    *websocket.Conn
	transport *rtcmedia.WebRTCTransport
	sessionID string
	interrupt chan os.Signal
	done      chan struct{}

	// Add mutex for thread safety
	mu sync.RWMutex
	// Add done channel for audio callback
	doneChan chan struct{}

	// Audio components
	streamPlayer *devices.StreamAudioPlayer
	pcmaDecoder  media2.EncoderFunc
	pcmaEncoder  media2.EncoderFunc
	txTrack      *webrtc.TrackLocalStaticSample

	// Microphone capture
	malgoCtx      *malgo.AllocatedContext
	captureDevice *malgo.Device

	// Track if we've started receiving audio (prevent duplicate processing)
	audioReceived bool
}

// NewClient creates a new WebRTC client
func NewClient() (*Client, error) {
	// Connect to WebSocket signaling server
	u := url.URL{
		Scheme: wsScheme,
		Host:   wsHost,
		Path:   wsPath,
	}

	conn, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to WebSocket server: %w", err)
	}

	// Create WebRTC transport
	transport := rtcmedia.NewWebRTCTransport(rtcmedia.WebRTCOption{
		Codec:      constants.CodecPCMA,
		ICETimeout: constants.DefaultICETimeout,
		ICEServers: []webrtc.ICEServer{
			{URLs: []string{"stun:stun.l.google.com:19302"}},
		},
		StreamID: "lingecho_ai_client",
	})

	// Setup signal handling
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt)

	return &Client{
		wsConn:    conn,
		transport: transport,
		interrupt: interrupt,
		done:      make(chan struct{}),
	}, nil
}

// Close closes the client connection
func (c *Client) Close() error {
	// Close the done channel to signal the audio callback to stop
	c.mu.Lock()
	if c.doneChan != nil {
		close(c.doneChan)
		c.doneChan = nil
	}
	c.mu.Unlock()

	if c.captureDevice != nil {
		c.captureDevice.Stop()
		c.captureDevice.Uninit()
		c.captureDevice = nil
	}
	if c.malgoCtx != nil {
		c.malgoCtx.Uninit()
		c.malgoCtx = nil
	}
	if c.streamPlayer != nil {
		c.streamPlayer.Close()
	}
	if c.transport != nil {
		c.transport.Close()
	}
	if c.wsConn != nil {
		return c.wsConn.Close()
	}
	return nil
}

// InitializeSession initializes the WebSocket session
func (c *Client) InitializeSession() (string, error) {
	_, initMsg, err := c.wsConn.ReadMessage()
	if err != nil {
		return "", fmt.Errorf("failed to read init message: %w", err)
	}

	var initSignal SignalMessage
	if err := json.Unmarshal(initMsg, &initSignal); err != nil {
		return "", fmt.Errorf("failed to unmarshal init message: %w", err)
	}

	if initSignal.Type != "init" {
		return "", fmt.Errorf("unexpected message type: %s", initSignal.Type)
	}

	c.sessionID = initSignal.SessionID
	fmt.Printf("[Client] Connected with session ID: %s\n", c.sessionID)
	return c.sessionID, nil
}

// CreateAndSendOffer creates a WebRTC offer and sends it to the server
func (c *Client) CreateAndSendOffer() error {
	c.transport.NewPeerConnection()

	// Get txTrack (created automatically in NewPeerConnection)
	c.txTrack = c.transport.GetTxTrack()
	if c.txTrack == nil {
		return fmt.Errorf("txTrack is nil after NewPeerConnection")
	}
	fmt.Printf("[Client] txTrack created: ID=%s\n", c.txTrack.ID())

	offer, candidates, err := c.transport.CreateOffer()
	if err != nil {
		return fmt.Errorf("failed to create offer: %w", err)
	}

	fmt.Printf("[Client] Created offer with %d candidates\n", len(candidates))

	offerMsg := SignalMessage{
		Type:      "offer",
		SessionID: c.sessionID,
		Data: map[string]interface{}{
			"sdp":        offer,
			"candidates": candidates,
		},
	}

	offerBytes, err := json.Marshal(offerMsg)
	if err != nil {
		return fmt.Errorf("failed to marshal offer: %w", err)
	}

	if err := c.wsConn.WriteMessage(websocket.TextMessage, offerBytes); err != nil {
		return fmt.Errorf("failed to send offer: %w", err)
	}

	fmt.Println("[Client] Offer sent to server")
	return nil
}

// WaitForConnection waits for the WebRTC connection to be established
func (c *Client) WaitForConnection() error {
	for i := 0; i < maxConnectionRetries; i++ {
		state := c.transport.GetConnectionState()
		if state == webrtc.PeerConnectionStateConnected {
			fmt.Println("[Client] WebRTC connection established")
			return nil
		}

		if i%connectionStateLogInterval == 0 {
			fmt.Printf("[Client] Waiting for connection... (state: %s)\n", state.String())
		}

		time.Sleep(connectionRetryDelay)
	}

	return fmt.Errorf("connection timeout after %d retries", maxConnectionRetries)
}

// WaitForTrack waits for the remote audio track to be available
func (c *Client) WaitForTrack() (*webrtc.TrackRemote, error) {
	for i := 0; i < maxConnectionRetries; i++ {
		rxTrack := c.transport.GetRxTrack()
		if rxTrack != nil {
			return rxTrack, nil
		}
		time.Sleep(connectionRetryDelay)
	}

	return nil, fmt.Errorf("rxTrack not available after %d retries", maxConnectionRetries)
}

// SetupAudioPlayback sets up audio playback components
func (c *Client) SetupAudioPlayback() error {
	// Create stream player
	streamPlayer, err := devices.NewStreamAudioPlayer(
		audioChannels,
		targetSampleRate,
		malgo.FormatS16,
	)
	if err != nil {
		return fmt.Errorf("failed to create stream player: %w", err)
	}

	// Start playback
	if err := streamPlayer.Play(); err != nil {
		streamPlayer.Close()
		return fmt.Errorf("failed to start playback: %w", err)
	}

	c.streamPlayer = streamPlayer

	fmt.Printf("[Client] Audio playback started: %dHz, %d channel(s)\n",
		targetSampleRate, audioChannels)

	// PCMA codec standard sample rate (RFC 3551)
	const pcmaSampleRate = 8000

	// Create PCMA decoder (for receiving audio from server)
	// Server sends PCMA at 8kHz, we decode to PCM at 16kHz for playback
	// Flow: PCMA (8kHz, 8-bit) -> PCM (8kHz) -> resample -> PCM (16kHz, 16-bit)
	// NOTE: CreateDecode uses src.Codec to find the decoder, so src must be "pcma"
	decodeFunc, err := encoder.CreateDecode(
		media2.CodecConfig{
			Codec:         "pcma",         // Source codec - used to find decoder
			SampleRate:    pcmaSampleRate, // 8kHz - PCMA standard (source)
			Channels:      audioChannels,
			BitDepth:      8, // PCMA uses 8-bit samples
			FrameDuration: "20ms",
		},
		media2.CodecConfig{
			Codec:         "pcm",            // Target codec
			SampleRate:    targetSampleRate, // 16kHz - target for playback
			Channels:      audioChannels,
			BitDepth:      audioBitDepth, // 16-bit PCM for playback
			FrameDuration: "20ms",
		},
	)
	if err != nil {
		streamPlayer.Close()
		return fmt.Errorf("failed to create decoder: %w", err)
	}

	c.pcmaDecoder = decodeFunc

	// Create PCMA encoder (for sending audio to server)
	// Microphone captures at 16kHz, we encode to PCMA at 8kHz for WebRTC
	// Flow: PCM (16kHz, 16-bit) -> resample -> PCM (8kHz) -> PCMA (8kHz, 8-bit)
	// NOTE: CreateEncode uses src.Codec to find the encoder, so src must be "pcma"
	//       The src/dst semantics are confusing - src is the codec type, not the input format
	encodeFunc, err := encoder.CreateEncode(
		media2.CodecConfig{
			Codec:         "pcma",           // Codec type - used to find encoder
			SampleRate:    targetSampleRate, // 16kHz - microphone capture rate (input)
			Channels:      audioChannels,
			BitDepth:      audioBitDepth, // 16-bit PCM from microphone
			FrameDuration: "20ms",
		},
		media2.CodecConfig{
			Codec:         "pcma",         // Target codec
			SampleRate:    pcmaSampleRate, // 8kHz - PCMA standard (output)
			Channels:      audioChannels,
			BitDepth:      8, // PCMA uses 8-bit samples
			FrameDuration: "20ms",
		},
	)
	if err != nil {
		streamPlayer.Close()
		return fmt.Errorf("failed to create encoder: %w", err)
	}

	c.pcmaEncoder = encodeFunc
	return nil
}

// ProcessAudioPacket processes a single RTP audio packet
func (c *Client) ProcessAudioPacket(
	packet *rtp.Packet,
	packetCount int,
) error {
	payload := packet.Payload
	if len(payload) == 0 {
		return nil
	}

	// Decode PCMA to PCM
	audioPacket := &media2.AudioPacket{Payload: payload}
	decodedPackets, err := c.pcmaDecoder(audioPacket)
	if err != nil {
		if packetCount%packetLogInterval == 0 {
			fmt.Printf("[Client] Error decoding frame %d: %v\n", packetCount, err)
		}
		return err
	}

	// Collect all decoded PCM data
	var allPCMData []byte
	for _, packet := range decodedPackets {
		if af, ok := packet.(*media2.AudioPacket); ok {
			if len(af.Payload) == 0 {
				continue
			}

			// Validate PCM data (should be 16-bit, so length must be even)
			if len(af.Payload)%2 != 0 {
				if packetCount <= 3 {
					fmt.Printf("[Client] Warning: Odd PCM length at packet %d: %d bytes\n",
						packetCount, len(af.Payload))
				}
				continue
			}

			allPCMData = append(allPCMData, af.Payload...)
		}
	}

	// Write to player
	if len(allPCMData) > 0 {
		if err := c.streamPlayer.Write(allPCMData); err != nil {
			if packetCount%packetLogInterval == 0 && err.Error() != "音频缓冲区已满" {
				fmt.Printf("[Client] Error writing to player: %v\n", err)
			}
		}
	}

	return nil
}

// startAudioReceiverFromTrack starts receiving and playing audio from a specific track
func (c *Client) startAudioReceiverFromTrack(rxTrack *webrtc.TrackRemote) error {
	if rxTrack == nil {
		return fmt.Errorf("rxTrack is nil")
	}

	// SetupAudioPlayback should already be called before this
	if c.streamPlayer == nil || c.pcmaDecoder == nil {
		return fmt.Errorf("audio playback not initialized")
	}

	codec := rxTrack.Codec()
	fmt.Printf("[Client] Received track: %s, %dHz\n", codec.MimeType, codec.ClockRate)

	packetCount := 0
	for {
		packet, _, err := rxTrack.ReadRTP()
		if err != nil {
			return fmt.Errorf("error reading RTP packet: %w", err)
		}

		if err := c.ProcessAudioPacket(packet, packetCount); err != nil {
			// Continue processing even if one packet fails
			continue
		}

		packetCount++
		if packetCount%packetLogInterval == 0 {
			fmt.Printf("[Client] Received and played %d RTP packets\n", packetCount)
		}
	}
}

// StartAudioReceiver starts receiving and playing audio packets (kept for backward compatibility)
func (c *Client) StartAudioReceiver(rxTrack *webrtc.TrackRemote) error {
	return c.startAudioReceiverFromTrack(rxTrack)
}

// StartAudioSender starts capturing audio from microphone and sending via WebRTC
func (c *Client) StartAudioSender() error {
	if c.txTrack == nil {
		return fmt.Errorf("txTrack is nil")
	}

	// Initialize malgo context
	malgoCtx, err := malgo.InitContext(nil, malgo.ContextConfig{}, func(message string) {
		log.Printf("[Client] Malgo: %s", message)
	})
	if err != nil {
		return fmt.Errorf("failed to initialize malgo context: %w", err)
	}
	c.malgoCtx = malgoCtx

	// Configure capture device
	deviceConfig := malgo.DefaultDeviceConfig(malgo.Capture)
	deviceConfig.Capture.Format = malgo.FormatS16
	deviceConfig.Capture.Channels = uint32(audioChannels)
	deviceConfig.SampleRate = uint32(targetSampleRate)

	// Audio capture callback
	frameDuration := 20 * time.Millisecond
	startTime := time.Now()
	frameCount := 0

	// Wait a bit to ensure pcmaEncoder is initialized
	// SetupAudioPlayback should have been called before this, but let's verify
	c.mu.RLock()
	encoderReady := c.pcmaEncoder != nil
	txTrackReady := c.txTrack != nil
	c.mu.RUnlock()

	if !encoderReady {
		// Wait a bit for encoder to be ready
		fmt.Printf("[Client] Waiting for pcmaEncoder to be initialized...\n")
		for i := 0; i < 50; i++ {
			time.Sleep(50 * time.Millisecond)
			c.mu.RLock()
			encoderReady = c.pcmaEncoder != nil
			c.mu.RUnlock()
			if encoderReady {
				fmt.Printf("[Client] pcmaEncoder is now ready\n")
				break
			}
		}
		if !encoderReady {
			return fmt.Errorf("pcmaEncoder is still nil after waiting")
		}
	}

	if !txTrackReady {
		return fmt.Errorf("txTrack is nil")
	}

	// Create local references to avoid potential race conditions
	c.mu.RLock()
	localTxTrack := c.txTrack
	localPcmaEncoder := c.pcmaEncoder
	c.mu.RUnlock()

	if localPcmaEncoder == nil {
		return fmt.Errorf("pcmaEncoder is nil after lock")
	}
	if localTxTrack == nil {
		return fmt.Errorf("txTrack is nil after lock")
	}

	fmt.Printf("[Client] Audio components ready: txTrack=%v, encoder=%v\n",
		localTxTrack != nil, localPcmaEncoder != nil)

	// Create a channel to signal when the client is closing
	doneChan := make(chan struct{})

	// Store the done channel in the client
	c.mu.Lock()
	c.doneChan = doneChan
	c.mu.Unlock()

	onSamples := func(pOutputSample, pInputSamples []byte, framecount uint32) {
		// Log first call to confirm callback is working
		if frameCount == 0 {
			fmt.Printf("[Client] ===== onSamples callback FIRED! =====\n")
			fmt.Printf("[Client] First frame: pInputSamples len=%d, framecount=%d\n",
				len(pInputSamples), framecount)
		}

		// Check if we should stop processing
		select {
		case <-doneChan:
			return
		default:
		}

		// Use local references to avoid potential race conditions
		if localTxTrack == nil {
			if frameCount < 3 {
				fmt.Printf("[Client] WARNING: localTxTrack is nil at frame %d!\n", frameCount)
			}
			frameCount++
			return
		}
		if localPcmaEncoder == nil {
			if frameCount < 3 {
				fmt.Printf("[Client] WARNING: localPcmaEncoder is nil at frame %d!\n", frameCount)
			}
			frameCount++
			return
		}

		// pInputSamples contains the captured PCM audio (16-bit, mono, 8kHz)
		if len(pInputSamples) == 0 {
			if frameCount < 3 {
				fmt.Printf("[Client] WARNING: pInputSamples is empty at frame %d!\n", frameCount)
			}
			frameCount++
			return
		}

		// Debug: Log input samples (log first few frames and then every 100th)
		if frameCount < 5 || frameCount%100 == 0 {
			fmt.Printf("[Client] Captured audio frame #%d, size: %d bytes, framecount: %d\n",
				frameCount, len(pInputSamples), framecount)

			// Calculate audio level (RMS) for debugging
			if len(pInputSamples) >= 2 {
				var sumSquares int64
				for i := 0; i < len(pInputSamples)-1; i += 2 {
					sample := int16(pInputSamples[i]) | int16(pInputSamples[i+1])<<8
					sumSquares += int64(sample) * int64(sample)
				}
				rms := float64(sumSquares) / float64(len(pInputSamples)/2)
				if rms > 0 {
					level := 20 * math.Log10(math.Sqrt(rms))
					fmt.Printf("[Client] Audio level: %.2f dB (RMS: %.0f)\n", level, rms)
					if level < -60 {
						fmt.Printf("[Client] WARNING: Audio level is very low! Consider increasing audioGain or microphone volume.\n")
					}
				} else {
					fmt.Printf("[Client] Audio level: SILENT (RMS: 0)\n")
				}
			}
		}

		// Apply audio gain if needed
		if audioGain != 1.0 && len(pInputSamples) >= 2 {
			for i := 0; i < len(pInputSamples)-1; i += 2 {
				sample := int16(pInputSamples[i]) | int16(pInputSamples[i+1])<<8
				// Apply gain
				amplified := float64(sample) * audioGain
				// Clamp to int16 range
				if amplified > 32767 {
					amplified = 32767
				} else if amplified < -32768 {
					amplified = -32768
				}
				sample = int16(amplified)
				pInputSamples[i] = byte(sample)
				pInputSamples[i+1] = byte(sample >> 8)
			}
		}

		// Encode PCM to PCMA
		audioPacket := &media2.AudioPacket{Payload: pInputSamples}
		encodedPackets, err := localPcmaEncoder(audioPacket)
		if err != nil {
			if frameCount%packetLogInterval == 0 {
				log.Printf("[Client] Encode error: %v", err)
			}
			frameCount++
			return
		}

		// Collect all encoded PCMA data
		var pcmaData []byte
		for _, packet := range encodedPackets {
			if af, ok := packet.(*media2.AudioPacket); ok {
				if len(af.Payload) > 0 {
					pcmaData = append(pcmaData, af.Payload...)
				}
			}
		}

		// Debug: Log encoded data
		if frameCount%100 == 0 && len(pcmaData) > 0 {
			fmt.Printf("[Client] Encoded PCMA data size: %d bytes\n", len(pcmaData))
		}

		// Warn if no encoded data
		if len(pcmaData) == 0 && frameCount < 10 {
			fmt.Printf("[Client] WARNING: No encoded data for frame #%d (input: %d bytes)\n",
				frameCount, len(pInputSamples))
		}

		// Send via WebRTC with precise timing
		if len(pcmaData) > 0 {
			expectedTime := startTime.Add(time.Duration(frameCount) * frameDuration)
			if now := time.Now(); expectedTime.After(now) {
				time.Sleep(expectedTime.Sub(now))
			}

			sample := media.Sample{
				Data:     pcmaData,
				Duration: frameDuration,
			}

			// Use local reference to txTrack
			if err := localTxTrack.WriteSample(sample); err != nil {
				if frameCount%packetLogInterval == 0 {
					log.Printf("[Client] Error writing sample: %v", err)
				}
				frameCount++
				return
			} else if frameCount%100 == 0 {
				fmt.Printf("[Client] Sent %d bytes via WebRTC\n", len(pcmaData))
			}

			frameCount++
			if frameCount%packetLogInterval == 0 {
				fmt.Printf("[Client] Sent %d audio frames\n", frameCount)
			}
		} else {
			frameCount++
		}
	}

	deviceCallbacks := malgo.DeviceCallbacks{
		Data: onSamples,
	}

	// Initialize and start capture device
	device, err := malgo.InitDevice(malgoCtx.Context, deviceConfig, deviceCallbacks)
	if err != nil {
		malgoCtx.Uninit()
		return fmt.Errorf("failed to initialize capture device: %w", err)
	}
	c.captureDevice = device

	if err := device.Start(); err != nil {
		device.Uninit()
		c.captureDevice = nil
		malgoCtx.Uninit()
		c.malgoCtx = nil
		return fmt.Errorf("failed to start capture device: %w", err)
	}

	fmt.Println("[Client] Microphone capture started, sending audio to server...")
	fmt.Printf("[Client] Device config: SampleRate=%d, Channels=%d, Format=%d\n",
		deviceConfig.SampleRate, deviceConfig.Capture.Channels, deviceConfig.Capture.Format)
	fmt.Printf("[Client] Waiting for audio samples from microphone...\n")

	// Keep the function running
	select {
	case <-c.done:
		// Close the done channel to signal the audio callback to stop
		c.mu.Lock()
		if c.doneChan != nil {
			close(c.doneChan)
			c.doneChan = nil
		}
		c.mu.Unlock()
		return nil
	case <-c.interrupt:
		// Close the done channel to signal the audio callback to stop
		c.mu.Lock()
		if c.doneChan != nil {
			close(c.doneChan)
			c.doneChan = nil
		}
		c.mu.Unlock()
		return nil
	}
}

// HandleAnswer handles the answer message from the server
func (c *Client) HandleAnswer(msg SignalMessage) error {
	answerData, ok := msg.Data.(map[string]interface{})
	if !ok {
		return fmt.Errorf("invalid answer data")
	}

	// Extract answer SDP
	answerStr, ok := answerData["sdp"].(string)
	if !ok {
		return fmt.Errorf("invalid answer SDP")
	}

	// Set up OnTrack callback before setting remote description
	// This is critical to ensure we don't miss the OnTrack event
	c.mu.Lock()
	c.audioReceived = false
	c.mu.Unlock()

	c.transport.OnTrack(func(track *webrtc.TrackRemote, receiver *webrtc.RTPReceiver) {
		c.mu.Lock()
		if !c.audioReceived {
			c.audioReceived = true
			c.mu.Unlock()
			fmt.Printf("[Client] ===== OnTrack callback FIRED! =====\n")
			fmt.Printf("[Client] Track codec: %s, SSRC: %d, ID: %s\n",
				track.Codec().MimeType, track.SSRC(), track.ID())

			// Start receiving audio in a separate goroutine
			go func() {
				defer func() {
					if r := recover(); r != nil {
						log.Printf("[Client] Recovered from panic in audio receiver: %v", r)
					}
				}()
				if err := c.startAudioReceiverFromTrack(track); err != nil {
					log.Printf("[Client] Error in audio receiver: %v", err)
				}
			}()
		} else {
			c.mu.Unlock()
		}
	})

	// Set remote description
	// Wrap the SDP in JSON format with type "answer" so SetRemoteDescription knows it's an answer
	answerJSON, err := json.Marshal(map[string]string{
		"type": "answer",
		"sdp":  answerStr,
	})
	if err != nil {
		return fmt.Errorf("failed to marshal answer SDP: %w", err)
	}
	if err := c.transport.SetRemoteDescription(string(answerJSON)); err != nil {
		return fmt.Errorf("failed to set remote description: %w", err)
	}

	// Extract and add ICE candidates
	candidates, ok := answerData["candidates"].([]interface{})
	if !ok {
		return fmt.Errorf("invalid candidates data")
	}

	candidateStrs := c.extractCandidates(candidates)
	for _, candidate := range candidateStrs {
		if err := c.transport.AddICECandidate(candidate); err != nil {
			log.Printf("[Client] Error adding ICE candidate: %v", err)
		}
	}

	// Send connected message
	connectedMsg := SignalMessage{
		Type:      "connected",
		SessionID: c.sessionID,
		Data:      map[string]interface{}{},
	}
	if err := c.wsConn.WriteJSON(connectedMsg); err != nil {
		return fmt.Errorf("failed to send connected message: %w", err)
	}

	fmt.Println("[Client] WebRTC connection should now be establishing...")

	// Wait for connection
	if err := c.WaitForConnection(); err != nil {
		return err
	}

	// Setup audio playback first (this initializes pcmaEncoder which is needed for sending)
	// We need this even if we're not receiving audio yet, because we need the encoder
	if err := c.SetupAudioPlayback(); err != nil {
		return fmt.Errorf("failed to setup audio playback: %w", err)
	}

	// Note: Audio receiving is now handled by the OnTrack callback
	// which is set up before SetRemoteDescription
	// No need to wait here - OnTrack will fire automatically when the track arrives
	fmt.Println("[Client] Audio playback setup complete, waiting for OnTrack callback to fire when server sends signal...")

	// Start sending audio from microphone (now pcmaEncoder should be ready)
	go func() {
		if err := c.StartAudioSender(); err != nil {
			log.Printf("[Client] Audio sender error: %v", err)
		}
	}()

	return nil
}

// extractCandidates extracts candidate strings from the interface slice
func (c *Client) extractCandidates(candidates []interface{}) []string {
	var candidateStrs []string
	for _, candidate := range candidates {
		if candidateStr, ok := candidate.(string); ok {
			candidateStrs = append(candidateStrs, candidateStr)
		}
	}
	return candidateStrs
}

// StartMessageListener starts listening for WebSocket messages
func (c *Client) StartMessageListener() {
	go func() {
		defer close(c.done)
		for {
			_, message, err := c.wsConn.ReadMessage()
			if err != nil {
				log.Printf("[Client] Error reading message: %v", err)
				return
			}

			var signal SignalMessage
			if err := json.Unmarshal(message, &signal); err != nil {
				log.Printf("[Client] Error unmarshaling message: %v", err)
				continue
			}

			switch signal.Type {
			case "answer":
				if err := c.HandleAnswer(signal); err != nil {
					log.Printf("[Client] Error handling answer: %v", err)
				}
			default:
				log.Printf("[Client] Unknown message type: %s", signal.Type)
			}
		}
	}()
}

// Run runs the client main loop
func (c *Client) Run() error {
	defer c.Close()

	// Initialize session
	if _, err := c.InitializeSession(); err != nil {
		return err
	}

	// Create and send offer
	if err := c.CreateAndSendOffer(); err != nil {
		return err
	}

	// Start message listener
	c.StartMessageListener()

	// Wait for interrupt or done signal
	fmt.Println("[Client] Waiting for connection to establish...")
	fmt.Println("[Client] Press Ctrl+C to exit")
	select {
	case <-c.interrupt:
		fmt.Println("\n[Client] Interrupted, closing connection...")
		return nil
	case <-c.done:
		fmt.Println("[Client] Connection closed")
		return nil
	}
}

func main() {
	client, err := NewClient()
	if err != nil {
		log.Fatalf("[Client] Failed to create client: %v", err)
	}

	if err := client.Run(); err != nil {
		log.Fatalf("[Client] Error: %v", err)
	}
}
