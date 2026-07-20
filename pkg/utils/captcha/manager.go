package captcha

import (
	"errors"
	"fmt"
	"strings"
	"time"
)

// Sentinel errors returned by ValidatePayload so callers can map to i18n responses.
var (
	ErrPayloadRequired = errors.New("captcha: id and type are required")
	ErrPayloadInvalid  = errors.New("captcha: verification failed")
)

// ValidatePayload trims and validates a captcha proof without touching any HTTP context.
// Returns nil on success, or ErrPayloadRequired / ErrPayloadInvalid so the caller can
// decide how to surface the error.
func ValidatePayload(id, typ string, value interface{}) error {
	payload := Payload{
		ID:    strings.TrimSpace(id),
		Type:  Type(strings.TrimSpace(typ)),
		Value: value,
	}
	if payload.ID == "" || payload.Type == "" {
		return ErrPayloadRequired
	}
	ok, err := VerifyPayload(payload)
	if err != nil || !ok {
		return ErrPayloadInvalid
	}
	return nil
}

// Manager is the unified captcha manager.
type Manager struct {
	imageCaptcha  *ImageCaptcha
	clickCaptcha  *ClickCaptcha
	sliderCaptcha *SliderCaptcha
	store         Store
}

// Config holds captcha settings.
type Config struct {
	ImageWidth  int
	ImageHeight int
	ImageLength int

	ClickWidth     int
	ClickHeight    int
	ClickCount     int
	ClickTolerance int

	SliderTrackWidth int
	SliderPassRatio  float64

	Expiration time.Duration
	Store      Store
}

// DefaultConfig returns sensible defaults.
func DefaultConfig() *Config {
	return &Config{
		ImageWidth:       200,
		ImageHeight:      60,
		ImageLength:      4,
		ClickWidth:       300,
		ClickHeight:      200,
		ClickCount:       3,
		ClickTolerance:   30,
		SliderTrackWidth: defaultSliderTrackWidth,
		SliderPassRatio:  0.92,
		Expiration:       5 * time.Minute,
	}
}

// NewManager creates a captcha manager.
func NewManager(config *Config) *Manager {
	if config == nil {
		config = DefaultConfig()
	}
	store := config.Store
	if store == nil {
		store = NewMemoryStore()
	}
	return &Manager{
		imageCaptcha:  NewImageCaptcha(config.ImageWidth, config.ImageHeight, config.ImageLength, config.Expiration, store),
		clickCaptcha:  NewClickCaptcha(config.ClickWidth, config.ClickHeight, config.ClickCount, config.ClickTolerance, config.Expiration, store),
		sliderCaptcha: NewSliderCaptcha(config.SliderTrackWidth, config.SliderPassRatio, config.Expiration, store),
		store:         store,
	}
}

// GenerateRandom creates a captcha using RandomType (slider, image, or click).
func (m *Manager) GenerateRandom() (*Result, error) {
	return m.Generate(RandomType())
}

// Generate creates a captcha of the given type.
func (m *Manager) Generate(captchaType Type) (*Result, error) {
	switch captchaType {
	case TypeImage:
		return m.imageCaptcha.Generate()
	case TypeClick:
		return m.clickCaptcha.Generate()
	case TypeSlider:
		return m.sliderCaptcha.Generate()
	default:
		return nil, fmt.Errorf("unsupported captcha type: %s", captchaType)
	}
}

// Verify validates a captcha proof and consumes it on success.
func (m *Manager) Verify(p Payload) (bool, error) {
	switch p.Type {
	case TypeImage:
		code, ok := p.Value.(string)
		if !ok {
			return false, fmt.Errorf("invalid captcha value for image")
		}
		return m.imageCaptcha.Verify(p.ID, code)
	case TypeClick:
		positions, err := parsePoints(p.Value)
		if err != nil {
			return false, err
		}
		return m.clickCaptcha.Verify(p.ID, positions)
	case TypeSlider:
		return m.sliderCaptcha.Verify(p.ID, intValue(p.Value))
	default:
		return false, fmt.Errorf("unsupported captcha type: %s", p.Type)
	}
}

// GlobalManager is the process-wide captcha manager.
var GlobalManager *Manager

// InitGlobalManager initializes GlobalManager once.
func InitGlobalManager(config *Config) {
	GlobalManager = NewManager(config)
}

// EnsureGlobalManager lazily initializes GlobalManager.
func EnsureGlobalManager() *Manager {
	if GlobalManager == nil {
		InitGlobalManager(DefaultConfig())
	}
	return GlobalManager
}

// VerifyPayload validates using GlobalManager.
func VerifyPayload(p Payload) (bool, error) {
	if p.ID == "" || p.Type == "" {
		return false, fmt.Errorf("captcha required")
	}
	return EnsureGlobalManager().Verify(p)
}

func parsePoints(v interface{}) ([]Point, error) {
	switch arr := v.(type) {
	case []Point:
		return arr, nil
	case []interface{}:
		out := make([]Point, 0, len(arr))
		for _, item := range arr {
			switch p := item.(type) {
			case map[string]interface{}:
				out = append(out, Point{X: intValue(p["x"]), Y: intValue(p["y"])})
			case Point:
				out = append(out, p)
			default:
				return nil, fmt.Errorf("invalid click point")
			}
		}
		return out, nil
	default:
		return nil, fmt.Errorf("invalid captcha value for click")
	}
}
