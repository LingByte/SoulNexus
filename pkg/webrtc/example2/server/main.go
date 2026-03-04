package main

import (
	"fmt"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/LingByte/SoulNexus/pkg/logger"
	"github.com/LingByte/SoulNexus/pkg/webrtc/constants"
	"github.com/LingByte/SoulNexus/pkg/webrtc/rtcmedia"
	transports "github.com/LingByte/SoulNexus/pkg/webrtc/transport"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/pion/webrtc/v3"
)

// Constants
const (
	serverPort = ":8080"
	wsPath     = "/websocket"
	// Connection configuration
	maxConnectionRetries       = 50
	connectionRetryDelay       = 100 * time.Millisecond
	connectionStateLogInterval = 10
	connectionReadyDelay       = 200 * time.Millisecond
)

// ClientManager manages WebRTC client connections
type ClientManager struct {
	clients map[string]*transports.AIClient
	mutex   sync.RWMutex
}

func NewClientManager() *ClientManager {
	return &ClientManager{
		clients: make(map[string]*transports.AIClient),
	}
}

func (m *ClientManager) AddClient(sessionID string, client *transports.AIClient) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	m.clients[sessionID] = client
}

func (m *ClientManager) RemoveClient(sessionID string) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	delete(m.clients, sessionID)
}

func (m *ClientManager) GetClient(sessionID string) (*transports.AIClient, bool) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	client, exists := m.clients[sessionID]
	return client, exists
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
			return true
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

	sessionID := fmt.Sprintf("session_%d", time.Now().UnixNano())

	// Create WebRTC transport
	transport := rtcmedia.NewWebRTCTransport(rtcmedia.WebRTCOption{
		Codec: constants.CodecPCMA,
		ICEServers: []webrtc.ICEServer{
			{URLs: []string{"stun:stun.l.google.com:19302"}},
		},
		StreamID:   "lingecho_ai_server",
		ICETimeout: constants.DefaultICETimeout,
	})
	transport.NewPeerConnection()

	// Initialize AI components
	aiClient, err := transports.NewAIClient(conn, transport, sessionID, "", nil, 0, 0, nil)
	if err != nil {
		log.Printf("[Server] Failed to create AI client: %v", err)
		return
	}

	// Set up OnTrack callback BEFORE handling any signaling messages
	// This is critical - OnTrack must be set up early to catch the track when it arrives
	transport.OnTrack(func(track *webrtc.TrackRemote, receiver *webrtc.RTPReceiver) {
		aiClient.Mu.Lock()
		if !aiClient.AudioReceived {
			aiClient.AudioReceived = true
			aiClient.Mu.Unlock()
			fmt.Printf("[Server] ===== OnTrack callback FIRED! =====\n")
			fmt.Printf("[Server] Track codec: %s, SSRC: %d, ID: %s\n",
				track.Codec().MimeType, track.SSRC(), track.ID())

			// Start audio receiver in a separate goroutine
			go func() {
				defer func() {
					if r := recover(); r != nil {
						log.Printf("[Server] Recovered from panic in audio receiver: %v", r)
					}
				}()
				if err := aiClient.StartAudioReceiverFromTrack(track); err != nil {
					log.Printf("[Server] Error in audio receiver: %v", err)
				}
			}()
		} else {
			aiClient.Mu.Unlock()
		}
	})
	fmt.Printf("[Server] OnTrack callback registered for client %s\n", sessionID)

	manager.AddClient(sessionID, aiClient)
	defer manager.RemoveClient(sessionID)
	defer aiClient.Close()

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

		handleSignalMessage(aiClient, msg)
	}
}

// handleSignalMessage routes signaling messages
func handleSignalMessage(client *transports.AIClient, msg SignalMessage) {
	switch msg.Type {
	case "offer":
		handleOffer(client, msg)
	case "connected":
		handleConnection(client, msg)
	default:
		log.Printf("[Server] Unknown message type: %s", msg.Type)
	}
}

// handleOffer handles the WebRTC offer
func handleOffer(client *transports.AIClient, msg SignalMessage) {
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

	// Debug: Check if offer SDP contains audio media
	if strings.Contains(offerStr, "m=audio") {
		fmt.Printf("[Server] Offer SDP contains audio media description\n")
	} else {
		fmt.Printf("[Server] WARNING: Offer SDP does NOT contain audio media description!\n")
		previewLen := 200
		if len(offerStr) < previewLen {
			previewLen = len(offerStr)
		}
		fmt.Printf("[Server] Offer SDP preview: %s...\n", offerStr[:previewLen])
	}

	if err := client.Transport.SetRemoteDescription(offerStr); err != nil {
		log.Printf("[Server] Error setting remote description: %v", err)
		return
	}
	fmt.Printf("[Server] Remote description set successfully\n")

	candidates, ok := offerData["candidates"].([]interface{})
	if !ok {
		log.Println("[Server] Invalid candidates data")
		return
	}

	candidateStrs := extractCandidates(candidates)
	answer, serverCandidates, err := client.Transport.CreateAnswer(candidateStrs)
	if err != nil {
		log.Printf("[Server] Error creating answer: %v", err)
		return
	}

	// Debug: Check if answer SDP contains audio media
	if strings.Contains(answer, "m=audio") {
		fmt.Printf("[Server] Answer SDP contains audio media description\n")
	} else {
		fmt.Printf("[Server] WARNING: Answer SDP does NOT contain audio media description!\n")
	}

	answerMsg := SignalMessage{
		Type:      "answer",
		SessionID: client.SessionID,
		Data: map[string]interface{}{
			"sdp":        answer,
			"candidates": serverCandidates,
		},
	}

	if err := client.Conn.WriteJSON(answerMsg); err != nil {
		log.Printf("[Server] Error sending answer: %v", err)
		return
	}

	fmt.Printf("[Server] Sent answer to client %s\n", client.SessionID)

	// Note: Audio receiving is now handled by the OnTrack callback
	// which is set up in websocketHandler before any signaling messages are processed
	// No need to wait here - OnTrack will fire automatically when the track arrives
	fmt.Println("[Server] Answer sent, waiting for OnTrack callback to fire when client sends signal...")
}

// extractCandidates extracts candidate strings
func extractCandidates(candidates []interface{}) []string {
	var candidateStrs []string
	for _, c := range candidates {
		if candidateStr, ok := c.(string); ok {
			candidateStrs = append(candidateStrs, candidateStr)
		}
	}
	return candidateStrs
}

// handleConnection handles connection established message (client confirmation)
func handleConnection(client *transports.AIClient, msg SignalMessage) {
	fmt.Printf("[Server] Client confirmed connection for session %s\n", client.SessionID)

	// Wait for connection to be established, then send greeting
	go func() {
		if err := waitForConnection(client.Transport); err != nil {
			log.Printf("[Server] Connection not established: %v", err)
			return
		}

		// Wait for txTrack to be ready
		maxWait := 50
		for i := 0; i < maxWait; i++ {
			txTrack := client.Transport.GetTxTrack()
			if txTrack != nil {
				fmt.Printf("[Server] txTrack is ready after %d attempts\n", i+1)
				break
			}
			if i == 0 {
				fmt.Printf("[Server] Waiting for txTrack to be ready...\n")
			}
			time.Sleep(50 * time.Millisecond)
		}

		// Wait a bit more for client's audio receiver to be ready
		time.Sleep(connectionReadyDelay * 2)

		// Send greeting to start the conversation
		greeting := "你好，我是AI助手，很高兴和你对话。"
		fmt.Printf("[Server] Sending greeting: %s\n", greeting)
		client.GenerateTTS(greeting)
	}()
}

// waitForConnection waits for WebRTC connection
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

	return fmt.Errorf("connection timeout")
}

func main() {
	// Initialize logger
	logCfg := &logger.LogConfig{
		Level:      "info",
		Filename:   "logs/ai-voice-server.log",
		MaxSize:    100,
		MaxAge:     7,
		MaxBackups: 3,
		Daily:      false,
	}
	if err := logger.Init(logCfg, "dev"); err != nil {
		log.Fatalf("[Server] Failed to initialize logger: %v", err)
	}
	defer logger.Sync()

	gin.SetMode(gin.ReleaseMode)

	router := gin.Default()
	router.GET(wsPath, websocketHandler)

	fmt.Printf("[Server] Starting AI voice server on %s\n", serverPort)
	if err := router.Run(serverPort); err != nil {
		log.Fatalf("[Server] Failed to start server: %v", err)
	}
}
