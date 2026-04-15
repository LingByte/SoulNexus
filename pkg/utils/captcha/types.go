package captcha

import "time"

// Type 验证码类型
type Type string

const (
	TypeImage Type = "image" // 图形验证码
	TypeClick Type = "click" // 点击验证码
)

// Result 验证码生成结果
type Result struct {
	ID      string                 `json:"id"`      // 验证码ID
	Type    Type                   `json:"type"`    // 验证码类型
	Data    map[string]interface{} `json:"data"`    // 验证码数据（根据类型不同而不同）
	Expires time.Time              `json:"expires"` // 过期时间
}

// ImageCaptchaData 图形验证码数据
type ImageCaptchaData struct {
	Image string `json:"image"` // Base64编码的图片
	Code  string `json:"code"`  // 验证码内容（仅用于测试，生产环境不应返回）
}

// ClickCaptchaData 点击验证码数据
type ClickCaptchaData struct {
	Image     string  `json:"image"`     // 图片Base64
	Positions []Point `json:"positions"` // 需要点击的位置列表
	Count     int     `json:"count"`     // 需要点击的数量
	Tolerance int     `json:"tolerance"` // 容差（像素）
}

// Point 坐标点
type Point struct {
	X int `json:"x"`
	Y int `json:"y"`
}
