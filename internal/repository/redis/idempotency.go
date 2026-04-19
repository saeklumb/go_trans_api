package redisrepo

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"
)

type IdempotencyStore struct {
	rdb *redis.Client
}

func NewIdempotencyStore(rdb *redis.Client) *IdempotencyStore {
	return &IdempotencyStore{rdb: rdb}
}

func (s *IdempotencyStore) Get(ctx context.Context, key string) (string, bool, error) {
	val, err := s.rdb.Get(ctx, "idem:"+key).Result()
	if err == redis.Nil {
		return "", false, nil
	}
	if err != nil {
		return "", false, err
	}
	return val, true, nil
}

func (s *IdempotencyStore) TryLock(ctx context.Context, key string, ttl time.Duration) (bool, error) {
	return s.rdb.SetNX(ctx, "idem_lock:"+key, "1", ttl).Result()
}

func (s *IdempotencyStore) Unlock(ctx context.Context, key string) error {
	return s.rdb.Del(ctx, "idem_lock:"+key).Err()
}

func (s *IdempotencyStore) Save(ctx context.Context, key string, payload string, ttl time.Duration) error {
	return s.rdb.Set(ctx, "idem:"+key, payload, ttl).Err()
}
