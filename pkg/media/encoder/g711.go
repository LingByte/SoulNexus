package encoder

const (
	biasValue        = 0x84
	segmentShift     = 4
	segmentMask      = 0x70
	quantizationMask = 0x0F
	signBitMask      = 0x80
)

var segmentBoundaries = []int{0xFF, 0x1FF, 0x3FF, 0x7FF, 0xFFF, 0x1FFF, 0x3FFF, 0x7FFF}

// findSegmentIndex uses optimized linear search (faster for small arrays due to cache locality)
func findSegmentIndex(value int, boundaries []int, count int) int {
	// Linear search is faster for 8 elements due to better cache locality
	// and no branch prediction misses
	for idx := 0; idx < count; idx++ {
		if value <= boundaries[idx] {
			return idx
		}
	}
	return count
}

// encodeToALaw uses lookup table approach for small values, computation for large values
func linear2alaw(pcmValue int) byte {
	// Handle sign and special cases first
	var signMask int
	var absValue int

	if pcmValue >= 0 {
		signMask = 0xD5
		absValue = pcmValue
	} else if pcmValue < -8 {
		signMask = 0x55
		absValue = -pcmValue - 8
	} else {
		// Special case: values in [-7, -1] range
		return 0xD5
	}

	// Use binary search for segment finding
	segment := findSegmentIndex(absValue, segmentBoundaries, 8)

	if segment >= 8 {
		return byte(0x7F ^ signMask)
	}

	// Calculate quantization bits using different bit manipulation
	var quantBits int
	if segment < 2 {
		quantBits = (absValue >> 4) & quantizationMask
	} else {
		shiftAmount := segment + 3
		quantBits = (absValue >> shiftAmount) & quantizationMask
	}

	// Combine segment and quantization
	encodedValue := (segment << segmentShift) | quantBits
	return byte(encodedValue ^ signMask)
}

// decodeFromALaw uses switch for better compiler optimization
func alaw2linear(alawByte byte) int16 {
	alawByte ^= 0x55
	temp := int16(alawByte&quantizationMask) << 4
	segment := int16((alawByte & segmentMask) >> segmentShift)

	// Switch statement allows compiler to optimize as jump table
	switch segment {
	case 0:
		temp += 8
	case 1:
		temp += 0x108
	default:
		temp += 0x108
		temp <<= (segment - 1)
	}

	if alawByte&signBitMask != 0 {
		return temp
	}
	return -temp
}

// encodeToULaw converts linear PCM to μ-law
func linear2ulaw(pcmValue int) byte {
	var signMask int
	var segment int
	var encodedValue int

	if pcmValue < 0 {
		pcmValue = biasValue - pcmValue
		signMask = 0x7F
	} else {
		pcmValue += biasValue
		signMask = 0xFF
	}

	segment = findSegmentIndex(pcmValue, segmentBoundaries, 8)

	if segment >= 8 {
		return byte(0x7F ^ signMask)
	} else {
		encodedValue = (segment << 4) | ((pcmValue >> (segment + 3)) & 0xF)
		return byte(encodedValue ^ signMask)
	}
}

// decodeFromULaw converts μ-law to linear PCM
func ulaw2linear(ulawByte byte) int {
	var temp int
	ulawByte = ^ulawByte

	temp = int((ulawByte&quantizationMask)<<3) + biasValue
	temp <<= (uint8(ulawByte) & segmentMask) >> segmentShift

	if ulawByte&signBitMask != 0 {
		return biasValue - temp
	}
	return temp - biasValue
}

// convertALawToPCM converts A-law encoded data to PCM
func pcma2pcm(alawData []byte) ([]byte, error) {
	pcmData := make([]byte, len(alawData)<<1)
	outputIdx := 0
	for _, alawByte := range alawData {
		pcmSample := alaw2linear(alawByte)
		pcmData[outputIdx] = byte(pcmSample)
		pcmData[outputIdx+1] = byte(pcmSample >> 8)
		outputIdx += 2
	}
	return pcmData, nil
}

// convertPCMToALaw converts PCM data to A-law encoding
// EncodePCMA encodes PCM data to PCMA format
func EncodePCMA(pcmData []byte) ([]byte, error) {
	return Pcm2pcma(pcmData)
}

func Pcm2pcma(pcmData []byte) ([]byte, error) {
	alawData := make([]byte, len(pcmData)>>1)
	outputIdx := 0
	for inputIdx := 0; inputIdx < len(pcmData); inputIdx += 2 {
		pcmSample := int16(pcmData[inputIdx+1])<<8 | int16(pcmData[inputIdx])
		alawData[outputIdx] = linear2alaw(int(pcmSample))
		outputIdx++
	}
	return alawData, nil
}

// convertULawToPCM converts μ-law encoded data to PCM
func pcmu2pcm(ulawData []byte) ([]byte, error) {
	pcmData := make([]byte, len(ulawData)<<1)
	outputIdx := 0
	for _, ulawByte := range ulawData {
		pcmSample := ulaw2linear(ulawByte)
		pcmData[outputIdx] = byte(pcmSample)
		pcmData[outputIdx+1] = byte(pcmSample >> 8)
		outputIdx += 2
	}
	return pcmData, nil
}

// convertPCMToULaw converts PCM data to μ-law encoding
func pcm2pcmu(pcmData []byte) ([]byte, error) {
	ulawData := make([]byte, len(pcmData)>>1)
	outputIdx := 0
	for inputIdx := 0; inputIdx < len(pcmData); inputIdx += 2 {
		pcmSample := int16(pcmData[inputIdx+1])<<8 | int16(pcmData[inputIdx])
		ulawData[outputIdx] = linear2ulaw(int(pcmSample))
		outputIdx++
	}
	return ulawData, nil
}
