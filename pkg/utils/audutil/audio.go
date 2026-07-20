package audutil

import (
	"encoding/binary"
	"fmt"
	"io"
	"math"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// MinRecordingWAVBytes is the smallest WAV object we persist (shorter blobs are decode noise).
const MinRecordingWAVBytes = 4096

// RMSPCM16LE returns the RMS level of signed 16-bit little-endian PCM samples.
func RMSPCM16LE(pcm []byte) float64 {
	if len(pcm) < 2 {
		return 0
	}
	n := len(pcm) / 2
	var sum float64
	for i := 0; i+1 < len(pcm); i += 2 {
		v := int16(uint16(pcm[i]) | (uint16(pcm[i+1]) << 8))
		f := float64(v)
		sum += f * f
	}
	return math.Sqrt(sum / float64(n))
}

// WAVDurationSec parses a minimal PCM WAV header and returns rounded seconds.
func WAVDurationSec(wav []byte) int {
	if len(wav) < 44 {
		return 0
	}
	if string(wav[0:4]) != "RIFF" || string(wav[8:12]) != "WAVE" {
		return 0
	}
	var byteRate uint32
	var dataSize uint32
	i := 12
	for i+8 <= len(wav) {
		chunkID := string(wav[i : i+4])
		chunkSize := int(binary.LittleEndian.Uint32(wav[i+4 : i+8]))
		if chunkSize < 0 {
			return 0
		}
		payloadStart := i + 8
		payloadEnd := payloadStart + chunkSize
		if payloadEnd > len(wav) {
			break
		}
		switch chunkID {
		case "fmt ":
			if chunkSize >= 16 {
				byteRate = binary.LittleEndian.Uint32(wav[payloadStart+8 : payloadStart+12])
			}
		case "data":
			dataSize = uint32(chunkSize)
		}
		advance := 8 + chunkSize
		if chunkSize%2 == 1 {
			advance++
		}
		i += advance
	}
	if byteRate == 0 || dataSize == 0 {
		return 0
	}
	sec := int((float64(dataSize) / float64(byteRate)) + 0.5)
	if sec < 0 {
		return 0
	}
	return sec
}

// RecordingObjectKey returns the object-storage key for a call recording artifact.
// Layout: recordings/{tenantID}/{YYYY-MM-DD}/{callID}.{ext}
func RecordingObjectKey(tenantID uint, callID string, at time.Time, ext string) string {
	callID = strings.TrimSpace(callID)
	ext = strings.TrimPrefix(strings.TrimSpace(ext), ".")
	if callID == "" || ext == "" {
		return ""
	}
	if at.IsZero() {
		at = time.Now()
	}
	date := at.UTC().Format("2006-01-02")
	tenant := strconv.FormatUint(uint64(tenantID), 10)
	name := sanitizeRecordingFilePart(callID) + "." + ext
	return filepath.ToSlash(filepath.Join("recordings", tenant, date, name))
}

// RecordingPartObjectKey is a rolling-chunk key beside the final WAV:
// recordings/{tenant}/{date}/{callID}-part-{seq}-{ts}.wav
func RecordingPartObjectKey(tenantID uint, callID string, at time.Time, seq int, ts int64) string {
	base := RecordingObjectKey(tenantID, callID, at, "wav")
	if base == "" {
		return ""
	}
	i := strings.LastIndex(base, "/")
	if i < 0 {
		return ""
	}
	stem := strings.TrimSuffix(base[i+1:], ".wav")
	return base[:i+1] + stem + fmt.Sprintf("-part-%d-%d.wav", seq, ts)
}

func sanitizeRecordingFilePart(s string) string {
	const repl = '_'
	b := []byte(s)
	for i, c := range b {
		switch {
		case c >= 'a' && c <= 'z', c >= 'A' && c <= 'Z', c >= '0' && c <= '9', c == '-', c == '.':
		default:
			b[i] = repl
		}
	}
	out := strings.Trim(string(b), "._")
	if out == "" {
		return "unknown"
	}
	return out
}

// FetchRecordingWAV loads recording bytes from recording_url (http(s), storage key, or local uploads path).
func FetchRecordingWAV(recordingURL string, httpClient *http.Client) ([]byte, error) {
	raw := strings.TrimSpace(recordingURL)
	if raw == "" {
		return nil, fmt.Errorf("empty recording url")
	}
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 60 * time.Second}
	}
	lower := strings.ToLower(raw)
	if strings.HasPrefix(lower, "http://") || strings.HasPrefix(lower, "https://") {
		return fetchHTTP(httpClient, raw)
	}
	return nil, fmt.Errorf("unsupported recording url: %s", raw)
}

func fetchHTTP(client *http.Client, u string) ([]byte, error) {
	req, err := http.NewRequest(http.MethodGet, u, nil)
	if err != nil {
		return nil, err
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("http %s: %s", resp.Status, u)
	}
	const maxWAV = 128 << 20 // 128 MiB safety cap
	return io.ReadAll(io.LimitReader(resp.Body, maxWAV))
}
