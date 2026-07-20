package validate

import (
	"fmt"
	"net/mail"
	"regexp"
	"strconv"
	"strings"

	apperr "github.com/LingByte/SoulNexus/pkg/errors"
)

var (
	mobileRegex = regexp.MustCompile(`^1[3-9]\d{9}$`)
	slugRegex   = regexp.MustCompile(`^[a-z0-9\-_]{3,64}$`)
	domainRegex = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9\-.]{1,61}[a-zA-Z0-9]\.[a-zA-Z]{2,}$`)
)

func IsEmail(email string) bool {
	_, err := mail.ParseAddress(strings.TrimSpace(email))
	return err == nil
}

func IsMobile(mobile string) bool {
	return mobileRegex.MatchString(mobile)
}

func IsDomain(domain string) bool {
	return domainRegex.MatchString(domain)
}

func IsSlug(slug string) bool {
	return slugRegex.MatchString(slug)
}

func IsEmpty(s string) bool {
	return strings.TrimSpace(s) == ""
}

func Trim(s string) string {
	return strings.TrimSpace(s)
}

func TrimAll(s string) string {
	return strings.ReplaceAll(strings.TrimSpace(s), " ", "")
}

func TrimLower(s string) string {
	return strings.ToLower(strings.TrimSpace(s))
}

func DefaultStr(s string, def string) string {
	if IsEmpty(s) {
		return def
	}
	return s
}

// NormalizePage clamps page/size into valid ranges.
// maxSize <= 0 defaults to 100.
func NormalizePage(page, size, maxSize int) (int, int) {
	if maxSize <= 0 {
		maxSize = 100
	}
	if page < 1 {
		page = 1
	}
	if size < 1 {
		size = 20
	}
	if size > maxSize {
		size = maxSize
	}
	return page, size
}

// ParseID parses a string into a uint ID. Returns error if empty, non-numeric, or zero.
func ParseID(s string) (uint, error) {
	v, err := strconv.ParseUint(strings.TrimSpace(s), 10, 64)
	if err != nil || v == 0 {
		return 0, fmt.Errorf("%w: %q", apperr.ErrInvalidPrimaryKey, s)
	}
	return uint(v), nil
}

// ParseOptionalID parses raw into uint; blank or invalid returns 0.
func ParseOptionalID(raw string) uint {
	id, err := ParseID(raw)
	if err != nil {
		return 0
	}
	return id
}

// RequireScopeID rejects blank and "0" before ParseID.
func RequireScopeID(raw, field string) (uint, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" || raw == "0" {
		return 0, fmt.Errorf("%w: %s", apperr.ErrMissingParams, field)
	}
	id, err := ParseID(raw)
	if err != nil {
		return 0, fmt.Errorf("%w: %s", apperr.ErrInvalidPrimaryKey, field)
	}
	return id, nil
}

// ParseIDStrings parses each element with ParseID.
func ParseIDStrings(raw []string) ([]uint, error) {
	out := make([]uint, 0, len(raw))
	for _, s := range raw {
		id, err := ParseID(s)
		if err != nil {
			return nil, err
		}
		out = append(out, id)
	}
	return out, nil
}

// ParseNonZeroIDStrings skips blank/"0" entries; empty input returns (nil, nil).
func ParseNonZeroIDStrings(raw []string, fieldName string) ([]uint, error) {
	if len(raw) == 0 {
		return nil, nil
	}
	out := make([]uint, 0, len(raw))
	for _, s := range raw {
		s = strings.TrimSpace(s)
		if s == "" || s == "0" {
			continue
		}
		n, err := strconv.ParseUint(s, 10, 64)
		if err != nil || n == 0 {
			return nil, fmt.Errorf("%w: %s", apperr.ErrInvalidPrimaryKey, s)
		}
		out = append(out, uint(n))
	}
	return out, nil
}

// FormatID returns the decimal string form of an ID.
func FormatID(id uint) string {
	return strconv.FormatUint(uint64(id), 10)
}

// FormatIDs returns decimal string forms for each ID.
func FormatIDs(ids []uint) []string {
	if len(ids) == 0 {
		return []string{}
	}
	out := make([]string, len(ids))
	for i, id := range ids {
		out[i] = FormatID(id)
	}
	return out
}

// ValidPassword checks minimum password length.
func ValidPassword(pw string, minLen int) bool {
	if minLen <= 0 {
		minLen = 8
	}
	return len(strings.TrimSpace(pw)) >= minLen
}
