package rtcmedia

import (
	"os"
	"strings"
)

// VerboseSDP enables SDP preview and ICE debug logs (set WEBRTC_DEBUG=1).
func VerboseSDP() bool {
	return strings.TrimSpace(os.Getenv("WEBRTC_DEBUG")) == "1" ||
		strings.EqualFold(strings.TrimSpace(os.Getenv("WEBRTC_DEBUG")), "true")
}
