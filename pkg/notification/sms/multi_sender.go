package sms

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync/atomic"

	"github.com/LingByte/SoulNexus/pkg/logger"
	"gorm.io/gorm"
)

// SenderChannel is one SMS send channel (provider + label).
type SenderChannel struct {
	Label    string
	Provider Provider
}

type senderRuntime struct {
	db        *gorm.DB
	orgID     uint
	userID    uint
	ipAddress string
}

type MultiSender struct {
	channels  []SenderChannel
	rrCounter uint32
	rt        senderRuntime
}

type SenderOption func(*senderRuntime)

func WithSMSLogOrgID(orgID uint) SenderOption {
	return func(rt *senderRuntime) { rt.orgID = orgID }
}
func WithSMSLogUserID(userID uint) SenderOption {
	return func(rt *senderRuntime) { rt.userID = userID }
}

func NewMultiSender(channels []SenderChannel, db *gorm.DB, ip string, opts ...SenderOption) (*MultiSender, error) {
	if len(channels) == 0 {
		return nil, errors.New("sms: at least one channel is required")
	}
	rt := senderRuntime{db: db, ipAddress: ip}
	for _, fn := range opts {
		fn(&rt)
	}
	return &MultiSender{channels: channels, rt: rt}, nil
}

func (s *MultiSender) Send(ctx context.Context, req SendRequest) error {
	ctx = ctxOrBackground(ctx)
	if err := ValidateBasic(req); err != nil {
		return err
	}
	n := len(s.channels)
	start := int(atomic.AddUint32(&s.rrCounter, 1)-1) % n
	var lastErr error
	var failParts []string
	for i := 0; i < n; i++ {
		slot := s.channels[(start+i)%n]
		res, err := slot.Provider.Send(ctx, req)
		if err == nil && res != nil && res.Accepted {
			if s.rt.db != nil {
				to := ""
				if len(req.To) > 0 {
					to = req.To[0].String()
				}
				_, _ = CreateSMSLog(
					s.rt.db,
					s.rt.orgID,
					s.rt.userID,
					string(slot.Provider.Kind()),
					slot.Label,
					to,
					strings.TrimSpace(req.Message.Template),
					strings.TrimSpace(req.Message.Content),
					strings.TrimSpace(res.MessageID),
					SmsStatusAccepted,
					res.Raw,
					s.rt.ipAddress,
				)
			}
			return nil
		}
		if err == nil && (res == nil || !res.Accepted) {
			err = errProviderRejected
		}
		lastErr = err
		failParts = append(failParts, fmt.Sprintf("[%s] %v", slot.Label, err))
		logger.Warnf("sms: send failed - channel=%s provider=%s err=%v", slot.Label, string(slot.Provider.Kind()), err)
	}
	errMsg := strings.Join(failParts, "; ")
	if s.rt.db != nil {
		to := ""
		if len(req.To) > 0 {
			to = req.To[0].String()
		}
		_, _ = CreateFailedSMSLog(
			s.rt.db,
			s.rt.orgID,
			s.rt.userID,
			"multi",
			"",
			to,
			strings.TrimSpace(req.Message.Template),
			strings.TrimSpace(req.Message.Content),
			errMsg,
			"",
			s.rt.ipAddress,
		)
	}
	if lastErr == nil {
		lastErr = errors.New("sms: all channels failed")
	}
	return lastErr
}
