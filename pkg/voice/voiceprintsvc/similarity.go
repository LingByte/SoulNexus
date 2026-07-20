package voiceprintsvc

import (
	"encoding/binary"
	"fmt"
	"math"

	"github.com/LingByte/lingllm/voiceprint"
)

func compareWAVScore(a, b []byte) (float64, error) {
	pcmA, infoA, err := voiceprint.ExtractWAVPCM(a)
	if err != nil {
		return 0, err
	}
	pcmB, infoB, err := voiceprint.ExtractWAVPCM(b)
	if err != nil {
		return 0, err
	}
	if infoA.BitsPerSample != 16 || infoB.BitsPerSample != 16 {
		return 0, fmt.Errorf("only 16-bit PCM WAV is supported")
	}
	samplesA := pcmToInt16(pcmA)
	samplesB := pcmToInt16(pcmB)
	if len(samplesA) == 0 || len(samplesB) == 0 {
		return 0, fmt.Errorf("empty audio samples")
	}
	n := min(len(samplesA), len(samplesB))
	return pearson(samplesA[:n], samplesB[:n]), nil
}

func pcmToInt16(pcm []byte) []int16 {
	n := len(pcm) / 2
	out := make([]int16, n)
	for i := 0; i < n; i++ {
		out[i] = int16(binary.LittleEndian.Uint16(pcm[i*2 : i*2+2]))
	}
	return out
}

func pearson(a, b []int16) float64 {
	n := len(a)
	if n == 0 || len(b) != n {
		return 0
	}
	var sumA, sumB, sumA2, sumB2, sumAB float64
	for i := 0; i < n; i++ {
		x := float64(a[i])
		y := float64(b[i])
		sumA += x
		sumB += y
		sumA2 += x * x
		sumB2 += y * y
		sumAB += x * y
	}
	denom := math.Sqrt((sumA2-sumA*sumA/float64(n)) * (sumB2-sumB*sumB/float64(n)))
	if denom == 0 {
		return 0
	}
	r := (sumAB - sumA*sumB/float64(n)) / denom
	if r < 0 {
		return 0
	}
	if r > 1 {
		return 1
	}
	return r
}
