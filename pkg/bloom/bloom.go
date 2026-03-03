package bloom

import (
	"context"
	"fmt"
	"hash/fnv"
	"math"
	"sync"

	"github.com/redis/go-redis/v9"
)

// Filter 布隆过滤器接口
type Filter interface {
	Add(ctx context.Context, data string) error
	Contains(ctx context.Context, data string) (bool, error)
	AddMulti(ctx context.Context, data []string) error
	ContainsMulti(ctx context.Context, data []string) ([]bool, error)
	Clear(ctx context.Context) error
	Close() error
	Stats() map[string]interface{}
}

// Config 布隆过滤器配置
type Config struct {
	Type              string  // "memory" 或 "redis"
	ExpectedElements  int     // 预期元素个数
	FalsePositiveRate float64 // 误判率（0.01 = 1%）
	RedisAddr         string  // Redis 地址
	RedisKey          string  // Redis key
}

// MemoryFilter 本地内存布隆过滤器
type MemoryFilter struct {
	bits      []byte
	size      uint32
	hashCount uint32
	mu        sync.RWMutex
}

// NewMemoryFilter 创建本地布隆过滤器
func NewMemoryFilter(expectedElements int, falsePositiveRate float64) *MemoryFilter {
	if expectedElements <= 0 {
		expectedElements = 1000
	}
	if falsePositiveRate <= 0 || falsePositiveRate >= 1 {
		falsePositiveRate = 0.01
	}

	// 计算位数组大小：m = -1 * n * ln(p) / (ln(2)^2)
	size := uint32(-1 * float64(expectedElements) * math.Log(falsePositiveRate) / (math.Log(2) * math.Log(2)))

	// 计算 hash 函数个数：k = m / n * ln(2)
	hashCount := uint32(float64(size) / float64(expectedElements) * math.Log(2))
	if hashCount < 1 {
		hashCount = 1
	}
	if hashCount > 16 {
		hashCount = 16
	}

	return &MemoryFilter{
		bits:      make([]byte, (size+7)/8),
		size:      size,
		hashCount: hashCount,
	}
}

func (mf *MemoryFilter) hash(data string, seed uint32) uint32 {
	h := fnv.New32a()
	h.Write([]byte(data))
	return (h.Sum32() ^ seed) % mf.size
}

func (mf *MemoryFilter) Add(ctx context.Context, data string) error {
	mf.mu.Lock()
	defer mf.mu.Unlock()

	for i := uint32(0); i < mf.hashCount; i++ {
		pos := mf.hash(data, i)
		byteIndex := pos / 8
		bitIndex := pos % 8
		mf.bits[byteIndex] |= 1 << bitIndex
	}
	return nil
}

func (mf *MemoryFilter) Contains(ctx context.Context, data string) (bool, error) {
	mf.mu.RLock()
	defer mf.mu.RUnlock()

	for i := uint32(0); i < mf.hashCount; i++ {
		pos := mf.hash(data, i)
		byteIndex := pos / 8
		bitIndex := pos % 8
		if (mf.bits[byteIndex] & (1 << bitIndex)) == 0 {
			return false, nil
		}
	}
	return true, nil
}

func (mf *MemoryFilter) AddMulti(ctx context.Context, data []string) error {
	mf.mu.Lock()
	defer mf.mu.Unlock()

	for _, item := range data {
		for i := uint32(0); i < mf.hashCount; i++ {
			pos := mf.hash(item, i)
			byteIndex := pos / 8
			bitIndex := pos % 8
			mf.bits[byteIndex] |= 1 << bitIndex
		}
	}
	return nil
}

func (mf *MemoryFilter) ContainsMulti(ctx context.Context, data []string) ([]bool, error) {
	mf.mu.RLock()
	defer mf.mu.RUnlock()

	results := make([]bool, len(data))
	for idx, item := range data {
		found := true
		for i := uint32(0); i < mf.hashCount; i++ {
			pos := mf.hash(item, i)
			byteIndex := pos / 8
			bitIndex := pos % 8
			if (mf.bits[byteIndex] & (1 << bitIndex)) == 0 {
				found = false
				break
			}
		}
		results[idx] = found
	}
	return results, nil
}

func (mf *MemoryFilter) Clear(ctx context.Context) error {
	mf.mu.Lock()
	defer mf.mu.Unlock()

	for i := range mf.bits {
		mf.bits[i] = 0
	}
	return nil
}

func (mf *MemoryFilter) Close() error {
	return nil
}

func (mf *MemoryFilter) Stats() map[string]interface{} {
	mf.mu.RLock()
	defer mf.mu.RUnlock()

	usedBits := 0
	for _, b := range mf.bits {
		for i := 0; i < 8; i++ {
			if (b & (1 << i)) != 0 {
				usedBits++
			}
		}
	}

	return map[string]interface{}{
		"type":            "memory",
		"size_bits":       mf.size,
		"size_bytes":      len(mf.bits),
		"hash_count":      mf.hashCount,
		"used_bits":       usedBits,
		"used_percentage": float64(usedBits) / float64(mf.size) * 100,
	}
}

// RedisFilter Redis 布隆过滤器
type RedisFilter struct {
	rdb *redis.Client
	key string
}

// NewRedisFilter 创建 Redis 布隆过滤器
func NewRedisFilter(rdb *redis.Client, key string) *RedisFilter {
	return &RedisFilter{rdb: rdb, key: key}
}

func (rf *RedisFilter) Add(ctx context.Context, data string) error {
	result := rf.rdb.Do(ctx, "BF.ADD", rf.key, data)
	if result.Err() != nil {
		return rf.addWithBitmap(ctx, data)
	}
	return nil
}

func (rf *RedisFilter) Contains(ctx context.Context, data string) (bool, error) {
	result := rf.rdb.Do(ctx, "BF.EXISTS", rf.key, data)
	if result.Err() != nil {
		return rf.containsWithBitmap(ctx, data)
	}

	val, err := result.Int64()
	if err != nil {
		return false, err
	}
	return val == 1, nil
}

func (rf *RedisFilter) AddMulti(ctx context.Context, data []string) error {
	args := []interface{}{"BF.MADD", rf.key}
	for _, item := range data {
		args = append(args, item)
	}

	result := rf.rdb.Do(ctx, args...)
	if result.Err() != nil {
		for _, item := range data {
			if err := rf.Add(ctx, item); err != nil {
				return err
			}
		}
	}
	return nil
}

func (rf *RedisFilter) ContainsMulti(ctx context.Context, data []string) ([]bool, error) {
	args := []interface{}{"BF.MEXISTS", rf.key}
	for _, item := range data {
		args = append(args, item)
	}

	result := rf.rdb.Do(ctx, args...)
	if result.Err() != nil {
		results := make([]bool, len(data))
		for i, item := range data {
			exists, err := rf.Contains(ctx, item)
			if err != nil {
				return nil, err
			}
			results[i] = exists
		}
		return results, nil
	}

	vals, err := result.Int64Slice()
	if err != nil {
		return nil, err
	}

	results := make([]bool, len(vals))
	for i, val := range vals {
		results[i] = val == 1
	}
	return results, nil
}

func (rf *RedisFilter) Clear(ctx context.Context) error {
	return rf.rdb.Del(ctx, rf.key).Err()
}

func (rf *RedisFilter) Close() error {
	return nil
}

func (rf *RedisFilter) addWithBitmap(ctx context.Context, data string) error {
	positions := rf.getPositions(data)
	pipe := rf.rdb.Pipeline()
	for _, pos := range positions {
		pipe.SetBit(ctx, rf.key, pos, 1)
	}
	_, err := pipe.Exec(ctx)
	return err
}

func (rf *RedisFilter) containsWithBitmap(ctx context.Context, data string) (bool, error) {
	positions := rf.getPositions(data)
	for _, pos := range positions {
		result, err := rf.rdb.GetBit(ctx, rf.key, pos).Result()
		if err != nil {
			return false, err
		}
		if result == 0 {
			return false, nil
		}
	}
	return true, nil
}

func (rf *RedisFilter) getPositions(data string) []int64 {
	positions := make([]int64, 4)
	maxBits := int64(1024 * 1024 * 8)

	for i := 0; i < 4; i++ {
		h := hashWithSeed(data, uint32(i))
		positions[i] = int64(h) % maxBits
	}
	return positions
}

func (rf *RedisFilter) Stats() map[string]interface{} {
	return map[string]interface{}{
		"type": "redis",
		"key":  rf.key,
	}
}

func hashWithSeed(data string, seed uint32) uint32 {
	hash := uint32(5381)
	for _, c := range data {
		hash = ((hash << 5) + hash) + uint32(c)
	}
	return hash ^ seed
}

// NewFilter 工厂函数
func NewFilter(config Config) (Filter, error) {
	switch config.Type {
	case "memory":
		return NewMemoryFilter(config.ExpectedElements, config.FalsePositiveRate), nil

	case "redis":
		if config.RedisAddr == "" {
			return nil, fmt.Errorf("redis address is required for redis filter")
		}
		if config.RedisKey == "" {
			return nil, fmt.Errorf("redis key is required for redis filter")
		}

		rdb := redis.NewClient(&redis.Options{Addr: config.RedisAddr})
		if err := rdb.Ping(context.Background()).Err(); err != nil {
			return nil, fmt.Errorf("failed to connect to redis: %w", err)
		}

		return NewRedisFilter(rdb, config.RedisKey), nil

	default:
		return nil, fmt.Errorf("unsupported filter type: %s", config.Type)
	}
}

// NewMemoryFilterWithDefaults 创建本地过滤器（使用默认参数）
func NewMemoryFilterWithDefaults() Filter {
	return NewMemoryFilter(1000000, 0.01)
}

// NewRedisFilterWithDefaults 创建 Redis 过滤器（使用默认参数）
func NewRedisFilterWithDefaults(addr, key string) (Filter, error) {
	return NewFilter(Config{
		Type:              "redis",
		ExpectedElements:  1000000,
		FalsePositiveRate: 0.01,
		RedisAddr:         addr,
		RedisKey:          key,
	})
}
