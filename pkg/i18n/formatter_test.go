package i18n

import (
	"testing"
	"time"
)

func TestFormatter_FormatNumber(t *testing.T) {
	formatter := NewFormatter("en")

	result := formatter.FormatNumber(1234.56, 2)
	if result != "1,234.56" {
		t.Errorf("expected '1,234.56', got '%s'", result)
	}

	formatterCN := NewFormatter("zh-CN")
	resultCN := formatterCN.FormatNumber(1234.56, 2)
	if resultCN != "1,234.56" {
		t.Errorf("expected '1,234.56', got '%s'", resultCN)
	}
}

func TestFormatter_FormatCurrency(t *testing.T) {
	formatter := NewFormatter("en")
	result := formatter.FormatCurrency(1234.56, "USD")
	if result != "$1,234.56" {
		t.Errorf("expected '$1,234.56', got '%s'", result)
	}

	formatterCN := NewFormatter("zh-CN")
	resultCN := formatterCN.FormatCurrency(1234.56, "CNY")
	if resultCN != "¥1,234.56" {
		t.Errorf("expected '¥1,234.56', got '%s'", resultCN)
	}
}

func TestFormatter_FormatDate(t *testing.T) {
	date := time.Date(2024, 1, 15, 14, 30, 0, 0, time.UTC)

	formatter := NewFormatter("en")
	result := formatter.FormatDate(date, "YYYY-MM-DD")
	if result != "2024-01-15" {
		t.Errorf("expected '2024-01-15', got '%s'", result)
	}

	formatterCN := NewFormatter("zh-CN")
	resultCN := formatterCN.FormatDate(date, "")
	if resultCN != "2024-01-15" {
		t.Errorf("expected '2024-01-15', got '%s'", resultCN)
	}
}

func TestFormatter_FormatRelativeTime(t *testing.T) {
	now := time.Now()

	formatter := NewFormatter("en")

	// Test seconds ago
	past := now.Add(-30 * time.Second)
	result := formatter.FormatRelativeTime(past)
	if result == "" {
		t.Error("expected non-empty result")
	}

	// Test minutes ago
	past = now.Add(-5 * time.Minute)
	result = formatter.FormatRelativeTime(past)
	if result == "" {
		t.Error("expected non-empty result")
	}

	// Test hours ago
	past = now.Add(-2 * time.Hour)
	result = formatter.FormatRelativeTime(past)
	if result == "" {
		t.Error("expected non-empty result")
	}

	// Test days ago
	past = now.Add(-3 * 24 * time.Hour)
	result = formatter.FormatRelativeTime(past)
	if result == "" {
		t.Error("expected non-empty result")
	}

	// Test Chinese locale
	formatterCN := NewFormatter("zh-CN")
	past = now.Add(-1 * time.Hour)
	resultCN := formatterCN.FormatRelativeTime(past)
	if resultCN == "" {
		t.Error("expected non-empty result")
	}
}

func TestFormatter_GetNumberFormat(t *testing.T) {
	formatterEN := NewFormatter("en")
	format := formatterEN.getNumberFormat()
	if format.DecimalSeparator != "." {
		t.Error("expected . as decimal separator for en")
	}

	formatterDE := NewFormatter("de")
	formatDE := formatterDE.getNumberFormat()
	if formatDE.DecimalSeparator != "," {
		t.Error("expected , as decimal separator for de")
	}
}

func TestFormatter_GetCurrencyFormat(t *testing.T) {
	formatterEN := NewFormatter("en")
	format := formatterEN.getCurrencyFormat()
	if format.Symbol != "$" {
		t.Errorf("expected $, got %s", format.Symbol)
	}

	formatterCN := NewFormatter("zh-CN")
	formatCN := formatterCN.getCurrencyFormat()
	if formatCN.Symbol != "¥" {
		t.Errorf("expected ¥, got %s", formatCN.Symbol)
	}
}

func TestFormatter_GetDateFormat(t *testing.T) {
	formatterEN := NewFormatter("en-US")
	format := formatterEN.getDateFormat()
	if format == "" {
		t.Error("expected non-empty date format")
	}

	formatterCN := NewFormatter("zh-CN")
	formatCN := formatterCN.getDateFormat()
	if formatCN != "YYYY-MM-DD" {
		t.Errorf("expected YYYY-MM-DD, got %s", formatCN)
	}
}

func TestFormatter_AddThousandSeparators(t *testing.T) {
	formatter := NewFormatter("en")

	result := formatter.addThousandSeparators("1234567", ",", ".")
	if result != "1,234,567" {
		t.Errorf("expected '1,234,567', got '%s'", result)
	}

	result = formatter.addThousandSeparators("1234.56", ",", ".")
	if result != "1,234.56" {
		t.Errorf("expected '1,234.56', got '%s'", result)
	}
}
