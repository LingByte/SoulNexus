package audit

import (
	"bytes"
	"encoding/json"
	"reflect"
	"strings"
)

var sensitiveKeys = map[string]struct{}{
	"password": {}, "passwordhash": {}, "passwd": {}, "secret": {}, "secretkey": {}, "sk": {},
	"token": {}, "accesstoken": {}, "refreshtoken": {}, "apikey": {},
	"apisecret": {}, "privatekey": {}, "totp": {}, "otp": {}, "totpsecret": {},
}

// Redact returns a JSON-safe copy with sensitive fields masked.
func Redact(v any) any {
	if v == nil {
		return nil
	}
	m := Snapshot(v)
	if m == nil {
		return nil
	}
	return m
}

// Snapshot converts a struct/map into a redacted map for diffing.
func Snapshot(v any) map[string]any {
	if v == nil {
		return nil
	}
	b, err := json.Marshal(v)
	if err != nil {
		return map[string]any{"_marshal_error": err.Error()}
	}
	dec := json.NewDecoder(bytes.NewReader(b))
	dec.UseNumber()
	var m map[string]any
	if err := dec.Decode(&m); err != nil {
		return map[string]any{"_raw": string(b)}
	}
	normalizeJSONNumbers(m)
	redactValue(m)
	return m
}

func normalizeJSONNumbers(v any) {
	switch x := v.(type) {
	case map[string]any:
		for k, val := range x {
			if n, ok := val.(json.Number); ok {
				x[k] = n.String()
				continue
			}
			normalizeJSONNumbers(val)
		}
	case []any:
		for i := range x {
			if n, ok := x[i].(json.Number); ok {
				x[i] = n.String()
				continue
			}
			normalizeJSONNumbers(x[i])
		}
	}
}

func redactValue(v any) {
	switch x := v.(type) {
	case map[string]any:
		for k, val := range x {
			if _, ok := sensitiveKeys[strings.ToLower(strings.TrimSpace(k))]; ok {
				x[k] = "***"
				continue
			}
			redactValue(val)
		}
	case []any:
		for i := range x {
			redactValue(x[i])
		}
	}
}

// ComputeChanges returns {field: {from, to}} for differing keys.
func ComputeChanges(before, after map[string]any) map[string]any {
	if before == nil && after == nil {
		return nil
	}
	if before == nil {
		before = map[string]any{}
	}
	if after == nil {
		after = map[string]any{}
	}
	changes := map[string]any{}
	seen := map[string]struct{}{}
	for k, newVal := range after {
		seen[k] = struct{}{}
		oldVal, ok := before[k]
		if !ok || !reflect.DeepEqual(oldVal, newVal) {
			from := any(nil)
			if ok {
				from = oldVal
			}
			changes[k] = map[string]any{"from": from, "to": newVal}
		}
	}
	for k, oldVal := range before {
		if _, ok := seen[k]; ok {
			continue
		}
		changes[k] = map[string]any{"from": oldVal, "to": nil}
	}
	if len(changes) == 0 {
		return nil
	}
	return changes
}

// BuildDetailJSON builds detail_json payload from before/after snapshots.
func BuildDetailJSON(before, after, request any, source string) any {
	detail := map[string]any{}
	if request != nil {
		detail["request"] = Redact(request)
	}
	if source != "" {
		detail["source"] = source
	}
	beforeSnap := Snapshot(before)
	afterSnap := Snapshot(after)
	if len(beforeSnap) > 0 {
		detail["before"] = beforeSnap
	}
	if len(afterSnap) > 0 {
		detail["after"] = afterSnap
	}
	isCreate := len(beforeSnap) == 0 && len(afterSnap) > 0
	isDelete := len(beforeSnap) > 0 && len(afterSnap) == 0
	if !isCreate && !isDelete {
		if ch := ComputeChanges(beforeSnap, afterSnap); len(ch) > 0 {
			detail["changes"] = ch
		}
	}
	if len(detail) == 0 {
		return nil
	}
	return detail
}

// MarshalDetailJSON serializes detail payload for operation_log.detail_json.
func MarshalDetailJSON(v any) string {
	const maxDetailBytes = 32 * 1024
	if v == nil {
		return ""
	}
	b, err := json.Marshal(v)
	if err != nil {
		return ""
	}
	if len(b) > maxDetailBytes {
		return string(b[:maxDetailBytes])
	}
	return string(b)
}
