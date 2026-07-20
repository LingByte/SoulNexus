package coerce

import (
	"encoding/json"
	"strconv"
	"strings"
)

// Float64FromAny coerces JSON or dynamic config values to float64.
func Float64FromAny(v any) (float64, bool) {
	switch t := v.(type) {
	case float64:
		return t, true
	case float32:
		return float64(t), true
	case int:
		return float64(t), true
	case int64:
		return float64(t), true
	case json.Number:
		f, err := t.Float64()
		return f, err == nil
	case string:
		f, err := strconv.ParseFloat(strings.TrimSpace(t), 64)
		return f, err == nil
	default:
		return 0, false
	}
}

// IntFromAny coerces JSON or dynamic config values to int.
func IntFromAny(v any) (int, bool) {
	switch t := v.(type) {
	case int:
		return t, true
	case int64:
		return int(t), true
	case float64:
		return int(t), true
	case float32:
		return int(t), true
	case json.Number:
		n, err := t.Int64()
		return int(n), err == nil
	case string:
		n, err := strconv.Atoi(strings.TrimSpace(t))
		return n, err == nil
	default:
		return 0, false
	}
}

// IntDefault returns def when v cannot be coerced or is not positive.
func IntDefault(v any, def int) int {
	n, ok := IntFromAny(v)
	if !ok || n <= 0 {
		return def
	}
	return n
}

// Float64Default returns def when v cannot be coerced.
func Float64Default(v any, def float64) float64 {
	f, ok := Float64FromAny(v)
	if !ok {
		return def
	}
	return f
}

// BoolFromAny coerces JSON or dynamic config values to bool.
func BoolFromAny(v any) bool {
	switch t := v.(type) {
	case bool:
		return t
	case string:
		s := strings.TrimSpace(strings.ToLower(t))
		return s == "true" || s == "1" || s == "yes"
	case float64:
		return t != 0
	case int:
		return t != 0
	case int64:
		return t != 0
	default:
		return false
	}
}

// IntFromAnyOrZero returns 0 when v cannot be coerced.
func IntFromAnyOrZero(v any) int {
	n, _ := IntFromAny(v)
	return n
}
