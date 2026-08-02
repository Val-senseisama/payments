package redis

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

type RedisStore struct {
	rdb *redis.Client
}

func NewRedisStore(rdb *redis.Client) *RedisStore {
	return &RedisStore{rdb: rdb}
}

func (s *RedisStore) SaveRefreshToken(ctx context.Context, userID string, tokenID string, ttl time.Duration) error {
	key := fmt.Sprintf("refresh_token:%s", userID)
	return s.rdb.Set(ctx, key, tokenID, ttl).Err()
}

func (s *RedisStore) GetRefreshToken(ctx context.Context, userID string) (string, error) {
	key := fmt.Sprintf("refresh_token:%s", userID)
	return s.rdb.Get(ctx, key).Result()
}

func (s *RedisStore) DeleteRefreshToken(ctx context.Context, userID string) error {
	key := fmt.Sprintf("refresh_token:%s", userID)
	return s.rdb.Del(ctx, key).Err()
}
