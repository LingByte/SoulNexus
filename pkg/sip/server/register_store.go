package server

import (
	"context"
	"net"
	"time"
)

// SIPRegisterStore persists REGISTER bindings for INVITE proxy and outbound dial lookup.
// Implementations must be safe for concurrent use (e.g. GORM).
type SIPRegisterStore interface {
	// SaveRegister stores the resolved Contact signaling target (UDP), same as INVITE proxy destination.
	SaveRegister(ctx context.Context, user, domain, contactURI string, sig *net.UDPAddr, expiresAt time.Time, userAgent string) error
	DeleteRegister(ctx context.Context, user, domain string) error
	// LookupRegister returns the UDP signaling target for a registered AOR (Contact / Via path).
	LookupRegister(ctx context.Context, user, domain string) (*net.UDPAddr, bool, error)
}
