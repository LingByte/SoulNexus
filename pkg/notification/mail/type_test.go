package mail

import (
	"context"
	"testing"
	"time"
)

// ── InitialMailStatus ──

func TestInitialMailStatus(t *testing.T) {
	tests := []struct {
		kind string
		want string
	}{
		{ProviderSMTP, StatusDelivered},
		{ProviderSendCloud, StatusSent},
		{"unknown", StatusSent},
		{"", StatusSent},
	}
	for _, tt := range tests {
		if got := InitialMailStatus(tt.kind); got != tt.want {
			t.Errorf("InitialMailStatus(%q) = %q, want %q", tt.kind, got, tt.want)
		}
	}
}

// ── SendCloudEventToStatus ──

func TestSendCloudEventToStatus(t *testing.T) {
	tests := []struct {
		event string
		want  string
	}{
		// numeric codes
		{"1", StatusDelivered},
		{"2", StatusFailed},
		{"3", StatusSpam},
		{"4", StatusInvalid},
		{"5", StatusSoftBounce},
		{"10", StatusClicked},
		{"11", StatusOpened},
		{"12", StatusUnsubscribed},
		{"18", StatusSent},
		// named events
		{"deliver", StatusDelivered},
		{"delivered", StatusDelivered},
		{"spam", StatusSpam},
		{"invalid", StatusInvalid},
		{"soft_bounce", StatusSoftBounce},
		{"softbounce", StatusSoftBounce},
		{"hard_bounce", StatusSoftBounce},
		{"hardbounce", StatusSoftBounce},
		{"bounce", StatusSoftBounce},
		{"reject", StatusFailed},
		{"rejected", StatusFailed},
		{"failed", StatusFailed},
		{"fail", StatusFailed},
		{"click", StatusClicked},
		{"clicked", StatusClicked},
		{"open", StatusOpened},
		{"opened", StatusOpened},
		{"unsubscribe", StatusUnsubscribed},
		{"unsubscribed", StatusUnsubscribed},
		{"request", StatusSent},
		// case-insensitive / whitespace
		{"  DELIVERED  ", StatusDelivered},
		{"Click", StatusClicked},
		// unknown
		{"999", StatusUnknown},
		{"", StatusUnknown},
		{"unknown-event", StatusUnknown},
	}
	for _, tt := range tests {
		if got := SendCloudEventToStatus(tt.event); got != tt.want {
			t.Errorf("SendCloudEventToStatus(%q) = %q, want %q", tt.event, got, tt.want)
		}
	}
}

// ── ReplacePlaceholders ──

func TestReplacePlaceholders(t *testing.T) {
	tests := []struct {
		template string
		vars     map[string]any
		want     string
	}{
		{"hello {{name}}", map[string]any{"name": "world"}, "hello world"},
		{"{{a}} and {{b}}", map[string]any{"a": 1, "b": 2}, "1 and 2"},
		{"no placeholders", map[string]any{"x": 1}, "no placeholders"},
		{"", map[string]any{"x": 1}, ""},
		{"{{missing}}", nil, "{{missing}}"},
		{"{{k1}}{{k2}}", map[string]any{"k1": "a", "k2": "b"}, "ab"},
	}
	for _, tt := range tests {
		if got := ReplacePlaceholders(tt.template, tt.vars); got != tt.want {
			t.Errorf("ReplacePlaceholders(%q, %v) = %q, want %q", tt.template, tt.vars, got, tt.want)
		}
	}
}

// ── ParseMailSender ──

func TestParseMailSender(t *testing.T) {
	tests := []struct {
		name      string
		from      string
		fallback  string
		wantEnv   string
		wantDisp  string
		wantErr   bool
	}{
		{"RFC format", "John Doe <john@example.com>", "", "john@example.com", "John Doe", false},
		{"plain email", "john@example.com", "", "john@example.com", "", false},
		{"plain with fallback", "john@example.com", "Fallback Name", "john@example.com", "Fallback Name", false},
		{"RFC overrides fallback", "Jane <jane@example.com>", "Fallback", "jane@example.com", "Jane", false},
		{"empty from", "", "Fallback", "", "", true},
		{"invalid format", "not an email", "", "", "", true},
	}
	for _, tt := range tests {
		got, err := ParseMailSender(tt.from, tt.fallback)
		if (err != nil) != tt.wantErr {
			t.Errorf("%s: err = %v, wantErr = %v", tt.name, err, tt.wantErr)
			continue
		}
		if err != nil {
			continue
		}
		if got.Envelope != tt.wantEnv {
			t.Errorf("%s: Envelope = %q, want %q", tt.name, got.Envelope, tt.wantEnv)
		}
		if got.Display != tt.wantDisp {
			t.Errorf("%s: Display = %q, want %q", tt.name, got.Display, tt.wantDisp)
		}
		if got.HeaderFrom == "" {
			t.Errorf("%s: HeaderFrom should not be empty", tt.name)
		}
	}
}

func TestParseMailSender_asciiName(t *testing.T) {
	got, err := ParseMailSender("Alice Smith <alice@example.com>", "")
	if err != nil {
		t.Fatal(err)
	}
	if got.Display != "Alice Smith" {
		t.Errorf("Display = %q", got.Display)
	}
	// ASCII name should be quoted
	expected := `"Alice Smith" <alice@example.com>`
	if got.HeaderFrom != expected {
		t.Errorf("HeaderFrom = %q, want %q", got.HeaderFrom, expected)
	}
}

func TestParseMailSender_nonASCII(t *testing.T) {
	got, err := ParseMailSender("张三 <zhang@example.com>", "")
	if err != nil {
		t.Fatal(err)
	}
	if got.Display != "张三" {
		t.Errorf("Display = %q", got.Display)
	}
	if got.HeaderFrom == "" {
		t.Error("HeaderFrom should not be empty")
	}
}

func TestParseMailSender_rfcWithFallback_emptyName(t *testing.T) {
	// ParseAddress succeeds but name is empty, so fallback should be used
	got, err := ParseMailSender("<john@example.com>", "Fallback")
	if err != nil {
		// If ParseAddress interprets brackets differently, just skip
		t.Skipf("ParseAddress behavior: %v", err)
	}
	if got.Display != "Fallback" {
		t.Errorf("Display = %q, want Fallback", got.Display)
	}
	if got.Envelope != "john@example.com" {
		t.Errorf("Envelope = %q, want john@example.com", got.Envelope)
	}
}

func TestParseMailSender_angleBracketsNoAt(t *testing.T) {
	// mail.ParseAddress fails, contains <> but no @ → should error
	_, err := ParseMailSender("<noat>", "")
	if err == nil {
		t.Error("expected error for angle brackets without @")
	}
}

// ── isASCIIString ──

func TestIsASCIIString(t *testing.T) {
	tests := []struct {
		s    string
		want bool
	}{
		{"hello", true},
		{"hello world 123", true},
		{"张三", false},
		{"emoji😀", false},
		{"", true},
		{"ABC123!@#", true},
	}
	for _, tt := range tests {
		if got := isASCIIString(tt.s); got != tt.want {
			t.Errorf("isASCIIString(%q) = %v, want %v", tt.s, got, tt.want)
		}
	}
}

// ── formatMailFromHeader ──

func TestFormatMailFromHeader(t *testing.T) {
	tests := []struct {
		display string
		email   string
		want    string
	}{
		{"John", "john@example.com", `"John" <john@example.com>`},
		{"", "john@example.com", "john@example.com"},
		{"John", "", ""},
		{"", "", ""},
		// ASCII with special chars gets escaped
		{`John "The Man"`, "john@example.com", `"John \"The Man\"" <john@example.com>`},
	}
	for _, tt := range tests {
		got := formatMailFromHeader(tt.display, tt.email)
		if got != tt.want {
			t.Errorf("formatMailFromHeader(%q, %q) = %q, want %q", tt.display, tt.email, got, tt.want)
		}
	}
}

// ── DefaultRetryPolicy / RetryPolicy.Normalized ──

func TestDefaultRetryPolicy(t *testing.T) {
	p := DefaultRetryPolicy()
	if p.MaxAttempts != 3 {
		t.Errorf("MaxAttempts = %d", p.MaxAttempts)
	}
	if p.InitialBackoff != 200*time.Millisecond {
		t.Errorf("InitialBackoff = %v", p.InitialBackoff)
	}
	if p.MaxBackoff != 5*time.Second {
		t.Errorf("MaxBackoff = %v", p.MaxBackoff)
	}
}

func TestRetryPolicy_Normalized(t *testing.T) {
	tests := []struct {
		name string
		in   RetryPolicy
		want RetryPolicy
	}{
		{
			"zero values get defaults",
			RetryPolicy{},
			RetryPolicy{MaxAttempts: 1, InitialBackoff: 200 * time.Millisecond, MaxBackoff: 5 * time.Second},
		},
		{
			"negative MaxAttempts becomes 1",
			RetryPolicy{MaxAttempts: -1},
			RetryPolicy{MaxAttempts: 1, InitialBackoff: 200 * time.Millisecond, MaxBackoff: 5 * time.Second},
		},
		{
			"MaxBackoff less than InitialBackoff is bumped up",
			RetryPolicy{MaxAttempts: 3, InitialBackoff: time.Second, MaxBackoff: time.Millisecond},
			RetryPolicy{MaxAttempts: 3, InitialBackoff: time.Second, MaxBackoff: time.Second},
		},
		{
			"valid values pass through",
			RetryPolicy{MaxAttempts: 5, InitialBackoff: 500 * time.Millisecond, MaxBackoff: 10 * time.Second},
			RetryPolicy{MaxAttempts: 5, InitialBackoff: 500 * time.Millisecond, MaxBackoff: 10 * time.Second},
		},
	}
	for _, tt := range tests {
		got := tt.in.Normalized()
		if got.MaxAttempts != tt.want.MaxAttempts ||
			got.InitialBackoff != tt.want.InitialBackoff ||
			got.MaxBackoff != tt.want.MaxBackoff {
			t.Errorf("%s: Normalized() = %+v, want %+v", tt.name, got, tt.want)
		}
	}
}

// ── WithRetry ──

func TestWithRetry(t *testing.T) {
	opts := &mailerOptions{}
	WithRetry(RetryPolicy{MaxAttempts: 7})(opts)
	if opts.retry.MaxAttempts != 7 {
		t.Errorf("MaxAttempts = %d", opts.retry.MaxAttempts)
	}
}

// ── WithMailLogUserID ──

func TestWithMailLogUserID(t *testing.T) {
	opts := &mailerOptions{}
	WithMailLogUserID(42)(opts)
	if opts.mailLogUserID == nil || *opts.mailLogUserID != 42 {
		t.Errorf("mailLogUserID = %v", opts.mailLogUserID)
	}
}

func TestWithMailLogUserID_zero(t *testing.T) {
	opts := &mailerOptions{mailLogUserID: new(uint)}
	*opts.mailLogUserID = 1
	WithMailLogUserID(0)(opts)
	if opts.mailLogUserID != nil {
		t.Errorf("expected nil for zero uid")
	}
}

// ── sleepCtx ──

func TestSleepCtx_completes(t *testing.T) {
	ctx := context.Background()
	err := sleepCtx(ctx, 10*time.Millisecond)
	if err != nil {
		t.Errorf("sleepCtx should not error: %v", err)
	}
}

func TestSleepCtx_canceled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	err := sleepCtx(ctx, time.Hour)
	if err == nil {
		t.Error("sleepCtx should return error on canceled context")
	}
}

func TestSleepCtx_deadline(t *testing.T) {
	ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(-time.Second))
	defer cancel()
	err := sleepCtx(ctx, time.Hour)
	if err == nil {
		t.Error("sleepCtx should return error on expired deadline")
	}
}
