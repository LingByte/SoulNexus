package transport

import (
	"context"
	"fmt"
	"log"
	"math"
	"strings"
	"sync"
	"time"

	"github.com/LingByte/SoulNexus/internal/models"
	"github.com/LingByte/SoulNexus/pkg/llm"
	media2 "github.com/LingByte/SoulNexus/pkg/media"
	"github.com/LingByte/SoulNexus/pkg/media/encoder"
	"github.com/LingByte/SoulNexus/pkg/recognizer"
	"github.com/LingByte/SoulNexus/pkg/synthesizer"
	"github.com/LingByte/SoulNexus/pkg/utils"
	"github.com/LingByte/SoulNexus/pkg/webrtc/rtcmedia"
	"github.com/gorilla/websocket"
	"github.com/pion/webrtc/v3"
	"github.com/pion/webrtc/v3/pkg/media"
	"gorm.io/gorm"
)

var (
	// Audio configuration
	// Use 16kHz for audio processing to match QCloud ASR (16k_zh model).
	// The PCMA codec handles the encoding internally.
	// Note: WebRTC PCMA uses 8kHz clock rate, but we process audio at 16kHz
	// and the codec resamples as needed.
	targetSampleRate = 16000
	audioChannels    = 1
	audioBitDepth    = 16
	// Frame size calculation for 16kHz:
	// - 20ms frame duration at 16000Hz = 320 samples
	// - For PCMA (8-bit), after resampling to 8kHz: 160 samples = 160 bytes
	// - For PCM (16-bit) at 16kHz: 320 samples = 640 bytes
	bytesPerFrame = 320 // PCMA frame size after encoding (accounts for resampling)
	// Connection configuration
	maxConnectionRetries       = 50
	connectionRetryDelay       = 100 * time.Millisecond
	connectionStateLogInterval = 10
	connectionReadyDelay       = 200 * time.Millisecond

	// Logging intervals
	packetLogInterval = 100
)

// AIClient represents an AI-powered WebRTC client connection
type AIClient struct {
	Conn      *websocket.Conn
	Transport *rtcmedia.WebRTCTransport
	SessionID string

	// AI components
	asrService  recognizer.TranscribeService
	llmProvider llm.LLMProvider
	ttsService  synthesizer.SynthesisService

	// Audio processing
	audioBuffer  chan []byte
	audioDecoder media2.EncoderFunc // Dynamic decoder based on actual codec

	// State
	Mu             sync.RWMutex
	isProcessing   bool
	lastText       string
	conversationID string

	// Add done channel for audio processing
	doneChan chan struct{}

	// Track if we've started receiving audio (prevent duplicate processing)
	AudioReceived bool

	// Knowledge base support
	knowledgeKey string   // Knowledge base identifier
	db           *gorm.DB // Database connection for knowledge base retrieval

	// User information for billing
	userID       uint  // User ID for usage tracking
	credentialID uint  // Credential ID for usage tracking
	assistantID  *uint // Assistant ID for usage tracking

	// Assistant configuration
	llmModel    string  // LLM model from assistant
	maxTokens   int     // Max tokens from assistant
	temperature float32 // Temperature from assistant

	// Echo cancellation: Half-duplex mode
	// When TTS is playing, we pause ASR to prevent AI from hearing itself
	isTTSPlaying  bool      // Whether TTS is currently playing
	ttsEndTime    time.Time // When TTS finished playing (for cooldown)
	ttsCooldownMs int       // Cooldown period after TTS ends (default 500ms)

	// Barge-in (interrupt) support with VAD
	enableVAD            bool          // Whether to enable VAD for barge-in detection
	vadThreshold         float64       // RMS threshold for voice activity detection (0-32768)
	ttsStopChan          chan struct{} // Channel to signal TTS to stop
	bargeInCooldown      int           // Cooldown after barge-in before processing (ms)
	vadConsecutiveFrames int           // Number of consecutive frames needed to trigger barge-in
	vadFrameCounter      int           // Current count of consecutive frames above threshold
}

// NewAIClient creates a new AI-powered client (legacy, uses environment variables)
func NewAIClient(conn *websocket.Conn, transport *rtcmedia.WebRTCTransport, sessionID string, knowledgeKey string, db *gorm.DB, userID uint, credentialID uint, assistantID *uint) (*AIClient, error) {
	// Initialize ASR (using QCloud as example, you can change to other providers)
	asrOpt := recognizer.NewQcloudASROption(
		utils.GetEnv("QCLOUD_APP_ID"),
		utils.GetEnv("QCLOUD_SECRET_ID"),
		utils.GetEnv("QCLOUD_SECRET"),
	)
	// Use 16k_zh model which expects 16kHz audio (default from NewQcloudASROption)
	// The decoded PCM audio is sent directly to ASR at 16kHz
	asrService := recognizer.NewQcloudASR(asrOpt)
	// Initialize LLM (legacy: use OpenAI provider)
	llmProvider, err := llm.NewLLMProvider(
		context.Background(),
		"openai",
		utils.GetEnv("OPENAI_TOKEN"),
		utils.GetEnv("OPENAI_BASE_URL"),
		"你是一个友好的AI助手，请用简洁明了的语言回答问题。",
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create LLM provider: %w", err)
	}

	// Initialize TTS (using QCloud as example)
	// Use 16kHz sample rate for better audio quality
	ttsOpt := synthesizer.NewQcloudTTSConfig(
		utils.GetEnv("QCLOUD_APP_ID"),
		utils.GetEnv("QCLOUD_SECRET_ID"),
		utils.GetEnv("QCLOUD_SECRET"),
		1005, // voice type
		"pcm",
		targetSampleRate, // 16kHz for TTS
	)
	ttsService := synthesizer.NewQCloudService(ttsOpt)

	// TTS player will be created when needed

	// Note: Audio decoder is created dynamically in StartAudioReceiverFromTrack
	// based on the actual codec negotiated with the client (PCMA, Opus, etc.)

	client := &AIClient{
		Conn:           conn,
		Transport:      transport,
		SessionID:      sessionID,
		asrService:     asrService,
		llmProvider:    llmProvider,
		ttsService:     ttsService,
		audioBuffer:    make(chan []byte, 100),
		audioDecoder:   nil, // Will be created when we know the actual codec
		conversationID: fmt.Sprintf("conv_%d", time.Now().UnixNano()),
		doneChan:       make(chan struct{}),
		AudioReceived:  false,
		knowledgeKey:   knowledgeKey,
		db:             db,
		userID:         userID,
		credentialID:   credentialID,
		assistantID:    assistantID,
		// Half-duplex mode: 500ms cooldown after TTS ends
		isTTSPlaying:  false,
		ttsCooldownMs: 500,
		// Barge-in with VAD: Enable by default
		// Threshold: 2000 (lowered for Opus codec which may have different amplitude range)
		// Consecutive frames: 5 frames (~100ms at 20ms/frame) for faster response
		enableVAD:            true,
		vadThreshold:         2000.0,
		ttsStopChan:          make(chan struct{}),
		bargeInCooldown:      100,
		vadConsecutiveFrames: 5,
		vadFrameCounter:      0,
	}

	// Note: OnTrack callback is now set up in websocketHandler after NewAIClient
	// This ensures it's set up before any signaling messages are processed

	// Initialize ASR service
	client.asrService.Init(
		func(text string, isLast bool, duration time.Duration, uuid string) {
			// 记录ASR使用量（当识别完成时）
			if isLast && client.db != nil && client.userID > 0 && client.credentialID > 0 && duration > 0 {
				go func() {
					// 估算音频大小（假设16kHz, 16bit, 单声道，约32KB/秒）
					audioSize := int64(duration.Seconds() * 32000)

					sessionID := uuid
					if sessionID == "" {
						sessionID = fmt.Sprintf("webrtc_%d_%d", client.userID, time.Now().Unix())
					}

					// 获取组织ID（如果助手属于组织）
					var groupID *uint
					if client.assistantID != nil {
						var assistant models.Assistant
						if err := client.db.Where("id = ?", *client.assistantID).First(&assistant).Error; err == nil {
							groupID = assistant.GroupID
						}
					}

					if err := models.RecordASRUsage(
						client.db,
						client.userID,
						client.credentialID,
						client.assistantID,
						groupID,
						sessionID,
						int(duration.Seconds()),
						audioSize,
					); err != nil {
						log.Printf("[Server] 记录ASR使用量失败: %v", err)
					}
				}()
			}

			client.handleASRResult(text, isLast, duration)
		},
		func(err error, isFatal bool) {
			log.Printf("[Server] ASR error: %v (fatal: %v)", err, isFatal)
			if isFatal {
				// Handle fatal error
			} else {
				client.asrService.RestartClient()
			}
		},
	)

	// Connect ASR
	if err := client.asrService.ConnAndReceive(client.conversationID); err != nil {
		return nil, fmt.Errorf("failed to connect ASR: %w", err)
	}

	return client, nil
}

// NewAIClientWithCredential creates a new AI-powered client using credential and assistant configuration
func NewAIClientWithCredential(
	conn *websocket.Conn,
	transport *rtcmedia.WebRTCTransport,
	sessionID string,
	knowledgeKey string,
	db *gorm.DB,
	userID uint,
	credentialID uint,
	assistantID *uint,
	cred *models.UserCredential,
	systemPrompt string,
	maxTokens int,
	temperature float32,
	language string,
	speaker string,
	llmModel string,
) (*AIClient, error) {
	// Initialize ASR service from credential
	factory := recognizer.GetGlobalFactory()

	// Use language from assistant, default to Chinese if not provided
	if language == "" {
		language = "zh"
	}

	// Build ASR configuration from credential
	asrProvider := cred.GetASRProvider()
	if asrProvider == "" {
		return nil, fmt.Errorf("ASR provider not configured in credential")
	}

	// Normalize provider name
	normalizedProvider := normalizeProviderName(asrProvider)

	// Build ASR config
	asrConfig := make(map[string]interface{})
	asrConfig["provider"] = normalizedProvider
	asrConfig["language"] = language

	if cred.AsrConfig != nil {
		for key, value := range cred.AsrConfig {
			asrConfig[key] = value
		}
	}

	// Parse ASR configuration using recognizer package (replacing voice package)
	asrParsedConfig, err := recognizer.NewTranscriberConfigFromMap(normalizedProvider, asrConfig, language)
	if err != nil {
		return nil, fmt.Errorf("failed to parse ASR configuration: %w", err)
	}

	// Create ASR service
	asrService, err := factory.CreateTranscriber(asrParsedConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create ASR service: %w", err)
	}

	// Initialize LLM from credential with assistant configuration
	// Use systemPrompt from assistant, with default fallback
	if systemPrompt == "" {
		systemPrompt = "你是一个友好的AI助手，请用简洁明了的语言回答问题。"
	}

	// Use factory function to create the correct LLM provider based on credential
	llmProvider, err := llm.NewLLMProvider(context.Background(), cred.LLMProvider, cred.LLMApiKey, cred.LLMApiURL, systemPrompt)
	if err != nil {
		return nil, fmt.Errorf("failed to create LLM provider: %w", err)
	}

	// Set LLM model if provided
	if llmModel != "" {
		log.Printf("[Server] LLM Model from assistant: %s", llmModel)
	}

	// Initialize TTS from credential with assistant configuration
	ttsProvider := cred.GetTTSProvider()
	if ttsProvider == "" {
		return nil, fmt.Errorf("TTS provider not configured in credential")
	}

	normalizedTTSProvider := normalizeProviderName(ttsProvider)

	// Build TTS config
	ttsConfig := make(synthesizer.TTSCredentialConfig)
	ttsConfig["provider"] = normalizedTTSProvider

	if cred.TtsConfig != nil {
		for key, value := range cred.TtsConfig {
			ttsConfig[key] = value
		}
	}

	// Set speaker/voice type from assistant if provided
	if speaker != "" {
		ttsConfig["voiceType"] = speaker
		ttsConfig["voice_type"] = speaker
	} else {
		// Set default voice type if not configured
		if _, exists := ttsConfig["voiceType"]; !exists {
			if _, exists = ttsConfig["voice_type"]; !exists {
				// Set default based on provider
				switch normalizedTTSProvider {
				case "tencent", "qcloud":
					ttsConfig["voiceType"] = 1005
				default:
					ttsConfig["voiceType"] = "default"
				}
			}
		}
	}

	// Set audio format and sample rate
	ttsConfig["format"] = "pcm"
	ttsConfig["sampleRate"] = targetSampleRate

	// Create TTS service
	ttsService, err := synthesizer.NewSynthesisServiceFromCredential(ttsConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create TTS service: %w", err)
	}

	// Create client
	client := &AIClient{
		Conn:           conn,
		Transport:      transport,
		SessionID:      sessionID,
		asrService:     asrService,
		llmProvider:    llmProvider,
		ttsService:     ttsService,
		audioBuffer:    make(chan []byte, 100),
		audioDecoder:   nil, // Will be created when we know the actual codec
		conversationID: fmt.Sprintf("conv_%d", time.Now().UnixNano()),
		doneChan:       make(chan struct{}),
		AudioReceived:  false,
		knowledgeKey:   knowledgeKey,
		db:             db,
		userID:         userID,
		credentialID:   credentialID,
		assistantID:    assistantID,
		// Assistant configuration
		llmModel:    llmModel,
		maxTokens:   maxTokens,
		temperature: temperature,
		// Half-duplex mode: 500ms cooldown after TTS ends
		isTTSPlaying:  false,
		ttsCooldownMs: 500,
		// Barge-in with VAD: Enable by default
		enableVAD:            true,
		vadThreshold:         2000.0,
		ttsStopChan:          make(chan struct{}),
		bargeInCooldown:      100,
		vadConsecutiveFrames: 5,
		vadFrameCounter:      0,
	}

	// Initialize ASR service
	client.asrService.Init(
		func(text string, isLast bool, duration time.Duration, uuid string) {
			// 记录ASR使用量（当识别完成时）
			if isLast && client.db != nil && client.userID > 0 && client.credentialID > 0 && duration > 0 {
				go func() {
					// 估算音频大小（假设16kHz, 16bit, 单声道，约32KB/秒）
					audioSize := int64(duration.Seconds() * 32000)

					sessionID := uuid
					if sessionID == "" {
						sessionID = fmt.Sprintf("webrtc_%d_%d", client.userID, time.Now().Unix())
					}

					// 获取组织ID（如果助手属于组织）
					var groupID *uint
					if client.assistantID != nil {
						var assistant models.Assistant
						if err := client.db.Where("id = ?", *client.assistantID).First(&assistant).Error; err == nil {
							groupID = assistant.GroupID
						}
					}

					if err := models.RecordASRUsage(
						client.db,
						client.userID,
						client.credentialID,
						client.assistantID,
						groupID,
						sessionID,
						int(duration.Seconds()),
						audioSize,
					); err != nil {
						log.Printf("[Server] 记录ASR使用量失败: %v", err)
					}
				}()
			}

			client.handleASRResult(text, isLast, duration)
		},
		func(err error, isFatal bool) {
			log.Printf("[Server] ASR error: %v (fatal: %v)", err, isFatal)
			if isFatal {
				// Handle fatal error
			} else {
				client.asrService.RestartClient()
			}
		},
	)

	// Connect ASR
	if err := client.asrService.ConnAndReceive(client.conversationID); err != nil {
		return nil, fmt.Errorf("failed to connect ASR: %w", err)
	}

	return client, nil
}

// normalizeProviderName normalizes provider name (e.g., "qcloud" -> "tencent", "qiniu" -> "qiniu")
func normalizeProviderName(provider string) string {
	provider = strings.ToLower(provider)
	switch provider {
	case "qcloud", "tencent", "tencentcloud":
		return "tencent"
	case "qiniu":
		return "qiniu"
	case "xunfei", "iflytek":
		return "xunfei"
	case "openai":
		return "openai"
	case "volcengine", "volcano":
		return "volcengine"
	case "minimax":
		return "minimax"
	default:
		return provider
	}
}

// Close closes the AI client
func (c *AIClient) Close() error {
	log.Printf("[Server] Closing AI client for session: %s", c.SessionID)

	// Stop TTS immediately
	c.stopTTS()

	// Close the done channel to signal audio processing to stop
	c.Mu.Lock()
	if c.doneChan != nil {
		close(c.doneChan)
		c.doneChan = nil
	}
	// Mark as closed to prevent further TTS generation
	c.isTTSPlaying = false
	c.Mu.Unlock()

	if c.asrService != nil {
		c.asrService.StopConn()
	}
	if c.ttsService != nil {
		c.ttsService.Close()
	}
	if c.Transport != nil {
		c.Transport.Close()
	}
	if c.Conn != nil {
		c.Conn.Close()
	}
	log.Printf("[Server] AI client closed for session: %s", c.SessionID)
	return nil
}

// setTTSPlaying sets the TTS playing state (for half-duplex echo cancellation)
func (c *AIClient) setTTSPlaying(playing bool) {
	c.Mu.Lock()
	defer c.Mu.Unlock()
	c.isTTSPlaying = playing
	if playing {
		// Create new stop channel for this TTS session
		c.ttsStopChan = make(chan struct{})
	} else {
		c.ttsEndTime = time.Now()
	}
	log.Printf("[Server] TTS playing state: %v", playing)
}

// stopTTS signals TTS to stop immediately (for barge-in)
func (c *AIClient) stopTTS() {
	c.Mu.Lock()
	defer c.Mu.Unlock()
	if c.isTTSPlaying && c.ttsStopChan != nil {
		select {
		case <-c.ttsStopChan:
			// Already closed
		default:
			close(c.ttsStopChan)
		}
		c.isTTSPlaying = false
		c.ttsEndTime = time.Now()
		log.Printf("[Server] TTS stopped by barge-in")
	}
}

// shouldStopTTS checks if TTS should stop
func (c *AIClient) shouldStopTTS() bool {
	c.Mu.RLock()
	defer c.Mu.RUnlock()
	if c.ttsStopChan == nil {
		return false
	}
	select {
	case <-c.ttsStopChan:
		return true
	default:
		return false
	}
}

// SetEnableVAD enables or disables VAD for barge-in detection
func (c *AIClient) SetEnableVAD(enable bool) {
	c.Mu.Lock()
	defer c.Mu.Unlock()
	c.enableVAD = enable
	log.Printf("[Server] VAD enabled: %v", enable)
}

// SetVADThreshold sets the VAD threshold (0-32768, typical speech ~500-5000)
func (c *AIClient) SetVADThreshold(threshold float64) {
	c.Mu.Lock()
	defer c.Mu.Unlock()
	c.vadThreshold = threshold
	log.Printf("[Server] VAD threshold set to: %.2f", threshold)
}

// SetVADConsecutiveFrames sets how many consecutive frames above threshold needed for barge-in
// Each frame is ~20ms, so 10 frames = ~200ms of sustained speech needed
func (c *AIClient) SetVADConsecutiveFrames(frames int) {
	c.Mu.Lock()
	defer c.Mu.Unlock()
	c.vadConsecutiveFrames = frames
	log.Printf("[Server] VAD consecutive frames set to: %d (~%dms)", frames, frames*20)
}

// calculateRMS calculates the Root Mean Square of 16-bit PCM audio data
func calculateRMS(pcmData []byte) float64 {
	if len(pcmData) < 2 {
		return 0
	}

	var sumSquares float64
	sampleCount := len(pcmData) / 2

	for i := 0; i < len(pcmData)-1; i += 2 {
		// Convert little-endian bytes to int16
		sample := int16(pcmData[i]) | int16(pcmData[i+1])<<8
		sumSquares += float64(sample) * float64(sample)
	}

	if sampleCount == 0 {
		return 0
	}

	return math.Sqrt(sumSquares / float64(sampleCount))
}

// checkBargeIn checks if user is speaking and should interrupt TTS
// Returns true if barge-in detected (TTS should stop)
// Uses consecutive frame detection to avoid false triggers from TTS echo
func (c *AIClient) checkBargeIn(pcmData []byte) bool {
	c.Mu.Lock()
	enableVAD := c.enableVAD
	vadThreshold := c.vadThreshold
	isTTSPlaying := c.isTTSPlaying
	consecutiveFramesNeeded := c.vadConsecutiveFrames

	// Only check for barge-in when VAD is enabled and TTS is playing
	if !enableVAD || !isTTSPlaying {
		c.vadFrameCounter = 0 // Reset counter when not in barge-in detection mode
		c.Mu.Unlock()
		return false
	}

	// Calculate audio energy (RMS)
	rms := calculateRMS(pcmData)

	// Debug: Log RMS values periodically to help tune threshold
	if c.vadFrameCounter%50 == 0 || rms > vadThreshold*0.5 {
		log.Printf("[Server] VAD: RMS=%.2f, Threshold=%.2f, Counter=%d, TTS=%v",
			rms, vadThreshold, c.vadFrameCounter, isTTSPlaying)
	}

	// Check if energy exceeds threshold
	if rms > vadThreshold {
		c.vadFrameCounter++
		// Only trigger barge-in after consecutive frames exceed threshold
		// This filters out short bursts from TTS echo
		if c.vadFrameCounter >= consecutiveFramesNeeded {
			log.Printf("[Server] Barge-in detected! RMS: %.2f, Threshold: %.2f, Consecutive frames: %d",
				rms, vadThreshold, c.vadFrameCounter)
			c.Mu.Unlock()
			c.stopTTS()
			return true
		}
	} else {
		// Reset counter if energy drops below threshold
		c.vadFrameCounter = 0
	}

	c.Mu.Unlock()
	return false
}

// shouldProcessAudio checks if we should process incoming audio
// Returns false during TTS playback and cooldown period (half-duplex mode)
// But if VAD detects barge-in, it will stop TTS and return true
func (c *AIClient) shouldProcessAudio() bool {
	c.Mu.RLock()
	defer c.Mu.RUnlock()

	// Don't process audio while TTS is playing (unless barge-in detected)
	if c.isTTSPlaying {
		return false
	}

	// Don't process audio during cooldown period after TTS ends
	if !c.ttsEndTime.IsZero() {
		elapsed := time.Since(c.ttsEndTime)
		if elapsed < time.Duration(c.ttsCooldownMs)*time.Millisecond {
			return false
		}
	}

	return true
}

// handleASRResult handles ASR recognition results
func (c *AIClient) handleASRResult(text string, isLast bool, duration time.Duration) {
	if text == "" {
		return
	}

	c.Mu.Lock()
	c.lastText = text
	c.Mu.Unlock()

	log.Printf("[Server] ASR Result: %s (isLast: %v, duration: %v)", text, isLast, duration)

	// Check if it's a complete sentence (even if isLast=false)
	isComplete := isCompleteSentence(text)

	if isLast {
		// Final result - process with LLM
		go c.processWithLLM(text)
	} else if isComplete {
		// Sentence end - also process if it's a meaningful sentence
		// Filter meaningless text
		filteredText := filterText(text)
		if filteredText != "" && !isMeaninglessText(filteredText) {
			log.Printf("[Server] Processing sentence end (isLast=false but complete): %s", filteredText)
			go c.processWithLLM(filteredText)
		}
	}
}

// isCompleteSentence checks if text contains sentence ending markers
func isCompleteSentence(text string) bool {
	// 简化处理：任何非空文本都视为完整句子
	return text != ""
}

// filterText removes whitespace only
func filterText(text string) string {
	return strings.TrimSpace(text)
}

// isMeaninglessText checks if text is meaningless (should be filtered)
func isMeaninglessText(text string) bool {
	if text == "" {
		return true
	}
	// Filter single character meaningless words
	meaninglessWords := []string{"嗯", "啊", "哦", "额", "呃", "0", "1", "2", "3", "4", "5", "6", "7", "8", "9"}
	cleaned := strings.TrimSpace(text)
	for _, word := range meaninglessWords {
		if cleaned == word {
			return true
		}
	}
	// Filter very short text (less than 2 characters after cleaning)
	if len([]rune(cleaned)) < 2 {
		return true
	}
	return false
}

// processWithLLM processes text with LLM and generates TTS
func (c *AIClient) processWithLLM(userText string) {
	c.Mu.Lock()
	if c.isProcessing {
		c.Mu.Unlock()
		return
	}
	c.isProcessing = true
	c.Mu.Unlock()

	defer func() {
		c.Mu.Lock()
		c.isProcessing = false
		c.Mu.Unlock()
	}()

	log.Printf("[Server] Processing with LLM: %s", userText)

	// Build query text (if knowledge base is provided, search knowledge base first)
	queryText := userText
	if c.knowledgeKey != "" && c.db != nil {
		// Search knowledge base
		knowledgeResults, err := models.SearchKnowledgeBase(c.db, c.knowledgeKey, userText, 5)
		if err != nil {
			log.Printf("[Server] Failed to search knowledge base: %v", err)
			// Use original query when search fails
			queryText = userText
		} else if len(knowledgeResults) > 0 {
			// Build context: use natural prompt template format, avoid AI mentioning "documents"
			var contextBuilder strings.Builder
			contextBuilder.WriteString(fmt.Sprintf("用户问题: %s\n\n", userText))
			// Directly provide information content, don't emphasize "documents" or "reference information"
			for i, result := range knowledgeResults {
				if i > 0 {
					contextBuilder.WriteString("\n\n")
				}
				contextBuilder.WriteString(result.Content)
			}
			contextBuilder.WriteString("\n\n请基于以上信息回答用户问题，回答要自然流畅，不要提及信息来源。")
			queryText = contextBuilder.String()
			log.Printf("[Server] Retrieved %d relevant documents from knowledge base (key: %s)", len(knowledgeResults), c.knowledgeKey)
		} else {
			// No relevant content found, use original query
			queryText = userText
		}
	}

	// Query LLM with assistant configuration
	c.Mu.RLock()
	model := c.llmModel
	maxTokens := c.maxTokens
	temp := c.temperature
	c.Mu.RUnlock()

	// Use model from assistant, fallback to environment variable if not set
	if model == "" {
		model = utils.GetEnv("OPENAI_MODEL")
		if model == "" {
			model = "gpt-4o" // Default model
		}
	}

	// Build query options
	options := llm.QueryOptions{
		Model: model,
	}

	// Set maxTokens if configured (0 means no limit)
	if maxTokens > 0 {
		options.MaxTokens = &maxTokens
	}

	// Set temperature if configured (0 means use default)
	if temp > 0 {
		options.Temperature = &temp
	} else {
		// Default temperature
		defaultTemp := float32(0.7)
		options.Temperature = &defaultTemp
	}

	// Query LLM with options
	response, err := c.llmProvider.QueryWithOptions(queryText, options)
	if err != nil {
		log.Printf("[Server] LLM error: %v", err)
		return
	}

	log.Printf("[Server] LLM Response: %s", response)

	// Generate TTS
	c.GenerateTTS(response)
}

// GenerateTTS generates TTS audio and sends it via WebRTC
func (c *AIClient) GenerateTTS(text string) {
	log.Printf("[Server] Generating TTS for: %s", text)

	ctx := context.Background()
	txTrack := c.Transport.GetTxTrack()
	if txTrack == nil {
		log.Printf("[Server] txTrack is nil, waiting...")
		// Wait for track
		for i := 0; i < maxConnectionRetries; i++ {
			txTrack = c.Transport.GetTxTrack()
			if txTrack != nil {
				break
			}
			time.Sleep(connectionRetryDelay)
		}
		if txTrack == nil {
			log.Printf("[Server] Failed to get txTrack")
			return
		}
	}

	// Create TTS handler
	ttsHandler := &TTSSender{
		txTrack:   txTrack,
		client:    c,
		audioSize: 0,
		startTime: time.Now(),
	}

	// Half-duplex mode: Set TTS playing state to pause ASR
	c.setTTSPlaying(true)

	// Synthesize
	if err := c.ttsService.Synthesize(ctx, ttsHandler, text); err != nil {
		log.Printf("[Server] TTS synthesis error: %v", err)
		c.setTTSPlaying(false) // Reset state on error
		return
	}

	// TTS finished, start cooldown period
	c.setTTSPlaying(false)

	// 记录TTS使用量
	if c.db != nil && c.userID > 0 && c.credentialID > 0 && ttsHandler.audioSize > 0 {
		go func() {
			ttsDuration := int(time.Since(ttsHandler.startTime).Seconds())
			if ttsDuration == 0 {
				// 如果时长为0，根据音频大小估算（假设16kHz, 16bit, 单声道）
				ttsDuration = int(float64(ttsHandler.audioSize) / 32000)
			}

			sessionID := fmt.Sprintf("webrtc_%d_%d", c.userID, time.Now().Unix())

			// 获取组织ID（如果助手属于组织）
			var groupID *uint
			if c.assistantID != nil {
				var assistant models.Assistant
				if err := c.db.Where("id = ?", *c.assistantID).First(&assistant).Error; err == nil {
					groupID = assistant.GroupID
				}
			}

			if err := models.RecordTTSUsage(
				c.db,
				c.userID,
				c.credentialID,
				c.assistantID,
				groupID,
				sessionID,
				ttsDuration,
				ttsHandler.audioSize,
			); err != nil {
				log.Printf("[Server] 记录TTS使用量失败: %v", err)
			}
		}()
	}
}

// TTSSender handles TTS audio data and sends it via WebRTC
type TTSSender struct {
	txTrack   *webrtc.TrackLocalStaticSample
	client    *AIClient
	buffer    []byte
	audioSize int64     // Track total audio size
	startTime time.Time // Track TTS start time
}

func (t *TTSSender) OnMessage(data []byte) {
	// Check if TTS should stop (barge-in detected or connection closed)
	// Skip processing if already interrupted to avoid log spam
	if t.client.shouldStopTTS() {
		return
	}

	// Check if connection is still valid
	t.client.Mu.RLock()
	connClosed := t.client.doneChan == nil
	transportClosed := t.client.Transport == nil
	t.client.Mu.RUnlock()

	if connClosed || transportClosed {
		log.Printf("[Server] TTS stopped: connection closed")
		return
	}

	// 累计音频数据大小
	if len(data) > 0 {
		t.audioSize += int64(len(data))
	}

	// Remove WAV header if present (if TTS returns WAV format)
	// Note: QCloud TTS returns PCM directly, but other providers might return WAV
	// data = encoder.StripWavHeader(data) // Uncomment if needed

	// PCMA standard sample rate (RFC 3551)
	const pcmaSampleRate = 8000

	// Get TTS format
	ttsFormat := t.client.ttsService.Format()

	// Resample from TTS sample rate to PCMA sample rate (8kHz)
	// TTS typically returns 16kHz, but PCMA needs 8kHz
	if ttsFormat.SampleRate != pcmaSampleRate {
		resampled, err := media2.ResamplePCM(data, ttsFormat.SampleRate, pcmaSampleRate)
		if err != nil {
			log.Printf("[Server] Resample error: %v", err)
			return
		}
		data = resampled
	}

	// Encode to PCMA (now at 8kHz)
	pcmaData, err := encoder.Pcm2pcma(data)
	if err != nil {
		log.Printf("[Server] Encode PCMA error: %v", err)
		return
	}

	// Send in frames
	t.sendPCMAFrames(pcmaData)
}

func (t *TTSSender) OnTimestamp(timestamp synthesizer.SentenceTimestamp) {
	// Not used for now
}

func (t *TTSSender) sendPCMAFrames(pcmaData []byte) {
	frameDuration := 20 * time.Millisecond
	startTime := time.Now()
	frameCount := 0

	// PCMA frame size at 8kHz: 20ms * 8000Hz = 160 samples = 160 bytes (8-bit)
	const pcmaFrameSize = 160

	for i := 0; i < len(pcmaData); i += pcmaFrameSize {
		// Check for barge-in: stop sending if user started speaking
		if t.client.shouldStopTTS() {
			log.Printf("[Server] TTS interrupted by barge-in after %d frames", frameCount)
			return
		}

		// Check if connection is still valid
		t.client.Mu.RLock()
		connClosed := t.client.doneChan == nil
		transportClosed := t.client.Transport == nil
		t.client.Mu.RUnlock()

		if connClosed || transportClosed {
			log.Printf("[Server] TTS stopped: connection closed after %d frames", frameCount)
			return
		}

		// Check if txTrack is still valid
		if t.txTrack == nil {
			log.Printf("[Server] TTS stopped: txTrack is nil after %d frames", frameCount)
			return
		}

		end := i + pcmaFrameSize
		if end > len(pcmaData) {
			end = len(pcmaData)
		}

		// Calculate exact send time
		expectedTime := startTime.Add(time.Duration(frameCount) * frameDuration)
		if now := time.Now(); expectedTime.After(now) {
			time.Sleep(expectedTime.Sub(now))
		}

		sample := media.Sample{
			Data:     pcmaData[i:end],
			Duration: frameDuration,
		}

		if err := t.txTrack.WriteSample(sample); err != nil {
			log.Printf("[Server] Error writing sample: %v", err)
			return
		}

		frameCount++
	}

	log.Printf("[Server] Sent %d TTS frames (%d bytes)", frameCount, len(pcmaData))
}

// createDecoderForCodec creates the appropriate decoder based on codec type
func (c *AIClient) createDecoderForCodec(mimeType string, clockRate int) (media2.EncoderFunc, error) {
	var codecName string
	var sourceSampleRate int

	// Determine codec name and sample rate from MIME type
	switch mimeType {
	case "audio/PCMA":
		codecName = "pcma"
		sourceSampleRate = 8000
	case "audio/PCMU":
		codecName = "pcmu"
		sourceSampleRate = 8000
	case "audio/opus":
		codecName = "opus"
		sourceSampleRate = 48000
	case "audio/G722":
		codecName = "g722"
		sourceSampleRate = 8000 // G.722 uses 16kHz but clock rate is 8000
	default:
		return nil, fmt.Errorf("unsupported codec: %s", mimeType)
	}

	fmt.Printf("[Server] Creating decoder: %s (%dHz) -> PCM (%dHz)\n",
		codecName, sourceSampleRate, targetSampleRate)

	// Determine bit depth based on codec
	bitDepth := 8
	if codecName == "opus" || codecName == "g722" {
		bitDepth = 16 // These codecs decode to 16-bit PCM
	}

	decoder, err := encoder.CreateDecode(
		media2.CodecConfig{
			Codec:         codecName,
			SampleRate:    sourceSampleRate,
			Channels:      audioChannels,
			BitDepth:      bitDepth,
			FrameDuration: "20ms",
		},
		media2.CodecConfig{
			Codec:         "pcm",
			SampleRate:    targetSampleRate, // 16kHz for ASR
			Channels:      audioChannels,
			BitDepth:      audioBitDepth, // 16-bit PCM for ASR
			FrameDuration: "20ms",
		},
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create %s decoder: %w", codecName, err)
	}

	return decoder, nil
}

// startAudioReceiver starts receiving and processing audio
// This function is kept for backward compatibility but the actual processing
// now happens in StartAudioReceiverFromTrack
func (c *AIClient) startAudioReceiver() error {
	// Wait for rxTrack to be available
	// OnTrack callback should fire when client's track is received
	// In example1, client waits for server's track - same logic here but reversed
	var rxTrack *webrtc.TrackRemote
	for i := 0; i < maxConnectionRetries; i++ {
		rxTrack = c.Transport.GetRxTrack()
		if rxTrack != nil {
			fmt.Printf("[Server] rxTrack received after %d attempts\n", i+1)
			break
		}
		if i%connectionStateLogInterval == 0 {
			fmt.Printf("[Server] Waiting for rxTrack... (attempt %d/%d, connection state: %s)\n",
				i+1, maxConnectionRetries, c.Transport.GetConnectionState().String())
		}
		time.Sleep(connectionRetryDelay)
	}

	if rxTrack == nil {
		return fmt.Errorf("rxTrack not available after %d retries (connection state: %s)",
			maxConnectionRetries, c.Transport.GetConnectionState().String())
	}

	return c.StartAudioReceiverFromTrack(rxTrack)
}

// StartAudioReceiverFromTrack starts receiving and processing audio from a specific track
func (c *AIClient) StartAudioReceiverFromTrack(rxTrack *webrtc.TrackRemote) error {
	if rxTrack == nil {
		return fmt.Errorf("rxTrack is nil")
	}

	codecParams := rxTrack.Codec()
	fmt.Printf("[Server] Received track: %s, %dHz\n", codecParams.MimeType, codecParams.ClockRate)

	// Create decoder based on actual codec type
	decoder, err := c.createDecoderForCodec(codecParams.MimeType, int(codecParams.ClockRate))
	if err != nil {
		return fmt.Errorf("failed to create decoder for %s: %w", codecParams.MimeType, err)
	}

	c.Mu.Lock()
	c.audioDecoder = decoder
	c.Mu.Unlock()

	fmt.Printf("[Server] Created decoder for codec: %s\n", codecParams.MimeType)

	packetCount := 0
	for {
		// Check if we should stop processing
		select {
		case <-c.doneChan:
			return nil
		default:
		}

		// Check if client is still valid
		c.Mu.RLock()
		currentDecoder := c.audioDecoder
		asrService := c.asrService
		c.Mu.RUnlock()

		if currentDecoder == nil {
			return fmt.Errorf("decoder is nil")
		}

		packet, _, err := rxTrack.ReadRTP()
		if err != nil {
			return fmt.Errorf("error reading RTP packet: %w", err)
		}

		// Debug: Log packet information
		if packetCount%100 == 0 {
			fmt.Printf("[Server] Received RTP packet #%d, payload size: %d, payload type: %d\n",
				packetCount, len(packet.Payload), packet.PayloadType)
		}

		// Decode audio to PCM (supports PCMA, PCMU, Opus, G722)
		audioFrame := &media2.AudioPacket{Payload: packet.Payload}
		decodedFrames, err := currentDecoder(audioFrame)
		if err != nil {
			if packetCount%packetLogInterval == 0 {
				log.Printf("[Server] Decode error: %v", err)
			}
			packetCount++
			continue
		}

		// Collect PCM data
		var pcmData []byte
		for _, frame := range decodedFrames {
			if af, ok := frame.(*media2.AudioPacket); ok && len(af.Payload) > 0 {
				pcmData = append(pcmData, af.Payload...)
			}
		}

		// Debug: Log decoded data
		if packetCount%100 == 0 && len(pcmData) > 0 {
			fmt.Printf("[Server] Decoded PCM data size: %d bytes\n", len(pcmData))
		}

		// Barge-in detection: Check if user is speaking while TTS is playing
		// If detected, stop TTS immediately and resume ASR processing
		if len(pcmData) > 0 {
			c.checkBargeIn(pcmData)
		}

		// Half-duplex mode: Skip sending to ASR while TTS is playing or during cooldown
		// This prevents AI from hearing itself and starting a self-conversation loop
		if !c.shouldProcessAudio() {
			packetCount++
			if packetCount%packetLogInterval == 0 {
				fmt.Printf("[Server] Skipped %d RTP packets (TTS playing or cooldown)\n", packetCount)
			}
			continue
		}

		// Send to ASR (check if ASR service is still available)
		if len(pcmData) > 0 && asrService != nil {
			if err := asrService.SendAudioBytes(pcmData); err != nil {
				log.Printf("[Server] ASR send error: %v", err)
				asrService.RestartClient()
			} else if packetCount%100 == 0 {
				fmt.Printf("[Server] Sent %d bytes to ASR\n", len(pcmData))
			}
		}

		packetCount++
		if packetCount%packetLogInterval == 0 {
			fmt.Printf("[Server] Processed %d RTP packets\n", packetCount)
		}
	}
}
