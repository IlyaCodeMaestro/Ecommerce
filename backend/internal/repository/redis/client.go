package redis

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

type Client struct {
	RDB         *redis.Client
	reserveSHA  string
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

	// Pre-load Lua script for ultra-fast EVALSHA execution
	sha, err := rdb.ScriptLoad(ctx, reserveStockScript).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to load stock reservation lua script: %w", err)
	}

	return &Client{
		RDB:        rdb,
		reserveSHA: sha,
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
		// Fallback to Eval if SHA was lost due to redis restart
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

func (c *Client) Close() error {
	return c.RDB.Close()
}
