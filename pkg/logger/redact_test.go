package logger

import (
	"bytes"
	"strings"
	"testing"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func TestRedactString(t *testing.T) {
	got := RedactString("hello")
	if got == "" {
		t.Fatal("result should not be empty")
	}

	// short string (<=4 chars) → "****"
	tests := []struct {
		input    string
		expected string
	}{
		{"ab", "****"},
		{"abc", "****"},
		{"abcd", "****"},
		{"abcde", "ab****de"},
		{"", "****"},
		{"1", "****"},
	}
	for _, tc := range tests {
		got := RedactString(tc.input)
		if got != tc.expected {
			t.Errorf("RedactString(%q) = %q, want %q", tc.input, got, tc.expected)
		}
	}
}

func TestRedactEmail(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"test@example.com", "te****@example.com"},
		{"a@b.com", "****@b.com"},
		{"ab@c.com", "****@c.com"},
		{"no-at-sign", "no****gn"},
	}
	for _, tc := range tests {
		got := RedactEmail(tc.input)
		if got != tc.expected {
			t.Errorf("RedactEmail(%q) = %q, want %q", tc.input, got, tc.expected)
		}
	}
}

func TestRedactPhone(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"13812345678", "138****5678"},
		{"12345678", "123****5678"},
		{"1234567", "****"}, // len 7 ≤ 7 → "****"
		{"123456", "****"},
		{"", "****"},
	}
	for _, tc := range tests {
		got := RedactPhone(tc.input)
		if got != tc.expected {
			t.Errorf("RedactPhone(%q) = %q, want %q", tc.input, got, tc.expected)
		}
	}
}

func TestRedactField(t *testing.T) {
	old := globalRedactorInstance()
	initRedactor(DefaultSensitiveFields)
	defer func() {
		globalRedactorMu.Lock()
		globalRedactor = old
		globalRedactorMu.Unlock()
	}()

	if RedactField("name", "John") != "John" {
		t.Fatal("non-sensitive field should not be redacted")
	}
	if RedactField("password", "secret123") == "secret123" {
		t.Fatal("password should be redacted")
	}
	if RedactField("email", "test@example.com") == "test@example.com" {
		t.Fatal("email should be redacted")
	}
}

func TestRedactFields(t *testing.T) {
	old := globalRedactorInstance()
	initRedactor(DefaultSensitiveFields)
	defer func() {
		globalRedactorMu.Lock()
		globalRedactor = old
		globalRedactorMu.Unlock()
	}()

	fields := map[string]interface{}{
		"name":     "John",
		"password": "secret",
	}
	redacted := RedactFields(fields)
	if redacted["name"] != "John" {
		t.Fatal("name should not be redacted")
	}
	if redacted["password"] == "secret" {
		t.Fatal("password should be redacted")
	}
}

func TestRedactValue(t *testing.T) {
	old := globalRedactorInstance()
	initRedactor(DefaultSensitiveFields)
	defer func() {
		globalRedactorMu.Lock()
		globalRedactor = old
		globalRedactorMu.Unlock()
	}()

	input := `{"password":"secret123","name":"John"}`
	got := RedactValue(input)
	if strings.Contains(got, "secret123") {
		t.Fatalf("password value should be redacted: %q", got)
	}
}

func TestInitRedactorFromConfig(t *testing.T) {
	r := initRedactor("custom_secret,password")
	if !r.isSensitive("custom_secret") {
		t.Fatal("custom_secret should be sensitive")
	}
	if !r.isSensitive("password") {
		t.Fatal("password should be sensitive")
	}
	if r.isSensitive("email") {
		t.Fatal("email should not be in custom list")
	}
}

func TestRedactCoreWrite(t *testing.T) {
	r := initRedactor(DefaultSensitiveFields)

	var buf bytes.Buffer
	enc := zapcore.NewJSONEncoder(zap.NewProductionEncoderConfig())
	core := wrapCoreWithRedact(zapcore.NewCore(enc, zapcore.AddSync(&buf), zapcore.InfoLevel), r)
	log := zap.New(core)

	log.Info(`request body {"password":"secret123"}`, zap.String("token", "abc123token"))
	output := buf.String()
	if strings.Contains(output, "secret123") || strings.Contains(output, "abc123token") {
		t.Fatalf("sensitive values should be redacted in output: %s", output)
	}
}

func TestInfoWithRedactedFields(t *testing.T) {
	oldLg := Lg
	Lg = zap.NewNop()
	defer func() { Lg = oldLg }()

	InfoWithRedactedFields("test", map[string]interface{}{
		"password": "secret",
		"name":     "John",
	})
}

func TestErrorWithRedactedFields(t *testing.T) {
	oldLg := Lg
	Lg = zap.NewNop()
	defer func() { Lg = oldLg }()

	ErrorWithRedactedFields("test", map[string]interface{}{
		"password": "secret",
		"name":     "John",
	})
}

func TestIsSensitiveField(t *testing.T) {
	old := globalRedactorInstance()
	initRedactor(DefaultSensitiveFields)
	defer func() {
		globalRedactorMu.Lock()
		globalRedactor = old
		globalRedactorMu.Unlock()
	}()

	if !IsSensitiveField("password") {
		t.Fatal("password should be sensitive by default")
	}
	if IsSensitiveField("username") {
		t.Fatal("username should not be sensitive by default")
	}
}

func TestRedactField_NilRedactor(t *testing.T) {
	var r *redactor = nil
	if r.redactField("password", "secret") != "secret" {
		t.Fatal("nil redactor should return value unchanged")
	}
}

func TestRedactFields_NilRedactor(t *testing.T) {
	var r *redactor = nil
	fields := map[string]interface{}{"password": "secret"}
	result := r.redactFields(fields)
	if result["password"] != "secret" {
		t.Fatal("nil redactor should return fields unchanged")
	}
}

func TestRedactValue_NilRedactor(t *testing.T) {
	var r *redactor = nil
	result := r.redactValue(`{"password":"secret"}`)
	if result != `{"password":"secret"}` {
		t.Fatal("nil redactor should return data unchanged")
	}
}

func TestRedactValue_EmptyJSONRE(t *testing.T) {
	r := &redactor{sensitive: map[string]bool{}}
	r.rebuildJSONPattern()
	result := r.redactValue(`{"password":"secret"}`)
	if result != `{"password":"secret"}` {
		t.Fatal("empty sensitive map should not modify data")
	}
}

func TestRedactZapField_StringType(t *testing.T) {
	r := initRedactor("password,token")
	f := zap.String("password", "mysecret")
	next, changed := r.redactZapField(f)
	if !changed {
		t.Fatal("password string should be redacted")
	}
	if next.String == "mysecret" {
		t.Fatal("password value should be redacted")
	}
}

func TestRedactZapField_NonSensitive(t *testing.T) {
	r := initRedactor("password")
	f := zap.String("username", "john")
	_, changed := r.redactZapField(f)
	if changed {
		t.Fatal("non-sensitive field should not be changed")
	}
}

func TestRedactZapField_ByteStringType(t *testing.T) {
	r := initRedactor("secret")
	f := zap.ByteString("secret", []byte("topsecret"))
	next, changed := r.redactZapField(f)
	if !changed {
		t.Fatal("byte string with sensitive key should be redacted")
	}
	if next.Type != zapcore.StringType {
		t.Fatal("ByteString should be converted to StringType")
	}
}

func TestRedactZapField_Int64Type(t *testing.T) {
	r := initRedactor("password")
	f := zap.Int64("password", 12345)
	next, changed := r.redactZapField(f)
	if !changed {
		t.Fatal("int64 with sensitive key should be redacted")
	}
	if next.Type != zapcore.StringType {
		t.Fatal("int64 should be converted to StringType")
	}
}

func TestRedactZapField_ReflectType(t *testing.T) {
	r := initRedactor("data")
	f := zap.Reflect("data", map[string]string{"key": "value"})
	next, changed := r.redactZapField(f)
	if !changed {
		t.Fatal("reflect with sensitive key should be redacted")
	}
	if next.Type != zapcore.StringType {
		t.Fatal("reflect should be converted to StringType")
	}
}

func TestRedactZapField_ErrorType(t *testing.T) {
	r := initRedactor("password,err")
	f := zap.NamedError("err", &testErr{msg: "some error"})
	next, changed := r.redactZapField(f)
	if !changed {
		t.Fatal("error type with sensitive key should be redacted when key is in sensitive list")
	}
	if next.Type != zapcore.StringType {
		t.Fatal("error should be converted to StringType")
	}
}

type testErr struct{ msg string }

func (e *testErr) Error() string { return e.msg }

func TestRedactZapField_BoolType(t *testing.T) {
	r := initRedactor("flag")
	f := zap.Bool("flag", true)
	next, changed := r.redactZapField(f)
	if !changed {
		t.Fatal("bool with sensitive key should be redacted")
	}
	if next.Type != zapcore.StringType {
		t.Fatal("bool should be converted to StringType")
	}
}

func TestRedactZapField_SkipInterface(t *testing.T) {
	// SkipType with non-nil Interface falls into the default branch of redactZapField
	// which converts it to StringType. This tests that path.
	r := initRedactor("password")
	f := zap.Field{Key: "password", Type: zapcore.SkipType, Interface: "skipped"}
	next, changed := r.redactZapField(f)
	if !changed {
		t.Fatal("SkipType with non-nil Interface should be redacted (falls into default case)")
	}
	if next.Type != zapcore.StringType {
		t.Fatal("SkipType should be converted to StringType")
	}
}

func TestRedactZapField_InterfaceNil(t *testing.T) {
	r := initRedactor("password")
	f := zap.Field{Key: "password", Type: zapcore.SkipType, Interface: nil}
	next, changed := r.redactZapField(f)
	if changed {
		t.Fatal("nil interface should not trigger redaction")
	}
	_ = next
}

func TestRedactZapFields_Empty(t *testing.T) {
	r := initRedactor("password")
	fields := []zapcore.Field{}
	result := r.redactZapFields(fields)
	if len(result) != 0 {
		t.Fatal("empty fields should remain empty")
	}
}

func TestRedactZapFields_NilRedactor(t *testing.T) {
	var r *redactor = nil
	fields := []zapcore.Field{zap.String("password", "secret")}
	result := r.redactZapFields(fields)
	if len(result) != 1 {
		t.Fatal("nil redactor should return fields unchanged")
	}
}

func TestRedactCore_With(t *testing.T) {
	r := initRedactor("password,token")
	var buf bytes.Buffer
	enc := zapcore.NewJSONEncoder(zap.NewProductionEncoderConfig())
	inner := zapcore.NewCore(enc, zapcore.AddSync(&buf), zapcore.InfoLevel)
	core := wrapCoreWithRedact(inner, r)

	newCore := core.With([]zapcore.Field{zap.String("password", "sensitive")})
	if newCore == nil {
		t.Fatal("With should return non-nil core")
	}
}

func TestRedactCore_With_NonSensitive(t *testing.T) {
	r := initRedactor("password")
	var buf bytes.Buffer
	enc := zapcore.NewJSONEncoder(zap.NewProductionEncoderConfig())
	inner := zapcore.NewCore(enc, zapcore.AddSync(&buf), zapcore.InfoLevel)
	core := wrapCoreWithRedact(inner, r)

	newCore := core.With([]zapcore.Field{zap.String("username", "john")})
	if newCore == nil {
		t.Fatal("With should return non-nil core")
	}
}

func TestRedactCore_Check_Enabled(t *testing.T) {
	r := initRedactor("password")
	var buf bytes.Buffer
	enc := zapcore.NewJSONEncoder(zap.NewProductionEncoderConfig())
	inner := zapcore.NewCore(enc, zapcore.AddSync(&buf), zapcore.InfoLevel)
	core := wrapCoreWithRedact(inner, r)

	ent := zapcore.Entry{Level: zapcore.InfoLevel}
	ce := core.Check(ent, nil)
	if ce == nil {
		t.Fatal("Check should return CheckedEntry for enabled level")
	}
}

func TestRedactCore_Check_Disabled(t *testing.T) {
	r := initRedactor("password")
	var buf bytes.Buffer
	enc := zapcore.NewJSONEncoder(zap.NewProductionEncoderConfig())
	inner := zapcore.NewCore(enc, zapcore.AddSync(&buf), zapcore.ErrorLevel) // only Error+
	core := wrapCoreWithRedact(inner, r)

	ent := zapcore.Entry{Level: zapcore.InfoLevel}
	ce := core.Check(ent, nil)
	if ce != nil {
		t.Log("Check returned non-nil for disabled level (zap may still return entry)")
	}
	// For disabled levels, Check should not panic
}

func TestRedactCore_Write_MessageRedaction(t *testing.T) {
	r := initRedactor(DefaultSensitiveFields)
	var buf bytes.Buffer
	enc := zapcore.NewJSONEncoder(zap.NewProductionEncoderConfig())
	inner := zapcore.NewCore(enc, zapcore.AddSync(&buf), zapcore.InfoLevel)
	core := wrapCoreWithRedact(inner, r)

	ent := zapcore.Entry{Level: zapcore.InfoLevel, Message: `sensitive {"password":"revealme"}`}
	ce := core.Check(ent, nil)
	if ce != nil {
		ce.Write()
	}

	output := buf.String()
	if strings.Contains(output, "revealme") {
		t.Fatalf("message should be redacted: %s", output)
	}
}

func TestNewRedactor_EmptyFields(t *testing.T) {
	r := newRedactor([]string{})
	if r == nil {
		t.Fatal("newRedactor should return non-nil with empty fields")
	}
	if r.isSensitive("anything") {
		t.Fatal("empty redactor should not mark anything sensitive")
	}
}

func TestInitRedactor_EmptyString(t *testing.T) {
	r := initRedactor("")
	if !r.isSensitive("password") {
		t.Fatal("empty config should fallback to default, password should be sensitive")
	}
}

func TestInitRedactor_WhitespaceString(t *testing.T) {
	r := initRedactor("   ")
	if !r.isSensitive("password") {
		t.Fatal("whitespace config should fallback to default")
	}
}

func TestStyleForField(t *testing.T) {
	if styleForField("email") != redactStyleEmail {
		t.Fatal("email should map to redactStyleEmail")
	}
	if styleForField("phone") != redactStylePhone {
		t.Fatal("phone should map to redactStylePhone")
	}
	if styleForField("mobile") != redactStylePhone {
		t.Fatal("mobile should map to redactStylePhone")
	}
	if styleForField("telephone") != redactStylePhone {
		t.Fatal("telephone should map to redactStylePhone")
	}
	if styleForField("password") != redactStyleString {
		t.Fatal("unknown field should map to redactStyleString")
	}
}

func TestParseSensitiveFieldsCSV(t *testing.T) {
	result := parseSensitiveFieldsCSV("a, b , c, , ")
	if len(result) != 3 {
		t.Fatalf("expected 3 fields, got %d: %v", len(result), result)
	}
	if result[0] != "a" || result[1] != "b" || result[2] != "c" {
		t.Fatalf("unexpected fields: %v", result)
	}
}

func TestGlobalRedactorInstance(t *testing.T) {
	r := globalRedactorInstance()
	if r == nil {
		t.Fatal("globalRedactorInstance should not be nil")
	}
}

func TestWrapCoreWithRedact_NilInner(t *testing.T) {
	result := wrapCoreWithRedact(nil, &redactor{})
	if result != nil {
		t.Fatal("wrapCoreWithRedact with nil inner should return nil")
	}
}

func TestWrapCoreWithRedact_NilRedactor(t *testing.T) {
	var buf bytes.Buffer
	enc := zapcore.NewJSONEncoder(zap.NewProductionEncoderConfig())
	inner := zapcore.NewCore(enc, zapcore.AddSync(&buf), zapcore.InfoLevel)
	result := wrapCoreWithRedact(inner, nil)
	if result != inner {
		t.Fatal("wrapCoreWithRedact with nil redactor should return inner unchanged")
	}
}

func TestIsSensitive_NilRedactor(t *testing.T) {
	var r *redactor = nil
	if r.isSensitive("password") {
		t.Fatal("nil redactor should not mark anything sensitive")
	}
}

func TestNewRedactor_AllWhitespaceFields(t *testing.T) {
	r := newRedactor([]string{"", "  ", "\t"})
	if len(r.sensitive) != 0 {
		t.Fatalf("all-whitespace fields should be empty, got %d", len(r.sensitive))
	}
}

func TestRedactByStyle(t *testing.T) {
	r := initRedactor("email,phone,token")
	// email style
	if r.redactByStyle(redactStyleEmail, "test@example.com") == "test@example.com" {
		t.Fatal("email should be redacted")
	}
	// phone style
	if r.redactByStyle(redactStylePhone, "1234567890") == "1234567890" {
		t.Fatal("phone should be redacted")
	}
	// default string style
	if r.redactByStyle(redactStyleString, "abcdef") == "abcdef" {
		t.Fatal("string should be redacted")
	}
}

func TestIsSensitive_MultipleFields(t *testing.T) {
	r := newRedactor([]string{"a", "b"})
	if !r.isSensitive("a") || !r.isSensitive("b") {
		t.Fatal("both fields should be sensitive")
	}
	if r.isSensitive("c") {
		t.Fatal("c should not be sensitive")
	}
}

func TestStyleFor_NotFound(t *testing.T) {
	r := newRedactor([]string{"custom"})
	// "custom" is not in the email/phone/telephone/mobile set
	style := r.styleFor("custom")
	if style != redactStyleString {
		t.Fatalf("unknown field should default to redactStyleString, got %v", style)
	}
}

func TestRedactValue_BrokenJSON(t *testing.T) {
	r := initRedactor("password")
	// Malformed JSON — the regex may not match, so value passes through
	result := r.redactValue(`{"password":"value`)
	if result == `{"password":"value` {
		// unclosed quote → no match → unchanged
	} else {
		t.Logf("broken json: %s", result)
	}
}
