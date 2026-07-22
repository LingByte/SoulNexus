package models

import (
	"strings"

	"github.com/LingByte/SoulNexus/internal/constants"
	"github.com/LingByte/SoulNexus/pkg/dialog/tenantcfg"
	"gorm.io/gorm"
)

func normalizeVoiceKey(v string) string {
	return strings.TrimSpace(strings.ToLower(v))
}

// voiceMatchKeys expands assistant timbre for pool routing (exact id, alias, provider:voice).
func voiceMatchKeys(voice string) []string {
	voice = normalizeVoiceKey(voice)
	if voice == "" {
		return nil
	}
	keys := []string{voice}
	if i := strings.Index(voice, ":"); i > 0 && i < len(voice)-1 {
		keys = append(keys, voice[i+1:])
	}
	if alias := voiceAliasCanonical[voice]; alias != "" {
		keys = append(keys, normalizeVoiceKey(alias))
	}
	seen := map[string]struct{}{}
	out := make([]string, 0, len(keys))
	for _, k := range keys {
		if k == "" {
			continue
		}
		if _, ok := seen[k]; ok {
			continue
		}
		seen[k] = struct{}{}
		out = append(out, k)
	}
	return out
}

// voiceAliasCanonical maps common display names / slugs to pool voice_ids entries.
var voiceAliasCanonical = map[string]string{
	"cherry": "zh_female_shuangkuaisisi_moon_bigtts",
	"serena": "zh_female_serena_moon_bigtts",
}

func poolVoiceIDMatches(poolID, voice string) bool {
	poolID = normalizeVoiceKey(poolID)
	if poolID == "*" {
		return true
	}
	if poolID == "" {
		return voice == ""
	}
	for _, k := range voiceMatchKeys(voice) {
		if k == poolID {
			return true
		}
	}
	return false
}

func poolMatchesVoice(ids []string, voice string) bool {
	voice = normalizeVoiceKey(voice)
	if len(ids) == 0 {
		return voice == ""
	}
	for _, id := range ids {
		id = normalizeVoiceKey(id)
		if id == "*" {
			return true
		}
		if voice != "" && poolVoiceIDMatches(id, voice) {
			return true
		}
	}
	return false
}

// poolVoiceMatchScore: higher = better match (exact voice > * > empty default).
func poolVoiceMatchScore(ids []string, voice string) int {
	voice = normalizeVoiceKey(voice)
	if len(ids) == 0 {
		if voice == "" {
			return 1
		}
		return 0
	}
	best := 0
	for _, id := range ids {
		id = normalizeVoiceKey(id)
		if id == "*" {
			if best < 2 {
				best = 2
			}
			continue
		}
		if voice != "" && poolVoiceIDMatches(id, voice) {
			return 10
		}
	}
	return best
}

type poolGrantRow struct {
	Pool  AIProviderPool
	Grant TenantAIPoolGrant
}

func loadTenantPoolGrantRows(db *gorm.DB, tenantID uint, modality string) []poolGrantRow {
	var grants []TenantAIPoolGrant
	if err := db.Where("tenant_id = ? AND enabled = ?", tenantID, true).Find(&grants).Error; err != nil || len(grants) == 0 {
		return nil
	}
	poolIDs := make([]uint, 0, len(grants))
	grantByPool := map[uint]TenantAIPoolGrant{}
	for _, g := range grants {
		if !grantHasQuota(g) {
			continue
		}
		poolIDs = append(poolIDs, g.PoolID)
		grantByPool[g.PoolID] = g
	}
	if len(poolIDs) == 0 {
		return nil
	}
	var pools []AIProviderPool
	if err := db.Where("id IN ? AND modality = ? AND enabled = ?", poolIDs, modality, true).
		Order("priority DESC, id ASC").
		Find(&pools).Error; err != nil {
		return nil
	}
	out := make([]poolGrantRow, 0, len(pools))
	for _, p := range pools {
		if !poolHasQuota(p) {
			continue
		}
		g, ok := grantByPool[p.ID]
		if !ok {
			continue
		}
		out = append(out, poolGrantRow{Pool: p, Grant: g})
	}
	return out
}

// SelectAIProviderPoolForTenant picks a pool granted to the tenant (中转站路由).
func SelectAIProviderPoolForTenant(db *gorm.DB, tenantID uint, modality, voiceID string) (AIProviderPool, bool) {
	if tenantID == 0 {
		return AIProviderPool{}, false
	}
	modality = NormalizeAIPoolModality(modality)
	if modality == "" {
		return AIProviderPool{}, false
	}
	rows := loadTenantPoolGrantRows(db, tenantID, modality)
	if len(rows) == 0 {
		return AIProviderPool{}, false
	}
	voiceID = strings.TrimSpace(voiceID)
	if modality != constants.AIPoolModalityTTS && modality != constants.AIPoolModalityRealtime {
		return rows[0].Pool, true
	}
	bestScore := -1
	var best AIProviderPool
	for _, row := range rows {
		ids := parsePoolVoiceIDs(row.Pool.VoiceIDs)
		score := poolVoiceMatchScore(ids, voiceID)
		if score > bestScore {
			bestScore = score
			best = row.Pool
		}
	}
	if bestScore <= 0 {
		return AIProviderPool{}, false
	}
	return best, true
}

// SelectAIProviderPool is deprecated for tenant traffic; kept for platform-wide checks.
func SelectAIProviderPool(db *gorm.DB, modality, voiceID string) (AIProviderPool, bool) {
	return AIProviderPool{}, false
}

// HasEnabledAIProviderPool — any enabled pool with global quota (admin UI).
func HasEnabledAIProviderPool(db *gorm.DB) bool {
	if db == nil {
		return false
	}
	var rows []AIProviderPool
	if err := db.Where("enabled = ?", true).Limit(32).Find(&rows).Error; err != nil {
		return false
	}
	for _, row := range rows {
		if poolHasQuota(row) {
			return true
		}
	}
	return false
}

// TenantPlatformKeyAllowed — platform keys require at least one active tenant 号池 grant.
func TenantPlatformKeyAllowed(db *gorm.DB, tenantID uint) bool {
	if tenantID == 0 {
		return false
	}
	return TenantHasActivePoolGrant(db, tenantID)
}

// PoolVoiceScope describes timbres exposed by a tenant's granted 号池 for one modality.
type PoolVoiceScope struct {
	Provider string   // first matching pool provider (for catalog lookup)
	VoiceIDs []string // explicit bindings (excluding "*")
	Wildcard bool     // any pool binds "*" → full catalog for Provider
}

// ListTenantPoolVoiceScope aggregates voice bindings from enabled tenant 号池 grants.
func ListTenantPoolVoiceScope(db *gorm.DB, tenantID uint, modality string) PoolVoiceScope {
	var out PoolVoiceScope
	if db == nil || tenantID == 0 {
		return out
	}
	rows := loadTenantPoolGrantRows(db, tenantID, modality)
	if len(rows) == 0 {
		return out
	}
	seen := map[string]struct{}{}
	for _, row := range rows {
		prov := strings.TrimSpace(strings.ToLower(row.Pool.Provider))
		if out.Provider == "" && prov != "" {
			out.Provider = prov
		}
		ids := parsePoolVoiceIDs(row.Pool.VoiceIDs)
		if containsStar(ids) {
			out.Wildcard = true
			continue
		}
		for _, id := range ids {
			id = strings.TrimSpace(id)
			if id == "" || normalizeVoiceKey(id) == "*" {
				continue
			}
			if _, ok := seen[id]; ok {
				continue
			}
			seen[id] = struct{}{}
			out.VoiceIDs = append(out.VoiceIDs, id)
		}
	}
	return out
}

// ResolvePoolConfigForVoice returns a copy of the granted pool config with voice applied.
func ResolvePoolConfigForVoice(db *gorm.DB, tenantID uint, modality, voiceID string) ([]byte, string, bool) {
	p, ok := SelectAIProviderPoolForTenant(db, tenantID, modality, voiceID)
	if !ok {
		return nil, "", false
	}
	raw := bytesTrimJSON(p.Config)
	if len(raw) == 0 {
		return nil, "", false
	}
	out := append([]byte(nil), raw...)
	voiceID = strings.TrimSpace(voiceID)
	if voiceID != "" {
		switch NormalizeAIPoolModality(modality) {
		case constants.AIPoolModalityTTS:
			out = tenantcfg.ApplyTTSVoice(out, voiceID)
		case constants.AIPoolModalityRealtime:
			out = tenantcfg.ApplyRealtimeVoice(out, voiceID)
		}
	}
	return out, strings.TrimSpace(strings.ToLower(p.Provider)), true
}
