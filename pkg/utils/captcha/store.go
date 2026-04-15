package captcha

import (
	"errors"
	"strings"
	"sync"
	"time"
)

// Store 验证码存储接口
type Store interface {
	Set(id string, data interface{}, expires time.Time) error
	Get(id string) (interface{}, error)
	Delete(id string) error
	VerifyWithFunc(id string, input interface{}, compareFunc func(stored, input interface{}) bool) (bool, error)
	VerifyWithFuncWithoutDelete(id string, input interface{}, compareFunc func(stored, input interface{}) bool) (bool, error)
}

// MemoryStore 内存存储实现
type MemoryStore struct {
	data map[string]storeData
	mu   sync.RWMutex
}

type storeData struct {
	data    interface{}
	expires time.Time
}

// NewMemoryStore 创建内存存储
func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		data: make(map[string]storeData),
	}
}

func (s *MemoryStore) Set(id string, data interface{}, expires time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.data[id] = storeData{
		data:    data,
		expires: expires,
	}
	// 清理过期数据
	go s.cleanup()
	return nil
}

func (s *MemoryStore) Get(id string) (interface{}, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	data, ok := s.data[id]
	if !ok {
		return nil, errors.New("captcha not found")
	}
	if time.Now().After(data.expires) {
		s.mu.RUnlock()
		s.mu.Lock()
		delete(s.data, id)
		s.mu.Unlock()
		s.mu.RLock()
		return nil, errors.New("captcha expired")
	}
	return data.data, nil
}

func (s *MemoryStore) Delete(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.data, id)
	return nil
}

func (s *MemoryStore) VerifyWithFunc(id string, input interface{}, compareFunc func(stored, input interface{}) bool) (bool, error) {
	stored, err := s.Get(id)
	if err != nil {
		return false, err
	}
	if compareFunc == nil {
		// 默认字符串比较（不区分大小写）
		if storedStr, ok := stored.(string); ok {
			if inputStr, ok := input.(string); ok {
				if strings.ToLower(storedStr) == strings.ToLower(inputStr) {
					s.Delete(id) // 验证成功后删除
					return true, nil
				}
			}
		}
		return false, nil
	}
	if compareFunc(stored, input) {
		s.Delete(id) // 验证成功后删除
		return true, nil
	}
	return false, nil
}

// VerifyWithFuncWithoutDelete 验证但不删除（用于预验证）
func (s *MemoryStore) VerifyWithFuncWithoutDelete(id string, input interface{}, compareFunc func(stored, input interface{}) bool) (bool, error) {
	stored, err := s.Get(id)
	if err != nil {
		return false, err
	}
	if compareFunc == nil {
		// 默认字符串比较（不区分大小写）
		if storedStr, ok := stored.(string); ok {
			if inputStr, ok := input.(string); ok {
				if strings.ToLower(storedStr) == strings.ToLower(inputStr) {
					return true, nil
				}
			}
		}
		return false, nil
	}
	return compareFunc(stored, input), nil
}

func (s *MemoryStore) cleanup() {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := time.Now()
	for id, data := range s.data {
		if now.After(data.expires) {
			delete(s.data, id)
		}
	}
}
