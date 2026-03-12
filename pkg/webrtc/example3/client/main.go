package main

// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/url"
	"os"
	"os/signal"
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
	"github.com/youpy/go-wav"
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
	targetSampleRate = 8000 // PCMA standard sample rate
	audioChannels    = 1
	audioBitDepth    = 16
	bytesPerSample   = 2 // 16-bit = 2 bytes

	// Frame configuration
	frameDurationMs = 20
	bytesPerFrame   = 160 // 20ms * 8000Hz = 160 samples = 160 bytes (PCMA is 1 byte per sample)

	// Logging intervals
	packetLogInterval = 100
	warningLogLimit   = 3

	// Audio file configuration
	clientAudioFile = "ringing.wav"
)

// SignalMessage represents a WebSocket signaling message
type SignalMessage struct {
	Type      string      `json:"type"`
	SessionID string      `json:"session_id,omitempty"`
	Data      interface{} `json:"data,omitempty"`
}

// Client represents a WebRTC client
type Client struct {
	wsConn    *websocket.Conn
	transport *rtcmedia.WebRTCTransport
	sessionID string
	interrupt chan os.Signal
	done      chan struct{}
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
		StreamID: "lingecho_client",
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
	if c.transport != nil {
		c.transport.Close()
	}
	if c.wsConn != nil {
		return c.wsConn.Close()
	}
	return nil
}

// InitializeSession initializes the WebSocket session and returns the session ID
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
func (c *Client) SetupAudioPlayback() (*devices.StreamAudioPlayer, media2.EncoderFunc, error) {
	// Create stream player
	streamPlayer, err := devices.NewStreamAudioPlayer(
		audioChannels,
		targetSampleRate,
		malgo.FormatS16,
	)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create stream player: %w", err)
	}

	// Start playback
	if err := streamPlayer.Play(); err != nil {
		streamPlayer.Close()
		return nil, nil, fmt.Errorf("failed to start playback: %w", err)
	}

	fmt.Printf("[Client] Audio playback started: %dHz, %d channel(s)\n",
		targetSampleRate, audioChannels)

	// Create PCMA decoder
	decodeFunc, err := encoder.CreateDecode(
		media2.CodecConfig{
			Codec:         "pcma",
			SampleRate:    targetSampleRate,
			Channels:      audioChannels,
			BitDepth:      8,
			FrameDuration: "20ms",
		},
		media2.CodecConfig{
			Codec:         "pcm",
			SampleRate:    targetSampleRate,
			Channels:      audioChannels,
			BitDepth:      audioBitDepth,
			FrameDuration: "20ms",
		},
	)
	if err != nil {
		streamPlayer.Close()
		return nil, nil, fmt.Errorf("failed to create decoder: %w", err)
	}

	return streamPlayer, decodeFunc, nil
}

// ProcessAudioPacket processes a single RTP audio packet
func (c *Client) ProcessAudioPacket(
	packet *rtp.Packet,
	decodeFunc media2.EncoderFunc,
	streamPlayer *devices.StreamAudioPlayer,
	packetCount int,
) error {
	payload := packet.Payload
	if len(payload) == 0 {
		return nil
	}

	// Decode PCMA to PCM
	audioPacket := &media2.AudioPacket{Payload: payload}
	decodedPackets, err := decodeFunc(audioPacket)
	if err != nil {
		if packetCount%packetLogInterval == 0 {
			fmt.Printf("[Client] Error decoding frame %d: %v\n", packetCount, err)
		}
		return err
	}

	// Collect all decoded PCM data and write at once to reduce discontinuity
	allPCMData := c.collectPCMData(decodedPackets, packetCount)
	if len(allPCMData) > 0 {
		if err := streamPlayer.Write(allPCMData); err != nil {
			// Buffer full is not critical, only log other errors
			if packetCount%packetLogInterval == 0 && err.Error() != "音频缓冲区已满" {
				fmt.Printf("[Client] Error writing to player: %v\n", err)
			}
		}
	}

	return nil
}

// collectPCMData collects and validates PCM data from decoded packets
func (c *Client) collectPCMData(decodedPackets []media2.MediaPacket, packetCount int) []byte {
	var allPCMData []byte

	for _, packet := range decodedPackets {
		af, ok := packet.(*media2.AudioPacket)
		if !ok {
			continue
		}

		// Skip empty frames
		if len(af.Payload) == 0 {
			continue
		}

		// Validate PCM data (should be 16-bit, so length must be even)
		if len(af.Payload)%2 != 0 {
			if packetCount <= warningLogLimit {
				fmt.Printf("[Client] Warning: Odd PCM length at packet %d: %d bytes\n",
					packetCount, len(af.Payload))
			}
			continue
		}

		allPCMData = append(allPCMData, af.Payload...)
	}

	return allPCMData
}

// SendAudioToServer sends audio data to the server via WebRTC
func (c *Client) SendAudioToServer() error {
	fmt.Println("[Client] Starting to send audio to server...")

	// Get transmit track
	txTrack := c.transport.GetTxTrack()
	if txTrack == nil {
		return fmt.Errorf("txTrack is nil")
	}

	// Load and process audio file
	pcmaData, err := c.loadAndProcessAudioFile()
	if err != nil {
		return fmt.Errorf("failed to load audio: %w", err)
	}

	// Send audio frames
	return c.sendAudioFrames(txTrack, pcmaData)
}

// loadAndProcessAudioFile loads and processes the audio file
func (c *Client) loadAndProcessAudioFile() ([]byte, error) {
	// Open audio file
	file, err := os.Open(clientAudioFile)
	if err != nil {
		return nil, fmt.Errorf("failed to open audio file: %w", err)
	}
	defer file.Close()

	// Read WAV format
	w := wav.NewReader(file)
	format, err := w.Format()
	if err != nil {
		return nil, fmt.Errorf("failed to get WAV format: %w", err)
	}

	fmt.Printf("[Client] WAV format: %dHz, %d channels, %d bits\n",
		format.SampleRate, format.NumChannels, format.BitsPerSample)

	// Read entire file
	allPCMData, err := c.readWAVFile(w)
	if err != nil {
		return nil, fmt.Errorf("failed to read WAV file: %w", err)
	}

	fmt.Printf("[Client] Read %d bytes from WAV file\n", len(allPCMData))

	// Convert to mono if needed
	channels := int(format.NumChannels)
	if channels > 1 {
		allPCMData = c.convertToMono(allPCMData, channels, int(format.BitsPerSample))
		channels = 1
	}

	// Resample if needed
	if int(format.SampleRate) != targetSampleRate {
		allPCMData, err = c.resampleAudio(allPCMData, int(format.SampleRate))
		if err != nil {
			return nil, err
		}
	}

	// Encode to PCMA
	pcmaData, err := encoder.EncodePCMA(allPCMData)
	if err != nil {
		return nil, fmt.Errorf("failed to encode to PCMA: %w", err)
	}

	fmt.Printf("[Client] Encoded %d bytes PCM to %d bytes PCMA\n",
		len(allPCMData), len(pcmaData))

	return pcmaData, nil
}

// readWAVFile reads the entire WAV file
func (c *Client) readWAVFile(w *wav.Reader) ([]byte, error) {
	var allPCMData []byte
	tempBuffer := make([]byte, 8192)

	for {
		n, err := w.Read(tempBuffer)
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
		allPCMData = append(allPCMData, tempBuffer[:n]...)
	}

	return allPCMData, nil
}

// convertToMono converts stereo audio to mono by averaging channels
func (c *Client) convertToMono(data []byte, channels, bytesPerSample int) []byte {
	if bytesPerSample != 2 {
		return data // Only support 16-bit
	}

	sampleCount := len(data) / (bytesPerSample * channels)
	monoData := make([]byte, sampleCount*bytesPerSample)

	for i := 0; i < sampleCount; i++ {
		leftIdx := i * bytesPerSample * channels
		rightIdx := leftIdx + bytesPerSample

		if rightIdx+bytesPerSample <= len(data) {
			leftSample := int16(data[leftIdx]) | int16(data[leftIdx+1])<<8
			rightSample := int16(data[rightIdx]) | int16(data[rightIdx+1])<<8

			// Average the two channels
			avg := int16((int32(leftSample) + int32(rightSample)) / 2)

			monoIdx := i * bytesPerSample
			monoData[monoIdx] = byte(avg)
			monoData[monoIdx+1] = byte(avg >> 8)
		}
	}

	fmt.Printf("[Client] Converted stereo to mono (%d samples)\n", sampleCount)
	return monoData
}

// resampleAudio resamples audio to target sample rate
func (c *Client) resampleAudio(data []byte, sourceRate int) ([]byte, error) {
	fmt.Printf("[Client] Resampling from %dHz to %dHz...\n", sourceRate, targetSampleRate)

	resampled, err := media2.ResamplePCM(data, sourceRate, targetSampleRate)
	if err != nil {
		return nil, fmt.Errorf("resampling failed: %w", err)
	}

	fmt.Printf("[Client] Resampled to %dHz (%d bytes)\n", targetSampleRate, len(resampled))
	return resampled, nil
}

// sendAudioFrames sends audio frames with precise timing
func (c *Client) sendAudioFrames(txTrack *webrtc.TrackLocalStaticSample, pcmaData []byte) error {
	frameDuration := time.Duration(frameDurationMs) * time.Millisecond
	startTime := time.Now()
	frameCount := 0

	for i := 0; i < len(pcmaData); i += bytesPerFrame {
		end := i + bytesPerFrame
		if end > len(pcmaData) {
			end = len(pcmaData)
		}

		// Calculate exact send time to maintain consistent frame rate
		expectedTime := startTime.Add(time.Duration(frameCount) * frameDuration)
		if now := time.Now(); expectedTime.After(now) {
			time.Sleep(expectedTime.Sub(now))
		}

		sample := media.Sample{
			Data:     pcmaData[i:end],
			Duration: frameDuration,
		}

		if err := txTrack.WriteSample(sample); err != nil {
			return fmt.Errorf("failed to write sample: %w", err)
		}

		frameCount++
		if frameCount%50 == 0 {
			fmt.Printf("[Client] Sent %d frames (PCMA: %d bytes)...\n",
				frameCount, len(pcmaData[i:end]))
		}
	}

	fmt.Printf("[Client] Finished sending audio (%d frames, %d bytes PCMA)\n",
		frameCount, len(pcmaData))

	return nil
}

// StartAudioReceiver starts receiving and playing audio packets
func (c *Client) StartAudioReceiver(rxTrack *webrtc.TrackRemote) error {
	streamPlayer, decodeFunc, err := c.SetupAudioPlayback()
	if err != nil {
		return err
	}
	defer streamPlayer.Close()

	codec := rxTrack.Codec()
	fmt.Printf("[Client] Received track: %s, %dHz\n", codec.MimeType, codec.ClockRate)

	packetCount := 0
	lastPacketTime := time.Now()
	noPacketTimeout := 2 * time.Second // 2秒没有收到数据包，认为播放完成

	// Use a goroutine to read packets with timeout detection
	packetChan := make(chan *rtp.Packet, 1)
	errChan := make(chan error, 1)
	done := make(chan bool, 1)

	go func() {
		for {
			packet, _, err := rxTrack.ReadRTP()
			if err != nil {
				errChan <- err
				return
			}
			select {
			case packetChan <- packet:
			case <-done:
				return
			}
		}
	}()

	for {
		select {
		case packet := <-packetChan:
			lastPacketTime = time.Now()
			if err := c.ProcessAudioPacket(packet, decodeFunc, streamPlayer, packetCount); err != nil {
				// Continue processing even if one packet fails
				continue
			}
			packetCount++
			if packetCount%packetLogInterval == 0 {
				fmt.Printf("[Client] Received and played %d RTP packets\n", packetCount)
			}
		case err := <-errChan:
			close(done)
			// If we get an error, wait a bit for playback to finish, then send audio
			fmt.Printf("[Client] Error reading RTP packet (may be end of stream): %v\n", err)
			time.Sleep(500 * time.Millisecond)
			return c.SendAudioToServer()
		case <-time.After(noPacketTimeout):
			// No packet received for timeout duration, check if we should consider playback finished
			if time.Since(lastPacketTime) >= noPacketTimeout {
				fmt.Printf("[Client] No packets received for %v, assuming playback finished\n", noPacketTimeout)
				close(done)
				// Wait a bit more to ensure all audio is played
				time.Sleep(500 * time.Millisecond)
				// Now send audio to server
				return c.SendAudioToServer()
			}
		}
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

	// Set remote description
	if err := c.transport.SetRemoteDescription(answerStr); err != nil {
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

	// Wait for track
	rxTrack, err := c.WaitForTrack()
	if err != nil {
		return err
	}

	// Start receiving audio
	return c.StartAudioReceiver(rxTrack)
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
