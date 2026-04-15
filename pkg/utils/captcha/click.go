package captcha

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"math/rand"
	"time"

	"github.com/golang/freetype"
	"github.com/golang/freetype/truetype"
	"golang.org/x/image/font/gofont/goregular"
)

// ClickCaptcha 点击验证码
type ClickCaptcha struct {
	width      int
	height     int
	count      int // 需要点击的数量
	tolerance  int // 容差（像素）
	expiration time.Duration
	store      Store
}

// NewClickCaptcha 创建点击验证码管理器
func NewClickCaptcha(width, height, count, tolerance int, expiration time.Duration, store Store) *ClickCaptcha {
	if store == nil {
		store = NewMemoryStore()
	}
	if count <= 0 {
		count = 3 // 默认3个点击点
	}
	return &ClickCaptcha{
		width:      width,
		height:     height,
		count:      count,
		tolerance:  tolerance,
		expiration: expiration,
		store:      store,
	}
}

// Generate 生成点击验证码
func (cc *ClickCaptcha) Generate() (*Result, error) {
	// 生成随机文字
	words := cc.generateWords(cc.count)

	// 生成图片和点击位置
	img, positions, err := cc.generateImageWithPositions(words)
	if err != nil {
		return nil, fmt.Errorf("failed to generate image: %w", err)
	}

	// 转换为base64
	imgBase64, err := cc.imageToBase64(img)
	if err != nil {
		return nil, fmt.Errorf("failed to encode image: %w", err)
	}

	// 生成ID
	id := cc.generateID()

	// 存储正确位置
	expires := time.Now().Add(cc.expiration)
	if err := cc.store.Set(id, positions, expires); err != nil {
		return nil, fmt.Errorf("failed to store captcha: %w", err)
	}

	return &Result{
		ID:      id,
		Type:    TypeClick,
		Data: map[string]interface{}{
			"image":     imgBase64,
			"positions": positions,
			"words":     words,
			"count":     cc.count,
			"tolerance": cc.tolerance,
		},
		Expires: expires,
	}, nil
}

// Verify 验证点击验证码
func (cc *ClickCaptcha) Verify(id string, userPositions []Point) (bool, error) {
	return cc.store.VerifyWithFunc(id, userPositions, func(stored, input interface{}) bool {
		storedPositions, ok1 := stored.([]Point)
		inputPositions, ok2 := input.([]Point)
		if !ok1 || !ok2 {
			return false
		}

		// 检查数量是否匹配
		if len(storedPositions) != len(inputPositions) {
			return false
		}

		// 检查每个位置是否在容差范围内
		// 使用最优匹配算法：找到所有可能的匹配组合，选择最优的
		// 对于每个存储的位置，找到所有在容差范围内的输入位置
		toleranceSquared := cc.tolerance * cc.tolerance

		// 计算所有可能的匹配（存储位置 -> 输入位置的距离）
		matches := make([][]int, len(storedPositions))
		for i, storedPos := range storedPositions {
			matches[i] = make([]int, 0)
			for j, inputPos := range inputPositions {
				dx := abs(inputPos.X - storedPos.X)
				dy := abs(inputPos.Y - storedPos.Y)
				distanceSquared := dx*dx + dy*dy
				if distanceSquared <= toleranceSquared {
					matches[i] = append(matches[i], j)
				}
			}
			// 如果某个存储位置没有任何匹配，直接返回false
			if len(matches[i]) == 0 {
				return false
			}
		}

		// 使用贪心算法找到匹配：优先匹配距离最近的点
		matched := make([]bool, len(inputPositions))
		for i, storedPos := range storedPositions {
			found := false
			bestMatch := -1
			bestDistance := toleranceSquared + 1

			// 找到最近的未匹配输入位置
			for _, j := range matches[i] {
				if !matched[j] {
					dx := abs(inputPositions[j].X - storedPos.X)
					dy := abs(inputPositions[j].Y - storedPos.Y)
					distanceSquared := dx*dx + dy*dy
					if distanceSquared < bestDistance {
						bestDistance = distanceSquared
						bestMatch = j
					}
				}
			}

			if bestMatch >= 0 {
				matched[bestMatch] = true
				found = true
			}

			if !found {
				return false
			}
		}

		return true
	})
}

// VerifyWithoutDelete 验证点击验证码但不删除（用于预验证）
func (cc *ClickCaptcha) VerifyWithoutDelete(id string, userPositions []Point) (bool, error) {
	return cc.store.VerifyWithFuncWithoutDelete(id, userPositions, func(stored, input interface{}) bool {
		storedPositions, ok1 := stored.([]Point)
		inputPositions, ok2 := input.([]Point)
		if !ok1 || !ok2 {
			return false
		}

		// 检查数量是否匹配
		if len(storedPositions) != len(inputPositions) {
			return false
		}

		// 检查每个位置是否在容差范围内
		// 使用最优匹配算法：找到所有可能的匹配组合，选择最优的
		// 对于每个存储的位置，找到所有在容差范围内的输入位置
		toleranceSquared := cc.tolerance * cc.tolerance

		// 计算所有可能的匹配（存储位置 -> 输入位置的距离）
		matches := make([][]int, len(storedPositions))
		for i, storedPos := range storedPositions {
			matches[i] = make([]int, 0)
			for j, inputPos := range inputPositions {
				dx := abs(inputPos.X - storedPos.X)
				dy := abs(inputPos.Y - storedPos.Y)
				distanceSquared := dx*dx + dy*dy
				if distanceSquared <= toleranceSquared {
					matches[i] = append(matches[i], j)
				}
			}
			// 如果某个存储位置没有任何匹配，直接返回false
			if len(matches[i]) == 0 {
				return false
			}
		}

		// 使用贪心算法找到匹配：优先匹配距离最近的点
		matched := make([]bool, len(inputPositions))
		for i, storedPos := range storedPositions {
			found := false
			bestMatch := -1
			bestDistance := toleranceSquared + 1

			// 找到最近的未匹配输入位置
			for _, j := range matches[i] {
				if !matched[j] {
					dx := abs(inputPositions[j].X - storedPos.X)
					dy := abs(inputPositions[j].Y - storedPos.Y)
					distanceSquared := dx*dx + dy*dy
					if distanceSquared < bestDistance {
						bestDistance = distanceSquared
						bestMatch = j
					}
				}
			}

			if bestMatch >= 0 {
				matched[bestMatch] = true
				found = true
			}

			if !found {
				return false
			}
		}

		return true
	})
}

// generateWords 生成随机文字
func (cc *ClickCaptcha) generateWords(count int) []string {
	// 验证码候选字符（使用字体稳定支持的 ASCII，避免出现方块字）
	chars := []string{
		"A", "B", "C", "D", "E", "F", "G", "H", "J", "K",
		"M", "N", "P", "Q", "R", "T", "U", "V", "W", "X",
		"Y", "Z", "2", "3", "4", "5", "6", "7", "8", "9",
	}
	rand.Seed(time.Now().UnixNano())

	words := make([]string, count)
	used := make(map[string]bool)
	for i := 0; i < count; i++ {
		for {
			word := chars[rand.Intn(len(chars))]
			if !used[word] {
				words[i] = word
				used[word] = true
				break
			}
		}
	}
	return words
}

// generateImageWithPositions 生成图片和点击位置
func (cc *ClickCaptcha) generateImageWithPositions(words []string) (image.Image, []Point, error) {
	// 创建图片
	img := image.NewRGBA(image.Rect(0, 0, cc.width, cc.height))

	// 填充背景色
	bgColor := color.RGBA{245, 245, 245, 255}
	for y := 0; y < cc.height; y++ {
		for x := 0; x < cc.width; x++ {
			img.Set(x, y, bgColor)
		}
	}

	// 添加干扰线
	rand.Seed(time.Now().UnixNano())
	for i := 0; i < 8; i++ {
		x1 := rand.Intn(cc.width)
		y1 := rand.Intn(cc.height)
		x2 := rand.Intn(cc.width)
		y2 := rand.Intn(cc.height)
		lineColor := color.RGBA{
			uint8(rand.Intn(200)),
			uint8(rand.Intn(200)),
			uint8(rand.Intn(200)),
			255,
		}
		drawLine(img, x1, y1, x2, y2, lineColor)
	}

	// 添加干扰点
	for i := 0; i < 100; i++ {
		x := rand.Intn(cc.width)
		y := rand.Intn(cc.height)
		dotColor := color.RGBA{
			uint8(rand.Intn(200)),
			uint8(rand.Intn(200)),
			uint8(rand.Intn(200)),
			255,
		}
		img.Set(x, y, dotColor)
	}

	// 加载字体
	fontData, err := truetype.Parse(goregular.TTF)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to parse font: %w", err)
	}

	c := freetype.NewContext()
	c.SetDPI(72)
	c.SetFont(fontData)
	c.SetFontSize(28)
	c.SetClip(img.Bounds())
	c.SetDst(img)

	// 计算每个文字的位置（随机分布，但确保不重叠）
	positions := make([]Point, len(words))
	usedAreas := make([]image.Rectangle, 0)

	for i, word := range words {
		var x, y int
		var rect image.Rectangle
		maxAttempts := 50

		for attempt := 0; attempt < maxAttempts; attempt++ {
			// 随机位置，但确保文字不会超出边界
			x = rand.Intn(cc.width-60) + 30
			y = rand.Intn(cc.height-40) + 30

			// 估算文字区域（大约30x30像素）
			rect = image.Rect(x-15, y-15, x+45, y+15)

			// 检查是否与已有位置重叠
			overlap := false
			for _, usedRect := range usedAreas {
				if rect.Overlaps(usedRect) {
					overlap = true
					break
				}
			}

			if !overlap {
				break
			}
		}

		positions[i] = Point{X: x, Y: y}
		usedAreas = append(usedAreas, rect)

		// 随机颜色
		textColor := color.RGBA{
			uint8(rand.Intn(100) + 30),
			uint8(rand.Intn(100) + 30),
			uint8(rand.Intn(100) + 30),
			255,
		}

		c.SetSrc(&image.Uniform{textColor})

		// 绘制文字
		pt := freetype.Pt(x, y)
		_, err := c.DrawString(word, pt)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to draw text: %w", err)
		}
	}

	return img, positions, nil
}

// imageToBase64 将图片转换为base64
func (cc *ClickCaptcha) imageToBase64(img image.Image) (string, error) {
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		return "", err
	}
	return "data:image/png;base64," + base64.StdEncoding.EncodeToString(buf.Bytes()), nil
}

// generateID 生成验证码ID
func (cc *ClickCaptcha) generateID() string {
	rand.Seed(time.Now().UnixNano())
	chars := "0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
	var id bytes.Buffer
	for i := 0; i < 32; i++ {
		id.WriteByte(chars[rand.Intn(len(chars))])
	}
	return id.String()
}
