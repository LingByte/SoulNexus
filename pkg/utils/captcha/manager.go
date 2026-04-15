package captcha

import (
	"fmt"
	"time"
)

// Manager 统一验证码管理器
type Manager struct {
	imageCaptcha *ImageCaptcha
	clickCaptcha *ClickCaptcha
	store        Store
}

// Config 验证码配置
type Config struct {
	// 图形验证码配置
	ImageWidth  int
	ImageHeight int
	ImageLength int

	// 点击验证码配置
	ClickWidth     int
	ClickHeight    int
	ClickCount     int
	ClickTolerance int

	// 通用配置
	Expiration time.Duration
	Store      Store
}

// DefaultConfig 默认配置
func DefaultConfig() *Config {
	return &Config{
		ImageWidth:     200,
		ImageHeight:    60,
		ImageLength:    4,
		ClickWidth:     300,
		ClickHeight:    200,
		ClickCount:     3,
		ClickTolerance: 30, // 提高容差，允许更大的点击误差
		Expiration:     5 * time.Minute,
		Store:          nil, // 使用默认内存存储
	}
}

// NewManager 创建统一验证码管理器
func NewManager(config *Config) *Manager {
	if config == nil {
		config = DefaultConfig()
	}

	store := config.Store
	if store == nil {
		store = NewMemoryStore()
	}

	return &Manager{
		imageCaptcha: NewImageCaptcha(config.ImageWidth, config.ImageHeight, config.ImageLength, config.Expiration, store),
		clickCaptcha: NewClickCaptcha(config.ClickWidth, config.ClickHeight, config.ClickCount, config.ClickTolerance, config.Expiration, store),
		store:        store,
	}
}

// GenerateImage 生成图形验证码
func (m *Manager) GenerateImage() (*Result, error) {
	return m.imageCaptcha.Generate()
}

// VerifyImage 验证图形验证码
func (m *Manager) VerifyImage(id, code string) (bool, error) {
	return m.imageCaptcha.Verify(id, code)
}

// GenerateClick 生成点击验证码
func (m *Manager) GenerateClick() (*Result, error) {
	return m.clickCaptcha.Generate()
}

// VerifyClick 验证点击验证码
func (m *Manager) VerifyClick(id string, positions []Point) (bool, error) {
	return m.clickCaptcha.Verify(id, positions)
}

// Generate 根据类型生成验证码
func (m *Manager) Generate(captchaType Type) (*Result, error) {
	switch captchaType {
	case TypeImage:
		return m.GenerateImage()
	case TypeClick:
		return m.GenerateClick()
	default:
		return nil, fmt.Errorf("unsupported captcha type: %s", captchaType)
	}
}

// Verify 根据类型验证验证码
func (m *Manager) Verify(captchaType Type, id string, data interface{}) (bool, error) {
	switch captchaType {
	case TypeImage:
		if code, ok := data.(string); ok {
			return m.VerifyImage(id, code)
		}
		return false, fmt.Errorf("invalid data type for image captcha, expected string")
	case TypeClick:
		if positions, ok := data.([]Point); ok {
			return m.VerifyClick(id, positions)
		}
		return false, fmt.Errorf("invalid data type for click captcha, expected []Point")
	default:
		return false, fmt.Errorf("unsupported captcha type: %s", captchaType)
	}
}

// VerifyImageWithoutDelete 验证图形验证码但不删除
func (m *Manager) VerifyImageWithoutDelete(id, code string) (bool, error) {
	return m.imageCaptcha.VerifyWithoutDelete(id, code)
}

// VerifyWithoutDelete 根据类型验证验证码但不删除
func (m *Manager) VerifyWithoutDelete(captchaType Type, id string, data interface{}) (bool, error) {
	switch captchaType {
	case TypeImage:
		if code, ok := data.(string); ok {
			return m.VerifyImageWithoutDelete(id, code)
		}
		return false, fmt.Errorf("invalid data type for image captcha, expected string")
	case TypeClick:
		// 尝试从 []map[string]interface{} 或 []Point 转换
		if positions, ok := data.([]Point); ok {
			return m.clickCaptcha.VerifyWithoutDelete(id, positions)
		}
		if positions, ok := data.([]interface{}); ok {
			points := make([]Point, 0, len(positions))
			for _, pos := range positions {
				if posMap, ok := pos.(map[string]interface{}); ok {
					var x, y int
					if xVal, ok := posMap["x"]; ok {
						if xFloat, ok := xVal.(float64); ok {
							x = int(xFloat)
						} else if xInt, ok := xVal.(int); ok {
							x = xInt
						}
					}
					if yVal, ok := posMap["y"]; ok {
						if yFloat, ok := yVal.(float64); ok {
							y = int(yFloat)
						} else if yInt, ok := yVal.(int); ok {
							y = yInt
						}
					}
					points = append(points, Point{X: x, Y: y})
				}
			}
			return m.clickCaptcha.VerifyWithoutDelete(id, points)
		}
		return false, fmt.Errorf("invalid data type for click captcha")
	default:
		return false, fmt.Errorf("unsupported captcha type: %s", captchaType)
	}
}

// GlobalManager 全局验证码管理器
var GlobalManager *Manager

// InitGlobalManager 初始化全局验证码管理器
func InitGlobalManager(config *Config) {
	GlobalManager = NewManager(config)
}
