package rtcmedia

import (
	"testing"

	"github.com/LingByte/SoulNexus/pkg/webrtc/constants"
	"github.com/stretchr/testify/assert"
)

func TestNewWebRTCTransport(t *testing.T) {
	transport := NewWebRTCTransport(WebRTCOption{
		Codec:      constants.DefaultCodec,
		ICETimeout: constants.DefaultICETimeout,
		StreamID:   constants.DefaultStreamID,
	})
	assert.Equal(t, transport.opt.Codec, constants.DefaultCodec)
	assert.Equal(t, transport.opt.ICETimeout, constants.DefaultICETimeout)
	assert.Equal(t, transport.opt.StreamID, constants.DefaultStreamID)
}

//
//func TestFullConnection(t *testing.T) {
//	client := NewWebRTCTransport(WebRTCOption{
//		Codec:      constants.CodecPCMA,
//		ICETimeout: constants.DefaultICETimeout,
//		StreamID:   constants.DefaultStreamID,
//		ICEServers: []webrtc.ICEServer{
//			{
//				URLs: []string{"stun:stun.l.google.com:19302"},
//			},
//		},
//	})
//	client.NewPeerConnection()
//	offer, candidates, err := client.CreateOffer()
//	if err != nil {
//		t.Fatal(err)
//	}
//
//	server := NewWebRTCTransport(WebRTCOption{
//		Codec:      constants.CodecPCMA,
//		ICETimeout: constants.DefaultICETimeout,
//		StreamID:   constants.DefaultStreamID,
//		ICEServers: []webrtc.ICEServer{
//			{
//				URLs: []string{"stun:stun.l.google.com:19302"},
//			},
//		},
//	})
//
//	server.NewPeerConnection()
//	server.SetRemoteDescription(offer)
//	answer, serverCandidates, err := server.CreateAnswer(candidates)
//	if err != nil {
//		t.Fatal(err)
//	}
//	client.SetRemoteDescription(answer)
//	for i := range serverCandidates {
//		client.AddICECandidate(serverCandidates[i])
//	}
//
//	// Wait for connection to establish
//	time.Sleep(2 * time.Second)
//
//	// Verify connection is established
//	if client.GetConnectionState() != webrtc.PeerConnectionStateConnected {
//		t.Fatal("Client connection not established")
//	}
//	if server.GetConnectionState() != webrtc.PeerConnectionStateConnected {
//		t.Fatal("Server connection not established")
//	}
//
//	t.Skip("Skipping voice test")
//	// Server reads file and sends audio data
//	go func() {
//		// 在开始发送数据前添加小延迟，确保连接完全建立
//		time.Sleep(100 * time.Millisecond)
//
//		// Open audio file
//		file, err := os.Open("ringing.wav")
//		if err != nil {
//			fmt.Println("Error opening file:", err)
//			return
//		}
//		defer file.Close()
//
//		w := wav.NewReader(file)
//		f, err := w.Format()
//		if err != nil {
//			fmt.Println("Error getting WAV format:", err)
//			return
//		}
//
//		// Calculate correct buffer size for PCMA (160 samples at 8kHz)
//		bytesPerSample := int(f.BitsPerSample) / 8
//		channels := int(f.NumChannels)
//		samplesPerFrame := 160 // PCMA standard frame size (20ms at 8kHz)
//		bufferSize := samplesPerFrame * bytesPerSample * channels
//
//		// Get server's transmit track
//		txTrack := server.GetTxTrack()
//		if txTrack == nil {
//			fmt.Println("Server txTrack is nil")
//			return
//		}
//
//		// Read and send audio data in chunks
//		buffer := make([]byte, bufferSize)
//		frameDuration := time.Millisecond * 20 // Standard for PCMA
//
//		for {
//			n, err := w.Read(buffer)
//			if err == io.EOF {
//				break
//			}
//			if err != nil {
//				fmt.Println("Error reading file:", err)
//				break
//			}
//
//			// Create media sample with correct duration
//			sample := media.Sample{
//				Data:     buffer[:n],
//				Duration: frameDuration,
//			}
//
//			// Send audio sample through WebRTC track
//			err = txTrack.WriteSample(sample)
//			if err != nil {
//				fmt.Println("Error writing sample:", err)
//				break
//			}
//
//			// Control sending rate
//			time.Sleep(frameDuration)
//		}
//
//		fmt.Println("Finished sending audio data")
//	}()
//
//	// Client receives and plays audio
//	go func() {
//		// Initialize miniaudio context for playback
//		ctx, err := malgo.InitContext(nil, malgo.ContextConfig{}, func(message string) {
//			fmt.Printf("LOG <%v>\n", message)
//		})
//		if err != nil {
//			fmt.Println("Error initializing audio context:", err)
//			return
//		}
//		defer func() {
//			_ = ctx.Uninit()
//			ctx.Free()
//		}()
//
//		// Configure audio device for PCMA (8kHz, mono)
//		deviceConfig := malgo.DefaultDeviceConfig(malgo.Playback)
//		deviceConfig.Playback.Format = malgo.FormatS16
//		deviceConfig.Playback.Channels = 1
//		deviceConfig.SampleRate = 8000 // PCMA standard sample rate
//		deviceConfig.Alsa.NoMMap = 1
//
//		// Buffer for audio playback (PCMA data)
//		audioBuffer := make(chan []byte, 500) // Increased buffer size
//
//		// Wait for rxTrack to be available (OnTrack callback may not have fired yet)
//		var rxTrack *webrtc.TrackRemote
//		for i := 0; i < 50; i++ { // Wait up to 5 seconds
//			rxTrack = client.GetRxTrack()
//			if rxTrack != nil {
//				break
//			}
//			time.Sleep(100 * time.Millisecond)
//		}
//
//		if rxTrack == nil {
//			fmt.Println("Client rxTrack is nil after waiting")
//			return
//		}
//
//		fmt.Println("Client rxTrack received, codec:", rxTrack.Codec().MimeType)
//
//		// Goroutine to receive audio packets
//		go func() {
//			packetCount := 0
//			for {
//				packet, _, err := rxTrack.ReadRTP()
//				if err != nil {
//					fmt.Println("Error reading RTP packet:", err)
//					break
//				}
//
//				// PCMA payload can be used directly
//				pcmaPayload := packet.Payload
//				if len(pcmaPayload) == 0 {
//					continue
//				}
//
//				packetCount++
//				if packetCount%50 == 0 {
//					fmt.Printf("Received %d RTP packets, payload size: %d bytes\n", packetCount, len(pcmaPayload))
//				}
//
//				// Send PCMA data directly to playback buffer
//				select {
//				case audioBuffer <- pcmaPayload:
//				default:
//					fmt.Println("Audio buffer full, dropping frames")
//				}
//			}
//		}()
//
//		// Audio playback callback
//		onSamples := func(pOutputSample, pInputSamples []byte, framecount uint32) {
//			// framecount is the number of frames requested
//			// Each frame is 1 byte for PCMA, so total bytes needed = framecount
//			bytesNeeded := int(framecount)
//			bytesCopied := 0
//
//			// Try to fill the buffer with PCMA data
//			for bytesCopied < bytesNeeded {
//				select {
//				case data := <-audioBuffer:
//					// Copy as much as we can
//					remaining := bytesNeeded - bytesCopied
//					if len(data) <= remaining {
//						copy(pOutputSample[bytesCopied:], data)
//						bytesCopied += len(data)
//					} else {
//						copy(pOutputSample[bytesCopied:], data[:remaining])
//						bytesCopied = bytesNeeded
//					}
//				default:
//					// No data available, fill with silence (0 for PCMA)
//					for i := bytesCopied; i < bytesNeeded; i++ {
//						pOutputSample[i] = 0
//					}
//					bytesCopied = bytesNeeded
//				}
//			}
//		}
//
//		deviceCallbacks := malgo.DeviceCallbacks{
//			Data: onSamples,
//		}
//
//		device, err := malgo.InitDevice(ctx.Context, deviceConfig, deviceCallbacks)
//		if err != nil {
//			fmt.Println("Error initializing audio device:", err)
//			return
//		}
//		defer device.Uninit()
//
//		err = device.Start()
//		if err != nil {
//			fmt.Println("Error starting audio device:", err)
//			return
//		}
//
//		// Keep playing for duration of test
//		time.Sleep(10 * time.Second)
//	}()
//
//	// Let the test run
//	time.Sleep(15 * time.Second)
//}
