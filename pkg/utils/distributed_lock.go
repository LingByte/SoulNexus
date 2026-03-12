package utils

import (
	"errors"
	"fmt"
	"time"
)

// DistributedLock 分布式锁接口
type DistributedLock interface {
	Lock(key string, ttl time.Duration) (bool, error)
	Unlock(key string) error
	TryLock(key string, ttl time.Duration) (bool, error)
}

// MemoryLock 内存锁实现（单机版）
type MemoryLock struct {
	locks map[string]time.Time
}

// NewMemoryLock 创建内存锁
func NewMemoryLock() *MemoryLock {
	return &MemoryLock{
		locks: make(map[string]time.Time),
	}
}

// Lock 获取锁
func (ml *MemoryLock) Lock(key string, ttl time.Duration) (bool, error) {
	now := time.Now()

	// 检查锁是否存在且未过期
	if expireTime, exists := ml.locks[key]; exists {
		if now.Before(expireTime) {
			return false, errors.New("lock already exists")
		}
		// 锁已过期，删除
		delete(ml.locks, key)
	}

	// 设置锁
	ml.locks[key] = now.Add(ttl)
	return true, nil
}

// Unlock 释放锁
func (ml *MemoryLock) Unlock(key string) error {
	delete(ml.locks, key)
	return nil
}

// TryLock 尝试获取锁（非阻塞）
func (ml *MemoryLock) TryLock(key string, ttl time.Duration) (bool, error) {
	return ml.Lock(key, ttl)
}

// GlobalDistributedLock 全局分布式锁实例
var GlobalDistributedLock DistributedLock

// InitGlobalDistributedLock 初始化全局分布式锁
func InitGlobalDistributedLock() {
	if GlobalCache != nil {
		// 如果使用Redis等分布式缓存，可以实现基于缓存的分布式锁
		GlobalDistributedLock = NewMemoryLock()
	} else {
		GlobalDistributedLock = NewMemoryLock()
	}
}

// AcquireRegistrationLock 获取注册锁（防止并发注册同一邮箱）
func AcquireRegistrationLock(email string) (bool, error) {
	if GlobalDistributedLock == nil {
		// 如果没有分布式锁，使用缓存作为简单锁
		if GlobalCache != nil {
			lockKey := fmt.Sprintf("reg:lock:%s", email)
			if _, exists := GlobalCache.Get(lockKey); exists {
				return false, errors.New("registration in progress for this email")
			}
			// 设置锁，5分钟过期
			GlobalCache.Add(lockKey, true)
			return true, nil
		}
		// 如果缓存也没有，返回true（允许注册，但无法防止并发）
		return true, nil
	}

	lockKey := fmt.Sprintf("reg:lock:%s", email)
	success, err := GlobalDistributedLock.Lock(lockKey, 5*time.Minute)
	return success, err
}

// ReleaseRegistrationLock 释放注册锁
func ReleaseRegistrationLock(email string) error {
	if GlobalDistributedLock == nil {
		if GlobalCache != nil {
			lockKey := fmt.Sprintf("reg:lock:%s", email)
			GlobalCache.Remove(lockKey)
		}
		return nil
	}

	lockKey := fmt.Sprintf("reg:lock:%s", email)
	return GlobalDistributedLock.Unlock(lockKey)
}
