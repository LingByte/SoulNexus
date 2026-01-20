package utils

import (
	"encoding/base64"
	"errors"
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"strconv"
	"sync"
	"time"

	"github.com/LingByte/SoulNexus/pkg/logger"
	"go.uber.org/zap"
)

var SnowflakeUtil *Snowflake
var letterRunes = []rune("0123456789abcdefghijklmnopqrstuvwxyz")
var numberRunes = []rune("0123456789")

func init() {
	rand.Seed(time.Now().UnixNano())
	SnowflakeUtil, _ = NewSnowflake()
}

func randRunes(n int, source []rune) string {
	b := make([]rune, n)
	for i := range b {
		b[i] = source[rand.Intn(len(source))]
	}
	return string(b)
}

func RandText(n int) string {
	return randRunes(n, letterRunes)
}

func RandNumberText(n int) string {
	return randRunes(n, numberRunes)
}

func RandString(n int) string {
	return randRunes(n, letterRunes)
}

func SafeCall(f func() error, failHandle func(error)) error {
	defer func() {
		if err := recover(); err != nil {
			if failHandle != nil {
				eo, ok := err.(error)
				if !ok {
					es, ok := err.(string)
					if ok {
						eo = errors.New(es)
					} else {
						eo = errors.New("unknown error type")
					}
				}
				failHandle(eo)
			} else {
				logger.Error("panic", zap.Any("error", err))
			}
		}
	}()
	return f()
}

func StructAsMap(form any, fields []string) (vals map[string]any) {
	vals = make(map[string]any)
	v := reflect.ValueOf(form)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}
	if v.Kind() != reflect.Struct {
		return vals
	}
	for i := 0; i < len(fields); i++ {
		k := v.FieldByName(fields[i])
		if !k.IsValid() || k.IsZero() {
			continue
		}
		if k.Kind() == reflect.Ptr {
			if !k.IsNil() {
				vals[fields[i]] = k.Elem().Interface()
			}
		} else {
			vals[fields[i]] = k.Interface()
		}
	}
	return vals
}

// GenerateSecureToken generate a fixed-length secure token
func GenerateSecureToken(length int) (string, error) {
	token := make([]byte, length)
	if _, err := rand.Read(token); err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(token), nil
}

const (
	epoch         int64 = 1609459200000000 // Microsecond timestamp start (2021-01-01)
	timestampBits uint  = 44
	machineIDBits uint  = 10
	sequenceBits  uint  = 9

	maxMachineID = -1 ^ (-1 << machineIDBits) // 1023
	maxSequence  = -1 ^ (-1 << sequenceBits)  // 511

	machineIDShift = sequenceBits
	timestampShift = machineIDBits + sequenceBits
)

type Snowflake struct {
	mu        sync.Mutex
	lastStamp int64
	sequence  int64
	machineID int64
}

func NewSnowflake() (*Snowflake, error) {
	id := getMachineID()
	if id < 0 || id > maxMachineID {
		return nil, errors.New("machineID out of range")
	}
	return &Snowflake{
		machineID: id,
	}, nil
}

func (s *Snowflake) NextID() int64 {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := currentMicro()
	if now < s.lastStamp {
		// Clock rollback protection
		return 0
	}

	if now == s.lastStamp {
		s.sequence = (s.sequence + 1) & maxSequence
		if s.sequence == 0 {
			// Sequence number for current microsecond is full, wait for next microsecond
			for now <= s.lastStamp {
				now = currentMicro()
			}
		}
	} else {
		s.sequence = 0
	}

	s.lastStamp = now

	id := ((now - epoch) << timestampShift) |
		(s.machineID << machineIDShift) |
		s.sequence

	return id
}

func currentMicro() int64 {
	return time.Now().UnixNano() / 1e3
}

func getMachineID() int64 {
	val := os.Getenv("MACHINE_ID")
	id, err := strconv.ParseInt(val, 10, 64)
	if err != nil {
		return 1 // fallback default value, recommended to modify according to actual situation
	}
	return id
}

// WriteFile write file
func WriteFile(filename string, data []byte) error {
	// Ensure directory exists
	dir := filepath.Dir(filename)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// Write file
	if err := os.WriteFile(filename, data, 0644); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	return nil
}

// ReadFile read file
func ReadFile(filename string) ([]byte, error) {
	return os.ReadFile(filename)
}

// removeEmoji 移除字符串中的 emoji 字符，避免数据库字符集不兼容问题
func RemoveEmoji(text string) string {
	var result []rune
	for _, r := range text {
		// 检查是否是 emoji 字符（常见的 emoji Unicode 范围）
		if (r >= 0x1F300 && r <= 0x1F9FF) || // 杂项符号和象形文字
			(r >= 0x1F600 && r <= 0x1F64F) || // 表情符号
			(r >= 0x1F680 && r <= 0x1F6FF) || // 交通和地图符号
			(r >= 0x2600 && r <= 0x26FF) || // 杂项符号
			(r >= 0x2700 && r <= 0x27BF) || // 装饰符号
			(r >= 0xFE00 && r <= 0xFE0F) || // 变体选择器
			(r >= 0x1F900 && r <= 0x1F9FF) || // 补充符号和象形文字
			(r >= 0x1F1E0 && r <= 0x1F1FF) { // 区域指示符号
			continue // 跳过 emoji
		}
		result = append(result, r)
	}
	return string(result)
}

// removeEmojiFromJSON 从 JSON 字符串中移除 emoji（仅从字符串值中移除，保持 JSON 结构）
func RemoveEmojiFromJSON(jsonStr string) string {
	// 使用正则表达式匹配 JSON 字符串值中的 emoji
	// 匹配 "key": "value" 中的 value 部分
	re := regexp.MustCompile(`("(?:[^"\\]|\\.)*")`)
	result := re.ReplaceAllStringFunc(jsonStr, func(match string) string {
		// 移除引号，清理 emoji，然后重新添加引号
		if len(match) > 2 {
			content := match[1 : len(match)-1]
			cleaned := RemoveEmoji(content)
			return `"` + cleaned + `"`
		}
		return match
	})
	return result
}

// ComputeSampleByteCount calculates bytes per millisecond for given audio parameters
func ComputeSampleByteCount(rate, depth, chans int) int {
	// Optimized: rate * depth / 8 / 1000 * chans
	// Reordered for better precision: (rate * depth * chans) / 8000
	return (rate * depth * chans) / 8000
}

// ValidateAndNormalizeDuration uses different validation logic with explicit bounds checking
func NormalizeFramePeriod(d string) time.Duration {
	parsed, err := time.ParseDuration(d)
	if err != nil {
		return 20 * time.Millisecond
	}
	if parsed == 0 {
		return 20 * time.Millisecond
	}

	// Use explicit range checks instead of compound condition
	if parsed < 10*time.Millisecond {
		return 20 * time.Millisecond
	}
	if parsed > 300*time.Millisecond {
		return 20 * time.Millisecond
	}
	return parsed
}
