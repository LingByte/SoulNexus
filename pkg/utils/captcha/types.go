package captcha

import "time"

// Type is the captcha challenge kind.
type Type string

const (
	TypeImage  Type = "image"  // distorted text image
	TypeClick  Type = "click"  // ordered click on characters
	TypeSlider Type = "slider" // drag slider to the end
	TypeRandom Type = "random"
)

// Result is returned when a captcha challenge is created.
type Result struct {
	ID      string                 `json:"id"`
	Type    Type                   `json:"type"`
	Data    map[string]interface{} `json:"data"`
	Expires time.Time              `json:"expires"`
}

// Point is a click coordinate in logical pixels.
type Point struct {
	X int `json:"x"`
	Y int `json:"y"`
}

// CharMarker is one character rendered on the click-captcha canvas.
type CharMarker struct {
	Char string `json:"char"`
	X    int    `json:"x"`
	Y    int    `json:"y"`
}

// Payload is the client proof submitted with protected actions.
type Payload struct {
	ID    string      `json:"captchaId"`
	Type  Type        `json:"captchaType"`
	Value interface{} `json:"captchaValue"`
}

// CaptchaFields is embedded in public auth requests that require human verification.
type CaptchaFields struct {
	CaptchaID    string      `json:"captchaId"`
	CaptchaType  string      `json:"captchaType"`
	CaptchaValue interface{} `json:"captchaValue"`
}
