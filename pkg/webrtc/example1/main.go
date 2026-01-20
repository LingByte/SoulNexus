package main

import (
	"fmt"
	"io"
	"os"
	"time"

	"github.com/LingByte/SoulNexus/pkg/devices"
	media2 "github.com/LingByte/SoulNexus/pkg/media"
	"github.com/LingByte/SoulNexus/pkg/media/encoder"
	"github.com/LingByte/SoulNexus/pkg/webrtc/constants"
	"github.com/LingByte/SoulNexus/pkg/webrtc/rtcmedia"
	"github.com/gen2brain/malgo"
	"github.com/pion/webrtc/v3"
	"github.com/pion/webrtc/v3/pkg/media"
	"github.com/youpy/go-wav"
)

func main() {
	fmt.Println("=== WebRTC Audio Transmission Example ===")
	fmt.Println("This example demonstrates:")
	fmt.Println("1. Server reads ringing.wav file")
	fmt.Println("2. Server sends audio to client via WebRTC")
	fmt.Println("3. Client receives and plays audio")
	fmt.Println()

	// Create client (will receive audio)
	client := rtcmedia.NewWebRTCTransport(rtcmedia.WebRTCOption{
		Codec:      constants.CodecPCMA, // Use PCMA for simplicity
		ICETimeout: constants.DefaultICETimeout,
		StreamID:   constants.DefaultStreamID,
		ICEServers: []webrtc.ICEServer{
			{
				URLs: []string{"stun:stun.l.google.com:19302"},
			},
		},
	})
	client.NewPeerConnection()

	// Create offer
	offer, candidates, err := client.CreateOffer()
	if err != nil {
		fmt.Printf("Error creating offer: %v\n", err)
		return
	}
	fmt.Printf("Client created offer with %d candidates\n", len(candidates))

	// Create server (will send audio)
	server := rtcmedia.NewWebRTCTransport(rtcmedia.WebRTCOption{
		Codec:      constants.CodecPCMA,
		ICETimeout: constants.DefaultICETimeout,
		StreamID:   constants.DefaultStreamID,
		ICEServers: []webrtc.ICEServer{
			{
				URLs: []string{"stun:stun.l.google.com:19302"},
			},
		},
	})
	server.NewPeerConnection()

	// Server sets remote description and creates answer
	server.SetRemoteDescription(offer)
	answer, serverCandidates, err := server.CreateAnswer(candidates)
	if err != nil {
		fmt.Printf("Error creating answer: %v\n", err)
		return
	}
	fmt.Printf("Server created answer with %d candidates\n", len(serverCandidates))

	// Client sets remote description and adds ICE candidates
	client.SetRemoteDescription(answer)
	for _, candidate := range serverCandidates {
		client.AddICECandidate(candidate)
	}

	// Wait for connection to establish
	fmt.Println("Waiting for connection...")
	timeout := time.After(10 * time.Second)
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	connected := false
	for !connected {
		select {
		case <-timeout:
			fmt.Println("Connection timeout!")
			return
		case <-ticker.C:
			if client.GetConnectionState() == webrtc.PeerConnectionStateConnected &&
				server.GetConnectionState() == webrtc.PeerConnectionStateConnected {
				connected = true
				fmt.Println("✓ Connection established!")
			}
		}
	}

	// Server: Read and send audio file
	go func() {
		// Wait a bit to ensure connection is fully established
		time.Sleep(200 * time.Millisecond)

		fmt.Println("\n[Server] Starting to send signal...")
		txTrack := server.GetTxTrack()
		if txTrack == nil {
			fmt.Println("[Server] Error: txTrack is nil")
			return
		}

		// Open WAV file (try ringring.wav first, then ringing.wav)
		var file *os.File
		file, err = os.Open("ringring.wav")
		if err != nil {
			file, err = os.Open("ringing.wav")
		}
		if err != nil {
			fmt.Printf("[Server] Error opening file: %v\n", err)
			return
		}
		defer file.Close()

		w := wav.NewReader(file)
		format, err := w.Format()
		if err != nil {
			fmt.Printf("[Server] Error getting WAV format: %v\n", err)
			return
		}

		fmt.Printf("[Server] WAV format: %dHz, %d channels, %d bits\n",
			format.SampleRate, format.NumChannels, format.BitsPerSample)

		// Read entire WAV file first for better quality processing
		var allPCMData []byte
		tempBuffer := make([]byte, 8192)
		for {
			n, err := w.Read(tempBuffer)
			if err == io.EOF {
				break
			}
			if err != nil {
				fmt.Printf("[Server] Error reading file: %v\n", err)
				return
			}
			allPCMData = append(allPCMData, tempBuffer[:n]...)
		}
		fmt.Printf("[Server] Read %d bytes from WAV file\n", len(allPCMData))

		// Convert to mono if stereo
		channels := int(format.NumChannels)
		bytesPerSample := int(format.BitsPerSample) / 8
		if channels > 1 && bytesPerSample == 2 {
			// Convert stereo to mono by averaging channels
			// For stereo: samples are interleaved [L R L R ...]
			sampleCount := len(allPCMData) / (bytesPerSample * channels)
			monoData := make([]byte, sampleCount*bytesPerSample)

			for i := 0; i < sampleCount; i++ {
				// Get left and right samples
				leftIdx := i * bytesPerSample * channels
				rightIdx := leftIdx + bytesPerSample

				if rightIdx+bytesPerSample <= len(allPCMData) {
					leftSample := int16(allPCMData[leftIdx]) | int16(allPCMData[leftIdx+1])<<8
					rightSample := int16(allPCMData[rightIdx]) | int16(allPCMData[rightIdx+1])<<8

					// Average the two channels
					avg := int16((int32(leftSample) + int32(rightSample)) / 2)

					monoIdx := i * bytesPerSample
					monoData[monoIdx] = byte(avg)
					monoData[monoIdx+1] = byte(avg >> 8)
				}
			}
			allPCMData = monoData
			channels = 1
			fmt.Printf("[Server] Converted stereo to mono (%d samples)\n", sampleCount)
		}

		// Resample to 8kHz if needed (do this once for better quality)
		targetSampleRate := 8000
		if int(format.SampleRate) != targetSampleRate {
			fmt.Printf("[Server] Resampling from %dHz to %dHz...\n", format.SampleRate, targetSampleRate)
			resampled, err := media2.ResamplePCM(allPCMData, int(format.SampleRate), targetSampleRate)
			if err != nil {
				fmt.Printf("[Server] Error resampling: %v\n", err)
				return
			}
			allPCMData = resampled
			fmt.Printf("[Server] Resampled to %dHz (%d bytes)\n", targetSampleRate, len(allPCMData))
		}

		// Encode entire PCM data to PCMA
		fmt.Printf("[Server] Encoding PCM to PCMA...\n")
		pcmaData, err := encoder.EncodePCMA(allPCMData)
		if err != nil {
			fmt.Printf("[Server] Error encoding to PCMA: %v\n", err)
			return
		}
		fmt.Printf("[Server] Encoded %d bytes PCM to %d bytes PCMA\n", len(allPCMData), len(pcmaData))

		// Send PCMA data in perfect 20ms frames (160 bytes per frame at 8kHz)
		frameDuration := 20 * time.Millisecond
		bytesPerFrame := 160 // 20ms * 8000Hz = 160 samples = 160 bytes (PCMA is 1 byte per sample)

		// Use high-precision timing
		startTime := time.Now()
		frameCount := 0

		for i := 0; i < len(pcmaData); i += bytesPerFrame {
			end := i + bytesPerFrame
			if end > len(pcmaData) {
				// Last frame might be shorter, pad with silence if needed
				end = len(pcmaData)
			}

			// Calculate exact send time to maintain consistent frame rate
			expectedTime := startTime.Add(time.Duration(frameCount) * frameDuration)
			now := time.Now()
			if expectedTime.After(now) {
				time.Sleep(expectedTime.Sub(now))
			}

			sample := media.Sample{
				Data:     pcmaData[i:end],
				Duration: frameDuration,
			}

			err = txTrack.WriteSample(sample)
			if err != nil {
				fmt.Printf("[Server] Error writing sample: %v\n", err)
				break
			}

			frameCount++
			if frameCount%50 == 0 {
				fmt.Printf("[Server] Sent %d frames (PCMA: %d bytes)...\n", frameCount, len(pcmaData[i:end]))
			}
		}

		fmt.Printf("[Server] Finished sending audio (%d frames, %d bytes PCMA)\n", frameCount, len(pcmaData))
	}()

	// Client: Receive and play audio
	go func() {
		// Wait for rxTrack to be available
		var rxTrack *webrtc.TrackRemote
		for i := 0; i < 100; i++ {
			rxTrack = client.GetRxTrack()
			if rxTrack != nil {
				break
			}
			time.Sleep(100 * time.Millisecond)
		}

		if rxTrack == nil {
			fmt.Println("[Client] Error: rxTrack is nil after waiting")
			return
		}

		codec := rxTrack.Codec()
		fmt.Printf("[Client] Received track: %s, %dHz\n", codec.MimeType, codec.ClockRate)

		// PCMA uses 8kHz, so we'll play at 8kHz
		// The decoder will output 16-bit PCM at the target sample rate
		targetSampleRate := 8000 // PCMA standard
		sampleRate := uint32(targetSampleRate)
		channels := uint32(1)

		// Create stream player for 8kHz playback
		streamPlayer, err := devices.NewStreamAudioPlayer(channels, sampleRate, malgo.FormatS16)
		if err != nil {
			fmt.Printf("[Client] Error creating stream player: %v\n", err)
			return
		}
		defer streamPlayer.Close()

		// Start playback
		err = streamPlayer.Play()
		if err != nil {
			fmt.Printf("[Client] Error starting playback: %v\n", err)
			return
		}

		fmt.Printf("[Client] Audio playback started: %dHz, %d channel(s)\n", sampleRate, channels)

		// Create PCMA decoder
		// Input: PCMA at 8kHz, Output: PCM at 8kHz (16-bit)
		// No resampling needed since both are 8kHz
		decodeFunc, err := encoder.CreateDecode(
			media2.CodecConfig{Codec: "pcma", SampleRate: 8000, Channels: 1, BitDepth: 8, FrameDuration: "20ms"},
			media2.CodecConfig{Codec: "pcm", SampleRate: targetSampleRate, Channels: 1, BitDepth: 16, FrameDuration: "20ms"},
		)
		if err != nil {
			fmt.Printf("[Client] Error creating decoder: %v\n", err)
			return
		}

		// Receive and play audio packets
		packetCount := 0
		for {
			packet, _, err := rxTrack.ReadRTP()
			if err != nil {
				fmt.Printf("[Client] Error reading RTP packet: %v\n", err)
				break
			}

			payload := packet.Payload
			if len(payload) == 0 {
				continue
			}

			packetCount++

			// Decode PCMA to PCM
			audioPacket := &media2.AudioPacket{
				Payload: payload,
			}

			decodedPackets, err := decodeFunc(audioPacket)
			if err != nil {
				if packetCount%100 == 0 {
					fmt.Printf("[Client] Error decoding packet %d: %v\n", packetCount, err)
				}
				continue
			}

			// Play each decoded packet
			for _, packet := range decodedPackets {
				af, ok := packet.(*media2.AudioPacket)
				if !ok {
					continue
				}

				// Write PCM data to stream player
				err = streamPlayer.Write(af.Payload)
				if err != nil {
					if packetCount%100 == 0 {
						fmt.Printf("[Client] Error writing to player: %v\n", err)
					}
				}
			}

			if packetCount%100 == 0 {
				fmt.Printf("[Client] Received and played %d RTP packets\n", packetCount)
			}
		}
	}()

	// Run for 30 seconds
	fmt.Println("\nRunning for 30 seconds...")
	time.Sleep(30 * time.Second)

	fmt.Println("\n=== Example completed ===")

	// Cleanup
	client.Close()
	server.Close()
}
