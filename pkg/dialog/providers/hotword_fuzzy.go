package providers

import (
	"github.com/mozillazg/go-pinyin"
)

var fuzzyShengmuSet = []string{
	"b", "p", "m", "f",
	"d", "t", "n", "l",
	"g", "k", "h",
	"j", "q", "x",
	"zh", "ch", "sh", "r",
	"z", "c", "s",
	"y", "w",
}

var fuzzyPhonemeMap = map[string]string{
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

func applyFuzzyCorrection(text string, fuzzyCanon map[string]string) string {
	if text == "" || len(fuzzyCanon) == 0 {
		return text
	}
	targets := make([]string, 0, len(fuzzyCanon))
	for w := range fuzzyCanon {
		if w != "" {
			targets = append(targets, w)
		}
	}
	runes := []rune(text)
	out := make([]rune, 0, len(runes))
	for i := 0; i < len(runes); {
		matched := false
		for _, canon := range targets {
			cr := []rune(canon)
			n := len(cr)
			if n == 0 || i+n > len(runes) {
				continue
			}
			seg := string(runes[i : i+n])
			if seg == canon || pinyinSimilar(seg, canon) {
				out = append(out, cr...)
				i += n
				matched = true
				break
			}
		}
		if !matched {
			out = append(out, runes[i])
			i++
		}
	}
	return string(out)
}

func pinyinSimilar(a, b string) bool {
	if len([]rune(a)) != len([]rune(b)) {
		return false
	}
	pyA := convertToPinyinArray(a)
	pyB := convertToPinyinArray(b)
	return computePinyinSimilarity(pyA, pyB) == 1.0
}

func convertToPinyinArray(s string) []string {
	args := pinyin.NewArgs()
	args.Style = pinyin.Normal
	pys := pinyin.Pinyin(s, args)
	res := make([]string, 0, len(pys))
	for _, p := range pys {
		if len(p) > 0 {
			res = append(res, p[0])
		}
	}
	return res
}

func computePinyinSimilarity(pys1, pys2 []string) float64 {
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
		if shengmuSimilarity(sm1, sm2) != 1.0 {
			return 0
		}
		if yunmuSimilarity(ym1, ym2) != 1.0 {
			return 0
		}
	}
	return 1.0
}

func isFuzzyShengmu(s string) bool {
	for _, sm := range fuzzyShengmuSet {
		if s == sm {
			return true
		}
	}
	return false
}

func splitShengmuYunmu(py string) (string, string) {
	if len(py) >= 2 {
		two := py[:2]
		if isFuzzyShengmu(two) {
			return two, py[2:]
		}
	}
	if len(py) >= 1 {
		one := py[:1]
		if isFuzzyShengmu(one) {
			return one, py[1:]
		}
	}
	return "", py
}

func shengmuSimilarity(sm1, sm2 string) float64 {
	if sm1 == sm2 {
		return 1.0
	}
	if fuzzyPhonemeMap[sm1] == sm2 || fuzzyPhonemeMap[sm2] == sm1 {
		return 1.0
	}
	return 0.0
}

func yunmuSimilarity(ym1, ym2 string) float64 {
	if ym1 == ym2 {
		return 1.0
	}
	if fuzzyPhonemeMap[ym1] == ym2 || fuzzyPhonemeMap[ym2] == ym1 {
		return 1.0
	}
	return 0.0
}
