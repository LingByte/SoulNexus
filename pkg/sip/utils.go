package sip

import (
	"fmt"
	"net"
	"strings"
	"time"

	"github.com/emiago/sipgo/sip"
	"github.com/pion/sdp/v3"
	"github.com/sirupsen/logrus"
)

// linearToMulaw converts 16-bit linear PCM to G.711 μ-law
func linearToMulaw(pcm int16) byte {
	// Get sign bit
	sign := byte(0)
	if pcm < 0 {
		sign = 0x80
		pcm = -pcm
	}

	// Limit range
	if pcm > 32635 {
		pcm = 32635
	}

	// Add bias
	pcm += 0x84

	// Find exponent
	exp := 7
	for exp > 0 && (pcm&0x4000) == 0 {
		exp--
		pcm <<= 1
	}

	// Extract mantissa (upper 4 bits)
	mantissa := byte((pcm >> 10) & 0x0F)

	// Combine result
	result := sign | byte(exp<<4) | mantissa

	// μ-law encoding requires inversion
	return ^result
}

// mulawToLinear 将 G.711 μ-law 转换为 16-bit 线性 PCM
func mulawToLinear(mulaw byte) int16 {
	// μ-law 解码需要先取反
	mulaw = ^mulaw

	// 提取符号位
	sign := int16(0)
	if (mulaw & 0x80) != 0 {
		sign = -1
	}

	// 提取指数（3位）
	exp := int16((mulaw >> 4) & 0x07)

	// 提取尾数（4位）
	mantissa := int16(mulaw & 0x0F)

	// 计算线性值
	linear := 33 + (2 * mantissa)
	linear <<= exp
	linear -= 33

	if sign != 0 {
		linear = -linear
	}

	return linear
}

func parseSDPForRTPAddress(sdpBody string) (string, error) {
	// Parse SDP to get client RTP address
	lines := strings.Split(sdpBody, "\r\n")
	if len(lines) == 1 {
		lines = strings.Split(sdpBody, "\n")
	}

	var ip, port string
	var foundMedia bool
	var mediaIP string

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// Find connection information c=IN IP4 x.x.x.x
		if strings.HasPrefix(line, "c=") {
			parts := strings.Fields(line[2:])
			if len(parts) >= 3 && parts[0] == "IN" && parts[1] == "IP4" {
				if foundMedia {
					// Media-level connection information
					mediaIP = parts[2]
				} else {
					// Session-level connection information
					ip = parts[2]
				}
			}
		}

		// Find media information m=audio PORT RTP/AVP ...
		if strings.HasPrefix(line, "m=audio") {
			foundMedia = true
			parts := strings.Fields(line[2:])
			if len(parts) >= 2 {
				port = parts[1]
			}
		}
	}

	// Prefer media-level IP
	if mediaIP != "" {
		ip = mediaIP
	}

	if ip == "" || port == "" {
		return "", fmt.Errorf("failed to parse IP and port from SDP: IP=%s, Port=%s", ip, port)
	}

	return fmt.Sprintf("%s:%s", ip, port), nil
}

func getServerIPFromRequest(req *sip.Request) string {
	// Try to get server IP from Via header's received parameter
	if via := req.Via(); via != nil {
		if received, exists := via.Params.Get("received"); exists && received != "" {
			ip := received
			if net.ParseIP(ip) != nil {
				logrus.WithField("ip", ip).Info("Using IP from Via header received parameter")
				return ip
			}
		}
	}

	// If no received parameter in Via header, use local IP address
	// This is the fallback when the SIP proxy/server doesn't add the received parameter
	localIP := getLocalIP()
	if localIP == "" {
		logrus.Warn("Failed to get local IP, using 127.0.0.1 as fallback")
		localIP = "127.0.0.1"
	} else {
		logrus.WithField("ip", localIP).Info("Using local IP address (Via received parameter not available)")
	}

	return localIP
}

func generateSDP(serverIP string, rtpPort int) string {
	// Use pion/sdp library to generate standard SDP response
	sessionID := time.Now().Unix()

	session := sdp.SessionDescription{
		Version: 0,
		Origin: sdp.Origin{
			Username:       "-",
			SessionID:      uint64(sessionID),
			SessionVersion: uint64(sessionID),
			NetworkType:    "IN",
			AddressType:    "IP4",
			UnicastAddress: serverIP,
		},
		SessionName: "SIP Call",
		ConnectionInformation: &sdp.ConnectionInformation{
			NetworkType: "IN",
			AddressType: "IP4",
			Address:     &sdp.Address{Address: serverIP},
		},
		TimeDescriptions: []sdp.TimeDescription{
			{Timing: sdp.Timing{StartTime: 0, StopTime: 0}},
		},
		MediaDescriptions: []*sdp.MediaDescription{
			{
				MediaName: sdp.MediaName{
					Media:   "audio",
					Port:    sdp.RangedPort{Value: rtpPort},
					Protos:  []string{"RTP", "AVP"},
					Formats: []string{"0"},
				},
				Attributes: []sdp.Attribute{
					{Key: "rtpmap", Value: "0 PCMU/8000/1"},
					{Key: "sendrecv", Value: ""},
				},
			},
		},
	}

	// Serialize to string
	sdpBytes, err := session.Marshal()
	if err != nil {
		logrus.WithError(err).Warn("Failed to generate SDP, using fallback method")
		// If serialization fails, use string concatenation as fallback
		return fmt.Sprintf("v=0\r\n"+
			"o=- %d %d IN IP4 %s\r\n"+
			"s=SIP Call\r\n"+
			"c=IN IP4 %s\r\n"+
			"t=0 0\r\n"+
			"m=audio %d RTP/AVP 0\r\n"+
			"a=rtpmap:0 PCMU/8000/1\r\n"+
			"a=sendrecv\r\n",
			sessionID, sessionID, serverIP, serverIP, rtpPort)
	}

	return string(sdpBytes)
}

func getLocalIP() string {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return ""
	}
	for _, addr := range addrs {
		if ipNet, ok := addr.(*net.IPNet); ok && !ipNet.IP.IsLoopback() {
			if ipNet.IP.To4() != nil {
				return ipNet.IP.String()
			}
		}
	}
	return ""
}
