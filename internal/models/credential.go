package models

import (
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"slices"
	"strings"
	"time"

	"github.com/LingByte/SoulNexus/internal/constants"
	constants2 "github.com/LingByte/SoulNexus/pkg/constants"
	apperror "github.com/LingByte/SoulNexus/pkg/errors"
	"github.com/LingByte/SoulNexus/pkg/utils"
	"github.com/LingByte/SoulNexus/pkg/utils/common"
	"gorm.io/gorm"
)

// API key token format: soulnexus_<43 base64url chars> (32 random bytes).
// AccessKey stores the lookup prefix (first APIKeyLookupLen runes).
// SecretKey stores SHA-256 hex of the full token (never the raw key after create).
// Legacy lex_<hex> tokens remain valid for lookup/auth until rotated.
const (
	APIKeyTokenPrefix       = "soulnexus_"
	APIKeyLegacyTokenPrefix = "lex_"
	APIKeySecretBytes       = 32
	APIKeyLookupSuffixLen   = 12
	APIKeyLookupLen         = len(APIKeyTokenPrefix) + APIKeyLookupSuffixLen // 22
	APIKeyLegacyLookupLen   = 16                                             // "lex_" + 12 hex
)

// Credential is a tenant-scoped machine API key with optional IP allowlist.
type Credential struct {
	common.BaseModel

	TenantID  uint   `json:"tenantId" gorm:"index;not null"`
	Name      string `json:"name" gorm:"size:128"`
	AccessKey string `json:"apiKeyPrefix" gorm:"column:access_key;size:64;uniqueIndex:idx_credential_ak;not null"` // lookup prefix
	SecretKey string `json:"-" gorm:"column:secret_key;size:256;not null"`                                         // sha256 hex of full API key
	Status    string `json:"status" gorm:"size:24;index;not null;default:active"`                                 // active | disabled
	AllowIP   string `json:"allowIp,omitempty" gorm:"type:text;comment:白名单IP，多个逗号分隔"`
	// PermissionCodes JSON array of catalog codes (e.g. ["api.assistants.read"]); ["*"] = all; empty/missing = no permissions.
	PermissionCodes string `json:"permissionCodes,omitempty" gorm:"column:permission_codes;type:text"`
	// AllowedRouteIDs JSON array of API route catalog ids; empty = no HTTP access.
	AllowedRouteIDs string     `json:"allowedRouteIds,omitempty" gorm:"column:allowed_route_ids;type:text"`
	ExpiresAt       *time.Time `json:"expiresAt,omitempty" gorm:"column:expires_at;index"`
	LastUsedAt      *time.Time `json:"lastUsedAt,omitempty" gorm:"column:last_used_at"`
	RequestCount    int64      `json:"requestCount" gorm:"column:request_count;not null;default:0"`
}

func (Credential) TableName() string {
	return constants2.CREDENTIAL_TABLE_NAME
}

// IsAPIKeyToken reports whether s looks like a SoulNexus API key (not a JWT).
func IsAPIKeyToken(s string) bool {
	s = strings.TrimSpace(s)
	if strings.HasPrefix(s, APIKeyTokenPrefix) {
		return len(s) > APIKeyLookupLen
	}
	// Legacy hex tokens issued before the soulnexus_ prefix change.
	return strings.HasPrefix(s, APIKeyLegacyTokenPrefix) && len(s) > APIKeyLegacyLookupLen
}

// APIKeyLookupPrefix returns the stored access_key lookup prefix for a full token.
func APIKeyLookupPrefix(fullKey string) string {
	fullKey = strings.TrimSpace(fullKey)
	n := APIKeyLookupLen
	if strings.HasPrefix(fullKey, APIKeyLegacyTokenPrefix) {
		n = APIKeyLegacyLookupLen
	}
	if len(fullKey) < n {
		return fullKey
	}
	return fullKey[:n]
}

// IsLegacyHMACCredential reports pre-API-key HMAC rows (access key "ak_…").
func IsLegacyHMACCredential(row Credential) bool {
	return strings.HasPrefix(strings.TrimSpace(row.AccessKey), "ak_")
}

// IssueAPIKey generates a raw API key, lookup prefix, and sha256 hash for storage.
func IssueAPIKey() (fullKey, lookupPrefix, keyHash string, err error) {
	buf := make([]byte, APIKeySecretBytes)
	if _, err = rand.Read(buf); err != nil {
		return "", "", "", err
	}
	fullKey = APIKeyTokenPrefix + base64.RawURLEncoding.EncodeToString(buf)
	lookupPrefix = APIKeyLookupPrefix(fullKey)
	keyHash = HashAPIKey(fullKey)
	return fullKey, lookupPrefix, keyHash, nil
}

// HashAPIKey returns lowercase hex SHA-256 of the full API key.
func HashAPIKey(fullKey string) string {
	sum := sha256.Sum256([]byte(strings.TrimSpace(fullKey)))
	return hex.EncodeToString(sum[:])
}

// APIKeyMatchesStoredHash constant-time compares HashAPIKey(fullKey) to storedHash.
func APIKeyMatchesStoredHash(fullKey, storedHash string) bool {
	want := HashAPIKey(fullKey)
	a := []byte(strings.ToLower(strings.TrimSpace(want)))
	b := []byte(strings.ToLower(strings.TrimSpace(storedHash)))
	if len(a) != len(b) {
		return false
	}
	return subtle.ConstantTimeCompare(a, b) == 1
}

// CredentialMatchesPermissionCodes checks API key permission JSON against required route codes (requireAll = AND).
func CredentialMatchesPermissionCodes(db *gorm.DB, credID uint, required []string, requireAll bool) (bool, error) {
	var row Credential
	if err := db.Where("id = ?", credID).First(&row).Error; err != nil {
		return false, err
	}
	raw := strings.TrimSpace(row.PermissionCodes)
	var codes []string
	if raw != "" {
		if err := json.Unmarshal([]byte(raw), &codes); err != nil {
			return false, err
		}
	}
	for _, c := range codes {
		if strings.TrimSpace(c) == "*" {
			return true, nil
		}
	}
	if len(required) == 0 {
		return true, nil
	}
	if requireAll {
		for _, req := range required {
			if !slices.Contains(codes, req) {
				return false, nil
			}
		}
		return true, nil
	}
	for _, req := range required {
		if slices.Contains(codes, req) {
			return true, nil
		}
	}
	return false, nil
}

// GetCredentialByIDForTenant loads one credential scoped to tenant (not deleted).
func GetCredentialByIDForTenant(db *gorm.DB, id, tenantID uint) (Credential, error) {
	var row Credential
	err := db.Where("id = ? AND tenant_id = ?", id, tenantID).First(&row).Error
	return row, err
}

// UpdateCredentialStatus sets status and optional update_by when status changes.
func UpdateCredentialStatus(db *gorm.DB, cred *Credential, status, updateBy string) error {
	if cred == nil || cred.ID == 0 {
		return gorm.ErrRecordNotFound
	}
	if cred.Status == status {
		return nil
	}
	meta := common.BaseModel{}
	meta.SetUpdateInfo(updateBy)
	updates := map[string]any{"status": status}
	if meta.UpdateBy != "" {
		updates["update_by"] = meta.UpdateBy
	}
	return db.Model(&Credential{}).Where("id = ?", cred.ID).Updates(updates).Error
}

// GetActiveCredentialByAPIKey resolves an active, non-legacy credential by full API key.
func GetActiveCredentialByAPIKey(db *gorm.DB, fullKey string) (Credential, error) {
	fullKey = strings.TrimSpace(fullKey)
	if !IsAPIKeyToken(fullKey) {
		return Credential{}, gorm.ErrRecordNotFound
	}
	prefix := APIKeyLookupPrefix(fullKey)
	var row Credential
	err := db.Model(&Credential{}).
		Where("access_key = ? AND status = ?", prefix, constants.CredentialStatusActive).
		First(&row).Error
	if err != nil {
		return Credential{}, err
	}
	if IsLegacyHMACCredential(row) || !APIKeyMatchesStoredHash(fullKey, row.SecretKey) {
		return Credential{}, gorm.ErrRecordNotFound
	}
	return row, nil
}

// CredentialIsExpired reports whether the key is past expires_at (nil = never expires).
func CredentialIsExpired(row Credential, now time.Time) bool {
	if row.ExpiresAt == nil {
		return false
	}
	return now.After(*row.ExpiresAt)
}

// RecordCredentialUse bumps request_count and last_used_at (best-effort).
func RecordCredentialUse(db *gorm.DB, id uint, now time.Time) {
	if db == nil || id == 0 {
		return
	}
	_ = db.Model(&Credential{}).Where("id = ?", id).Updates(map[string]any{
		"last_used_at":  now,
		"request_count": gorm.Expr("request_count + 1"),
		"updated_at":    now,
	}).Error
}

// ParseCredentialAllowedRouteIDs parses the credential route scope JSON array.
func ParseCredentialAllowedRouteIDs(raw string) []string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	var ids []string
	if err := json.Unmarshal([]byte(raw), &ids); err != nil {
		return nil
	}
	return NormalizeAKSKRouteIDs(ids)
}

// MarshalCredentialAllowedRouteIDs serializes route ids for storage.
func MarshalCredentialAllowedRouteIDs(ids []string) (string, error) {
	ids = NormalizeAKSKRouteIDs(ids)
	if len(ids) == 0 {
		return "", apperror.ErrAKSKRouteIDsRequired
	}
	b, err := json.Marshal(ids)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

// CredentialClientIPAllowed reports whether clientIP is permitted by AllowIP (comma-separated).
// Empty allowlist denies all clients unless CREDENTIAL_ALLOW_EMPTY_ALLOW_IP=true (dev only).
func CredentialClientIPAllowed(allowList, clientIP string) bool {
	allowList = strings.TrimSpace(allowList)
	if allowList == "" {
		return strings.EqualFold(strings.TrimSpace(utils.GetEnv(constants.ENVCredentialAllowEmptyAllowIP)), "true")
	}
	clientIP = strings.TrimSpace(clientIP)
	if clientIP == "" {
		return false
	}
	for _, part := range strings.Split(allowList, ",") {
		if strings.TrimSpace(part) == clientIP {
			return true
		}
	}
	return false
}

// MaskAPIKeyPrefix returns a short UI hint for a stored lookup prefix.
func MaskAPIKeyPrefix(prefix string) string {
	prefix = strings.TrimSpace(prefix)
	if prefix == "" {
		return "—"
	}
	if len(prefix) <= 8 {
		return prefix + "…"
	}
	return prefix[:8] + "…"
}
