package protocol

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// SDPCodec represents one codec mapping in SDP (via a=rtpmap).
type SDPCodec struct {
	PayloadType uint8
	Name        string
	ClockRate   int
	Channels    int
}

// SDPInfo holds minimal audio media information from SDP.
type SDPInfo struct {
	IP     string
	Port   int
	Proto  string
	Codecs []SDPCodec
}

var (
	reIP      = regexp.MustCompile(`c=IN IP4 ([0-9.]+)`)
	reMAudio  = regexp.MustCompile(`m=audio\s+(\d+)\s+([A-Za-z0-9/]+)\s+(.+)`)
	reRtpMap  = regexp.MustCompile(`^a=rtpmap:(\d+)\s+([^/]+)/(\d+)`)
	reRtpMapV = regexp.MustCompile(`^a=rtpmap:(\d+)\s+([^/]+)/(\d+)(?:/(\d+))?$`)
)

func normalizeCodecName(name string) string {
	n := strings.TrimSpace(name)
	n = strings.ToLower(n)
	switch n {
	case "pcmu":
		return "pcmu"
	case "pcma":
		return "pcma"
	case "g722":
		return "g722"
	case "opus":
		return "opus"
	case "pcm":
		return "pcm"
	case "telephone-event":
		return "telephone-event"
	default:
		return n
	}
}

// staticPayloadCodec returns the RFC 3551 static audio mapping when no a=rtpmap is present.
func staticPayloadCodec(pt uint8) (SDPCodec, bool) {
	switch pt {
	case 0:
		return SDPCodec{PayloadType: 0, Name: "pcmu", ClockRate: 8000, Channels: 1}, true
	case 8:
		return SDPCodec{PayloadType: 8, Name: "pcma", ClockRate: 8000, Channels: 1}, true
	case 9:
		return SDPCodec{PayloadType: 9, Name: "g722", ClockRate: 8000, Channels: 1}, true
	default:
		return SDPCodec{}, false
	}
}

// ParseSDP extracts IP/port and codec mappings from SDP body.
func ParseSDP(body string) (*SDPInfo, error) {
	body = strings.TrimSpace(body)
	if body == "" {
		return nil, fmt.Errorf("sip: empty sdp body")
	}

	info := &SDPInfo{}

	// Connection IP.
	if m := reIP.FindStringSubmatch(body); len(m) >= 2 {
		info.IP = m[1]
	}

	// m=audio line: grab port and payload types list.
	var payloadTypes []uint8
	var mediaProto string
	lines := strings.Split(strings.ReplaceAll(body, "\r\n", "\n"), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		if strings.HasPrefix(line, "m=audio") {
			m := reMAudio.FindStringSubmatch(line)
			if len(m) >= 4 {
				port, err := strconv.Atoi(strings.TrimSpace(m[1]))
				if err != nil {
					return nil, fmt.Errorf("sip: invalid m=audio port: %w", err)
				}
				info.Port = port
				mediaProto = strings.ToUpper(strings.TrimSpace(m[2]))
				info.Proto = mediaProto

				pts := strings.Fields(strings.TrimSpace(m[3]))
				for _, ptStr := range pts {
					ptInt, err := strconv.Atoi(ptStr)
					if err != nil {
						continue
					}
					if ptInt < 0 || ptInt > 255 {
						continue
					}
					payloadTypes = append(payloadTypes, uint8(ptInt))
				}
			}
		}

		// a=rtpmap lines.
		if strings.HasPrefix(line, "a=rtpmap:") {
			m := reRtpMapV.FindStringSubmatch(line)
			if len(m) >= 4 {
				ptInt, err := strconv.Atoi(m[1])
				if err != nil {
					continue
				}
				name := m[2]
				clock, err := strconv.Atoi(m[3])
				if err != nil {
					continue
				}
				if ptInt < 0 || ptInt > 255 {
					continue
				}
				channels := 1
				if len(m) >= 5 && strings.TrimSpace(m[4]) != "" {
					if ch, err := strconv.Atoi(strings.TrimSpace(m[4])); err == nil && ch > 0 {
						channels = ch
					}
				}

				info.Codecs = append(info.Codecs, SDPCodec{
					PayloadType: uint8(ptInt),
					Name:        normalizeCodecName(name),
					ClockRate:   clock,
					Channels:    channels,
				})
			} else if m2 := reRtpMap.FindStringSubmatch(line); len(m2) >= 4 {
				ptInt, err := strconv.Atoi(m2[1])
				if err != nil {
					continue
				}
				name := m2[2]
				clock, err := strconv.Atoi(m2[3])
				if err != nil {
					continue
				}
				if ptInt < 0 || ptInt > 255 {
					continue
				}
				info.Codecs = append(info.Codecs, SDPCodec{
					PayloadType: uint8(ptInt),
					Name:        normalizeCodecName(name),
					ClockRate:   clock,
					Channels:    1,
				})
			}
		}
	}

	// Reject secure RTP offers for now (no SRTP support in this module yet).
	// Common values: RTP/SAVP, RTP/SAVPF.
	if mediaProto != "" && strings.Contains(mediaProto, "SAVP") {
		return nil, fmt.Errorf("sip: unsupported media proto: %s", mediaProto)
	}

	// Peers often list static PTs (0/8/9) on m=audio but omit a=rtpmap for them while still
	// sending a=rtpmap only for dynamic types (e.g. 101 telephone-event). Without this, we would
	// keep only the dynamic codec and fail CallSession negotiation (see staticPayloadCodec).
	if len(payloadTypes) > 0 {
		seen := make(map[uint8]struct{}, len(info.Codecs)+len(payloadTypes))
		for _, c := range info.Codecs {
			seen[c.PayloadType] = struct{}{}
		}
		for _, pt := range payloadTypes {
			if _, ok := seen[pt]; ok {
				continue
			}
			if sc, ok := staticPayloadCodec(pt); ok {
				info.Codecs = append(info.Codecs, sc)
				seen[pt] = struct{}{}
			}
		}
	}

	// Many SIP clients omit every a=rtpmap; infer static codecs from m= alone.
	if len(info.Codecs) == 0 && len(payloadTypes) > 0 {
		for _, pt := range payloadTypes {
			if sc, ok := staticPayloadCodec(pt); ok {
				info.Codecs = append(info.Codecs, sc)
			}
		}
	}

	// If we failed to collect codecs, still return error to avoid silent misbehavior.
	if len(info.Codecs) == 0 {
		return nil, fmt.Errorf("sip: no codec found in SDP")
	}

	// If payload types were extracted, filter codecs to those payload types and order by m= line.
	if len(payloadTypes) > 0 {
		want := make(map[uint8]struct{}, len(payloadTypes))
		for _, pt := range payloadTypes {
			want[pt] = struct{}{}
		}
		filtered := make([]SDPCodec, 0, len(info.Codecs))
		for _, c := range info.Codecs {
			if _, ok := want[c.PayloadType]; ok {
				filtered = append(filtered, c)
			}
		}
		if len(filtered) > 0 {
			byPT := make(map[uint8]SDPCodec, len(filtered))
			for _, c := range filtered {
				byPT[c.PayloadType] = c
			}
			ordered := make([]SDPCodec, 0, len(filtered))
			for _, pt := range payloadTypes {
				if c, ok := byPT[pt]; ok {
					ordered = append(ordered, c)
				}
			}
			info.Codecs = ordered
		}
	}

	return info, nil
}

// GenerateSDP generates a minimal SDP body for audio RTP.
func GenerateSDP(localIP string, localPort int, codecs []SDPCodec) string {
	return GenerateSDPWithProto(localIP, localPort, "RTP/AVP", codecs)
}

// GenerateSDPWithProto generates a minimal SDP body for audio RTP with a specific m=audio proto
// (e.g. RTP/AVP, RTP/AVPF). SRTP protos (SAVP/SAVPF) are not supported by this package.
func GenerateSDPWithProto(localIP string, localPort int, proto string, codecs []SDPCodec) string {
	if localPort <= 0 {
		localPort = 49172
	}
	if localIP == "" {
		localIP = "127.0.0.1"
	}
	proto = strings.ToUpper(strings.TrimSpace(proto))
	if proto == "" {
		proto = "RTP/AVP"
	}

	pts := make([]string, 0, len(codecs))
	for _, c := range codecs {
		pts = append(pts, strconv.Itoa(int(c.PayloadType)))
	}

	var b strings.Builder
	b.WriteString("v=0\r\n")
	sess := time.Now().Unix()
	b.WriteString(fmt.Sprintf("o=- %d %d IN IP4 %s\r\n", sess, sess, localIP))
	b.WriteString("s=SoulNexus SIP\r\n")
	b.WriteString(fmt.Sprintf("c=IN IP4 %s\r\n", localIP))
	b.WriteString("t=0 0\r\n")

	b.WriteString(fmt.Sprintf("m=audio %d %s %s\r\n", localPort, proto, strings.Join(pts, " ")))
	// Default RTCP is RTP+1 if not specified, but many clients behave better
	// when rtcp is explicit.
	b.WriteString(fmt.Sprintf("a=rtcp:%d IN IP4 %s\r\n", localPort+1, localIP))
	b.WriteString("a=sendrecv\r\n")
	b.WriteString("a=ptime:20\r\n")
	for _, c := range codecs {
		if c.Channels > 1 {
			b.WriteString(fmt.Sprintf("a=rtpmap:%d %s/%d/%d\r\n", c.PayloadType, strings.ToUpper(c.Name), c.ClockRate, c.Channels))
		} else {
			b.WriteString(fmt.Sprintf("a=rtpmap:%d %s/%d\r\n", c.PayloadType, strings.ToUpper(c.Name), c.ClockRate))
		}
		// Common OPUS fmtp improves interoperability with softphones.
		if strings.EqualFold(c.Name, "opus") {
			b.WriteString(fmt.Sprintf("a=fmtp:%d minptime=10;useinbandfec=1\r\n", c.PayloadType))
		}
		if strings.EqualFold(c.Name, "telephone-event") {
			b.WriteString(fmt.Sprintf("a=fmtp:%d 0-15\r\n", c.PayloadType))
		}
	}
	// Ensure the SDP body ends with CRLF.
	return b.String()
}

// PickTelephoneEventFromOffer returns the best telephone-event codec from the remote offer.
// Prefer payload type whose clock rate matches the negotiated audio codec (e.g. 48000 with Opus).
func PickTelephoneEventFromOffer(offer []SDPCodec, matchClockRate int) (SDPCodec, bool) {
	var fallback SDPCodec
	var hasFallback bool
	for _, c := range offer {
		if !strings.EqualFold(strings.TrimSpace(c.Name), "telephone-event") {
			continue
		}
		if matchClockRate > 0 && c.ClockRate == matchClockRate {
			return c, true
		}
		if !hasFallback {
			fallback = c
			hasFallback = true
		}
	}
	if hasFallback {
		return fallback, true
	}
	return SDPCodec{}, false
}

