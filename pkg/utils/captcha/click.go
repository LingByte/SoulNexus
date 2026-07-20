package captcha

import (
	"fmt"
	"image"
	"image/color"
	"math/rand"
	"time"
)

const defaultDecoyCount = 2

// clickStored holds ordered target click positions (server-side only).
type clickStored struct {
	Positions []Point
}

// ClickCaptcha is an ordered click challenge rendered on the client.
type ClickCaptcha struct {
	width      int
	height     int
	count      int
	decoys     int
	tolerance  int
	expiration time.Duration
	store      Store
}

// NewClickCaptcha creates a click captcha manager.
func NewClickCaptcha(width, height, count, tolerance int, expiration time.Duration, store Store) *ClickCaptcha {
	if store == nil {
		store = NewMemoryStore()
	}
	if count <= 0 {
		count = 3
	}
	return &ClickCaptcha{
		width:      width,
		height:     height,
		count:      count,
		decoys:     defaultDecoyCount,
		tolerance:  tolerance,
		expiration: expiration,
		store:      store,
	}
}

// Generate creates a click challenge. Client renders chars; targets must be clicked in order.
func (cc *ClickCaptcha) Generate() (*Result, error) {
	total := cc.count + cc.decoys
	allWords := cc.generateWords(total)
	targets := append([]string(nil), allWords[:cc.count]...)

	rng := rand.New(rand.NewSource(time.Now().UnixNano()))
	rng.Shuffle(len(allWords), func(i, j int) { allWords[i], allWords[j] = allWords[j], allWords[i] })

	chars, targetPositions, err := cc.layoutChars(allWords, targets, rng)
	if err != nil {
		return nil, err
	}

	id := generateID()
	expires := time.Now().Add(cc.expiration)
	stored := clickStored{Positions: targetPositions}
	if err := cc.store.Set(id, stored, expires); err != nil {
		return nil, fmt.Errorf("failed to store captcha: %w", err)
	}

	markers := make([]CharMarker, len(chars))
	for i, c := range chars {
		markers[i] = CharMarker{Char: c.Char, X: c.X, Y: c.Y}
	}

	bg, err := cc.generateBackground(rng)
	if err != nil {
		return nil, fmt.Errorf("failed to generate background: %w", err)
	}

	return &Result{
		ID:   id,
		Type: TypeClick,
		Data: map[string]interface{}{
			"width":      cc.width,
			"height":     cc.height,
			"targets":    targets,
			"chars":      markers,
			"tolerance":  cc.tolerance,
			"background": bg,
		},
		Expires: expires,
	}, nil
}

// Verify validates clicks in the exact order of the target sequence.
func (cc *ClickCaptcha) Verify(id string, userPositions []Point) (bool, error) {
	return cc.store.VerifyWithFunc(id, userPositions, cc.compareOrdered)
}

func (cc *ClickCaptcha) compareOrdered(stored, input interface{}) bool {
	data, ok1 := stored.(clickStored)
	inputPositions, ok2 := input.([]Point)
	if !ok1 || !ok2 {
		return false
	}
	return cc.positionsMatchOrdered(data.Positions, inputPositions)
}

func (cc *ClickCaptcha) positionsMatchOrdered(stored, input []Point) bool {
	if len(stored) != len(input) {
		return false
	}
	toleranceSquared := cc.tolerance * cc.tolerance
	for i := range stored {
		dx := abs(input[i].X - stored[i].X)
		dy := abs(input[i].Y - stored[i].Y)
		if dx*dx+dy*dy > toleranceSquared {
			return false
		}
	}
	return true
}

type placedChar struct {
	Char string
	X    int
	Y    int
}

func (cc *ClickCaptcha) layoutChars(allWords, targets []string, rng *rand.Rand) ([]placedChar, []Point, error) {
	usedAreas := make([]rect, 0, len(allWords))
	placed := make([]placedChar, 0, len(allWords))

	for _, word := range allWords {
		x, y, area, ok := cc.randomOpenSlot(usedAreas, rng)
		if !ok {
			return nil, nil, fmt.Errorf("failed to place character %q", word)
		}
		usedAreas = append(usedAreas, area)
		placed = append(placed, placedChar{Char: word, X: x, Y: y})
	}

	byChar := make(map[string]Point, len(placed))
	for _, p := range placed {
		byChar[p.Char] = Point{X: p.X, Y: p.Y}
	}
	targetPositions := make([]Point, 0, len(targets))
	for _, target := range targets {
		pos, ok := byChar[target]
		if !ok {
			return nil, nil, fmt.Errorf("target %q not placed", target)
		}
		targetPositions = append(targetPositions, pos)
	}
	return placed, targetPositions, nil
}

type rect struct{ x1, y1, x2, y2 int }

func (r rect) overlaps(o rect) bool {
	return r.x1 < o.x2 && r.x2 > o.x1 && r.y1 < o.y2 && r.y2 > o.y1
}

func (cc *ClickCaptcha) randomOpenSlot(used []rect, rng *rand.Rand) (x, y int, area rect, ok bool) {
	const (
		charW = 36
		charH = 36
	)
	maxAttempts := 80
	for attempt := 0; attempt < maxAttempts; attempt++ {
		x = rng.Intn(max(1, cc.width-charW-20)) + 10
		y = rng.Intn(max(1, cc.height-charH-20)) + 10
		area = rect{x1: x, y1: y, x2: x + charW, y2: y + charH}
		overlap := false
		for _, u := range used {
			if area.overlaps(u) {
				overlap = true
				break
			}
		}
		if !overlap {
			return x, y, area, true
		}
	}
	return 0, 0, rect{}, false
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// generateWords picks unique characters for targets/decoys.
func (cc *ClickCaptcha) generateWords(count int) []string {
	pool := []string{
		"春", "夏", "秋", "冬", "东", "西", "南", "北",
		"红", "绿", "蓝", "黄", "山", "水", "火", "土",
		"金", "木", "日", "月", "星", "云", "雨", "风",
		"花", "草", "树", "叶", "鸟", "鱼", "马", "牛",
		"A", "B", "C", "D", "E", "F", "G", "H", "J", "K",
		"M", "N", "P", "Q", "R", "T", "U", "V", "W", "X",
		"Y", "Z", "2", "3", "4", "5", "6", "7", "8", "9",
	}
	rng := rand.New(rand.NewSource(time.Now().UnixNano()))
	words := make([]string, 0, count)
	used := make(map[string]struct{})
	for len(words) < count {
		word := pool[rng.Intn(len(pool))]
		if _, ok := used[word]; ok {
			continue
		}
		used[word] = struct{}{}
		words = append(words, word)
	}
	return words
}

func (cc *ClickCaptcha) generateBackground(rng *rand.Rand) (string, error) {
	img := image.NewRGBA(image.Rect(0, 0, cc.width, cc.height))
	top := color.RGBA{
		uint8(170 + rng.Intn(50)),
		uint8(200 + rng.Intn(40)),
		uint8(210 + rng.Intn(35)),
		255,
	}
	bottom := color.RGBA{
		uint8(225 + rng.Intn(25)),
		uint8(230 + rng.Intn(20)),
		uint8(235 + rng.Intn(20)),
		255,
	}
	for y := 0; y < cc.height; y++ {
		t := float64(y) / float64(max(1, cc.height-1))
		for x := 0; x < cc.width; x++ {
			noise := rng.Intn(13) - 6
			r := clampByte(int(float64(top.R)*(1-t)+float64(bottom.R)*t), noise)
			g := clampByte(int(float64(top.G)*(1-t)+float64(bottom.G)*t), noise)
			b := clampByte(int(float64(top.B)*(1-t)+float64(bottom.B)*t), noise)
			img.Set(x, y, color.RGBA{r, g, b, 255})
		}
	}
	for i := 0; i < 10; i++ {
		x1 := rng.Intn(cc.width)
		y1 := rng.Intn(cc.height)
		x2 := rng.Intn(cc.width)
		y2 := rng.Intn(cc.height)
		lineColor := color.RGBA{
			uint8(rng.Intn(160) + 60),
			uint8(rng.Intn(160) + 60),
			uint8(rng.Intn(160) + 60),
			uint8(rng.Intn(80) + 40),
		}
		drawLine(img, x1, y1, x2, y2, lineColor)
	}
	for i := 0; i < 120; i++ {
		x := rng.Intn(cc.width)
		y := rng.Intn(cc.height)
		dotColor := color.RGBA{
			uint8(rng.Intn(180) + 40),
			uint8(rng.Intn(180) + 40),
			uint8(rng.Intn(180) + 40),
			uint8(rng.Intn(100) + 80),
		}
		img.Set(x, y, dotColor)
	}
	return imagePNGDataURL(img)
}
