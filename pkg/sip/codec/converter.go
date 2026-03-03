package codec

// PCMUToPCM16 将 PCMU (G.711 μ-law) 转换为 PCM16
func PCMUToPCM16(pcmuData []byte) []byte {
	pcm16Data := make([]byte, len(pcmuData)*2)

	for i, mulaw := range pcmuData {
		// 解码 μ-law 为 16-bit PCM
		pcm := muLawDecompressTable[mulaw]

		// 写入 16-bit PCM (小端序)
		pcm16Data[i*2] = byte(pcm & 0xFF)
		pcm16Data[i*2+1] = byte((pcm >> 8) & 0xFF)
	}

	return pcm16Data
}

// PCM16ToPCMU 将 PCM16 转换为 PCMU (G.711 μ-law)
func PCM16ToPCMU(pcm16Data []byte) []byte {
	if len(pcm16Data)%2 != 0 {
		// 确保是偶数长度
		pcm16Data = append(pcm16Data, 0)
	}

	pcmuData := make([]byte, len(pcm16Data)/2)

	for i := 0; i < len(pcm16Data); i += 2 {
		// 读取 16-bit PCM (小端序)
		pcm := int16(pcm16Data[i]) | (int16(pcm16Data[i+1]) << 8)

		// 编码为 μ-law
		pcmuData[i/2] = linearToMuLaw(pcm)
	}

	return pcmuData
}

// ResampleAudio 简单的音频重采样（线性插值）
// 从 fromRate 重采样到 toRate
func ResampleAudio(audioData []byte, fromRate, toRate int) []byte {
	if fromRate == toRate {
		return audioData
	}

	// 计算采样比率
	ratio := float64(toRate) / float64(fromRate)
	newLength := int(float64(len(audioData)) * ratio)

	// 确保是偶数长度（16-bit 样本）
	if newLength%2 != 0 {
		newLength++
	}

	resampled := make([]byte, newLength)

	// 简单的线性插值
	for i := 0; i < newLength/2; i++ {
		srcPos := float64(i) / ratio
		srcIdx := int(srcPos) * 2

		if srcIdx+3 < len(audioData) {
			// 线性插值
			frac := srcPos - float64(int(srcPos))

			sample1 := int16(audioData[srcIdx]) | (int16(audioData[srcIdx+1]) << 8)
			sample2 := int16(audioData[srcIdx+2]) | (int16(audioData[srcIdx+3]) << 8)

			interpolated := int16(float64(sample1)*(1-frac) + float64(sample2)*frac)

			resampled[i*2] = byte(interpolated & 0xFF)
			resampled[i*2+1] = byte((interpolated >> 8) & 0xFF)
		} else if srcIdx+1 < len(audioData) {
			// 边界情况：直接复制
			resampled[i*2] = audioData[srcIdx]
			resampled[i*2+1] = audioData[srcIdx+1]
		}
	}

	return resampled
}

// ConvertPCMUToPCM16WithResampling 将 PCMU 转换为 PCM16 并重采样
// pcmuData: PCMU 数据
// fromRate: PCMU 采样率 (通常是 8000)
// toRate: 目标 PCM16 采样率 (如 16000)
func ConvertPCMUToPCM16WithResampling(pcmuData []byte, fromRate, toRate int) []byte {
	// 1. PCMU 转 PCM16
	pcm16Data := PCMUToPCM16(pcmuData)

	// 2. 重采样
	if fromRate != toRate {
		pcm16Data = ResampleAudio(pcm16Data, fromRate, toRate)
	}

	return pcm16Data
}

// ConvertPCM16ToPCMUWithResampling 将 PCM16 转换为 PCMU 并重采样
// pcm16Data: PCM16 数据
// fromRate: 源 PCM16 采样率 (如 16000)
// toRate: 目标 PCMU 采样率 (通常是 8000)
func ConvertPCM16ToPCMUWithResampling(pcm16Data []byte, fromRate, toRate int) []byte {
	// 1. 重采样
	if fromRate != toRate {
		pcm16Data = ResampleAudio(pcm16Data, fromRate, toRate)
	}

	// 2. PCM16 转 PCMU
	return PCM16ToPCMU(pcm16Data)
}
