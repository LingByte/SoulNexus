package logger

import (
	"fmt"
	"regexp"
	"strings"
	"sync"

	"go.uber.org/zap/zapcore"
)

const DefaultSensitiveFields = "password,passwd,pwd,dsn,secret,token,access_token,refresh_token,api_key,apikey,ssn,phone,email"

var (
	globalRedactor   = newRedactor(parseSensitiveFieldsCSV(DefaultSensitiveFields))
	globalRedactorMu sync.RWMutex
)

type redactStyle int

const (
	redactStyleString redactStyle = iota
	redactStyleEmail
	redactStylePhone
)

// redactor applies field-name based log redaction. Enabled by default.
type redactor struct {
	mu        sync.RWMutex
	sensitive map[string]bool
	styles    map[string]redactStyle
	jsonRE    *regexp.Regexp
}

func initRedactor(sensitiveFieldsCSV string) *redactor {
	raw := strings.TrimSpace(sensitiveFieldsCSV)
	if raw == "" {
		raw = DefaultSensitiveFields
	}
	r := newRedactor(parseSensitiveFieldsCSV(raw))
	globalRedactorMu.Lock()
	globalRedactor = r
	globalRedactorMu.Unlock()
	return r
}

func globalRedactorInstance() *redactor {
	globalRedactorMu.RLock()
	defer globalRedactorMu.RUnlock()
	return globalRedactor
}

func newRedactor(fields []string) *redactor {
	r := &redactor{
		sensitive: make(map[string]bool, len(fields)),
		styles:    make(map[string]redactStyle, len(fields)),
	}
	for _, field := range fields {
		key := strings.ToLower(strings.TrimSpace(field))
		if key == "" {
			continue
		}
		r.sensitive[key] = true
		r.styles[key] = styleForField(key)
	}
	r.rebuildJSONPattern()
	return r
}

func parseSensitiveFieldsCSV(raw string) []string {
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			out = append(out, part)
		}
	}
	return out
}

func styleForField(key string) redactStyle {
	switch key {
	case "email":
		return redactStyleEmail
	case "phone", "mobile", "telephone":
		return redactStylePhone
	default:
		return redactStyleString
	}
}

func (r *redactor) isSensitive(key string) bool {
	if r == nil {
		return false
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.sensitive[strings.ToLower(key)]
}

func (r *redactor) redactString(s string) string {
	if len(s) <= 4 {
		return "****"
	}
	return s[:2] + "****" + s[len(s)-2:]
}

func (r *redactor) redactEmail(email string) string {
	parts := strings.Split(email, "@")
	if len(parts) != 2 {
		return r.redactString(email)
	}
	username := parts[0]
	domain := parts[1]
	if len(username) <= 2 {
		return "****@" + domain
	}
	return username[:2] + "****@" + domain
}

func (r *redactor) redactPhone(phone string) string {
	if len(phone) <= 7 {
		return "****"
	}
	return phone[:3] + "****" + phone[len(phone)-4:]
}

func (r *redactor) styleFor(key string) redactStyle {
	keyLower := strings.ToLower(key)
	r.mu.RLock()
	style, ok := r.styles[keyLower]
	r.mu.RUnlock()
	if ok {
		return style
	}
	return redactStyleString
}

func (r *redactor) redactByStyle(style redactStyle, value string) string {
	switch style {
	case redactStyleEmail:
		return r.redactEmail(value)
	case redactStylePhone:
		return r.redactPhone(value)
	default:
		return r.redactString(value)
	}
}

func (r *redactor) redactField(key string, value interface{}) interface{} {
	if r == nil || !r.isSensitive(key) {
		return value
	}
	return r.redactByStyle(r.styleFor(key), fmt.Sprintf("%v", value))
}

func (r *redactor) redactFields(fields map[string]interface{}) map[string]interface{} {
	if r == nil {
		return fields
	}
	out := make(map[string]interface{}, len(fields))
	for k, v := range fields {
		out[k] = r.redactField(k, v)
	}
	return out
}

func (r *redactor) redactValue(data string) string {
	if r == nil {
		return data
	}
	r.mu.RLock()
	re := r.jsonRE
	r.mu.RUnlock()
	if re == nil {
		return data
	}
	return re.ReplaceAllStringFunc(data, func(match string) string {
		submatches := re.FindStringSubmatch(match)
		if len(submatches) >= 3 {
			key := submatches[1]
			value := submatches[2]
			redacted, _ := r.redactField(key, value).(string)
			return fmt.Sprintf(`"%s":"%s"`, key, redacted)
		}
		return match
	})
}

func (r *redactor) rebuildJSONPattern() {
	r.mu.Lock()
	defer r.mu.Unlock()
	if len(r.sensitive) == 0 {
		r.jsonRE = nil
		return
	}
	keys := make([]string, 0, len(r.sensitive))
	for key := range r.sensitive {
		keys = append(keys, regexp.QuoteMeta(key))
	}
	pattern := `"(?i)(` + strings.Join(keys, "|") + `)"\s*:\s*"([^"]*)"`
	r.jsonRE = regexp.MustCompile(pattern)
}

// RedactString redacts a generic string value.
func RedactString(s string) string {
	return globalRedactorInstance().redactString(s)
}

// RedactEmail redacts an email address.
func RedactEmail(email string) string {
	return globalRedactorInstance().redactEmail(email)
}

// RedactPhone redacts a phone number.
func RedactPhone(phone string) string {
	return globalRedactorInstance().redactPhone(phone)
}

// RedactField redacts value when key is a configured sensitive field.
func RedactField(key string, value interface{}) interface{} {
	return globalRedactorInstance().redactField(key, value)
}

// RedactFields redacts sensitive keys in a map.
func RedactFields(fields map[string]interface{}) map[string]interface{} {
	return globalRedactorInstance().redactFields(fields)
}

// RedactValue redacts JSON-like sensitive key/value pairs in a string.
func RedactValue(data string) string {
	return globalRedactorInstance().redactValue(data)
}

// IsSensitiveField reports whether key is configured as sensitive.
func IsSensitiveField(key string) bool {
	return globalRedactorInstance().isSensitive(key)
}

type redactCore struct {
	zapcore.Core
	redactor *redactor
}

func wrapCoreWithRedact(inner zapcore.Core, r *redactor) zapcore.Core {
	if inner == nil || r == nil {
		return inner
	}
	return &redactCore{Core: inner, redactor: r}
}

func (c *redactCore) With(fields []zapcore.Field) zapcore.Core {
	return &redactCore{
		Core:     c.Core.With(c.redactor.redactZapFields(fields)),
		redactor: c.redactor,
	}
}

func (c *redactCore) Check(ent zapcore.Entry, ce *zapcore.CheckedEntry) *zapcore.CheckedEntry {
	if c.Enabled(ent.Level) {
		return ce.AddCore(ent, c)
	}
	return ce
}

func (c *redactCore) Write(ent zapcore.Entry, fields []zapcore.Field) error {
	ent.Message = c.redactor.redactValue(ent.Message)
	return c.Core.Write(ent, c.redactor.redactZapFields(fields))
}

func (r *redactor) redactZapFields(fields []zapcore.Field) []zapcore.Field {
	if r == nil || len(fields) == 0 {
		return fields
	}

	changed := false
	out := fields
	for i, field := range fields {
		next, ok := r.redactZapField(field)
		if !ok {
			continue
		}
		if !changed {
			out = make([]zapcore.Field, len(fields))
			copy(out, fields)
			changed = true
		}
		out[i] = next
	}
	return out
}

func (r *redactor) redactZapField(field zapcore.Field) (zapcore.Field, bool) {
	if !r.isSensitive(field.Key) {
		return field, false
	}

	style := r.styleFor(field.Key)
	switch field.Type {
	case zapcore.StringType:
		field.String = r.redactByStyle(style, field.String)
		return field, true
	case zapcore.ByteStringType:
		field.String = r.redactByStyle(style, string(field.String))
		field.Type = zapcore.StringType
		return field, true
	case zapcore.Complex128Type, zapcore.Complex64Type, zapcore.DurationType,
		zapcore.Float64Type, zapcore.Float32Type, zapcore.Int64Type, zapcore.Int32Type,
		zapcore.Int16Type, zapcore.Int8Type, zapcore.Uint64Type, zapcore.Uint32Type,
		zapcore.Uint16Type, zapcore.Uint8Type, zapcore.UintptrType, zapcore.BoolType,
		zapcore.TimeType, zapcore.TimeFullType:
		field.Type = zapcore.StringType
		field.String = r.redactByStyle(style, fmt.Sprint(field.Interface))
		field.Interface = nil
		return field, true
	case zapcore.ReflectType, zapcore.ObjectMarshalerType, zapcore.ArrayMarshalerType,
		zapcore.BinaryType, zapcore.StringerType, zapcore.ErrorType:
		field.Type = zapcore.StringType
		field.String = r.redactByStyle(style, fmt.Sprint(field.Interface))
		field.Interface = nil
		return field, true
	default:
		if field.Interface != nil {
			field.Type = zapcore.StringType
			field.String = r.redactByStyle(style, fmt.Sprint(field.Interface))
			field.Interface = nil
			return field, true
		}
		return field, false
	}
}
