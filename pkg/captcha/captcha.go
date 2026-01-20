package captcha

import (
	"bytes"
	"encoding/base64"
	"errors"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"math/rand"
	"strings"
	"time"

	"github.com/golang/freetype"
	"github.com/golang/freetype/truetype"
	"golang.org/x/image/font/gofont/goregular"
)

// Captcha 验证码结构
type Captcha struct {
	ID      string    // 验证码ID
	Code    string    // 验证码内容
	Image   string    // Base64编码的图片
	Expires time.Time // 过期时间
}

// CaptchaManager 验证码管理器
type CaptchaManager struct {
	width      int
	height     int
	length     int
	expiration time.Duration
	store      CaptchaStore
}

// CaptchaStore 验证码存储接口
type CaptchaStore interface {
	Set(id, code string, expires time.Time) error
	Get(id string) (string, error)
	Delete(id string) error
	Verify(id, code string) (bool, error)
}

// MemoryCaptchaStore 内存存储实现
type MemoryCaptchaStore struct {
	data map[string]captchaData
}

type captchaData struct {
	code    string
	expires time.Time
}

// NewMemoryCaptchaStore 创建内存存储
func NewMemoryCaptchaStore() *MemoryCaptchaStore {
	return &MemoryCaptchaStore{
		data: make(map[string]captchaData),
	}
}

func (s *MemoryCaptchaStore) Set(id, code string, expires time.Time) error {
	s.data[id] = captchaData{
		code:    code,
		expires: expires,
	}
	// 清理过期数据
	go s.cleanup()
	return nil
}

func (s *MemoryCaptchaStore) Get(id string) (string, error) {
	data, ok := s.data[id]
	if !ok {
		return "", errors.New("captcha not found")
	}
	if time.Now().After(data.expires) {
		delete(s.data, id)
		return "", errors.New("captcha expired")
	}
	return data.code, nil
}

func (s *MemoryCaptchaStore) Delete(id string) error {
	delete(s.data, id)
	return nil
}

func (s *MemoryCaptchaStore) Verify(id, code string) (bool, error) {
	storedCode, err := s.Get(id)
	if err != nil {
		return false, err
	}
	// 不区分大小写比较
	if strings.ToLower(storedCode) == strings.ToLower(code) {
		s.Delete(id) // 验证成功后删除
		return true, nil
	}
	return false, nil
}

func (s *MemoryCaptchaStore) cleanup() {
	now := time.Now()
	for id, data := range s.data {
		if now.After(data.expires) {
			delete(s.data, id)
		}
	}
}

// NewCaptchaManager 创建验证码管理器
func NewCaptchaManager(width, height, length int, expiration time.Duration, store CaptchaStore) *CaptchaManager {
	if store == nil {
		store = NewMemoryCaptchaStore()
	}
	return &CaptchaManager{
		width:      width,
		height:     height,
		length:     length,
		expiration: expiration,
		store:      store,
	}
}

// Generate 生成验证码
func (cm *CaptchaManager) Generate() (*Captcha, error) {
	// 生成随机验证码
	code := cm.generateCode()

	// 生成图片
	img, err := cm.generateImage(code)
	if err != nil {
		return nil, fmt.Errorf("failed to generate image: %w", err)
	}

	// 转换为base64
	imgBase64, err := cm.imageToBase64(img)
	if err != nil {
		return nil, fmt.Errorf("failed to encode image: %w", err)
	}

	// 生成ID
	id := cm.generateID()

	// 存储验证码
	expires := time.Now().Add(cm.expiration)
	if err := cm.store.Set(id, code, expires); err != nil {
		return nil, fmt.Errorf("failed to store captcha: %w", err)
	}

	return &Captcha{
		ID:      id,
		Code:    code, // 仅用于测试，生产环境不应返回
		Image:   imgBase64,
		Expires: expires,
	}, nil
}

// Verify 验证验证码
func (cm *CaptchaManager) Verify(id, code string) (bool, error) {
	return cm.store.Verify(id, code)
}

// generateCode 生成随机验证码
func (cm *CaptchaManager) generateCode() string {
	// 使用数字和字母（排除容易混淆的字符：0, O, I, 1, l）
	chars := "23456789ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijkmnopqrstuvwxyz"
	rand.Seed(time.Now().UnixNano())

	var code strings.Builder
	for i := 0; i < cm.length; i++ {
		code.WriteByte(chars[rand.Intn(len(chars))])
	}
	return code.String()
}

// generateID 生成验证码ID
func (cm *CaptchaManager) generateID() string {
	rand.Seed(time.Now().UnixNano())
	chars := "0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
	var id strings.Builder
	for i := 0; i < 32; i++ {
		id.WriteByte(chars[rand.Intn(len(chars))])
	}
	return id.String()
}

// generateImage 生成验证码图片
func (cm *CaptchaManager) generateImage(code string) (image.Image, error) {
	// 创建图片
	img := image.NewRGBA(image.Rect(0, 0, cm.width, cm.height))

	// 填充背景色（浅色）
	bgColor := color.RGBA{240, 240, 240, 255}
	for y := 0; y < cm.height; y++ {
		for x := 0; x < cm.width; x++ {
			img.Set(x, y, bgColor)
		}
	}

	// 添加干扰线
	rand.Seed(time.Now().UnixNano())
	for i := 0; i < 5; i++ {
		x1 := rand.Intn(cm.width)
		y1 := rand.Intn(cm.height)
		x2 := rand.Intn(cm.width)
		y2 := rand.Intn(cm.height)
		lineColor := color.RGBA{
			uint8(rand.Intn(200)),
			uint8(rand.Intn(200)),
			uint8(rand.Intn(200)),
			255,
		}
		drawLine(img, x1, y1, x2, y2, lineColor)
	}

	// 添加干扰点
	for i := 0; i < 50; i++ {
		x := rand.Intn(cm.width)
		y := rand.Intn(cm.height)
		dotColor := color.RGBA{
			uint8(rand.Intn(200)),
			uint8(rand.Intn(200)),
			uint8(rand.Intn(200)),
			255,
		}
		img.Set(x, y, dotColor)
	}

	// 绘制文字
	if err := cm.drawText(img, code); err != nil {
		return nil, err
	}

	return img, nil
}

// drawText 绘制文字
func (cm *CaptchaManager) drawText(img *image.RGBA, text string) error {
	// 加载字体
	fontData, err := truetype.Parse(goregular.TTF)
	if err != nil {
		return fmt.Errorf("failed to parse font: %w", err)
	}

	c := freetype.NewContext()
	c.SetDPI(72)
	c.SetFont(fontData)
	c.SetFontSize(32)
	c.SetClip(img.Bounds())
	c.SetDst(img)
	c.SetSrc(image.Black)

	// 计算文字位置（居中）
	charWidth := float64(cm.width) / float64(len(text))
	y := float64(cm.height)/2 + 12 // 垂直居中

	rand.Seed(time.Now().UnixNano())

	// 绘制每个字符
	for i, char := range text {
		x := float64(i)*charWidth + charWidth/2 - 8

		// 随机颜色
		textColor := color.RGBA{
			uint8(rand.Intn(100) + 50),
			uint8(rand.Intn(100) + 50),
			uint8(rand.Intn(100) + 50),
			255,
		}

		c.SetSrc(&image.Uniform{textColor})

		// 绘制字符
		pt := freetype.Pt(int(x), int(y))
		_, err := c.DrawString(string(char), pt)
		if err != nil {
			return fmt.Errorf("failed to draw text: %w", err)
		}
	}

	return nil
}

// drawLine 绘制直线
func drawLine(img *image.RGBA, x1, y1, x2, y2 int, c color.Color) {
	dx := abs(x2 - x1)
	dy := abs(y2 - y1)
	sx := 1
	if x1 > x2 {
		sx = -1
	}
	sy := 1
	if y1 > y2 {
		sy = -1
	}
	err := dx - dy

	x, y := x1, y1
	for {
		img.Set(x, y, c)
		if x == x2 && y == y2 {
			break
		}
		e2 := 2 * err
		if e2 > -dy {
			err -= dy
			x += sx
		}
		if e2 < dx {
			err += dx
			y += sy
		}
	}
}

func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

// imageToBase64 将图片转换为base64
func (cm *CaptchaManager) imageToBase64(img image.Image) (string, error) {
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		return "", err
	}
	return "data:image/png;base64," + base64.StdEncoding.EncodeToString(buf.Bytes()), nil
}

// GlobalCaptchaManager 全局验证码管理器
var GlobalCaptchaManager *CaptchaManager

// InitGlobalCaptchaManager 初始化全局验证码管理器
func InitGlobalCaptchaManager(store CaptchaStore) {
	GlobalCaptchaManager = NewCaptchaManager(200, 60, 4, 5*time.Minute, store)
}
