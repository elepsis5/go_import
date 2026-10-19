package queue

import (
	"context"
	"encoding/json"
	"time"

	"github.com/redis/go-redis/v9"
)

type ImportJob struct {
	ImportID   string   `json:"import_id"`
	SupplierID string   `json:"supplier_id"`
	ProductIDs []string `json:"product_ids"`
	Attempt    int      `json:"attempt"`
	EnqueuedAt string   `json:"enqueued_at"`
}

type Client struct {
	rdb *redis.Client
	key string
}

func New(addr, password string, db int, key string) *Client {
	return &Client{
		rdb: redis.NewClient(&redis.Options{
			Addr:     addr,
			Password: password,
			DB:       db,
		}),
		key: key,
	}
}

// Consume блокируется до появления сообщения (BLPOP).
// Таймаут 5 сек нужен, чтобы корректно реагировать на shutdown.
func (c *Client) Consume(ctx context.Context) (*ImportJob, error) {
	res, err := c.rdb.BLPop(ctx, 5*time.Second, c.key).Result()
	if err != nil {
		if err == redis.Nil || ctx.Err() != nil {
			return nil, nil // тихий таймаут — не ошибка
		}
		return nil, err
	}
	// res[0] — имя ключа, res[1] — payload
	var job ImportJob
	if err := json.Unmarshal([]byte(res[1]), &job); err != nil {
		return nil, err // битое сообщение — логируем и идём дальше
	}
	return &job, nil
}
