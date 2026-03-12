package main

// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"sync"
	"time"

	media2 "github.com/LingByte/SoulNexus/pkg/media"
	"github.com/LingByte/SoulNexus/pkg/media/encoder"
	"github.com/LingByte/SoulNexus/pkg/webrtc/constants"
	"github.com/LingByte/SoulNexus/pkg/webrtc/rtcmedia"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/pion/webrtc/v3"
	"github.com/pion/webrtc/v3/pkg/media"
	"github.com/youpy/go-wav"
)

// Constants
const (
	// Server configuration
	serverPort = ":8080"
	wsPath     = "/websocket"

	// Connection retry configuration
	maxConnectionRetries       = 50
	connectionRetryDelay       = 100 * time.Millisecond
	connectionStateLogInterval = 10
	connectionReadyDelay       = 200 * time.Millisecond

	// Audio configuration
	targetSampleRate = 8000 // PCMA standard sample rate
	audioChannels    = 1
	audioBitDepth    = 16
	bytesPerSample   = 2 // 16-bit = 2 bytes

	// Frame configuration
	frameDurationMs = 20
	bytesPerFrame   = 160 // 20ms * 8000Hz = 160 samples = 160 bytes (PCMA is 1 byte per sample)

	// File configuration
	readBufferSize    = 8192
	audioFilePrimary  = "ringring.wav"
	audioFileFallback = "ringing.wav"

	// Logging intervals
	frameLogInterval = 50
)

// ClientManager manages WebRTC client connections
type ClientManager struct {
	clients map[string]*Client
	mutex   sync.RWMutex
}

// NewClientManager creates a new client manager
func NewClientManager() *ClientManager {
	return &ClientManager{
		clients: make(map[string]*Client),
	}
}

// AddClient adds a client to the manager
func (m *ClientManager) AddClient(sessionID string, client *Client) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	m.clients[sessionID] = client
}

// RemoveClient removes a client from the manager
func (m *ClientManager) RemoveClient(sessionID string) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	delete(m.clients, sessionID)
}

// GetClient retrieves a client by session ID
func (m *ClientManager) GetClient(sessionID string) (*Client, bool) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	client, exists := m.clients[sessionID]
	return client, exists
}

// Client represents a WebRTC client connection
type Client struct {
	conn      *websocket.Conn
	transport *rtcmedia.WebRTCTransport
	sessionID string
}

// SignalMessage represents a WebSocket signaling message
type SignalMessage struct {
	Type      string      `json:"type"`
	SessionID string      `json:"session_id,omitempty"`
	Data      interface{} `json:"data,omitempty"`
}

var (
	manager  = NewClientManager()
	upgrader = websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool {
			return true // Allow all origins in development
		},
	}
)

// websocketHandler handles WebSocket connections
func websocketHandler(c *gin.Context) {
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Printf("[Server] Failed to upgrade connection: %v", err)
		return
	}
	defer conn.Close()

	// Create session ID
	sessionID := fmt.Sprintf("session_%d", time.Now().UnixNano())

	// Create WebRTC transport
	transport := rtcmedia.NewWebRTCTransport(rtcmedia.WebRTCOption{
		Codec: constants.CodecPCMA,
		ICEServers: []webrtc.ICEServer{
			{URLs: []string{"stun:stun.l.google.com:19302"}},
		},
		StreamID:   "lingecho_server",
		ICETimeout: constants.DefaultICETimeout,
	})
	transport.NewPeerConnection()

	client := &Client{
		conn:      conn,
		transport: transport,
		sessionID: sessionID,
	}

	manager.AddClient(sessionID, client)
	defer manager.RemoveClient(sessionID)

	// Send session ID to client
	initMsg := SignalMessage{
		Type:      "init",
		SessionID: sessionID,
	}
	if err := conn.WriteJSON(initMsg); err != nil {
		log.Printf("[Server] Failed to send init message: %v", err)
		return
	}

	// Handle incoming messages
	for {
		var msg SignalMessage
		if err := conn.ReadJSON(&msg); err != nil {
			log.Printf("[Server] Error reading message: %v", err)
			break
		}

		handleSignalMessage(client, msg)
	}
}

// handleSignalMessage routes signaling messages to appropriate handlers
func handleSignalMessage(client *Client, msg SignalMessage) {
	switch msg.Type {
	case "offer":
		handleOffer(client, msg)
	case "connected":
		handleConnection(client, msg)
	default:
		log.Printf("[Server] Unknown message type: %s", msg.Type)
	}
}

// handleOffer handles the WebRTC offer from the client
func handleOffer(client *Client, msg SignalMessage) {
	offerData, ok := msg.Data.(map[string]interface{})
	if !ok {
		log.Println("[Server] Invalid offer data")
		return
	}

	offerStr, ok := offerData["sdp"].(string)
	if !ok {
		log.Println("[Server] Invalid offer SDP")
		return
	}

	// Set remote description
	if err := client.transport.SetRemoteDescription(offerStr); err != nil {
		log.Printf("[Server] Error setting remote description: %v", err)
		return
	}

	// Extract candidates
	candidates, ok := offerData["candidates"].([]interface{})
	if !ok {
		log.Println("[Server] Invalid candidates data")
		return
	}

	candidateStrs := extractCandidates(candidates)
	answer, serverCandidates, err := client.transport.CreateAnswer(candidateStrs)
	if err != nil {
		log.Printf("[Server] Error creating answer: %v", err)
		return
	}

	// Send answer back to client
	answerMsg := SignalMessage{
		Type:      "answer",
		SessionID: client.sessionID,
		Data: map[string]interface{}{
			"sdp":        answer,
			"candidates": serverCandidates,
		},
	}

	if err := client.conn.WriteJSON(answerMsg); err != nil {
		log.Printf("[Server] Error sending answer: %v", err)
		return
	}

	fmt.Printf("[Server] Sent answer to client %s\n", client.sessionID)
}

// extractCandidates extracts candidate strings from the interface slice
func extractCandidates(candidates []interface{}) []string {
	var candidateStrs []string
	for _, c := range candidates {
		if candidateStr, ok := c.(string); ok {
			candidateStrs = append(candidateStrs, candidateStr)
		}
	}
	return candidateStrs
}

// handleConnection handles the connection established message and starts sending audio
func handleConnection(client *Client, msg SignalMessage) {
	// Run in goroutine to avoid blocking
	go func() {
		if err := sendAudioToClient(client); err != nil {
			log.Printf("[Server] Error sending audio: %v", err)
		}
	}()
}

// sendAudioToClient sends audio data to the client via WebRTC
func sendAudioToClient(client *Client) error {
	// Wait for connection to be fully established
	if err := waitForConnection(client.transport); err != nil {
		return fmt.Errorf("connection not established: %w", err)
	}

	// Additional delay to ensure everything is ready
	time.Sleep(connectionReadyDelay)

	fmt.Println("[Server] Starting to send signal...")

	// Get transmit track
	txTrack := client.transport.GetTxTrack()
	if txTrack == nil {
		return fmt.Errorf("txTrack is nil")
	}

	// Load and process audio file
	pcmaData, err := loadAndProcessAudioFile()
	if err != nil {
		return fmt.Errorf("failed to load audio: %w", err)
	}

	// Send audio frames
	return sendAudioFrames(txTrack, pcmaData)
}

// waitForConnection waits for the WebRTC connection to be established
func waitForConnection(transport *rtcmedia.WebRTCTransport) error {
	for i := 0; i < maxConnectionRetries; i++ {
		state := transport.GetConnectionState()
		if state == webrtc.PeerConnectionStateConnected {
			return nil
		}

		if i%connectionStateLogInterval == 0 {
			fmt.Printf("[Server] Waiting for connection... (state: %s)\n", state.String())
		}

		time.Sleep(connectionRetryDelay)
	}

	return fmt.Errorf("connection timeout after %d retries", maxConnectionRetries)
}

// loadAndProcessAudioFile loads and processes the audio file
func loadAndProcessAudioFile() ([]byte, error) {
	// Open audio file
	file, err := openAudioFile()
	if err != nil {
		return nil, err
	}
	defer file.Close()

	// Read WAV format
	w := wav.NewReader(file)
	format, err := w.Format()
	if err != nil {
		return nil, fmt.Errorf("failed to get WAV format: %w", err)
	}

	fmt.Printf("[Server] WAV format: %dHz, %d channels, %d bits\n",
		format.SampleRate, format.NumChannels, format.BitsPerSample)

	// Read entire file
	allPCMData, err := readWAVFile(w)
	if err != nil {
		return nil, fmt.Errorf("failed to read WAV file: %w", err)
	}

	fmt.Printf("[Server] Read %d bytes from WAV file\n", len(allPCMData))

	// Convert to mono if needed
	channels := int(format.NumChannels)
	if channels > 1 {
		allPCMData = convertToMono(allPCMData, channels, int(format.BitsPerSample))
		channels = 1
	}

	// Resample if needed
	if int(format.SampleRate) != targetSampleRate {
		allPCMData, err = resampleAudio(allPCMData, int(format.SampleRate))
		if err != nil {
			return nil, err
		}
	}

	// Encode to PCMA
	pcmaData, err := encoder.EncodePCMA(allPCMData)
	if err != nil {
		return nil, fmt.Errorf("failed to encode to PCMA: %w", err)
	}

	fmt.Printf("[Server] Encoded %d bytes PCM to %d bytes PCMA\n",
		len(allPCMData), len(pcmaData))

	return pcmaData, nil
}

// openAudioFile opens the audio file with fallback
func openAudioFile() (*os.File, error) {
	file, err := os.Open(audioFilePrimary)
	if err != nil {
		file, err = os.Open(audioFileFallback)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to open audio file: %w", err)
	}
	return file, nil
}

// readWAVFile reads the entire WAV file
func readWAVFile(w *wav.Reader) ([]byte, error) {
	var allPCMData []byte
	tempBuffer := make([]byte, readBufferSize)

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
func convertToMono(data []byte, channels, bytesPerSample int) []byte {
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

	fmt.Printf("[Server] Converted stereo to mono (%d samples)\n", sampleCount)
	return monoData
}

// resampleAudio resamples audio to target sample rate
func resampleAudio(data []byte, sourceRate int) ([]byte, error) {
	fmt.Printf("[Server] Resampling from %dHz to %dHz...\n", sourceRate, targetSampleRate)

	resampled, err := media2.ResamplePCM(data, sourceRate, targetSampleRate)
	if err != nil {
		return nil, fmt.Errorf("resampling failed: %w", err)
	}

	fmt.Printf("[Server] Resampled to %dHz (%d bytes)\n", targetSampleRate, len(resampled))
	return resampled, nil
}

// sendAudioFrames sends audio frames with precise timing
func sendAudioFrames(txTrack *webrtc.TrackLocalStaticSample, pcmaData []byte) error {
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
		if frameCount%frameLogInterval == 0 {
			fmt.Printf("[Server] Sent %d frames (PCMA: %d bytes)...\n",
				frameCount, len(pcmaData[i:end]))
		}
	}

	fmt.Printf("[Server] Finished sending audio (%d frames, %d bytes PCMA)\n",
		frameCount, len(pcmaData))

	return nil
}

func main() {
	// Set Gin to release mode
	gin.SetMode(gin.ReleaseMode)

	// Create router
	router := gin.Default()
	router.GET(wsPath, websocketHandler)

	// Start server
	fmt.Printf("[Server] Starting server on %s\n", serverPort)
	if err := router.Run(serverPort); err != nil {
		log.Fatalf("[Server] Failed to start server: %v", err)
	}
}
