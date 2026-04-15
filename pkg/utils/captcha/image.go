package captcha

import (
	"bytes"
	"encoding/base64"
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

// ImageCaptcha 图形验证码
type ImageCaptcha struct {
	width      int
	height     int
	length     int
	expiration time.Duration
	store      Store
}

// NewImageCaptcha 创建图形验证码管理器
func NewImageCaptcha(width, height, length int, expiration time.Duration, store Store) *ImageCaptcha {
	if store == nil {
		store = NewMemoryStore()
	}
	return &ImageCaptcha{
		width:      width,
		height:     height,
		length:     length,
		expiration: expiration,
		store:      store,
	}
}

// Generate 生成图形验证码
func (ic *ImageCaptcha) Generate() (*Result, error) {
	// 生成随机验证码
	code := ic.generateCode()

	// 生成图片
	img, err := ic.generateImage(code)
	if err != nil {
		return nil, fmt.Errorf("failed to generate image: %w", err)
	}

	// 转换为base64
	imgBase64, err := ic.imageToBase64(img)
	if err != nil {
		return nil, fmt.Errorf("failed to encode image: %w", err)
	}

	// 生成ID
	id := ic.generateID()

	// 存储验证码
	expires := time.Now().Add(ic.expiration)
	if err := ic.store.Set(id, code, expires); err != nil {
		return nil, fmt.Errorf("failed to store captcha: %w", err)
	}

	return &Result{
		ID:      id,
		Type:    TypeImage,
		Data:    map[string]interface{}{"image": imgBase64, "code": code},
		Expires: expires,
	}, nil
}

// Verify 验证图形验证码
func (ic *ImageCaptcha) Verify(id, code string) (bool, error) {
	return ic.store.VerifyWithFunc(id, code, func(stored, input interface{}) bool {
		storedStr, ok1 := stored.(string)
		inputStr, ok2 := input.(string)
		if !ok1 || !ok2 {
			return false
		}
		return strings.ToLower(storedStr) == strings.ToLower(inputStr)
	})
}

// VerifyWithoutDelete 验证图形验证码但不删除（用于预验证）
func (ic *ImageCaptcha) VerifyWithoutDelete(id, code string) (bool, error) {
	return ic.store.VerifyWithFuncWithoutDelete(id, code, func(stored, input interface{}) bool {
		storedStr, ok1 := stored.(string)
		inputStr, ok2 := input.(string)
		if !ok1 || !ok2 {
			return false
		}
		return strings.ToLower(storedStr) == strings.ToLower(inputStr)
	})
}

// generateCode 生成随机验证码
func (ic *ImageCaptcha) generateCode() string {
	// 使用数字和字母（排除容易混淆的字符：0, O, I, 1, l）
	chars := "23456789ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijkmnopqrstuvwxyz"
	rand.Seed(time.Now().UnixNano())

	var code strings.Builder
	for i := 0; i < ic.length; i++ {
		code.WriteByte(chars[rand.Intn(len(chars))])
	}
	return code.String()
}

// generateID 生成验证码ID
func (ic *ImageCaptcha) generateID() string {
	rand.Seed(time.Now().UnixNano())
	chars := "0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
	var id strings.Builder
	for i := 0; i < 32; i++ {
		id.WriteByte(chars[rand.Intn(len(chars))])
	}
	return id.String()
}

// generateImage 生成验证码图片
func (ic *ImageCaptcha) generateImage(code string) (image.Image, error) {
	// 创建图片
	img := image.NewRGBA(image.Rect(0, 0, ic.width, ic.height))

	// 填充背景色（浅色）
	bgColor := color.RGBA{240, 240, 240, 255}
	for y := 0; y < ic.height; y++ {
		for x := 0; x < ic.width; x++ {
			img.Set(x, y, bgColor)
		}
	}

	// 添加干扰线
	rand.Seed(time.Now().UnixNano())
	for i := 0; i < 5; i++ {
		x1 := rand.Intn(ic.width)
		y1 := rand.Intn(ic.height)
		x2 := rand.Intn(ic.width)
		y2 := rand.Intn(ic.height)
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
		x := rand.Intn(ic.width)
		y := rand.Intn(ic.height)
		dotColor := color.RGBA{
			uint8(rand.Intn(200)),
			uint8(rand.Intn(200)),
			uint8(rand.Intn(200)),
			255,
		}
		img.Set(x, y, dotColor)
	}

	// 绘制文字
	if err := ic.drawText(img, code); err != nil {
		return nil, err
	}

	return img, nil
}

// drawText 绘制文字
func (ic *ImageCaptcha) drawText(img *image.RGBA, text string) error {
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
	charWidth := float64(ic.width) / float64(len(text))
	y := float64(ic.height)/2 + 12 // 垂直居中

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
func (ic *ImageCaptcha) imageToBase64(img image.Image) (string, error) {
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		return "", err
	}
	return "data:image/png;base64," + base64.StdEncoding.EncodeToString(buf.Bytes()), nil
}
