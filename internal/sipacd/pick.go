package sipacd

// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

import (
	"context"
	"strings"
	"time"

	"github.com/LingByte/SoulNexus/internal/models"
	"github.com/LingByte/SoulNexus/internal/sipreg"
	"github.com/LingByte/SoulNexus/pkg/sip/outbound"
	"gorm.io/gorm"
)

// PickTransferDialTarget selects one row from acd_pool_targets for blind transfer (DTMF).
// Eligible: not deleted, weight > 0, work_state = available, route_type sip or web.
// Ordering: weight DESC, id ASC (highest weight wins; tie-break lower id first).
//   - web → WebSeat (browser agent leg).
//   - sip trunk → DialTargetFromACDTrunk; sip internal → reg.DialTargetForUsername.
// SIP rows: sipCallerId / sipCallerDisplayName copied onto DialTarget when set.
func PickTransferDialTarget(ctx context.Context, db *gorm.DB, reg *sipreg.GormStore) (outbound.DialTarget, bool) {
	if db == nil {
		return outbound.DialTarget{}, false
	}
	freshWebSince := time.Now().Add(-models.WebSeatStaleAfter)
	var row models.ACDPoolTarget
	err := db.WithContext(ctx).
		Where("is_deleted = ? AND weight > ? AND work_state = ? AND route_type IN ?",
			models.SoftDeleteStatusActive, 0, models.ACDWorkStateAvailable,
			[]string{models.ACDPoolRouteTypeSIP, models.ACDPoolRouteTypeWeb}).
		Where("(route_type != ? OR (web_seat_last_seen_at IS NOT NULL AND web_seat_last_seen_at > ?))",
			models.ACDPoolRouteTypeWeb, freshWebSince).
		Order("weight DESC").Order("id ASC").
		First(&row).Error
	if err != nil {
		return outbound.DialTarget{}, false
	}

	if row.RouteType == models.ACDPoolRouteTypeWeb {
		return outbound.DialTarget{WebSeat: true}, true
	}

	var dt outbound.DialTarget
	src := strings.ToLower(strings.TrimSpace(row.SipSource))
	switch src {
	case models.ACDSipSourceTrunk:
		t, ok := outbound.DialTargetFromACDTrunk(row.TargetValue, row.SipTrunkHost, row.SipTrunkSignalingAddr, row.SipTrunkPort)
		if !ok {
			return outbound.DialTarget{}, false
		}
		dt = t
	default:
		if reg == nil {
			return outbound.DialTarget{}, false
		}
		u := strings.TrimSpace(row.TargetValue)
		if u == "" {
			return outbound.DialTarget{}, false
		}
		t, ok := reg.DialTargetForUsername(ctx, u)
		if !ok {
			return outbound.DialTarget{}, false
		}
		dt = t
	}
	dt.CallerUser = strings.TrimSpace(row.SipCallerID)
	dt.CallerDisplayName = strings.TrimSpace(row.SipCallerDisplayName)
	return dt, true
}
