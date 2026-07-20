package timeutil

// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

import (
	"strings"
	"time"
)

const queryDayLayout = "2006-01-02"

var queryTimeLayouts = []string{
	time.RFC3339,
	"2006-01-02 15:04:05",
	"2006-01-02T15:04:05",
	"2006-01-02T15:04:05Z07:00",
	queryDayLayout,
}

// ParseQueryTime parses common API date/time query values.
func ParseQueryTime(raw string) (time.Time, bool) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return time.Time{}, false
	}
	for _, layout := range queryTimeLayouts {
		if t, err := time.Parse(layout, raw); err == nil {
			return t, true
		}
	}
	return time.Time{}, false
}

// ParseOptionalQueryTimePtr parses raw into *time.Time; nil when blank or invalid.
func ParseOptionalQueryTimePtr(raw string) *time.Time {
	if t, ok := ParseQueryTime(raw); ok {
		return &t
	}
	return nil
}

// ParseQueryDay parses YYYY-MM-DD in the configured business timezone.
func ParseQueryDay(raw string) (time.Time, bool) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return time.Time{}, false
	}
	if len(raw) > 10 {
		raw = raw[:10]
	}
	t, err := time.ParseInLocation(queryDayLayout, raw, Location())
	return t, err == nil
}

// ParseOptionalQueryDayPtr parses YYYY-MM-DD; nil when blank or invalid.
func ParseOptionalQueryDayPtr(raw string) *time.Time {
	if t, ok := ParseQueryDay(raw); ok {
		return &t
	}
	return nil
}

// InclusiveEndOfDay returns 23:59:59.999999999 on the same calendar day.
func InclusiveEndOfDay(t time.Time) time.Time {
	loc := t.Location()
	if loc == nil {
		loc = Location()
	}
	return time.Date(t.Year(), t.Month(), t.Day(), 23, 59, 59, 999999999, loc)
}

// StartOfNextDay returns midnight at the start of the day after t.
func StartOfNextDay(t time.Time) time.Time {
	loc := t.Location()
	if loc == nil {
		loc = Location()
	}
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, loc).Add(24 * time.Hour)
}

// ParseOptionalQueryDayRange parses optional start/end day strings.
// end is inclusive through end-of-day when set.
func ParseOptionalQueryDayRange(startRaw, endRaw string) (start, end *time.Time) {
	start = ParseOptionalQueryDayPtr(startRaw)
	end = ParseOptionalQueryDayPtr(endRaw)
	if end != nil {
		e := InclusiveEndOfDay(*end)
		end = &e
	}
	return start, end
}

// DefaultRollingDateRange returns [from, to) ending at start of tomorrow by default.
func DefaultRollingDateRange(fromRaw, toRaw string, defaultDays int) (from, to time.Time) {
	if defaultDays <= 0 {
		defaultDays = 7
	}
	loc := Location()
	now := Now().In(loc)
	to = StartOfNextDay(now)
	from = to.AddDate(0, 0, -defaultDays)
	if s := strings.TrimSpace(fromRaw); s != "" {
		if t, ok := ParseQueryDay(s); ok {
			from = t
		}
	}
	if s := strings.TrimSpace(toRaw); s != "" {
		if t, ok := ParseQueryDay(s); ok {
			to = StartOfNextDay(t)
		}
	}
	return from, to
}

// SlidingWindowNow returns [now-days, now].
func SlidingWindowNow(days, defaultDays, maxDays int) (from, to time.Time) {
	if defaultDays <= 0 {
		defaultDays = 30
	}
	if maxDays <= 0 {
		maxDays = 365
	}
	if days <= 0 {
		days = defaultDays
	}
	if days > maxDays {
		days = maxDays
	}
	to = time.Now()
	from = to.AddDate(0, 0, -days)
	return from, to
}

// ParseOptionalRFC3339 parses *s when set; nil or blank returns (nil, nil).
func ParseOptionalRFC3339(s *string) (*time.Time, error) {
	if s == nil {
		return nil, nil
	}
	v := strings.TrimSpace(*s)
	if v == "" {
		return nil, nil
	}
	t, err := time.Parse(time.RFC3339, v)
	if err != nil {
		return nil, err
	}
	return &t, nil
}

// FormatRFC3339Ptr formats *time.Time; empty string when nil.
func FormatRFC3339Ptr(t *time.Time) string {
	if t == nil {
		return ""
	}
	return t.Format(time.RFC3339)
}
