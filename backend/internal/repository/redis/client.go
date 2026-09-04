package redis

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

type Client struct {
	RDB           *redis.Client
	reserveSHA    string
	rateLimitSHA  string
}

// Lua script to atomically check and reserve stock without race conditions
const reserveStockScript = `
local stock_key = KEYS[1]
local requested = tonumber(ARGV[1])
local current = tonumber(redis.call('get', stock_key) or '-1')
if current == -1 then
    return -1
end
if current >= requested then
    redis.call('decrby', stock_key, requested)
    return 1
else
    return 0
end
`

// Lua script for atomic sliding window rate limiting
const rateLimitScript = `
local key = KEYS[1]
local limit = tonumber(ARGV[1])
local window = tonumber(ARGV[2])
local current = redis.call('INCR', key)
if current == 1 then
    redis.call('EXPIRE', key, window)
end
if current > limit then
    return 0
else
    return 1
end
`

func NewClient(addr string) (*Client, error) {
	rdb := redis.NewClient(&redis.Options{
		Addr:         addr,
		Password:     "",
		DB:           0,
		PoolSize:     300,
		MinIdleConns: 50,
		DialTimeout:  2 * time.Second,
		ReadTimeout:  1 * time.Second,
		WriteTimeout: 1 * time.Second,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	if err := rdb.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to redis at %s: %w", addr, err)
	}

	// Pre-load Lua scripts for ultra-fast EVALSHA execution
	shaReserve, err := rdb.ScriptLoad(ctx, reserveStockScript).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to load stock reservation lua script: %w", err)
	}

	shaRate, err := rdb.ScriptLoad(ctx, rateLimitScript).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to load rate limit lua script: %w", err)
	}

	return &Client{
		RDB:          rdb,
		reserveSHA:   shaReserve,
		rateLimitSHA: shaRate,
	}, nil
}

func (c *Client) Get(ctx context.Context, key string) (string, error) {
	return c.RDB.Get(ctx, key).Result()
}

func (c *Client) Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error {
	return c.RDB.Set(ctx, key, value, ttl).Err()
}

func (c *Client) SetStock(ctx context.Context, productID int64, stock int) error {
	key := fmt.Sprintf("stock:%d", productID)
	return c.RDB.Set(ctx, key, stock, 24*time.Hour).Err()
}

// ReserveStock returns 1 if successfully reserved, 0 if out of stock, -1 if stock not found in Redis
func (c *Client) ReserveStock(ctx context.Context, productID int64, quantity int) (int64, error) {
	key := fmt.Sprintf("stock:%d", productID)
	val, err := c.RDB.EvalSha(ctx, c.reserveSHA, []string{key}, quantity).Result()
	if err != nil {
		val, err = c.RDB.Eval(ctx, reserveStockScript, []string{key}, quantity).Result()
		if err != nil {
			return 0, err
		}
	}
	res, ok := val.(int64)
	if !ok {
		return 0, fmt.Errorf("unexpected return type from lua script: %T", val)
	}
	return res, nil
}

// AllowRequest checks rate limit atomically using Token/Counter bucket
func (c *Client) AllowRequest(ctx context.Context, key string, limit int, windowSeconds int) (bool, error) {
	val, err := c.RDB.EvalSha(ctx, c.rateLimitSHA, []string{key}, limit, windowSeconds).Result()
	if err != nil {
		val, err = c.RDB.Eval(ctx, rateLimitScript, []string{key}, limit, windowSeconds).Result()
		if err != nil {
			return true, err // Fallback to allowing on Redis failure to prevent total outage
		}
	}
	res, ok := val.(int64)
	if !ok {
		return true, nil
	}
	return res == 1, nil
}

// Publish broadcasts event to a Redis Pub/Sub channel
func (c *Client) Publish(ctx context.Context, channel string, message interface{}) error {
	return c.RDB.Publish(ctx, channel, message).Err()
}

// Subscribe returns a PubSub subscription to specified channels
func (c *Client) Subscribe(ctx context.Context, channels ...string) *redis.PubSub {
	return c.RDB.Subscribe(ctx, channels...)
}

func (c *Client) Close() error {
	return c.RDB.Close()
}
