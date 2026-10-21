package queue

import (
	"context"
	"encoding/json"
	"time"

	"test/internal/config"

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
	Client *redis.Client
	key    string
}

func New(cfg config.Config) *Client {
	return &Client{
		Client: redis.NewClient(&redis.Options{
			Addr:     cfg.RedisAddr,
			Password: cfg.RedisPassword,
			DB:       cfg.RedisDB,
		}),
		key: cfg.QueueKey,
	}
}

// Consume блокируется до появления сообщения (BLPOP).
// Таймаут 5 сек нужен, чтобы корректно реагировать на shutdown.
func (c *Client) Consume(ctx context.Context) (*ImportJob, error) {
	res, err := c.Client.BLPop(ctx, 5*time.Second, c.key).Result()
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
