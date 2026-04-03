package sipreg

import (
	"context"
	"errors"
	"fmt"
	"net"
	"strconv"
	"strings"
	"time"

	"github.com/LingByte/SoulNexus/internal/models"
	"github.com/LingByte/SoulNexus/pkg/sip/outbound"
	"github.com/LingByte/SoulNexus/pkg/utils"

	"gorm.io/gorm"
)

// GormStore implements server.SIPRegisterStore using sip_users.
type GormStore struct {
	db *gorm.DB
}

func NewGormStore(db *gorm.DB) *GormStore {
	return &GormStore{db: db}
}

func (s *GormStore) SaveRegister(ctx context.Context, user, domain, contactURI string, sig *net.UDPAddr, expiresAt time.Time, userAgent string) error {
	if s == nil || s.db == nil || sig == nil {
		return nil
	}
	user = strings.TrimSpace(user)
	domain = strings.TrimSpace(domain)
	if user == "" || domain == "" {
		return nil
	}
	now := time.Now()
	exp := expiresAt
	var row models.SIPUser
	err := s.db.WithContext(ctx).Where("username = ? AND domain = ? AND is_deleted = ?", user, domain, models.SoftDeleteStatusActive).First(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		row = models.SIPUser{
			Username:   user,
			Domain:     domain,
			ContactURI: contactURI,
			RemoteIP:   sig.IP.String(),
			RemotePort: sig.Port,
			UserAgent:  userAgent,
			Online:     true,
			ExpiresAt:  &exp,
			LastSeenAt: &now,
		}
		return s.db.WithContext(ctx).Create(&row).Error
	}
	if err != nil {
		return err
	}
	return s.db.WithContext(ctx).Model(&row).Updates(map[string]interface{}{
		"contact_uri":  contactURI,
		"remote_ip":    sig.IP.String(),
		"remote_port":  sig.Port,
		"user_agent":   userAgent,
		"online":       true,
		"expires_at":   exp,
		"last_seen_at": now,
	}).Error
}

func (s *GormStore) DeleteRegister(ctx context.Context, user, domain string) error {
	if s == nil || s.db == nil {
		return nil
	}
	user = strings.TrimSpace(user)
	domain = strings.TrimSpace(domain)
	if user == "" || domain == "" {
		return nil
	}
	return s.db.WithContext(ctx).Model(&models.SIPUser{}).
		Where("username = ? AND domain = ? AND is_deleted = ?", user, domain, models.SoftDeleteStatusActive).
		Updates(map[string]interface{}{
			"online":       false,
			"expires_at":   nil,
			"last_seen_at": time.Now(),
		}).Error
}

func (s *GormStore) LookupRegister(ctx context.Context, user, domain string) (*net.UDPAddr, bool, error) {
	if s == nil || s.db == nil {
		return nil, false, nil
	}
	user = strings.TrimSpace(user)
	domain = strings.TrimSpace(domain)
	if user == "" || domain == "" {
		return nil, false, nil
	}
	var row models.SIPUser
	err := s.db.WithContext(ctx).Where("username = ? AND domain = ? AND is_deleted = ? AND online = ?", user, domain, models.SoftDeleteStatusActive, true).
		Where("expires_at IS NULL OR expires_at > ?", time.Now()).
		First(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, err
	}
	if row.RemoteIP == "" || row.RemotePort <= 0 {
		return nil, false, nil
	}
	ip := net.ParseIP(row.RemoteIP)
	if ip == nil {
		return nil, false, nil
	}
	return &net.UDPAddr{IP: ip, Port: row.RemotePort}, true, nil
}

// DialTargetForUsername returns outbound.DialTarget for a registered extension (username).
// SIP_DEFAULT_DOMAIN optionally restricts to one AOR when multiple domains exist.
func (s *GormStore) DialTargetForUsername(ctx context.Context, username string) (outbound.DialTarget, bool) {
	var zero outbound.DialTarget
	if s == nil || s.db == nil {
		return zero, false
	}
	username = strings.TrimSpace(username)
	if username == "" {
		return zero, false
	}
	domain := strings.TrimSpace(utils.GetEnv(outbound.EnvSIPDefaultDomain))
	q := s.db.WithContext(ctx).Model(&models.SIPUser{}).
		Where("username = ? AND is_deleted = ? AND online = ?", username, models.SoftDeleteStatusActive, true).
		Where("expires_at IS NULL OR expires_at > ?", time.Now())
	if domain != "" {
		q = q.Where("domain = ?", domain)
	}
	var row models.SIPUser
	if err := q.First(&row).Error; err != nil {
		return zero, false
	}
	if row.RemoteIP == "" || row.RemotePort <= 0 {
		return zero, false
	}
	d := row.Domain
	if d == "" {
		d = "localhost"
	}
	port := 5060
	if ps := strings.TrimSpace(utils.GetEnv(outbound.EnvSIPDefaultURIPort)); ps != "" {
		if p, err := strconv.Atoi(ps); err == nil && p > 0 && p < 65536 {
			port = p
		}
	}
	reqURI := fmt.Sprintf("sip:%s@%s:%d", row.Username, d, port)
	sig := net.JoinHostPort(row.RemoteIP, strconv.Itoa(row.RemotePort))
	return outbound.DialTarget{RequestURI: reqURI, SignalingAddr: sig}, true
}
