package i18n

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

// Formatter handles locale-specific formatting
type Formatter struct {
	locale Locale
}

// NewFormatter creates a new formatter for a locale
func NewFormatter(locale Locale) *Formatter {
	return &Formatter{locale: locale}
}

// FormatNumber formats a number according to locale
func (f *Formatter) FormatNumber(number float64, decimals int) string {
	// Get locale-specific number format
	format := f.getNumberFormat()

	// Format number
	formatted := fmt.Sprintf("%."+strconv.Itoa(decimals)+"f", number)

	// Apply locale-specific formatting
	if format.DecimalSeparator != "." {
		parts := strings.Split(formatted, ".")
		if len(parts) == 2 {
			formatted = parts[0] + format.DecimalSeparator + parts[1]
		}
	}

	// Add thousand separators
	if format.ThousandSeparator != "" {
		formatted = f.addThousandSeparators(formatted, format.ThousandSeparator, format.DecimalSeparator)
	}

	return formatted
}

// FormatCurrency formats currency according to locale
func (f *Formatter) FormatCurrency(amount float64, currency string) string {
	format := f.getCurrencyFormat()

	formattedAmount := f.FormatNumber(amount, 2)

	// Apply currency format
	if format.SymbolPosition == "before" {
		return format.Symbol + formattedAmount
	} else if format.SymbolPosition == "after" {
		return formattedAmount + " " + format.Symbol
	}

	return formattedAmount
}

// FormatDate formats date according to locale
func (f *Formatter) FormatDate(date time.Time, format string) string {
	if format == "" {
		format = f.getDateFormat()
	}

	// Replace format tokens with locale-specific values
	formatted := format
	formatted = strings.ReplaceAll(formatted, "YYYY", fmt.Sprintf("%04d", date.Year()))
	formatted = strings.ReplaceAll(formatted, "MM", fmt.Sprintf("%02d", int(date.Month())))
	formatted = strings.ReplaceAll(formatted, "DD", fmt.Sprintf("%02d", date.Day()))
	formatted = strings.ReplaceAll(formatted, "HH", fmt.Sprintf("%02d", date.Hour()))
	formatted = strings.ReplaceAll(formatted, "mm", fmt.Sprintf("%02d", date.Minute()))
	formatted = strings.ReplaceAll(formatted, "ss", fmt.Sprintf("%02d", date.Second()))

	return formatted
}

// FormatRelativeTime formats relative time (e.g., "2 hours ago")
func (f *Formatter) FormatRelativeTime(t time.Time) string {
	now := time.Now()
	diff := now.Sub(t)

	// Get locale-specific time units
	units := f.getTimeUnits()

	if diff < time.Minute {
		seconds := int(diff.Seconds())
		if seconds <= 0 {
			return units.JustNow
		}
		return fmt.Sprintf("%d %s", seconds, units.Seconds)
	} else if diff < time.Hour {
		minutes := int(diff.Minutes())
		return fmt.Sprintf("%d %s", minutes, units.Minutes)
	} else if diff < 24*time.Hour {
		hours := int(diff.Hours())
		return fmt.Sprintf("%d %s", hours, units.Hours)
	} else if diff < 30*24*time.Hour {
		days := int(diff.Hours() / 24)
		return fmt.Sprintf("%d %s", days, units.Days)
	} else if diff < 365*24*time.Hour {
		months := int(diff.Hours() / (24 * 30))
		return fmt.Sprintf("%d %s", months, units.Months)
	} else {
		years := int(diff.Hours() / (24 * 365))
		return fmt.Sprintf("%d %s", years, units.Years)
	}
}

// NumberFormat represents number formatting rules
type NumberFormat struct {
	DecimalSeparator  string
	ThousandSeparator string
}

// CurrencyFormat represents currency formatting rules
type CurrencyFormat struct {
	Symbol         string
	SymbolPosition string // "before" or "after"
}

// TimeUnits represents time unit translations
type TimeUnits struct {
	JustNow string
	Seconds string
	Minutes string
	Hours   string
	Days    string
	Months  string
	Years   string
}

func (f *Formatter) getNumberFormat() NumberFormat {
	switch f.locale {
	case "zh-CN", "zh-TW":
		return NumberFormat{
			DecimalSeparator:  ".",
			ThousandSeparator: ",",
		}
	case "de", "fr", "es", "it", "pt":
		return NumberFormat{
			DecimalSeparator:  ",",
			ThousandSeparator: ".",
		}
	default:
		return NumberFormat{
			DecimalSeparator:  ".",
			ThousandSeparator: ",",
		}
	}
}

func (f *Formatter) getCurrencyFormat() CurrencyFormat {
	switch f.locale {
	case "zh-CN":
		return CurrencyFormat{
			Symbol:         "¥",
			SymbolPosition: "before",
		}
	case "zh-TW":
		return CurrencyFormat{
			Symbol:         "NT$",
			SymbolPosition: "before",
		}
	case "en", "en-US", "en-GB":
		return CurrencyFormat{
			Symbol:         "$",
			SymbolPosition: "before",
		}
	case "ja":
		return CurrencyFormat{
			Symbol:         "¥",
			SymbolPosition: "before",
		}
	case "ko":
		return CurrencyFormat{
			Symbol:         "₩",
			SymbolPosition: "before",
		}
	case "ru":
		return CurrencyFormat{
			Symbol:         "₽",
			SymbolPosition: "after",
		}
	case "eu", "de", "fr", "es", "it", "pt":
		return CurrencyFormat{
			Symbol:         "€",
			SymbolPosition: "after",
		}
	case "gb", "uk":
		return CurrencyFormat{
			Symbol:         "£",
			SymbolPosition: "before",
		}
	default:
		return CurrencyFormat{
			Symbol:         "$",
			SymbolPosition: "before",
		}
	}
}

func (f *Formatter) getDateFormat() string {
	switch f.locale {
	case "zh-CN", "zh-TW":
		return "YYYY-MM-DD"
	case "en-US":
		return "MM/DD/YYYY"
	case "en-GB", "en":
		return "DD/MM/YYYY"
	case "de", "fr", "es", "it", "pt":
		return "DD.MM.YYYY"
	case "ja":
		return "YYYY年MM月DD日"
	default:
		return "YYYY-MM-DD"
	}
}

func (f *Formatter) getTimeUnits() TimeUnits {
	switch f.locale {
	case "zh-CN":
		return TimeUnits{
			JustNow: "刚刚",
			Seconds: "秒前",
			Minutes: "分钟前",
			Hours:   "小时前",
			Days:    "天前",
			Months:  "个月前",
			Years:   "年前",
		}
	case "zh-TW":
		return TimeUnits{
			JustNow: "剛剛",
			Seconds: "秒前",
			Minutes: "分鐘前",
			Hours:   "小時前",
			Days:    "天前",
			Months:  "個月前",
			Years:   "年前",
		}
	case "ja":
		return TimeUnits{
			JustNow: "たった今",
			Seconds: "秒前",
			Minutes: "分前",
			Hours:   "時間前",
			Days:    "日前",
			Months:  "ヶ月前",
			Years:   "年前",
		}
	case "ko":
		return TimeUnits{
			JustNow: "방금",
			Seconds: "초 전",
			Minutes: "분 전",
			Hours:   "시간 전",
			Days:    "일 전",
			Months:  "개월 전",
			Years:   "년 전",
		}
	default:
		return TimeUnits{
			JustNow: "just now",
			Seconds: "seconds ago",
			Minutes: "minutes ago",
			Hours:   "hours ago",
			Days:    "days ago",
			Months:  "months ago",
			Years:   "years ago",
		}
	}
}

func (f *Formatter) addThousandSeparators(number, separator, decimalSep string) string {
	parts := strings.Split(number, decimalSep)
	integerPart := parts[0]

	// Add thousand separators from right to left
	result := ""
	for i, digit := range integerPart {
		if i > 0 && (len(integerPart)-i)%3 == 0 {
			result += separator
		}
		result += string(digit)
	}

	if len(parts) > 1 {
		result += decimalSep + parts[1]
	}

	return result
}
