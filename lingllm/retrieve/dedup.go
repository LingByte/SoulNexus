package retrieve

import (
	"crypto/md5"
	"encoding/hex"
	"sort"
	"strings"
)

// DeduplicateDocuments removes duplicate chunk IDs and near-duplicate content signatures.
func DeduplicateDocuments(docs []*Document) []*Document {
	if len(docs) <= 1 {
		return docs
	}
	seenID := make(map[string]bool, len(docs))
	contentSig := make(map[string]string)
	out := make([]*Document, 0, len(docs))
	for _, d := range docs {
		if d == nil {
			continue
		}
		if d.ID != "" {
			if seenID[d.ID] {
				continue
			}
			seenID[d.ID] = true
		}
		sig := BuildContentSignature(d.Content)
		if sig != "" {
			if first, exists := contentSig[sig]; exists {
				_ = first
				continue
			}
			contentSig[sig] = d.ID
		}
		out = append(out, d)
	}
	return out
}

// RemovePartialOverlaps drops lower-scored docs largely contained in higher-scored ones.
func RemovePartialOverlaps(docs []*Document, threshold float64) []*Document {
	if threshold <= 0 {
		threshold = 0.85
	}
	if len(docs) <= 1 {
		return docs
	}
	type normEntry struct {
		norm string
		doc  *Document
	}
	entries := make([]normEntry, 0, len(docs))
	for _, d := range docs {
		if d == nil {
			continue
		}
		entries = append(entries, normEntry{
			norm: NormalizeContent(d.Content),
			doc:  d,
		})
	}
	removed := make(map[int]bool)
	for i := 0; i < len(entries); i++ {
		if removed[i] {
			continue
		}
		for j := i + 1; j < len(entries); j++ {
			if removed[j] {
				continue
			}
			a, b := entries[i], entries[j]
			shortIdx, longIdx := i, j
			if len(a.norm) > len(b.norm) {
				shortIdx, longIdx = j, i
			}
			contained := IsContentContained(entries[shortIdx].norm, entries[longIdx].norm)
			if contained {
				removed[shortIdx] = true
				continue
			}
			ratio := ContentOverlapRatio(entries[shortIdx].doc.Content, entries[longIdx].doc.Content)
			if ratio < threshold {
				continue
			}
			victim := shortIdx
			if entries[shortIdx].doc.Score > entries[longIdx].doc.Score {
				victim = longIdx
			}
			removed[victim] = true
		}
	}
	out := make([]*Document, 0, len(entries)-len(removed))
	for i, e := range entries {
		if !removed[i] {
			out = append(out, e.doc)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Score > out[j].Score })
	return out
}

// BuildContentSignature fingerprints normalized content for deduplication.
func BuildContentSignature(content string) string {
	c := NormalizeContent(content)
	if c == "" {
		return ""
	}
	hash := md5.Sum([]byte(c))
	return hex.EncodeToString(hash[:])
}

// NormalizeContent lowercases and collapses whitespace.
func NormalizeContent(s string) string {
	c := strings.ToLower(strings.TrimSpace(s))
	if c == "" {
		return ""
	}
	return strings.Join(strings.Fields(c), " ")
}

// IsContentContained reports whether normalizedShort is a substring of normalizedLong.
func IsContentContained(normalizedShort, normalizedLong string) bool {
	if normalizedShort == "" || normalizedLong == "" {
		return false
	}
	if len(normalizedShort) > len(normalizedLong) {
		return false
	}
	return strings.Contains(normalizedLong, normalizedShort)
}

// ContentOverlapRatio returns |intersection| / |smaller token set|.
func ContentOverlapRatio(a, b string) float64 {
	tokA := tokenizeSimple(a)
	tokB := tokenizeSimple(b)
	if len(tokA) == 0 || len(tokB) == 0 {
		return 0
	}
	small, large := tokA, tokB
	if len(tokA) > len(tokB) {
		small, large = tokB, tokA
	}
	inter := 0
	for k := range small {
		if _, ok := large[k]; ok {
			inter++
		}
	}
	return float64(inter) / float64(len(small))
}
