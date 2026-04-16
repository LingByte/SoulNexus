package protocol

import (
	"fmt"
	"sort"
	"strings"
)

// Message represents a parsed SIP message.
type Message struct {
	Method       string
	RequestURI   string
	StatusCode   int
	StatusText   string
	Version      string
	Headers      map[string]string   // Headers stores the first value for each header name (case-insensitive).
	HeadersMulti map[string][]string // HeadersMulti stores all values for each header name (case-insensitive).
	Body         string
	IsRequest    bool
}

func canonicalHeaderKey(name string) string {
	return strings.ToLower(strings.TrimSpace(name))
}

// Parse parses a raw SIP message string into a Message.
//
// It supports both request and response messages.
// - Header lines use ":" separator.
// - Message body starts after the first empty line.
func Parse(raw string) (*Message, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, fmt.Errorf("sip: empty message")
	}

	// SIP usually uses CRLF, but tolerate LF.
	raw = strings.ReplaceAll(raw, "\r\n", "\n")
	lines := strings.Split(raw, "\n")
	if len(lines) == 0 {
		return nil, fmt.Errorf("sip: empty message lines")
	}

	firstLine := strings.TrimSpace(lines[0])
	if firstLine == "" {
		return nil, fmt.Errorf("sip: empty first line")
	}

	msg := &Message{
		Headers:      make(map[string]string),
		HeadersMulti: make(map[string][]string),
	}

	// Request line or status line.
	if strings.HasPrefix(firstLine, "SIP/") {
		// Response
		msg.IsRequest = false
		parts := strings.SplitN(firstLine, " ", 3)
		if len(parts) < 2 {
			return nil, fmt.Errorf("sip: invalid response line: %s", firstLine)
		}
		msg.Version = strings.TrimSpace(parts[0])
		fmt.Sscanf(parts[1], "%d", &msg.StatusCode)
		if len(parts) >= 3 {
			msg.StatusText = strings.TrimSpace(parts[2])
		}
	} else {
		// Request
		msg.IsRequest = true
		parts := strings.SplitN(firstLine, " ", 3)
		if len(parts) < 2 {
			return nil, fmt.Errorf("sip: invalid request line: %s", firstLine)
		}
		msg.Method = strings.ToUpper(strings.TrimSpace(parts[0]))
		msg.RequestURI = strings.TrimSpace(parts[1])
		if len(parts) >= 3 {
			msg.Version = strings.TrimSpace(parts[2])
		} else {
			msg.Version = "SIP/2.0"
		}
	}

	// Parse headers until empty line.
	bodyStart := -1
	for i := 1; i < len(lines); i++ {
		line := strings.TrimSpace(lines[i])
		if line == "" {
			bodyStart = i + 1
			break
		}

		parts := strings.SplitN(line, ":", 2)
		if len(parts) == 2 {
			key := strings.TrimSpace(parts[0])
			val := strings.TrimSpace(parts[1])
			if key != "" {
				ck := canonicalHeaderKey(key)
				// keep first value in Headers, keep all in HeadersMulti
				if _, exists := msg.Headers[ck]; !exists {
					msg.Headers[ck] = val
				}
				msg.HeadersMulti[ck] = append(msg.HeadersMulti[ck], val)
			}
		}
	}

	// Body.
	if bodyStart > 0 && bodyStart < len(lines) {
		// Re-join with "\n"; body is typically treated as opaque SDP text.
		msg.Body = strings.Join(lines[bodyStart:], "\n")
	}

	return msg, nil
}

// String converts a Message back to a SIP raw message.
func (m *Message) String() string {
	if m == nil {
		return ""
	}

	var b strings.Builder
	if m.IsRequest {
		b.WriteString(fmt.Sprintf("%s %s %s\r\n", m.Method, m.RequestURI, m.Version))
	} else {
		b.WriteString(fmt.Sprintf("%s %d %s\r\n", m.Version, m.StatusCode, m.StatusText))
	}

	// SIP header names are case-insensitive, but some clients are picky about
	// common casing and ordering. Emit a stable, conventional order first.
	preferred := []string{
		"via",
		"max-forwards",
		"from",
		"to",
		"call-id",
		"cseq",
		"contact",
		"allow",
		"supported",
		"user-agent",
		"content-type",
		"content-length",
	}

	emitted := make(map[string]struct{}, 32)
	for _, k := range preferred {
		vals := m.HeadersMulti[k]
		if len(vals) == 0 {
			if v, ok := m.Headers[k]; ok && v != "" {
				vals = []string{v}
			}
		}
		if len(vals) == 0 {
			continue
		}
		for _, v := range vals {
			b.WriteString(fmt.Sprintf("%s: %s\r\n", prettyHeaderName(k), v))
		}
		emitted[k] = struct{}{}
	}

	// Emit the rest deterministically (sorted by canonical key).
	restKeys := make([]string, 0, len(m.HeadersMulti))
	for k := range m.HeadersMulti {
		if _, ok := emitted[k]; ok {
			continue
		}
		restKeys = append(restKeys, k)
	}
	// In case HeadersMulti is empty but Headers has values, include them too.
	for k := range m.Headers {
		if _, ok := emitted[k]; ok {
			continue
		}
		found := false
		for _, rk := range restKeys {
			if rk == k {
				found = true
				break
			}
		}
		if !found {
			restKeys = append(restKeys, k)
		}
	}
	sort.Strings(restKeys)
	for _, k := range restKeys {
		vals := m.HeadersMulti[k]
		if len(vals) == 0 {
			if v, ok := m.Headers[k]; ok && v != "" {
				vals = []string{v}
			}
		}
		for _, v := range vals {
			b.WriteString(fmt.Sprintf("%s: %s\r\n", prettyHeaderName(k), v))
		}
	}

	b.WriteString("\r\n")
	if m.Body != "" {
		// Body is opaque; normalize to CRLF without producing "\r\r\n".
		body := normalizeCRLF(m.Body)
		b.WriteString(body)
	}

	return b.String()
}

// normalizeCRLF converts any mixture of "\r\n" and "\n" to "\r\n".
func normalizeCRLF(s string) string {
	if s == "" {
		return ""
	}
	// First collapse CRLF to LF, then expand LF to CRLF.
	s = strings.ReplaceAll(s, "\r\n", "\n")
	s = strings.ReplaceAll(s, "\r", "\n")
	return strings.ReplaceAll(s, "\n", "\r\n")
}

// BodyBytesLen returns the length in bytes of the normalized CRLF body.
func BodyBytesLen(body string) int {
	return len([]byte(normalizeCRLF(body)))
}

func prettyHeaderName(canonical string) string {
	switch strings.ToLower(strings.TrimSpace(canonical)) {
	case "via":
		return "Via"
	case "max-forwards":
		return "Max-Forwards"
	case "from":
		return "From"
	case "to":
		return "To"
	case "call-id":
		return "Call-ID"
	case "cseq":
		return "CSeq"
	case "contact":
		return "Contact"
	case "allow":
		return "Allow"
	case "supported":
		return "Supported"
	case "user-agent":
		return "User-Agent"
	case "content-type":
		return "Content-Type"
	case "content-length":
		return "Content-Length"
	default:
		// Best-effort Title-Case for unknown headers.
		parts := strings.Split(canonical, "-")
		for i := range parts {
			if parts[i] == "" {
				continue
			}
			parts[i] = strings.ToUpper(parts[i][:1]) + parts[i][1:]
		}
		return strings.Join(parts, "-")
	}
}

func (m *Message) GetHeader(name string) string {
	if m == nil {
		return ""
	}
	return m.Headers[canonicalHeaderKey(name)]
}

// GetHeaders returns all values for a header name (case-insensitive).
func (m *Message) GetHeaders(name string) []string {
	if m == nil {
		return nil
	}
	return m.HeadersMulti[canonicalHeaderKey(name)]
}

func (m *Message) SetHeader(name, value string) {
	if m == nil {
		return
	}
	if m.Headers == nil {
		m.Headers = make(map[string]string)
	}
	if m.HeadersMulti == nil {
		m.HeadersMulti = make(map[string][]string)
	}
	ck := canonicalHeaderKey(name)
	m.Headers[ck] = value
	m.HeadersMulti[ck] = []string{value}
}

// AddHeader appends a header value (multi-value headers like Via, Record-Route).
func (m *Message) AddHeader(name, value string) {
	if m == nil {
		return
	}
	if m.Headers == nil {
		m.Headers = make(map[string]string)
	}
	if m.HeadersMulti == nil {
		m.HeadersMulti = make(map[string][]string)
	}
	ck := canonicalHeaderKey(name)
	if _, exists := m.Headers[ck]; !exists {
		m.Headers[ck] = value
	}
	m.HeadersMulti[ck] = append(m.HeadersMulti[ck], value)
}
