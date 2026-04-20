package models

import (
	"context"
	"strings"
	"time"

	"github.com/LingByte/SoulNexus/pkg/constants"
	"gorm.io/gorm"
)

// ACD pool route types (single table for SIP accounts and Web/WebRTC seats).
const (
	ACDPoolRouteTypeSIP = "sip"
	ACDPoolRouteTypeWeb = "web"
)

// SipSource applies when RouteType is SIP (empty for web rows).
const (
	ACDSipSourceInternal = "internal" // TargetValue = registered sip_users.username
	ACDSipSourceTrunk    = "trunk"    // TargetValue = external dial string (PSTN / carrier)
)

// WebSeatStaleAfter is the max age of WebSeatLastSeenAt for "链路在线" and sipacd.PickTransferDialTarget web eligibility.
const WebSeatStaleAfter = 90 * time.Second

// WorkState is real-time seat eligibility for this row (agent UI, admin, or SIP/Web gateway updates it).
// cmd/sip blind transfer (DTMF) picks only rows with WorkState == ACDWorkStateAvailable (plus weight>0).
// Use ringing/busy/acw/break to take a row out of the next-offer set without deleting it.
const (
	ACDWorkStateOffline   = "offline"   // not signed in / not bound
	ACDWorkStateAvailable = "available" // idle, eligible for next offer
	ACDWorkStateRinging   = "ringing"   // offer in progress; do not assign another call
	ACDWorkStateBusy      = "busy"      // media connected / on call
	ACDWorkStateACW       = "acw"       // after-call work (wrap-up)
	ACDWorkStateBreak     = "break"     // rest / pause
)

// ACDPoolTarget is one row in the transfer routing table (acd_pool_targets) when cmd/sip uses a database.
// Selection: highest Weight first, then lowest id; only Weight>0 and WorkState==available.
// SIP rows: internal TargetValue = sip_users.username; trunk = dial string + trunk host fields.
// Web rows: TargetValue usually empty; WebSeat handoff when this row wins over SIP rows by Weight.
type ACDPoolTarget struct {
	BaseModel
	Name                  string     `json:"name" gorm:"size:128"`                               // optional admin label
	RouteType             string     `json:"routeType" gorm:"size:16;not null;index"`            // RouteType is ACDPoolRouteTypeSIP or ACDPoolRouteTypeWeb.
	TargetValue           string     `json:"targetValue" gorm:"size:256"`                        // TargetValue: sip internal → sip_users.username; sip trunk → dial digits / URI; web → usually empty.
	SipSource             string     `json:"sipSource" gorm:"size:16;not null;default:'';index"` // SipSource: internal | trunk when RouteType is SIP; empty when web.
	SipTrunkHost          string     `json:"sipTrunkHost" gorm:"size:128"`                       // Sip trunk only: next SIP hop for INVITE (Request-URI host:port + optional signaling override).
	SipTrunkPort          int        `json:"sipTrunkPort" gorm:"not null;default:0"`
	SipTrunkSignalingAddr string     `json:"sipTrunkSignalingAddr" gorm:"size:160"` // host:port, optional; default host:port from above
	SipCallerID           string     `json:"sipCallerId" gorm:"size:64"`            // SIP only: optional outbound From user / display (like SIP_CALLER_ID); empty → cmd/sip uses global .env default.
	SipCallerDisplayName  string     `json:"sipCallerDisplayName" gorm:"size:128"`
	Weight                int        `json:"weight" gorm:"not null;default:0;index"`                  // Weight: higher = higher priority when selecting; 0 = disabled (not eligible).
	WorkState             string     `json:"workState" gorm:"size:24;not null;default:offline;index"` // WorkState: see ACDWorkState*; default offline until sign-in or integration sets available.
	WorkStateAt           *time.Time `json:"workStateAt"`                                             // WorkStateAt: optional last transition (ring timeouts, metrics).
	WebSeatLastSeenAt     *time.Time `json:"webSeatLastSeenAt" gorm:"column:web_seat_last_seen_at"`   // WebSeatLastSeenAt: route_type=web only; last heartbeat from browser (keepalive). Used for pick + admin "链路在线".
}

// WebSeatLastSeenFresh reports whether a web seat heartbeat is recent enough to treat the row as reachable.
func WebSeatLastSeenFresh(t *time.Time) bool {
	if t == nil {
		return false
	}
	return time.Since(*t) <= WebSeatStaleAfter
}

func (ACDPoolTarget) TableName() string {
	return constants.ACD_POOL_TARGET_TABLE_NAME
}

// --- Normalization (admin API + transfer pick) ---

// ParseACDRouteType returns canonical route type or false.
func ParseACDRouteType(s string) (string, bool) {
	s = strings.ToLower(strings.TrimSpace(s))
	switch s {
	case ACDPoolRouteTypeSIP, ACDPoolRouteTypeWeb:
		return s, true
	default:
		return "", false
	}
}

// NormalizeACDSipSource returns internal or trunk.
func NormalizeACDSipSource(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	if s == ACDSipSourceTrunk || s == "external" {
		return ACDSipSourceTrunk
	}
	return ACDSipSourceInternal
}

// NormalizeACDWorkState returns a known work_state or offline.
func NormalizeACDWorkState(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	switch s {
	case ACDWorkStateOffline, ACDWorkStateAvailable, ACDWorkStateRinging,
		ACDWorkStateBusy, ACDWorkStateACW, ACDWorkStateBreak:
		return s
	default:
		return ACDWorkStateOffline
	}
}

// NormalizeACDTrunkPort returns a valid SIP port or 5060.
func NormalizeACDTrunkPort(p int) int {
	if p <= 0 || p >= 65536 {
		return 5060
	}
	return p
}

// ACDSipInternalLiveLineEligible is true when list UI may show SIP registration hint.
func ACDSipInternalLiveLineEligible(row ACDPoolTarget) bool {
	if row.RouteType != ACDPoolRouteTypeSIP || strings.TrimSpace(row.TargetValue) == "" {
		return false
	}
	src := strings.ToLower(strings.TrimSpace(row.SipSource))
	if src == ACDSipSourceTrunk || src == "external" {
		return false
	}
	return true
}

// ACDTrunkStorageFields returns DB trunk columns; empty when not sip+trunk.
func ACDTrunkStorageFields(routeType, sipSource, trunkHost string, trunkPort int, trunkSig string) (host string, port int, sig string) {
	if routeType != ACDPoolRouteTypeSIP || sipSource != ACDSipSourceTrunk {
		return "", 0, ""
	}
	return strings.TrimSpace(trunkHost), NormalizeACDTrunkPort(trunkPort), strings.TrimSpace(trunkSig)
}

// ACDCallerStorageFields returns outbound CLI fields for SIP rows only.
func ACDCallerStorageFields(routeType, sipCallerID, sipCallerDisplayName string) (id, disp string) {
	if routeType != ACDPoolRouteTypeSIP {
		return "", ""
	}
	return strings.TrimSpace(sipCallerID), strings.TrimSpace(sipCallerDisplayName)
}

// ValidateACDTrunkCreateUpdate requires dial target + gateway for trunk SIP rows.
func ValidateACDTrunkCreateUpdate(routeType, sipSource, targetValue, trunkHost string) bool {
	if routeType != ACDPoolRouteTypeSIP || sipSource != ACDSipSourceTrunk {
		return true
	}
	return strings.TrimSpace(targetValue) != "" && strings.TrimSpace(trunkHost) != ""
}

// --- Queries ---

// ActiveACDPoolTargets is the non-deleted scope.
func ActiveACDPoolTargets(db *gorm.DB) *gorm.DB {
	return db.Model(&ACDPoolTarget{}).Where("is_deleted = ?", SoftDeleteStatusActive)
}

// ListACDPoolTargetsPage lists active targets; routeType empty skips filter.
func ListACDPoolTargetsPage(db *gorm.DB, page, size int, routeType string) ([]ACDPoolTarget, int64, error) {
	q := ActiveACDPoolTargets(db)
	if rt := strings.TrimSpace(routeType); rt != "" {
		if t, ok := ParseACDRouteType(rt); ok {
			q = q.Where("route_type = ?", t)
		}
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	offset := (page - 1) * size
	var list []ACDPoolTarget
	if err := q.Order("weight DESC, id DESC").Offset(offset).Limit(size).Find(&list).Error; err != nil {
		return nil, 0, err
	}
	return list, total, nil
}

// GetActiveACDPoolTargetByID returns one active row.
func GetActiveACDPoolTargetByID(db *gorm.DB, id uint) (ACDPoolTarget, error) {
	var row ACDPoolTarget
	err := ActiveACDPoolTargets(db).Where("id = ?", id).First(&row).Error
	return row, err
}

// ReloadACDPoolTargetByID refetches by primary key (any delete state).
func ReloadACDPoolTargetByID(db *gorm.DB, id uint) (ACDPoolTarget, error) {
	var row ACDPoolTarget
	err := db.Where("id = ?", id).First(&row).Error
	return row, err
}

// SoftDeleteACDPoolTargetByID soft-deletes an active row.
func SoftDeleteACDPoolTargetByID(db *gorm.DB, id uint, updateBy string) (int64, error) {
	u := map[string]any{
		"is_deleted": SoftDeleteStatusDeleted,
		"updated_at": time.Now(),
	}
	if updateBy != "" {
		u["update_by"] = updateBy
	}
	res := db.Model(&ACDPoolTarget{}).Where("id = ? AND is_deleted = ?", id, SoftDeleteStatusActive).Updates(u)
	return res.RowsAffected, res.Error
}

// NewACDPoolTargetForCreate builds a row after request normalization.
func NewACDPoolTargetForCreate(
	name, routeType, sipSource, targetValue string,
	trunkHost string, trunkPort int, trunkSig string,
	sipCallerID, sipCallerDisplayName string,
	weight int, workState string,
	now time.Time,
	webSeatLastSeen *time.Time,
) ACDPoolTarget {
	th, tp, ts := ACDTrunkStorageFields(routeType, sipSource, trunkHost, trunkPort, trunkSig)
	cid, cdn := ACDCallerStorageFields(routeType, sipCallerID, sipCallerDisplayName)
	return ACDPoolTarget{
		Name:                  strings.TrimSpace(name),
		RouteType:             routeType,
		SipSource:             sipSource,
		TargetValue:           strings.TrimSpace(targetValue),
		SipTrunkHost:          th,
		SipTrunkPort:          tp,
		SipTrunkSignalingAddr: ts,
		SipCallerID:           cid,
		SipCallerDisplayName:  cdn,
		Weight:                weight,
		WorkState:             workState,
		WorkStateAt:           &now,
		WebSeatLastSeenAt:     webSeatLastSeen,
	}
}

// BuildACDPoolTargetUpdateMap builds GORM Updates for admin PUT.
func BuildACDPoolTargetUpdateMap(
	existing ACDPoolTarget,
	name, routeType, sipSource, targetValue string,
	trunkHost string, trunkPort int, trunkSig string,
	sipCallerID, sipCallerDisplayName string,
	weight int, workState string,
	now time.Time,
	updateBy string,
) map[string]any {
	th, tp, ts := ACDTrunkStorageFields(routeType, sipSource, trunkHost, trunkPort, trunkSig)
	cid, cdn := ACDCallerStorageFields(routeType, sipCallerID, sipCallerDisplayName)
	u := map[string]any{
		"name":                     strings.TrimSpace(name),
		"route_type":               routeType,
		"sip_source":               sipSource,
		"target_value":             strings.TrimSpace(targetValue),
		"sip_trunk_host":           th,
		"sip_trunk_port":           tp,
		"sip_trunk_signaling_addr": ts,
		"sip_caller_id":            cid,
		"sip_caller_display_name":  cdn,
		"weight":                   weight,
		"work_state":               workState,
		"updated_at":               now,
	}
	if existing.WorkState != workState {
		u["work_state_at"] = now
	}
	if routeType == ACDPoolRouteTypeWeb && workState == ACDWorkStateAvailable {
		u["web_seat_last_seen_at"] = now
	}
	if updateBy != "" {
		u["update_by"] = updateBy
	}
	return u
}

// ClearACDPoolTargetWebSeatLastSeen sets web_seat_last_seen_at NULL (web row offline).
func ClearACDPoolTargetWebSeatLastSeen(db *gorm.DB, id uint) error {
	return db.Model(&ACDPoolTarget{}).Where("id = ?", id).Update("web_seat_last_seen_at", nil).Error
}

// UpdateACDPoolTargetWebSeatHeartbeat refreshes web seat presence only.
// It must not overwrite work_state (ringing/busy/acw etc.) — those are driven by transfer + webseat lifecycle.
func UpdateACDPoolTargetWebSeatHeartbeat(db *gorm.DB, id uint, operator string, now time.Time) error {
	u := map[string]any{
		"web_seat_last_seen_at": now,
		"updated_at":            now,
		"update_by":             strings.TrimSpace(operator),
	}
	return db.Model(&ACDPoolTarget{}).Where("id = ?", id).Updates(u).Error
}

// UpdateACDPoolTargetWorkState updates work_state for an active row (admin, SIP transfer, or webseat).
func UpdateACDPoolTargetWorkState(ctx context.Context, db *gorm.DB, id uint, workState string, updateBy string) error {
	if db == nil || id == 0 {
		return nil
	}
	ws := NormalizeACDWorkState(workState)
	now := time.Now()
	u := map[string]any{
		"work_state":    ws,
		"work_state_at": now,
		"updated_at":    now,
	}
	if s := strings.TrimSpace(updateBy); s != "" {
		u["update_by"] = s
	}
	return ActiveACDPoolTargets(db.WithContext(ctx)).Where("id = ?", id).Updates(u).Error
}

// WebSeatActorMayTouchRow allows heartbeat when CreateBy is empty or matches operator.
func WebSeatActorMayTouchRow(row ACDPoolTarget, operator string) bool {
	operator = strings.TrimSpace(operator)
	cb := strings.TrimSpace(row.CreateBy)
	if cb == "" {
		return true
	}
	return cb == operator
}

// PickEligibleACDPoolTargetForTransfer loads highest-weight available target (same rule as sipserver.PickTransferDialTarget query).
func PickEligibleACDPoolTargetForTransfer(ctx context.Context, db *gorm.DB) (ACDPoolTarget, error) {
	freshWebSince := time.Now().Add(-WebSeatStaleAfter)
	var row ACDPoolTarget
	err := db.WithContext(ctx).
		Where("is_deleted = ? AND weight > ? AND work_state = ? AND route_type IN ?",
			SoftDeleteStatusActive, 0, ACDWorkStateAvailable,
			[]string{ACDPoolRouteTypeSIP, ACDPoolRouteTypeWeb}).
		Where("(route_type != ? OR (web_seat_last_seen_at IS NOT NULL AND web_seat_last_seen_at > ?))",
			ACDPoolRouteTypeWeb, freshWebSince).
		Order("weight DESC").Order("id ASC").
		First(&row).Error
	return row, err
}

// ListActiveWebACDPoolTargetsByCreateBy lists active web rows owned by the same creator.
func ListActiveWebACDPoolTargetsByCreateBy(ctx context.Context, db *gorm.DB, createBy string) ([]ACDPoolTarget, error) {
	createBy = strings.TrimSpace(createBy)
	if createBy == "" {
		return nil, nil
	}
	var rows []ACDPoolTarget
	err := ActiveACDPoolTargets(db.WithContext(ctx)).
		Where("route_type = ? AND create_by = ?", ACDPoolRouteTypeWeb, createBy).
		Order("updated_at DESC").Order("id DESC").
		Find(&rows).Error
	return rows, err
}

// SoftDeleteACDPoolTargetsByIDs soft-deletes rows by ids.
func SoftDeleteACDPoolTargetsByIDs(ctx context.Context, db *gorm.DB, ids []uint, updateBy string) (int64, error) {
	if len(ids) == 0 {
		return 0, nil
	}
	u := map[string]any{
		"is_deleted": SoftDeleteStatusDeleted,
		"updated_at": time.Now(),
	}
	if s := strings.TrimSpace(updateBy); s != "" {
		u["update_by"] = s
	}
	res := db.WithContext(ctx).Model(&ACDPoolTarget{}).
		Where("id IN ? AND is_deleted = ?", ids, SoftDeleteStatusActive).
		Updates(u)
	return res.RowsAffected, res.Error
}

// MarkStaleWebACDPoolTargetsOffline sets stale web seats to offline for abnormal-disconnect fallback.
func MarkStaleWebACDPoolTargetsOffline(ctx context.Context, db *gorm.DB, now time.Time) (int64, error) {
	freshSince := now.Add(-WebSeatStaleAfter)
	res := ActiveACDPoolTargets(db.WithContext(ctx)).
		Where("route_type = ?", ACDPoolRouteTypeWeb).
		Where("work_state <> ?", ACDWorkStateOffline).
		Where("web_seat_last_seen_at IS NULL OR web_seat_last_seen_at <= ?", freshSince).
		Updates(map[string]any{
			"work_state":    ACDWorkStateOffline,
			"work_state_at": now,
			"updated_at":    now,
		})
	return res.RowsAffected, res.Error
}
