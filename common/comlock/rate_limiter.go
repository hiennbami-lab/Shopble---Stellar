package comlock

import (
	"context"
	"time"

	"shopble/common/comerr"
	"shopble/common/comtypes"
	"shopble/glib/gconsts"

	"github.com/go-redis/redis_rate/v9"
)

type (
	RateLimiter interface {
		AllowOnce(ctx context.Context, key string, interval time.Duration) (*redis_rate.Result, error)
		Allow(ctx context.Context, key string, limit redis_rate.Limit) (*redis_rate.Result, error)
		AllowN(ctx context.Context, key string, limit redis_rate.Limit, n int) (*redis_rate.Result, error)
		Reset(ctx context.Context, key string) error
	}

	RateLimitRunner func(context.Context, func() error) error
)

var DefaultRateLimiter = comtypes.NewSingleton(func() *tOurRateLimiter {
	redisClient := GetRedisClient()
	var (
		limiter = redis_rate.NewLimiter(redisClient)
	)
	return &tOurRateLimiter{
		Limiter: limiter,
	}
})

type tOurRateLimiter struct {
	*redis_rate.Limiter
}

func (l *tOurRateLimiter) AllowOnce(
	ctx context.Context,
	key string, interval time.Duration,
) (*redis_rate.Result, error) {
	limit := redis_rate.Limit{
		Rate:   1,
		Burst:  1,
		Period: interval,
	}
	return l.Allow(ctx, key, limit)
}

func (l *tOurRateLimiter) Allow(
	ctx context.Context,
	key string, limit redis_rate.Limit,
) (*redis_rate.Result, error) {
	return l.AllowN(ctx, key, limit, 1)
}

func (l *tOurRateLimiter) AllowN(
	ctx context.Context,
	key string, limit redis_rate.Limit, n int,
) (*redis_rate.Result, error) {
	result, err := l.Limiter.AllowN(ctx, key, limit, n)
	if err != nil {
		return nil, err
	}
	return result, nil
}

func (l *tOurRateLimiter) Reset(ctx context.Context, key string) error {
	return l.Limiter.Reset(ctx, key)
}

func GenRateLimitRunner(
	limiter RateLimiter,
	key string, limit redis_rate.Limit, timeout time.Duration,
) RateLimitRunner {
	return func(ctx context.Context, runner func() error) error {
		return RunInRateLimit(ctx, limiter, key, limit, timeout, runner)
	}
}

func GenDefaultRateLimiterRunner(
	key string, limit redis_rate.Limit,
	timeout time.Duration,
) RateLimitRunner {
	return GenRateLimitRunner(DefaultRateLimiter.GetF(), key, limit, timeout)
}

func RunInRateLimit(
	ctx context.Context,
	limiter RateLimiter, key string, limit redis_rate.Limit,
	timeout time.Duration, runner func() error,
) error {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	for {
		select {
		case <-ctx.Done():
			return comerr.WrapMessage(gconsts.ErrorTimeOut, "limiter timeout")
		default:
			result, err := limiter.Allow(ctx, key, limit)
			if err != nil {
				return err
			}
			if result.Allowed > 0 {
				return runner()
			}
			time.Sleep(result.RetryAfter)
		}
	}
}
