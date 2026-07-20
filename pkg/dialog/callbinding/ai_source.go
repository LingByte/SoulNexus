package callbinding

import (
	"strings"
	"sync"
)

var callAISources sync.Map // callID -> string

// SetAISource stores the product surface that opened this session/call
// (assistant_debug_text / assistant_debug_voice / js_template / js_embed / voice…).
func SetAISource(callID, source string) {
	callID = strings.TrimSpace(callID)
	source = strings.TrimSpace(source)
	if callID == "" || source == "" {
		return
	}
	callAISources.Store(callID, source)
}

// GetAISource returns the bound AI invocation source (empty if unset).
func GetAISource(callID string) string {
	callID = strings.TrimSpace(callID)
	if callID == "" {
		return ""
	}
	if v, ok := callAISources.Load(callID); ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

// ClearAISource removes the bound source.
func ClearAISource(callID string) {
	callID = strings.TrimSpace(callID)
	if callID == "" {
		return
	}
	callAISources.Delete(callID)
}

var callJSSourceIDs sync.Map // callID -> js_source_id

// SetJSSourceID binds an embed JS template id to the session.
func SetJSSourceID(callID, jsSourceID string) {
	callID = strings.TrimSpace(callID)
	jsSourceID = strings.TrimSpace(jsSourceID)
	if callID == "" || jsSourceID == "" {
		return
	}
	callJSSourceIDs.Store(callID, jsSourceID)
}

// GetJSSourceID returns the bound js_source_id (empty if unset).
func GetJSSourceID(callID string) string {
	callID = strings.TrimSpace(callID)
	if callID == "" {
		return ""
	}
	if v, ok := callJSSourceIDs.Load(callID); ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

// ClearJSSourceID removes the bound template id.
func ClearJSSourceID(callID string) {
	callID = strings.TrimSpace(callID)
	if callID == "" {
		return
	}
	callJSSourceIDs.Delete(callID)
}
