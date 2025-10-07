package middleware

import (
	"context"
	"fmt"
	"time"

	"github.com/conan-flynn/cronnect/database"
	"github.com/redis/go-redis/v9"
)

const (
	// Maybe need to lower this? 
	MaxPingsPerHour = 100
	RateLimitWindow = time.Hour
)

type RateLimiter struct {
	redis *redis.Client
	ctx   context.Context
}

func NewRateLimiter() *RateLimiter {
	return &RateLimiter{
		redis: database.RedisClient,
		ctx:   context.Background(),
	}
}

func (rl *RateLimiter) CheckRateLimit(userID string) (allowed bool, remaining int, resetAt time.Time, err error) {
	key := fmt.Sprintf("rate_limit:%s", userID)
	now := time.Now()
	windowStart := now.Add(-RateLimitWindow)

	_, err = rl.redis.ZRemRangeByScore(rl.ctx, key, "0", fmt.Sprintf("%d", windowStart.Unix())).Result()
	if err != nil {
		return false, 0, time.Time{}, err
	}

	count, err := rl.redis.ZCard(rl.ctx, key).Result()
	if err != nil {
		return false, 0, time.Time{}, err
	}

	remaining = MaxPingsPerHour - int(count)
	if remaining < 0 {
		remaining = 0
	}

	oldestEntries, err := rl.redis.ZRangeWithScores(rl.ctx, key, 0, 0).Result()
	if err != nil {
		return false, remaining, time.Time{}, err
	}

	if len(oldestEntries) > 0 {
		oldestTime := time.Unix(int64(oldestEntries[0].Score), 0)
		resetAt = oldestTime.Add(RateLimitWindow)
	} else {
		resetAt = now.Add(RateLimitWindow)
	}

	if count >= MaxPingsPerHour {
		return false, 0, resetAt, nil
	}

	return true, remaining, resetAt, nil
}

func (rl *RateLimiter) RecordPing(userID string) error {
	key := fmt.Sprintf("rate_limit:%s", userID)
	now := time.Now()

	_, err := rl.redis.ZAdd(rl.ctx, key, redis.Z{
		Score:  float64(now.Unix()),
		Member: fmt.Sprintf("%d", now.UnixNano()),
	}).Result()
	if err != nil {
		return err
	}

	_, err = rl.redis.Expire(rl.ctx, key, RateLimitWindow*2).Result()
	return err
}

func (rl *RateLimiter) GetRateLimitStatus(userID string) (used int, remaining int, limit int, resetAt time.Time, err error) {
	key := fmt.Sprintf("rate_limit:%s", userID)
	now := time.Now()
	windowStart := now.Add(-RateLimitWindow)

	_, err = rl.redis.ZRemRangeByScore(rl.ctx, key, "0", fmt.Sprintf("%d", windowStart.Unix())).Result()
	if err != nil {
		return 0, 0, MaxPingsPerHour, time.Time{}, err
	}

	count, err := rl.redis.ZCard(rl.ctx, key).Result()
	if err != nil {
		return 0, 0, MaxPingsPerHour, time.Time{}, err
	}

	used = int(count)
	remaining = MaxPingsPerHour - used
	if remaining < 0 {
		remaining = 0
	}
	limit = MaxPingsPerHour

	oldestEntries, err := rl.redis.ZRangeWithScores(rl.ctx, key, 0, 0).Result()
	if err != nil {
		return used, remaining, limit, time.Time{}, err
	}

	if len(oldestEntries) > 0 {
		oldestTime := time.Unix(int64(oldestEntries[0].Score), 0)
		resetAt = oldestTime.Add(RateLimitWindow)
	} else {
		resetAt = now.Add(RateLimitWindow)
	}

	return used, remaining, limit, resetAt, nil
}

