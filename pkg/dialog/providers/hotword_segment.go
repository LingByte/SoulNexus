package providers

import (
	"strings"
	"sync"

	"github.com/go-ego/gse"
)

var (
	segOnce sync.Once
	segGse  *gse.Segmenter
)

func hotwordSegmenter() *gse.Segmenter {
	segOnce.Do(func() {
		var seg gse.Segmenter
		seg.LoadDict()
		segGse = &seg
	})
	return segGse
}

func correctByTokens(text string, replaceWords, fuzzyCanon map[string]string) string {
	if text == "" {
		return text
	}
	seg := hotwordSegmenter()
	if seg == nil {
		return correctWithoutSegmenter(text, replaceWords, fuzzyCanon)
	}
	words := seg.Cut(text, true)
	for i, word := range words {
		if canon, ok := replaceWords[word]; ok {
			wr, wc := []rune(word), []rune(canon)
			if len(wr) > 0 && len(wr) < len(wc) {
				words[i] = string(wc[:len(wr)])
			} else {
				words[i] = canon
			}
			continue
		}
		for _, fw := range fuzzyCanon {
			if len([]rune(fw)) != len([]rune(word)) {
				continue
			}
			if pinyinSimilar(word, fw) {
				words[i] = fw
				break
			}
		}
	}
	return strings.Join(words, "")
}

func correctWithoutSegmenter(text string, replaceWords, fuzzyCanon map[string]string) string {
	t := text
	if len(replaceWords) > 0 {
		rp := make([]string, 0, len(replaceWords)*2)
		for from, to := range replaceWords {
			if from != "" && to != "" && from != to {
				rp = append(rp, from, to)
			}
		}
		if len(rp) > 0 {
			t = strings.NewReplacer(rp...).Replace(t)
		}
	}
	if len(fuzzyCanon) > 0 {
		t = applyFuzzyCorrection(t, fuzzyCanon)
	}
	return t
}
