package captcha

import (
	"fmt"
	"time"
)

const defaultSliderTrackWidth = 300

// SliderCaptcha is a drag-to-end slider challenge.
type SliderCaptcha struct {
	trackWidth int
	passRatio  float64 // minimum x / trackWidth to pass (0..1)
	expiration time.Duration
	store      Store
}

// NewSliderCaptcha creates a slider captcha manager.
func NewSliderCaptcha(trackWidth int, passRatio float64, expiration time.Duration, store Store) *SliderCaptcha {
	if store == nil {
		store = NewMemoryStore()
	}
	if trackWidth <= 0 {
		trackWidth = defaultSliderTrackWidth
	}
	if passRatio <= 0 || passRatio > 1 {
		passRatio = 0.92
	}
	return &SliderCaptcha{
		trackWidth: trackWidth,
		passRatio:  passRatio,
		expiration: expiration,
		store:      store,
	}
}

// Generate creates a slider challenge.
func (sc *SliderCaptcha) Generate() (*Result, error) {
	id := generateID()
	expires := time.Now().Add(sc.expiration)
	if err := sc.store.Set(id, sc.trackWidth, expires); err != nil {
		return nil, fmt.Errorf("failed to store captcha: %w", err)
	}
	return &Result{
		ID:   id,
		Type: TypeSlider,
		Data: map[string]interface{}{
			"trackWidth": sc.trackWidth,
		},
		Expires: expires,
	}, nil
}

// Verify checks that the slider was dragged near the right edge.
func (sc *SliderCaptcha) Verify(id string, x int) (bool, error) {
	return sc.store.VerifyWithFunc(id, x, sc.compare)
}

func (sc *SliderCaptcha) compare(stored, input interface{}) bool {
	trackWidth := intValue(stored)
	sliderX := intValue(input)
	if trackWidth <= 0 {
		return false
	}
	minX := int(float64(trackWidth) * sc.passRatio)
	return sliderX >= minX && sliderX <= trackWidth
}

func intValue(v interface{}) int {
	switch n := v.(type) {
	case int:
		return n
	case int32:
		return int(n)
	case int64:
		return int(n)
	case float64:
		return int(n)
	case float32:
		return int(n)
	default:
		return 0
	}
}
