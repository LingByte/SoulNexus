package ua

import (
	"net"
	"strconv"
	"sync"
	"time"

	"github.com/LingByte/SoulNexus/internal/models"
	"github.com/emiago/sipgo/sip"
	"gorm.io/gorm"
)

// StorageType define storage type
type StorageType string

const (
	StorageTypeFile     StorageType = "file"     // storage by file
	StorageTypeMemory   StorageType = "memory"   // storage by memory
	StorageTypeDatabase StorageType = "database" // storage by database
)

// UAConfig represents the configuration for a User Agent
type UAConfig struct {
	Host                  string        // sip server host address
	Port                  int           // sip server port
	UserAgentName         string        // represent client user agent name
	LocalRTPPort          int           // local RTP port
	RegisterTimeout       time.Duration // register timeout
	TransactionTimeout    time.Duration // transaction timeout
	KeepAliveInterval     time.Duration // keep alive interval
	MaxForwards           int           // max forwards times
	EnableAuthentication  bool          // if enable authentication
	AuthenticationRealm   string        // authentication realm
	EnableTLS             bool          // if enable tls
	TLSCertFile           string        // tls cert file
	TLSKeyFile            string        // tls key file
	LogLevel              string        // log level
	LogFile               string        // log file
	RTPBufferSize         int           // rtp buffer size
	MaxConcurrentSessions int           // max concurrent sessions
	SessionTimeout        time.Duration // session timeout
	NetworkInterface      string        // network interface
	EnableICE             bool          // enable ice
	StorageType           StorageType   // storage type
	Db                    *gorm.DB
	registeredUsers       map[string]string // username -> Contact address (从 REGISTER 请求中获取)
	registerMutex         sync.RWMutex
	StoragePath           string                     // storage path for file
	pendingSessions       map[string]string          // Call-ID -> client RTP address (for memory storage)
	sessionsMutex         sync.RWMutex               // Protects concurrent access to pendingSessions
	memoryCalls           map[string]*models.SipCall // Call-ID -> SipCall (for memory storage)
	memoryCallsMutex      sync.RWMutex               // Protects concurrent access to memoryCalls
	activeSessions        map[string]*SessionInfo    // Call-ID -> session info
	activeMutex           sync.RWMutex
}

type SessionInfo struct {
	ClientRTPAddr *net.UDPAddr
	StopRecording chan bool
	DTMFChannel   chan string // DTMF 按键通道
	RecordingFile string      // 录音文件路径
}

// DefaultUAConfig return default ua config
func DefaultUAConfig() *UAConfig {
	return &UAConfig{
		Host:                  "0.0.0.0",
		Port:                  5060,
		UserAgentName:         DEFAULT_USER_AGENT,
		LocalRTPPort:          10000, // Default RPT Port
		RegisterTimeout:       30 * time.Second,
		TransactionTimeout:    30 * time.Second,
		KeepAliveInterval:     60 * time.Second,
		MaxForwards:           70,
		EnableAuthentication:  false,
		AuthenticationRealm:   DEFAULT_REALM_NAME,
		EnableTLS:             false,
		TLSCertFile:           "",
		TLSKeyFile:            "",
		LogLevel:              "info",
		LogFile:               "",
		RTPBufferSize:         1500, // standard 以太网 MTU Size
		MaxConcurrentSessions: 100,
		SessionTimeout:        10 * time.Minute,
		NetworkInterface:      "",
		EnableICE:             false,
		StorageType:           StorageTypeMemory,
		StoragePath:           "./sip_data",
		registeredUsers:       make(map[string]string),
		pendingSessions:       make(map[string]string),
		memoryCalls:           make(map[string]*models.SipCall),
		activeSessions:        make(map[string]*SessionInfo),
	}
}

// ApplyDefaults applies default values to current config
func (c *UAConfig) ApplyDefaults() {
	defaultConfig := DefaultUAConfig()

	if c.Host == "" {
		c.Host = defaultConfig.Host
	}

	if c.Port == 0 {
		c.Port = defaultConfig.Port
	}

	if c.UserAgentName == "" {
		c.UserAgentName = defaultConfig.UserAgentName
	}

	if c.LocalRTPPort == 0 {
		c.LocalRTPPort = defaultConfig.LocalRTPPort
	}

	if c.RegisterTimeout == 0 {
		c.RegisterTimeout = defaultConfig.RegisterTimeout
	}

	if c.TransactionTimeout == 0 {
		c.TransactionTimeout = defaultConfig.TransactionTimeout
	}

	if c.KeepAliveInterval == 0 {
		c.KeepAliveInterval = defaultConfig.KeepAliveInterval
	}

	if c.MaxForwards == 0 {
		c.MaxForwards = defaultConfig.MaxForwards
	}

	if c.RTPBufferSize == 0 {
		c.RTPBufferSize = defaultConfig.RTPBufferSize
	}

	if c.MaxConcurrentSessions == 0 {
		c.MaxConcurrentSessions = defaultConfig.MaxConcurrentSessions
	}

	if c.SessionTimeout == 0 {
		c.SessionTimeout = defaultConfig.SessionTimeout
	}

	if c.EnableICE == false {
		c.EnableICE = defaultConfig.EnableICE
	}

	if c.StorageType == "" {
		c.StorageType = defaultConfig.StorageType
	}

	if c.StoragePath == "" {
		c.StoragePath = defaultConfig.StoragePath
	}

	// Initialize registeredUsers map if not initialized
	if c.registeredUsers == nil {
		c.registeredUsers = make(map[string]string)
	}

	// Initialize pendingSessions map if not initialized
	if c.pendingSessions == nil {
		c.pendingSessions = make(map[string]string)
	}

	// Initialize memoryCalls map if not initialized
	if c.memoryCalls == nil {
		c.memoryCalls = make(map[string]*models.SipCall)
	}

	// Initialize activeSessions map if not initialized
	if c.activeSessions == nil {
		c.activeSessions = make(map[string]*SessionInfo)
	}
}

// GetSIPAddress returns the full SIP address
func (c *UAConfig) GextSIPAddress() string {
	return c.Host + ":" + string(rune(c.Port))
}

// SetRegisteredUser sets a registered user's contact address
func (c *UAConfig) SetRegisteredUser(username, contact string) {
	c.registerMutex.Lock()
	defer c.registerMutex.Unlock()
	c.registeredUsers[username] = contact
}

// GetRegisteredUser gets a registered user's contact address
func (c *UAConfig) GetRegisteredUser(username string) (string, bool) {
	c.registerMutex.RLock()
	defer c.registerMutex.RUnlock()
	contact, exists := c.registeredUsers[username]
	return contact, exists
}

// RemoveRegisteredUser removes a registered user
func (c *UAConfig) RemoveRegisteredUser(username string) {
	c.registerMutex.Lock()
	defer c.registerMutex.Unlock()
	delete(c.registeredUsers, username)
}

// GetRTPAddress returns the RTP address
func (c *UAConfig) GetRTPAddress() string {
	return c.Host + ":" + string(rune(c.LocalRTPPort))
}

// SetDBConfig set db config
func (as *UAConfig) SetDBConfig(db *gorm.DB) {
	as.Db = db
}

// ConfigError Config Error Type
type ConfigError struct {
	Field   string      // error field
	Value   interface{} // error value
	Message string      // error message
}

func (e *ConfigError) Error() string {
	return "Config Error [" + e.Field + " = " + e.Value.(string) + "]: " + e.Message
}

// Validate validates the configuration
func (c *UAConfig) Validate() error {
	if c.Port <= 0 || c.Port > 65535 {
		return &ConfigError{Field: "Port", Value: c.Port, Message: "Port must be between 1-65535"}
	}

	if c.LocalRTPPort <= 0 || c.LocalRTPPort > 65535 {
		return &ConfigError{Field: "LocalRTPPort", Value: c.LocalRTPPort, Message: "RTP Port must be between 1-65535"}
	}

	if c.TransactionTimeout <= 0 {
		return &ConfigError{Field: "TransactionTimeout", Value: c.TransactionTimeout, Message: "Transaction timeout must be greater than 0"}
	}

	if c.MaxForwards <= 0 {
		return &ConfigError{Field: "MaxForwards", Value: c.MaxForwards, Message: "Max forwards must be greater than 0"}
	}

	return nil
}

// ExtractRegistrationInfo extracts registration information from SIP REGISTER request
func (c *UAConfig) ExtractRegistrationInfo(req *sip.Request) *RegistrationInfo {
	info := &RegistrationInfo{
		Expires: 3600, // Default 1 hour
	}

	// Extract username from From header
	if from := req.From(); from != nil {
		info.Username = from.Address.User
	}

	// Extract contact information
	if contact := req.Contact(); contact != nil {
		info.ContactStr = contact.Address.String()
		info.ContactIP = contact.Address.Host
		info.ContactPort = contact.Address.Port
		if info.ContactPort == 0 {
			info.ContactPort = 5060 // Default SIP port
		}
	}

	// Extract expires from request
	if expiresHeader := req.GetHeader("Expires"); expiresHeader != nil {
		if expiresValue, err := strconv.Atoi(expiresHeader.Value()); err == nil {
			info.Expires = expiresValue
		}
	}

	// Extract User-Agent
	if uaHeader := req.GetHeader("User-Agent"); uaHeader != nil {
		info.UserAgent = uaHeader.Value()
	}

	// Extract remote IP from Via header
	if via := req.Via(); via != nil {
		if received, exists := via.Params.Get("received"); exists && received != "" {
			info.RemoteIP = received
		} else if via.Host != "" {
			info.RemoteIP = via.Host
		}
	}

	return info
}
