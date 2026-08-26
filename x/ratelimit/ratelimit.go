package ratelimit

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
)

type Limit struct {
	Rate   int
	Burst  int
	Period time.Duration
}

func (l Limit) String() string {
	return fmt.Sprintf("%d req/%s (burst %d)", l.Rate, fmtDur(l.Period), l.Burst)
}

func (l Limit) IsZero() bool {
	return l == Limit{}
}

func fmtDur(d time.Duration) string {
	switch d {
	case time.Second:
		return "s"
	case time.Minute:
		return "m"
	case time.Hour:
		return "h"
	}

	return d.String()
}

func PerSecond(rate int) Limit {
	return Limit{
		Rate:   rate,
		Period: time.Second,
		Burst:  rate,
	}
}

func PerMinute(rate int) Limit {
	return Limit{
		Rate:   rate,
		Period: time.Minute,
		Burst:  rate,
	}
}

func PerHour(rate int) Limit {
	return Limit{
		Rate:   rate,
		Period: time.Hour,
		Burst:  rate,
	}
}

//------------------------------------------------------------------------------

// Limiter controls how frequently events are allowed to happen.
type Limiter struct {
	rdb *redis.Client
}

// NewLimiter returns a new Limiter.
func NewLimiter(rdb *redis.Client) *Limiter {
	return &Limiter{
		rdb: rdb,
	}
}

// Allow is a shortcut for AllowN(ctx, key, limit, 1).
func (l Limiter) Allow(ctx context.Context, key string, limit Limit) (*Result, error) {
	return l.AllowN(ctx, key, limit, 1)
}

// AllowN reports whether n events may happen at time now.
func (l Limiter) AllowN(
	ctx context.Context,
	key string,
	limit Limit,
	n int,
) (*Result, error) {
	values := []any{limit.Burst, limit.Rate, limit.Period.Seconds(), n}

	v, err := luaAllowN.Run(ctx, l.rdb, []string{key}, values...).Result()
	if err != nil {
		return nil, err
	}

	values, ok := v.([]any)
	if !ok || len(values) < 4 {
		return nil, fmt.Errorf("ratelimit: unexpected redis result %T", v)
	}

	return parseAllowResult(limit, values)
}

// AllowAtMost reports whether at most n events may happen at time now.
// It returns the number of allowed events that is less than or equal to n.
func (l Limiter) AllowAtMost(
	ctx context.Context,
	key string,
	limit Limit,
	n int,
) (*Result, error) {
	values := []any{limit.Burst, limit.Rate, limit.Period.Seconds(), n}

	v, err := luaAllowAtMost.Run(ctx, l.rdb, []string{key}, values...).Result()
	if err != nil {
		return nil, err
	}

	values, ok := v.([]any)
	if !ok || len(values) < 4 {
		return nil, fmt.Errorf("ratelimit: unexpected redis result %T", v)
	}

	return parseAllowResult(limit, values)
}

func parseAllowResult(limit Limit, values []any) (*Result, error) {
	allowed, err := redisInt(values[0])
	if err != nil {
		return nil, fmt.Errorf("ratelimit allowed: %w", err)
	}

	remaining, err := redisInt(values[1])
	if err != nil {
		return nil, fmt.Errorf("ratelimit remaining: %w", err)
	}

	retryAfter, err := redisFloat(values[2])
	if err != nil {
		return nil, fmt.Errorf("ratelimit retry_after: %w", err)
	}

	resetAfter, err := redisFloat(values[3])
	if err != nil {
		return nil, fmt.Errorf("ratelimit reset_after: %w", err)
	}

	return &Result{
		Limit:      limit,
		Allowed:    allowed,
		Remaining:  remaining,
		RetryAfter: dur(retryAfter),
		ResetAfter: dur(resetAfter),
	}, nil
}

// redisInt converts Redis Lua numeric results. Whole numbers often arrive as
// int64; allow_n.lua returns remaining as a float (division), which go-redis
// surfaces as float64 — a hard int64 assert panics on every successful Allow.
func redisInt(v any) (int, error) {
	switch n := v.(type) {
	case int64:
		return int(n), nil
	case int:
		return n, nil
	case float64:
		return int(n), nil
	case string:
		i, err := strconv.Atoi(n)

		return i, err
	default:
		return 0, fmt.Errorf("unsupported numeric type %T", v)
	}
}

func redisFloat(v any) (float64, error) {
	switch n := v.(type) {
	case string:
		return strconv.ParseFloat(n, 64)
	case float64:
		return n, nil
	case int64:
		return float64(n), nil
	case int:
		return float64(n), nil
	default:
		return 0, fmt.Errorf("unsupported float type %T", v)
	}
}

// Reset gets a key and reset all limitations and previous usages
func (l *Limiter) Reset(ctx context.Context, key string) error {
	return l.rdb.Del(ctx, key).Err()
}

func dur(f float64) time.Duration {
	if f == -1 {
		return -1
	}

	return time.Duration(f * float64(time.Second))
}

type Result struct {
	// Limit is the limit that was used to obtain this result.
	Limit Limit

	// Allowed is the number of events that may happen at time now.
	Allowed int

	// Remaining is the maximum number of requests that could be
	// permitted instantaneously for this key given the current
	// state. For example, if a rate limiter allows 10 requests per
	// second and has already received 6 requests for this key this
	// second, the Remaining would be 4.
	Remaining int

	// RetryAfter is the time until the next request will be permitted.
	// It should be -1 unless the rate limit has been exceeded.
	RetryAfter time.Duration

	// ResetAfter is the time until the RateLimiter returns to its
	// initial state for a given key. For example, if a rate limiter
	// manages requests per second and received one request 200ms ago,
	// Reset would return 800ms. You can also think of this as the time
	// until Limit and Remaining will be equal.
	ResetAfter time.Duration
}
