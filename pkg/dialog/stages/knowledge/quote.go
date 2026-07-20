package knowledge

import (
	"regexp"
	"strings"
	"sync"
)

var (
	quoteTagRe = regexp.MustCompile(`(?is)<quote>(.*?)</quote>`)
	quoteMu    sync.RWMutex
	// lastHitsByCall stores most recent retrieval hits for quote validation.
	lastHitsByCall = map[string][]Hit{}
)

// StoreLastHits caches hits for quote validation on the same call leg.
func StoreLastHits(callID string, hits []Hit) {
	callID = strings.TrimSpace(callID)
	if callID == "" {
		return
	}
	quoteMu.Lock()
	cp := make([]Hit, len(hits))
	copy(cp, hits)
	lastHitsByCall[callID] = cp
	quoteMu.Unlock()
}

// ClearQuoteState removes cached hits on hangup.
func ClearQuoteState(callID string) {
	callID = strings.TrimSpace(callID)
	if callID == "" {
		return
	}
	quoteMu.Lock()
	delete(lastHitsByCall, callID)
	quoteMu.Unlock()
}

// ExtractQuotes parses <quote>...</quote> segments from assistant text.
func ExtractQuotes(text string) []string {
	text = strings.TrimSpace(text)
	if text == "" {
		return nil
	}
	matches := quoteTagRe.FindAllStringSubmatch(text, -1)
	if len(matches) == 0 {
		return nil
	}
	out := make([]string, 0, len(matches))
	for _, m := range matches {
		if len(m) < 2 {
			continue
		}
		q := strings.TrimSpace(m[1])
		if q != "" {
			out = append(out, q)
		}
	}
	return out
}

// ValidateQuotes marks hits as quoted when assistant output contains matching <quote> tags.
func ValidateQuotes(callID, assistantText string, hits []Hit) []Hit {
	quotes := ExtractQuotes(assistantText)
	if len(quotes) == 0 {
		return hits
	}
	quoteMu.RLock()
	cached := lastHitsByCall[strings.TrimSpace(callID)]
	quoteMu.RUnlock()
	pool := hits
	if len(pool) == 0 {
		pool = cached
	}
	if len(pool) == 0 {
		return hits
	}
	out := make([]Hit, len(hits))
	copy(out, hits)
	for i := range out {
		for _, q := range quotes {
			if quoteMatches(out[i], q) {
				out[i].Quoted = true
				break
			}
		}
	}
	return out
}

func quoteMatches(h Hit, quote string) bool {
	quote = strings.TrimSpace(quote)
	if quote == "" {
		return false
	}
	content := strings.TrimSpace(h.Content)
	title := strings.TrimSpace(h.Title)
	if content != "" && (strings.Contains(content, quote) || strings.Contains(quote, content)) {
		return true
	}
	if title != "" && strings.Contains(quote, title) {
		return true
	}
	return false
}

// PrepareAssistantSpeech validates <quote> citations and returns TTS-safe text.
func PrepareAssistantSpeech(callID, raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	ValidateQuotes(callID, raw, nil)
	return StripQuoteTags(raw)
}

// StripQuoteTags removes quote markup from spoken text.
func StripQuoteTags(text string) string {
	return strings.TrimSpace(quoteTagRe.ReplaceAllString(text, ""))
}

// QuotePromptAddon instructs the model to cite knowledge with <quote> tags.
func QuotePromptAddon() string {
	return "引用知识库内容作答时，用 <quote>被引用原文片段</quote> 包裹引用片段（勿向用户宣读标签本身）。"
}
