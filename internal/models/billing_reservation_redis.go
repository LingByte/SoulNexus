package models

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
)

type redisReserveBackend struct {
	rdb *redis.Client
}

func newRedisReserveBackend(url string) (*redisReserveBackend, error) {
	opts, err := redis.ParseURL(strings.TrimSpace(url))
	if err != nil {
		return nil, err
	}
	rdb := redis.NewClient(opts)
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if err := rdb.Ping(ctx).Err(); err != nil {
		_ = rdb.Close()
		return nil, err
	}
	return &redisReserveBackend{rdb: rdb}, nil
}

func (r *redisReserveBackend) lockTenant(ctx context.Context, tenantID uint) (func(), error) {
	key := tenantAdmitLockKey(tenantID)
	token := strconv.FormatInt(time.Now().UnixNano(), 10)
	for i := 0; i < 40; i++ {
		ok, err := r.rdb.SetNX(ctx, key, token, 5*time.Second).Result()
		if err != nil {
			return noopUnlock, err
		}
		if ok {
			return func() {
				script := redis.NewScript(`
if redis.call("GET", KEYS[1]) == ARGV[1] then
  return redis.call("DEL", KEYS[1])
end
return 0
`)
				_, _ = script.Run(context.Background(), r.rdb, []string{key}, token).Result()
			}, nil
		}
		time.Sleep(25 * time.Millisecond)
	}
	return noopUnlock, fmt.Errorf("billing admit lock busy for tenant %d", tenantID)
}

func (r *redisReserveBackend) heldMinutes(ctx context.Context, tenantID uint) (int64, error) {
	n, err := r.rdb.Get(ctx, tenantHeldKey(tenantID)).Int64()
	if err == redis.Nil {
		return 0, nil
	}
	return n, err
}

func (r *redisReserveBackend) putReservation(ctx context.Context, tenantID uint, callID string, minutes int64) error {
	key := callReserveKey(callID)
	exists, err := r.rdb.Exists(ctx, key).Result()
	if err != nil {
		return err
	}
	if exists > 0 {
		return nil
	}
	pipe := r.rdb.TxPipeline()
	pipe.Set(ctx, key, reservationPayload(tenantID, minutes), reservationTTL)
	pipe.IncrBy(ctx, tenantHeldKey(tenantID), minutes)
	_, err = pipe.Exec(ctx)
	return err
}

func (r *redisReserveBackend) dropReservation(ctx context.Context, callID string) error {
	key := callReserveKey(callID)
	raw, err := r.rdb.Get(ctx, key).Result()
	if err == redis.Nil {
		return nil
	}
	if err != nil {
		return err
	}
	tenantID, minutes, ok := parseReservationPayload(raw)
	if !ok {
		_ = r.rdb.Del(ctx, key).Err()
		return nil
	}
	pipe := r.rdb.TxPipeline()
	pipe.Del(ctx, key)
	pipe.DecrBy(ctx, tenantHeldKey(tenantID), minutes)
	_, err = pipe.Exec(ctx)
	return err
}

func (r *redisReserveBackend) setBalanceHint(ctx context.Context, tenantID uint, balance int64) error {
	return r.rdb.Set(ctx, tenantBalanceKey(tenantID), balance, 0).Err()
}

var _ reserveBackend = (*redisReserveBackend)(nil)
