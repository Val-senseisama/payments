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

func (s *RedisStore) SaveOTP(ctx context.Context, email string, otp string, ttl time.Duration) error {
	key := fmt.Sprintf("otp:%s", email)
	return s.rdb.Set(ctx, key, otp, ttl).Err()
}

func (s *RedisStore) GetOTP(ctx context.Context, email string) (string, error) {
	key := fmt.Sprintf("otp:%s", email)
	return s.rdb.Get(ctx, key).Result()
}

func (s *RedisStore) DeleteOTP(ctx context.Context, email string) error {
	key := fmt.Sprintf("otp:%s", email)
	return s.rdb.Del(ctx, key).Err()
}

func (s *RedisStore) SetIdempotencyKey(ctx context.Context, key string, value []byte, ttl time.Duration) error {
	redisKey := fmt.Sprintf("idempotency:%s", key)
	return s.rdb.Set(ctx, redisKey, value, ttl).Err()
}

func (s *RedisStore) GetIdempotencyKey(ctx context.Context, key string) ([]byte, error) {
	redisKey := fmt.Sprintf("idempotency:%s", key)
	return s.rdb.Get(ctx, redisKey).Bytes()
}

func (s *RedisStore) AcquirePostingLock(ctx context.Context, txnID string, ttl time.Duration) (bool, error) {
	key := fmt.Sprintf("lock:posting:%s", txnID)
	return s.rdb.SetNX(ctx, key, "1", ttl).Result()
}

func (s *RedisStore) ReleasePostingLock(ctx context.Context, txnID string) error {
	key := fmt.Sprintf("lock:posting:%s", txnID)
	return s.rdb.Del(ctx, key).Err()
}

func (s *RedisStore) PublishTransactionUpdate(ctx context.Context, companyID string, payload []byte) error {
	channel := fmt.Sprintf("txn:updates:%s", companyID)
	return s.rdb.Publish(ctx, channel, payload).Err()
}

func (s *RedisStore) EnqueueLedgerJob(ctx context.Context, payload []byte) error {
	return s.rdb.RPush(ctx, "queue:ledger:posts", payload).Err()
}

func (s *RedisStore) DequeueLedgerJob(ctx context.Context, timeout time.Duration) ([]byte, error) {
	res, err := s.rdb.BLPop(ctx, timeout, "queue:ledger:posts").Result()
	if err != nil {
		return nil, err
	}
	if len(res) < 2 {
		return nil, fmt.Errorf("invalid list pop result")
	}
	return []byte(res[1]), nil
}

func (s *RedisStore) EnqueueDLQJob(ctx context.Context, payload []byte) error {
	return s.rdb.RPush(ctx, "queue:ledger:dlq", payload).Err()
}

func (s *RedisStore) SubscribeTransactionUpdates(ctx context.Context, companyID string) (<-chan string, func()) {
	channel := fmt.Sprintf("txn:updates:%s", companyID)
	pubsub := s.rdb.Subscribe(ctx, channel)
	out := make(chan string, 10)

	go func() {
		defer close(out)
		ch := pubsub.Channel()
		for msg := range ch {
			out <- msg.Payload
		}
	}()

	cleanup := func() {
		_ = pubsub.Close()
	}

	return out, cleanup
}


