package knowledge

import (
	"crypto/md5"
	"math"
	"strings"
)

// GenerateEmbedding 生成文本的embedding向量
// 使用 TF-IDF 方法生成向量
func GenerateEmbedding(text string, dimension int) []float32 {
	if dimension <= 0 {
		dimension = 384
	}

	// 简单的分词（按空格和标点符号分割）
	words := tokenize(text)
	if len(words) == 0 {
		// 如果没有单词，返回零向量
		return make([]float32, dimension)
	}

	// 计算词频
	wordFreq := make(map[string]int)
	for _, word := range words {
		wordFreq[word]++
	}

	// 生成向量：使用词频和词的哈希值
	embedding := make([]float32, dimension)

	for word, freq := range wordFreq {
		// 计算词的哈希值
		hash := md5.Sum([]byte(word))

		// 使用哈希值确定这个词对哪些维度有贡献
		for i := 0; i < dimension; i++ {
			// 使用哈希的字节生成伪随机索引
			idx := (int(hash[i%len(hash)]) + i) % dimension

			// 计算这个词对该维度的贡献
			// 使用词频和哈希值的组合
			contribution := float32(freq) * float32(hash[i%len(hash)]) / 255.0
			embedding[idx] += contribution
		}
	}

	// L2 归一化
	var norm float32
	for _, v := range embedding {
		norm += v * v
	}
	norm = float32(math.Sqrt(float64(norm)))

	if norm > 0 {
		for i := range embedding {
			embedding[i] /= norm
		}
	}

	return embedding
}

// GenerateEmbeddingFromBytes 从字节生成embedding
func GenerateEmbeddingFromBytes(data []byte, dimension int) []float32 {
	return GenerateEmbedding(string(data), dimension)
}

// tokenize 简单的分词函数
func tokenize(text string) []string {
	// 转换为小写
	text = strings.ToLower(text)

	// 替换标点符号为空格
	replacer := strings.NewReplacer(
		".", " ", ",", " ", "!", " ", "?", " ",
		";", " ", ":", " ", "'", " ", "\"", " ",
		"(", " ", ")", " ", "[", " ", "]", " ",
		"{", " ", "}", " ", "-", " ", "_", " ",
		"/", " ", "\\", " ", "|", " ", "@", " ",
		"#", " ", "$", " ", "%", " ", "^", " ",
		"&", " ", "*", " ", "+", " ", "=", " ",
	)
	text = replacer.Replace(text)

	// 按空格分割
	words := strings.Fields(text)

	// 过滤空字符串和短词
	var result []string
	for _, word := range words {
		if len(word) > 2 { // 只保留长度大于2的词
			result = append(result, word)
		}
	}

	return result
}
