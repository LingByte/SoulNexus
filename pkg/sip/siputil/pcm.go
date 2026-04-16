package siputil

import "math"

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
