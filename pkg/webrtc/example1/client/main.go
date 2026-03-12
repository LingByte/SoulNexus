package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/url"
	"os"
	"os/signal"
	"time"

	"github.com/LingByte/SoulNexus/pkg/devices"
	"github.com/LingByte/SoulNexus/pkg/media"
	"github.com/LingByte/SoulNexus/pkg/media/encoder"
	"github.com/LingByte/SoulNexus/pkg/webrtc/constants"
	"github.com/LingByte/SoulNexus/pkg/webrtc/rtcmedia"
	"github.com/gen2brain/malgo"
	"github.com/gorilla/websocket"
	"github.com/pion/rtp"
	"github.com/pion/webrtc/v3"
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

	// Logging intervals
	packetLogInterval = 100
	warningLogLimit   = 3
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
func (c *Client) SetupAudioPlayback() (*devices.StreamAudioPlayer, media.EncoderFunc, error) {
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
		media.CodecConfig{
			Codec:         "pcma",
			SampleRate:    targetSampleRate,
			Channels:      audioChannels,
			BitDepth:      8,
			FrameDuration: "20ms",
		},
		media.CodecConfig{
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
	decodeFunc media.EncoderFunc,
	streamPlayer *devices.StreamAudioPlayer,
	packetCount int,
) error {
	payload := packet.Payload
	if len(payload) == 0 {
		return nil
	}

	// Decode PCMA to PCM
	audioPacket := &media.AudioPacket{Payload: payload}
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
func (c *Client) collectPCMData(decodedPackets []media.MediaPacket, packetCount int) []byte {
	var allPCMData []byte

	for _, packet := range decodedPackets {
		af, ok := packet.(*media.AudioPacket)
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
	for {
		packet, _, err := rxTrack.ReadRTP()
		if err != nil {
			return fmt.Errorf("error reading RTP packet: %w", err)
		}

		if err := c.ProcessAudioPacket(packet, decodeFunc, streamPlayer, packetCount); err != nil {
			// Continue processing even if one packet fails
			continue
		}

		packetCount++
		if packetCount%packetLogInterval == 0 {
			fmt.Printf("[Client] Received and played %d RTP packets\n", packetCount)
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
