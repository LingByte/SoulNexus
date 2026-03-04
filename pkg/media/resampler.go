package media

import (
	"io"
)

// SampleRateConverter defines interface for converting sample rates
type SampleRateConverter interface {
	io.WriteCloser
	Samples() []byte
}

// ConverterFactory creates a sample rate converter
type ConverterFactory func(inputRate, outputRate int) SampleRateConverter

var defaultConverterFactory ConverterFactory = NewInterpolatingConverter

// DefaultResampler creates a default sample rate converter
func DefaultResampler(inputRate, outputRate int) SampleRateConverter {
	return defaultConverterFactory(inputRate, outputRate)
}

// SetDefaultResampler sets the default converter factory
func SetDefaultResampler(factory ConverterFactory) {
	defaultConverterFactory = factory
}

// ResamplePCM converts audio data from one sample rate to another
func ResamplePCM(data []byte, inputRate, outputRate int) ([]byte, error) {
	if inputRate == outputRate {
		return data, nil
	}
	converter := DefaultResampler(inputRate, outputRate)
	_, err := converter.Write(data)
	if err != nil {
		return nil, err
	}
	err = converter.Close()
	if err != nil {
		return nil, err
	}
	return converter.Samples(), nil
}

// InterpolatingConverter performs optimized interpolation for sample rate conversion
type InterpolatingConverter struct {
	sourceRate int
	targetRate int
	buffer     []byte
	useCubic   bool // Use cubic interpolation for better quality (slower)
}

// NewInterpolatingConverter creates a new interpolating converter with linear interpolation (fast)
func NewInterpolatingConverter(sourceRate, targetRate int) SampleRateConverter {
	return &InterpolatingConverter{
		sourceRate: sourceRate,
		targetRate: targetRate,
		useCubic:   false, // Default to linear for performance
	}
}

// NewCubicInterpolatingConverter creates a converter with cubic interpolation (better quality)
func NewCubicInterpolatingConverter(sourceRate, targetRate int) SampleRateConverter {
	return &InterpolatingConverter{
		sourceRate: sourceRate,
		targetRate: targetRate,
		useCubic:   true,
	}
}

// ConvertSamples performs interpolation (linear by default, cubic if enabled)
func (ic *InterpolatingConverter) ConvertSamples(samples []byte) []byte {
	if ic.sourceRate == ic.targetRate {
		return samples
	}
	if len(samples)&1 != 0 {
		return nil
	}

	rateRatio := float64(ic.targetRate) / float64(ic.sourceRate)
	sampleCount := len(samples) >> 1
	outputSampleCount := int(float64(sampleCount) * rateRatio)
	outputLength := outputSampleCount << 1
	output := make([]byte, outputLength)

	// Helper to extract sample value
	getSample := func(idx int) int16 {
		if idx*2+1 < len(samples) {
			return int16(samples[idx*2]) | (int16(samples[idx*2+1]) << 8)
		}
		return 0
	}

	for outputIdx := 0; outputIdx < outputLength; outputIdx += 2 {
		targetSampleIdx := outputIdx >> 1
		sourcePos := float64(targetSampleIdx) / rateRatio
		sourceIdx := int(sourcePos)
		fractional := sourcePos - float64(sourceIdx)

		var interpolated int16

		if ic.useCubic && sourceIdx+2 < sampleCount && sourceIdx > 0 {
			// Cubic Hermite interpolation (better quality, slower)
			p0 := float64(getSample(sourceIdx - 1))
			p1 := float64(getSample(sourceIdx))
			p2 := float64(getSample(sourceIdx + 1))
			p3 := float64(getSample(sourceIdx + 2))

			t := fractional
			t2 := t * t
			t3 := t2 * t

			interpolated = int16(
				(2*t3-3*t2+1)*p1 +
					(t3-2*t2+t)*(p2-p0)/2 +
					(-2*t3+3*t2)*p2 +
					(t3-t2)*(p3-p1)/2)
		} else if sourceIdx+1 < sampleCount {
			// Linear interpolation (fast, default)
			val1 := float64(getSample(sourceIdx))
			val2 := float64(getSample(sourceIdx + 1))
			interpolated = int16(val1*(1.0-fractional) + val2*fractional)
		} else if sourceIdx < sampleCount {
			// Use last available sample
			interpolated = getSample(sourceIdx)
		}

		output[outputIdx] = byte(interpolated)
		output[outputIdx+1] = byte(interpolated >> 8)
	}
	return output
}

func (ic *InterpolatingConverter) Close() error {
	return nil
}

func (ic *InterpolatingConverter) Samples() []byte {
	result := ic.buffer
	ic.buffer = nil
	return result
}

// Write implements SampleRateConverter
func (ic *InterpolatingConverter) Write(p []byte) (n int, err error) {
	converted := ic.ConvertSamples(p)
	ic.buffer = append(ic.buffer, converted...)
	return len(p), nil
}
