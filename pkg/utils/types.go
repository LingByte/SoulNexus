package utils

// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

import (
	apperr "github.com/LingByte/SoulNexus/pkg/errors"
	"github.com/LingByte/SoulNexus/pkg/utils/audutil"
	"github.com/LingByte/SoulNexus/pkg/utils/coerce"
	"github.com/LingByte/SoulNexus/pkg/utils/common"
	"github.com/LingByte/SoulNexus/pkg/utils/phone"
	"github.com/LingByte/SoulNexus/pkg/utils/security"
	"github.com/LingByte/SoulNexus/pkg/utils/validate"
	"gorm.io/gorm"
)

// --- type aliases ---

type Config = common.Config
type Error = apperr.Error
type PageParams = common.PageParams
type Snowflake = common.Snowflake
type LoginGeo = common.LoginGeo
type Level = security.Level
type Check = security.Check
type Snapshot = security.Snapshot
type Options = security.Options
type SignalHandler = common.SignalHandler
type SigHandler = common.SigHandler
type SigHandlerEvent = common.SigHandlerEvent
type Signals = common.Signals
type IPLocationResponse = common.IPLocationResponse
type IPGeolocationResponse = common.IPGeolocationResponse
type SSRFSafeHTTPClientConfig = common.SSRFSafeHTTPClientConfig

// --- vars ---

var SnowflakeUtil = common.SnowflakeUtil

var (
	ErrUnauthorized              = apperr.ErrUnauthorized
	ErrAttachmentNotExist        = apperr.ErrAttachmentNotExist
	ErrNotAttachmentOwner        = apperr.ErrNotAttachmentOwner
	ErrQuotaExceeded             = apperr.ErrQuotaExceeded
	ErrLLMCallFailed             = apperr.ErrLLMCallFailed
	ErrEmptyPassword             = apperr.ErrEmptyPassword
	ErrEmptyEmail                = apperr.ErrEmptyEmail
	ErrSameEmail                 = apperr.ErrSameEmail
	ErrEmailExists               = apperr.ErrEmailExists
	ErrUserNotExists             = apperr.ErrUserNotExists
	ErrForbidden                 = apperr.ErrForbidden
	ErrUserNotAllowLogin         = apperr.ErrUserNotAllowLogin
	ErrUserNotAllowSignup        = apperr.ErrUserNotAllowSignup
	ErrNotActivated              = apperr.ErrNotActivated
	ErrTokenRequired             = apperr.ErrTokenRequired
	ErrInvalidToken              = apperr.ErrInvalidToken
	ErrBadToken                  = apperr.ErrBadToken
	ErrTokenExpired              = apperr.ErrTokenExpired
	ErrEmailRequired             = apperr.ErrEmailRequired
	ErrNotFound                  = apperr.ErrNotFound
	ErrNotChanged                = apperr.ErrNotChanged
	ErrInvalidView               = apperr.ErrInvalidView
	ErrOnlySuperUser             = apperr.ErrOnlySuperUser
	ErrInvalidPrimaryKey         = apperr.ErrInvalidPrimaryKey
	ErrInvalidToolListFormat     = apperr.ErrInvalidToolListFormat
	ErrInvalidToolFormat         = apperr.ErrInvalidToolFormat
	ErrToolNotFound              = apperr.ErrToolNotFound
	ErrInvalidToolParams         = apperr.ErrInvalidToolParams
	ErrParseJSONRPC              = apperr.ErrParseJSONRPC
	ErrInvalidJSONRPCFormat      = apperr.ErrInvalidJSONRPCFormat
	ErrInvalidJSONRPCResponse    = apperr.ErrInvalidJSONRPCResponse
	ErrInvalidJSONRPCRequest     = apperr.ErrInvalidJSONRPCRequest
	ErrInvalidJSONRPCParams      = apperr.ErrInvalidJSONRPCParams
	ErrInvalidResourceFormat     = apperr.ErrInvalidResourceFormat
	ErrResourceNotFound          = apperr.ErrResourceNotFound
	ErrInvalidPromptFormat       = apperr.ErrInvalidPromptFormat
	ErrPromptNotFound            = apperr.ErrPromptNotFound
	ErrEmptyToolName             = apperr.ErrEmptyToolName
	ErrToolAlreadyRegistered     = apperr.ErrToolAlreadyRegistered
	ErrToolExecutionFailed       = apperr.ErrToolExecutionFailed
	ErrEmptyResourceURI          = apperr.ErrEmptyResourceURI
	ErrEmptyPromptName           = apperr.ErrEmptyPromptName
	ErrSessionAlreadyInitialized = apperr.ErrSessionAlreadyInitialized
	ErrSessionNotInitialized     = apperr.ErrSessionNotInitialized
	ErrInvalidParams             = apperr.ErrInvalidParams
	ErrMissingParams             = apperr.ErrMissingParams
	ErrAlreadyInitialized        = apperr.ErrAlreadyInitialized
	ErrNotInitialized            = apperr.ErrNotInitialized
	ErrInvalidServerURL          = apperr.ErrInvalidServerURL
	ErrSSRFRedirectBlocked       = common.ErrSSRFRedirectBlocked
)

// --- constants ---

const (
	DefaultPageSize    = common.DefaultPageSize
	DefaultMaxPageSize = common.DefaultMaxPageSize
	MaxPageSize100     = common.MaxPageSize100
	MaxPageSize200     = common.MaxPageSize200

	MinRecordingWAVBytes = audutil.MinRecordingWAVBytes

	MaxKnownLoginCities = common.MaxKnownLoginCities

	DefaultSessionIdleTimeoutHours = common.DefaultSessionIdleTimeoutHours
	DefaultSessionMaxLifetimeHours = common.DefaultSessionMaxLifetimeHours
	MinSessionIdleTimeoutHours     = common.MinSessionIdleTimeoutHours
	MaxSessionIdleTimeoutHours     = common.MaxSessionIdleTimeoutHours
	MinSessionMaxLifetimeHours     = common.MinSessionMaxLifetimeHours
	MaxSessionMaxLifetimeHours     = common.MaxSessionMaxLifetimeHours
	TrustDeviceLoginDays           = common.TrustDeviceLoginDays

	LoginFailMaxAttempts = common.LoginFailMaxAttempts
	LoginFailLockTTL     = common.LoginFailLockTTL
	LoginFailCountTTL    = common.LoginFailCountTTL

	CategoryMobile  = common.CategoryMobile
	CategoryDesktop = common.CategoryDesktop
	CategoryTablet  = common.CategoryTablet

	PCONLINE_IP_URL = common.PCONLINE_IP_URL
	IP_API_URL      = common.IP_API_URL
	UNKNOWN         = common.UNKNOWN
	INTERNAL_IP     = common.INTERNAL_IP
	LOCAL_NETWORK   = common.LOCAL_NETWORK

	LevelOK    = security.LevelOK
	LevelWarn  = security.LevelWarn
	LevelError = security.LevelError
)

// --- coerce ---

var (
	Float64FromAny   = coerce.Float64FromAny
	IntFromAny       = coerce.IntFromAny
	IntDefault       = coerce.IntDefault
	Float64Default   = coerce.Float64Default
	BoolFromAny      = coerce.BoolFromAny
	IntFromAnyOrZero = coerce.IntFromAnyOrZero
)

// --- env ---

var (
	GetEnv                 = common.GetEnv
	LookupEnv              = common.LookupEnv
	PurgeEnvCacheForTest   = common.PurgeEnvCacheForTest
	GetBoolEnv             = common.GetBoolEnv
	GetFloatEnv            = common.GetFloatEnv
	GetIntEnv              = common.GetIntEnv
	EnvInt                 = common.EnvInt
	PositiveIntEnv         = common.PositiveIntEnv
	EnvFloat               = common.EnvFloat
	GetFloatEnvWithDefault = common.GetFloatEnvWithDefault
	GetIntEnvWithDefault   = common.GetIntEnvWithDefault
	GetStringOrDefault     = common.GetStringOrDefault
	GetBoolOrDefault       = common.GetBoolOrDefault
	GetIntOrDefault        = common.GetIntOrDefault
	GetFloatOrDefault      = common.GetFloatOrDefault
	ParseDuration          = common.ParseDuration
	LoadEnvs               = common.LoadEnvs
	LoadEnv                = common.LoadEnv
)

// --- sysconfig ---

var (
	PurgeConfigCache  = common.PurgeConfigCache
	SetValue          = common.SetValue
	GetValue          = common.GetValue
	GetIntValue       = common.GetIntValue
	GetBoolValue      = common.GetBoolValue
	CheckValue        = common.CheckValue
	LoadAutoloads     = common.LoadAutoloads
	LoadPublicConfigs = common.LoadPublicConfigs
)

// --- pagination ---

var (
	NormalizePageParams = common.NormalizePageParams
	TotalPages          = common.TotalPages
	PagePayload         = common.PagePayload
)

func FindPage[T any](q *gorm.DB, page, size int, orderExpr string, maxSize int) ([]T, int64, error) {
	return common.FindPage[T](q, page, size, orderExpr, maxSize)
}

func FindPageQuery[T any](q *gorm.DB, page, size, maxSize int, apply func(*gorm.DB) *gorm.DB) ([]T, int64, error) {
	return common.FindPageQuery[T](q, page, size, maxSize, apply)
}

// --- database ---

var (
	InitDatabase            = common.InitDatabase
	ConfigureConnectionPool = common.ConfigureConnectionPool
	MakeMigrates            = common.MakeMigrates
)

// --- core ---

var (
	RandText                         = common.RandText
	RandNumberText                   = common.RandNumberText
	SafeCall                         = common.SafeCall
	StructAsMap                      = common.StructAsMap
	GenerateSecureToken              = common.GenerateSecureToken
	NewSnowflake                     = common.NewSnowflake
	NextSnowflakeUint                = common.NextSnowflakeUint
	ClampSnowflakeUint               = common.ClampSnowflakeUint
	WriteFile                        = common.WriteFile
	ReadFile                         = common.ReadFile
	ComputeSampleByteCount           = common.ComputeSampleByteCount
	NormalizeFramePeriod             = common.NormalizeFramePeriod
	PickImageExtFromContentType      = common.PickImageExtFromContentType
	JSONValueFromBytes               = common.JSONValueFromBytes
	MarshalStringSliceJSON           = common.MarshalStringSliceJSON
	MustMarshalJSON                  = common.MustMarshalJSON
	NonEmptyOr                       = common.NonEmptyOr
	CloneRawMessage                  = common.CloneRawMessage
	ParseOptionalRFC3339        = common.ParseOptionalRFC3339
	DeriveTenantSlug            = common.DeriveTenantSlug
	ValidTenantSlug             = common.ValidTenantSlug
	DedupeUint                  = common.DedupeUint
)

var (
	ParseCommaList         = coerce.ParseCommaList
	FirstNonEmpty          = coerce.FirstNonEmpty
	ParseCommaSeparatedIDs = coerce.ParseCommaSeparatedIDs
	ContainsFourByteRune   = coerce.ContainsFourByteRune
	StripFourByteRunes     = coerce.StripFourByteRunes
	ReplaceFourByteRunes   = coerce.ReplaceFourByteRunes
	RemoveEmoji            = coerce.RemoveEmoji
	RemoveEmojiFromJSON    = coerce.RemoveEmojiFromJSON
)

// --- validate ---

var (
	IsEmail               = validate.IsEmail
	IsMobile              = validate.IsMobile
	IsDomain              = validate.IsDomain
	IsSlug                = validate.IsSlug
	IsEmpty               = validate.IsEmpty
	Trim                  = validate.Trim
	TrimAll               = validate.TrimAll
	TrimLower             = validate.TrimLower
	DefaultStr            = validate.DefaultStr
	NormalizePage         = validate.NormalizePage
	ParseID               = validate.ParseID
	ParseOptionalID       = validate.ParseOptionalID
	RequireScopeID        = validate.RequireScopeID
	ParseIDStrings        = validate.ParseIDStrings
	ParseNonZeroIDStrings = validate.ParseNonZeroIDStrings
	FormatID              = validate.FormatID
	FormatIDs             = validate.FormatIDs
	ValidPassword         = validate.ValidPassword
)

// --- security ---

var (
	SanitizeHTML        = security.SanitizeHTML
	EscapeHTML          = security.EscapeHTML
	ValidateInput       = security.ValidateInput
	IsValidURL          = security.IsValidURL
	CleanMarkdown       = security.CleanMarkdown
	SanitizeForDisplay  = security.SanitizeForDisplay
	SanitizeForLog      = security.SanitizeForLog
	SanitizeForLogArray = security.SanitizeForLogArray
)

// --- ua ---

var (
	CategoryFromUserAgent    = common.CategoryFromUserAgent
	LoginLimitCategory       = common.LoginLimitCategory
	DisplayNameFromUserAgent = common.DisplayNameFromUserAgent
)

// --- net ---

var (
	IsIP                            = common.IsIP
	ParseIP                         = common.ParseIP
	IsPrivateIP                     = common.IsPrivateIP
	IsIpInCIDRList                  = common.IsIpInCIDRList
	IsInternalIP                    = common.IsInternalIP
	GetIPLocation                   = common.GetIPLocation
	GetIPLocationCN                 = common.GetIPLocationCN
	GetIPLocationGlobal             = common.GetIPLocationGlobal
	GetRealAddressByIP              = common.GetRealAddressByIP
	IsPublicIP                      = common.IsPublicIP
	SetSSRFWhitelistFromRaw         = common.SetSSRFWhitelistFromRaw
	IsSSRFWhitelisted               = common.IsSSRFWhitelisted
	ResetSSRFWhitelistForTest       = common.ResetSSRFWhitelistForTest
	ValidateURLForSSRF              = common.ValidateURLForSSRF
	ValidateTenantConfiguredURL     = common.ValidateTenantConfiguredURL
	IsSystemProxy                   = common.IsSystemProxy
	DefaultSSRFSafeHTTPClientConfig = common.DefaultSSRFSafeHTTPClientConfig
	NewSSRFSafeHTTPClient           = common.NewSSRFSafeHTTPClient
	NewTenantToolHTTPClient         = common.NewTenantToolHTTPClient
	SSRFSafeDialContext             = common.SSRFSafeDialContext
)

// --- login ---

var (
	CheckLoginAccountLocked = common.CheckLoginAccountLocked
	RecordLoginFailure      = common.RecordLoginFailure
	ClearLoginFailures      = common.ClearLoginFailures
	LoginGeoFromIP          = common.LoginGeoFromIP
	LoginCityFromIP         = common.LoginCityFromIP
	ParseKnownLoginCities   = common.ParseKnownLoginCities
	IsKnownLoginCity        = common.IsKnownLoginCity
	NeedsRemoteLoginVerify  = common.NeedsRemoteLoginVerify
	AddKnownLoginCity       = common.AddKnownLoginCity
)

// --- audutil ---

var (
	RMSPCM16LE                      = audutil.RMSPCM16LE
	WAVDurationSec                  = audutil.WAVDurationSec
	RecordingObjectKey              = audutil.RecordingObjectKey
	RecordingPartObjectKey          = audutil.RecordingPartObjectKey
	FetchRecordingWAV               = audutil.FetchRecordingWAV
	RecordingJitterSnapNs           = audutil.RecordingJitterSnapNs
	RecordingLegObjectKey           = audutil.RecordingLegObjectKey
	SplitSN3ToSOA1                  = audutil.SplitSN3ToSOA1
	PackOggOpus                     = audutil.PackOggOpus
	SOA1ToWAV                       = audutil.SOA1ToWAV
	TrimSOA1AfterWallNs             = audutil.TrimSOA1AfterWallNs
	MergeSOA1StereoWAV              = audutil.MergeSOA1StereoWAV
	EncodePCM16MonoToSOA1           = audutil.EncodePCM16MonoToSOA1
	EncodeStereoWAVLegsToSOA1       = audutil.EncodeStereoWAVLegsToSOA1
)

const (
	RecordingFormatSOA1 = audutil.RecordingFormatSOA1
	RecordingFormatWAV  = audutil.RecordingFormatWAV
)

// --- phone ---

var (
	NormalizePhoneDigits     = phone.NormalizePhoneDigits
	NormalizePhone           = phone.NormalizePhoneDigits
	FormatPhoneLocation      = phone.FormatPhoneLocation
	LookupPhoneLocationParts = phone.LookupPhoneLocationParts
	LookupPhoneLocation      = phone.LookupPhoneLocation
)

// --- session ---

var (
	SessionIdleTimeout = common.SessionIdleTimeout
	SessionMaxLifetime = common.SessionMaxLifetime
)

// --- signal ---

var (
	Sig        = common.Sig
	NewSignals = common.NewSignals
)

// --- preflight ---

var (
	StoreSnapshot = security.StoreSnapshot
	GetSnapshot   = security.GetSnapshot
	Run           = security.Run
)
