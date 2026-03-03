package codec

// PCMU (G.711 μ-law) 编码器/解码器 - 纯 Go 实现

var (
	muLawCompressTable = [256]byte{
		0, 0, 1, 1, 2, 2, 2, 2, 3, 3, 3, 3, 3, 3, 3, 3,
		4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4,
		5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5,
		5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5,
		6, 6, 6, 6, 6, 6, 6, 6, 6, 6, 6, 6, 6, 6, 6, 6,
		6, 6, 6, 6, 6, 6, 6, 6, 6, 6, 6, 6, 6, 6, 6, 6,
		6, 6, 6, 6, 6, 6, 6, 6, 6, 6, 6, 6, 6, 6, 6, 6,
		6, 6, 6, 6, 6, 6, 6, 6, 6, 6, 6, 6, 6, 6, 6, 6,
		7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7,
		7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7,
		7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7,
		7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7,
		7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7,
		7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7,
		7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7,
		7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7,
	}

	muLawDecompressTable = [256]int16{
		-32124, -31100, -30076, -29052, -28028, -27004, -25980, -24956,
		-23932, -22908, -21884, -20860, -19836, -18812, -17788, -16764,
		-15996, -15484, -14972, -14460, -13948, -13436, -12924, -12412,
		-11900, -11388, -10876, -10364, -9852, -9340, -8828, -8316,
		-7932, -7676, -7420, -7164, -6908, -6652, -6396, -6140,
		-5884, -5628, -5372, -5116, -4860, -4604, -4348, -4092,
		-3900, -3772, -3644, -3516, -3388, -3260, -3132, -3004,
		-2876, -2748, -2620, -2492, -2364, -2236, -2108, -1980,
		-1884, -1820, -1756, -1692, -1628, -1564, -1500, -1436,
		-1372, -1308, -1244, -1180, -1116, -1052, -988, -924,
		-876, -844, -812, -780, -748, -716, -684, -652,
		-620, -588, -556, -524, -492, -460, -428, -396,
		-372, -356, -340, -324, -308, -292, -276, -260,
		-244, -228, -212, -196, -180, -164, -148, -132,
		-120, -112, -104, -96, -88, -80, -72, -64,
		-56, -48, -40, -32, -24, -16, -8, 0,
		32124, 31100, 30076, 29052, 28028, 27004, 25980, 24956,
		23932, 22908, 21884, 20860, 19836, 18812, 17788, 16764,
		15996, 15484, 14972, 14460, 13948, 13436, 12924, 12412,
		11900, 11388, 10876, 10364, 9852, 9340, 8828, 8316,
		7932, 7676, 7420, 7164, 6908, 6652, 6396, 6140,
		5884, 5628, 5372, 5116, 4860, 4604, 4348, 4092,
		3900, 3772, 3644, 3516, 3388, 3260, 3132, 3004,
		2876, 2748, 2620, 2492, 2364, 2236, 2108, 1980,
		1884, 1820, 1756, 1692, 1628, 1564, 1500, 1436,
		1372, 1308, 1244, 1180, 1116, 1052, 988, 924,
		876, 844, 812, 780, 748, 716, 684, 652,
		620, 588, 556, 524, 492, 460, 428, 396,
		372, 356, 340, 324, 308, 292, 276, 260,
		244, 228, 212, 196, 180, 164, 148, 132,
		120, 112, 104, 96, 88, 80, 72, 64,
		56, 48, 40, 32, 24, 16, 8, 0,
	}
)

// PCMUEncoder 将 PCM 编码为 PCMU (G.711 μ-law)
type PCMUEncoder struct {
	sampleRate int
	channels   int
}

// PCMUDecoder 将 PCMU 解码为 PCM
type PCMUDecoder struct {
	sampleRate int
	channels   int
}

// NewPCMUEncoder 创建新的 PCMU 编码器
func NewPCMUEncoder(sampleRate, channels int) (*PCMUEncoder, error) {
	return &PCMUEncoder{
		sampleRate: sampleRate,
		channels:   channels,
	}, nil
}

// NewPCMUDecoder 创建新的 PCMU 解码器
func NewPCMUDecoder(sampleRate, channels int) (*PCMUDecoder, error) {
	return &PCMUDecoder{
		sampleRate: sampleRate,
		channels:   channels,
	}, nil
}

// Encode 将 PCM 样本编码为 PCMU
func (e *PCMUEncoder) Encode(pcm []int16) ([]byte, error) {
	output := make([]byte, len(pcm))
	for i, sample := range pcm {
		output[i] = linearToMuLaw(sample)
	}
	return output, nil
}

// Decode 将 PCMU 解码为 PCM 样本
func (d *PCMUDecoder) Decode(data []byte) ([]int16, error) {
	output := make([]int16, len(data))
	for i, mulaw := range data {
		output[i] = muLawDecompressTable[mulaw]
	}
	return output, nil
}

// linearToMuLaw 将 16 位线性 PCM 样本转换为 μ-law
func linearToMuLaw(sample int16) byte {
	const BIAS = 0x84
	const CLIP = 32635

	var sign byte
	var exponent byte
	var mantissa byte
	var mulaw byte

	// 获取符号
	if sample < 0 {
		sample = -sample
		sign = 0x80
	} else {
		sign = 0
	}

	// 限幅
	if sample > CLIP {
		sample = CLIP
	}

	// 添加偏置
	sample = sample + BIAS

	// 查找指数
	exponent = muLawCompressTable[(sample>>7)&0xFF]

	// 查找尾数
	mantissa = byte((sample >> (exponent + 3)) & 0x0F)

	// 组合符号、指数和尾数
	mulaw = ^(sign | (exponent << 4) | mantissa)

	return mulaw
}

// GetSampleRate 返回采样率
func (e *PCMUEncoder) GetSampleRate() int {
	return e.sampleRate
}

// GetChannels 返回声道数
func (e *PCMUEncoder) GetChannels() int {
	return e.channels
}

// GetSampleRate 返回采样率
func (d *PCMUDecoder) GetSampleRate() int {
	return d.sampleRate
}

// GetChannels 返回声道数
func (d *PCMUDecoder) GetChannels() int {
	return d.channels
}

// Close 关闭编码器（PCMU 无操作）
func (e *PCMUEncoder) Close() {}

// Close 关闭解码器（PCMU 无操作）
func (d *PCMUDecoder) Close() {}
