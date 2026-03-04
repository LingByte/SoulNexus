package recognizer

import (
	"strings"
	"sync"

	"github.com/go-ego/gse"
	"github.com/mozillazg/go-pinyin"
)

var (
	once      sync.Once
	sharedSeg *gse.Segmenter
)

type AsrCorrector struct {
	segmenter    *gse.Segmenter
	ReplaceWords map[string]string
	FuzzyWords   map[string]string
}

type AsrCorrectorOption struct {
	ReplaceWords map[string]string `json:"replaceWords"`
	FuzzyWords   map[string]string `json:"fuzzyWords"`
}

func NewAsrCorrector(opt AsrCorrectorOption) *AsrCorrector {
	once.Do(func() {
		var seg gse.Segmenter
		seg.Dict = gse.NewDict()
		seg.Load = true
		seg.Init()
		_ = seg.Read("gse_data/dict/zh/s_1.txt")
		_ = seg.Read("gse_data/dict/zh/t_1.txt")
		sharedSeg = &seg
	})

	if opt.ReplaceWords == nil {
		opt.ReplaceWords = map[string]string{}
	}

	if opt.FuzzyWords == nil {
		opt.FuzzyWords = map[string]string{}
	}

	return &AsrCorrector{
		segmenter:    sharedSeg,
		ReplaceWords: opt.ReplaceWords,
		FuzzyWords:   opt.FuzzyWords,
	}
}

var shengmuSet = []string{
	"b", "p", "m", "f",
	"d", "t", "n", "l",
	"g", "k", "h",
	"j", "q", "x",
	"zh", "ch", "sh", "r",
	"z", "c", "s",
	"y", "w",
}

func (ac *AsrCorrector) Correct(text string) string {
	words := ac.segmenter.Cut(text, true)

	for i, word := range words {
		// 1. text exact replace
		if rw, ok := ac.ReplaceWords[word]; ok {
			words[i] = rw
			continue
		}

		// 2. pinyin fuzzy replace
		for _, fw := range ac.FuzzyWords {
			if len(fw) != len(word) {
				continue
			}

			if ac.isSimilar(word, fw) {
				words[i] = fw
				break
			}
		}

	}

	return strings.Join(words, "")
}

func (ac *AsrCorrector) isSimilar(a, b string) bool {
	pyA := convertToPinyinArray(a)
	pyB := convertToPinyinArray(b)

	sim := computeSimilarity(pyA, pyB)

	return sim == 1.0
}

func convertToPinyinArray(s string) []string {
	args := pinyin.NewArgs()
	args.Style = pinyin.Normal
	pys := pinyin.Pinyin(s, args)
	res := []string{}
	for _, p := range pys {
		if len(p) > 0 {
			res = append(res, p[0])
		}
	}
	return res
}

func computeSimilarity(pys1, pys2 []string) float64 {
	n := len(pys1)
	if len(pys2) < n {
		n = len(pys2)
	}
	if n == 0 {
		return 0
	}

	for i := 0; i < n; i++ {
		sm1, ym1 := splitShengmuYunmu(pys1[i])
		sm2, ym2 := splitShengmuYunmu(pys2[i])
		smSim := shengmuSimilarity(sm1, sm2)
		if smSim != 1.0 {
			return 0.0
		}

		ymSim := yunmuSimilarity(ym1, ym2)
		if ymSim != 1.0 {
			return 0.0
		}

	}

	return 1.0
}

func isShengmu(s string) bool {
	for _, sm := range shengmuSet {
		if s == sm {
			return true
		}
	}
	return false
}

// e.g: "long" -> ("l", "ong"), "ai" -> ("", "ai")
func splitShengmuYunmu(py string) (string, string) {
	// at least 2（zh, ch, sh）
	if len(py) >= 2 {
		two := py[:2]
		if isShengmu(two) {
			return two, py[2:]
		}
	}
	if len(py) >= 1 {
		one := py[:1]
		if isShengmu(one) {
			return one, py[1:]
		}
	}

	return "", py
}

var FussyMap = map[string]string{
	"s": "sh", "sh": "s",
	"c": "ch", "ch": "c",
	"z": "zh", "zh": "z",
	"l": "n", "n": "l",
	"f": "h", "h": "f",
	"r":  "l",
	"an": "ang", "ang": "an",
	"en": "eng", "eng": "en",
	"in": "ing", "ing": "in",
	"ian": "iang", "iang": "ian",
	"uan": "uang", "uang": "uan",
}

func shengmuSimilarity(sm1, sm2 string) float64 {
	if sm1 == sm2 {
		return 1.0
	}
	if FussyMap[sm1] == sm2 || FussyMap[sm2] == sm1 {
		return 1.0
	}

	return 0.0
}

func yunmuSimilarity(ym1, ym2 string) float64 {
	if ym1 == ym2 {
		return 1.0
	}
	if FussyMap[ym1] == ym2 || FussyMap[ym2] == ym1 {
		return 1.0
	}

	return 0.0
}

func (ac *AsrCorrector) SegmentWords(s string) []string {
	tokens := ac.segmenter.Cut(s, true) // true 表示精确分词
	return tokens
}
