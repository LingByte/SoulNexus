package dialog

import (
	"strings"
)

// AppendTagAfterNameAddr inserts ;tag=tag immediately after the closing '>' of a name-addr,
// or appends ;tag=tag when no angle brackets are present. If a tag already exists, header is unchanged.
func AppendTagAfterNameAddr(header, tag string) string {
	header = strings.TrimSpace(header)
	tag = strings.TrimSpace(tag)
	if header == "" || tag == "" {
		return header
	}
	if TagFromHeader(header) != "" {
		return header
	}
	if idx := strings.LastIndex(header, ">"); idx >= 0 {
		return header[:idx+1] + ";tag=" + tag + header[idx+1:]
	}
	return header + ";tag=" + tag
}

// TagFromHeader extracts the SIP tag parameter from a From or To header value.
func TagFromHeader(header string) string {
	header = strings.TrimSpace(header)
	if header == "" {
		return ""
	}
	lower := strings.ToLower(header)
	idx := strings.Index(lower, ";tag=")
	if idx < 0 {
		return ""
	}
	v := header[idx+len(";tag="):]
	if cut := strings.IndexAny(v, ";>"); cut >= 0 {
		v = v[:cut]
	}
	return strings.TrimSpace(strings.Trim(strings.TrimSpace(v), "\""))
}
