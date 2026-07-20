package timeutil

import (
	"log"
	"strings"
	"sync"
	"time"
)

// DefaultTimezone is used when SYSTEM_TIMEZONE is unset or invalid.
const DefaultTimezone = "Asia/Shanghai"

var (
	mu     sync.RWMutex
	tzName = DefaultTimezone
)

// Init loads IANA timezone name, sets process time.Local, and stores the name
// for modules that need an explicit zone (realtime tools, scheduling).
func Init(name string) {
	name = strings.TrimSpace(name)
	if name == "" {
		name = DefaultTimezone
	}
	loc, err := time.LoadLocation(name)
	if err != nil {
		log.Printf("timeutil: invalid SYSTEM_TIMEZONE %q: %v; falling back to %s", name, err, DefaultTimezone)
		loc, _ = time.LoadLocation(DefaultTimezone)
		name = DefaultTimezone
	}
	mu.Lock()
	tzName = name
	time.Local = loc
	mu.Unlock()
}

// Name returns the configured IANA timezone (e.g. Asia/Shanghai).
func Name() string {
	mu.RLock()
	defer mu.RUnlock()
	return tzName
}

// Location returns the configured business timezone.
func Location() *time.Location {
	mu.RLock()
	name := tzName
	mu.RUnlock()
	loc, err := time.LoadLocation(name)
	if err != nil {
		loc, _ = time.LoadLocation(DefaultTimezone)
	}
	return loc
}

// Now returns the current time in the configured business timezone.
func Now() time.Time {
	return time.Now()
}

// FormatLocaleDateTime formats t for user-facing notifications (e.g. 2026年06月28日 19:12).
func FormatLocaleDateTime(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.In(Location()).Format("2006年01月02日 15:04")
}
