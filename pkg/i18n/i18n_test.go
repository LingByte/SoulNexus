package i18n

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestT_enUS(t *testing.T) {
	if got := T(LocaleEnUS, KeyAuthInvalidCredentials); got != "Invalid email or password" {
		t.Fatalf("got %q", got)
	}
}

func TestT_zhTW(t *testing.T) {
	if got := T(LocaleZhTW, KeyAuthInvalidCredentials); got != "郵箱或密碼錯誤" {
		t.Fatalf("got %q", got)
	}
}

func TestT_jaJP(t *testing.T) {
	if got := T(LocaleJaJP, KeyAuthInvalidCredentials); got != "メールアドレスまたはパスワードが間違っています" {
		t.Fatalf("got %q", got)
	}
}

func TestTGin_acceptLanguage(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/", nil)
	c.Request.Header.Set("Accept-Language", "en-US,en;q=0.9")
	SetLocaleOnGin(c, ParseAcceptLanguage(c.Request.Header.Get("Accept-Language")))
	if got := TGin(c, KeyTenantRegisterDisabled); got == "" || got == KeyTenantRegisterDisabled {
		t.Fatalf("unexpected %q", got)
	}
	if got := TGin(c, KeyTenantRegisterDisabled); got != enUS[KeyTenantRegisterDisabled] {
		t.Fatalf("got %q", got)
	}
}

func TestResolveLocale(t *testing.T) {
	tests := []struct{ raw, want string }{
		{"", LocaleZhCN},
		{"", LocaleZhCN},
		{"zh-CN", LocaleZhCN},
		{"zh-TW", LocaleZhTW},
		{"en-US", LocaleEnUS},
		{"ja-JP", LocaleJaJP},
		{"zh", LocaleZhCN},    // unknown → default
		{"en", LocaleZhCN},    // unknown → default
		{"ja", LocaleZhCN},    // unknown → default
		{"fr", LocaleZhCN},    // unknown → default
	}
	for _, tt := range tests {
		if got := ResolveLocale(tt.raw); got != tt.want {
			t.Errorf("ResolveLocale(%q) = %q, want %q", tt.raw, got, tt.want)
		}
	}
}
