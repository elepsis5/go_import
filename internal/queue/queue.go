package queue

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"os"
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
	logger *slog.Logger
}

func New(cfg config.Config) *Client {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level:     slog.LevelDebug,
		AddSource: true, // ← вот переключатель файла и строки
	}))

	logger.Info("", "cfg.RedisAddr", cfg.RedisAddr)
	logger.Info("", "cfg.RedisPassword", cfg.RedisPassword)
	logger.Info("", "cfg.RedisDB", cfg.RedisDB)

	// --- Redis ---
	redisAddr := os.Getenv("REDIS_ADDR")
	rdb := redis.NewClient(&redis.Options{
		Addr:     redisAddr,
		Password: os.Getenv("REDIS_PASSWORD"),
		DB:       0,
	})
	if err := rdb.Ping(ctx).Err(); err != nil {
		slog.Error("redis ping FAILED:", "error", err)
	}
	logger.Info("redis ping OK!", "host", redisAddr)

	return &Client{
		Client: rdb,
		key:    cfg.QueueKey,
		logger: logger,
	}
	// return &Client{
	// 	Client: redis.NewClient(&redis.Options{
	// 		Addr:     cfg.RedisAddr,
	// 		Password: cfg.RedisPassword,
	// 		DB:       cfg.RedisDB,
	// 	}),
	// 	key: cfg.QueueKey,
	// }
}

// Consume блокируется до появления сообщения (BLPOP).
// Таймаут 5 сек нужен, чтобы корректно реагировать на shutdown.
func (c *Client) Consume(ctx context.Context) (*ImportJob, error) {
	res, err := c.Client.BLPop(ctx, 5*time.Second, c.key).Result()
	if err != nil {
		if ctx.Err() != nil {
			// Для BLPop таймаут — это нормальное поведение, логируем как «тихий таймаут»
			c.logger.Info("тихий таймаут BLPop")
			return nil, nil // считаем это не ошибкой, а нормальным завершением цикла воркера
		}
		// Таймаут BLPop — нормальная ситуация для воркера, очередь пуста
		if errors.Is(err, context.DeadlineExceeded) {
			return nil, nil
		}
		// Реальная ошибка: сеть, Redis упал и т.п. — сообщаем наверх
		c.logger.Error("BLPop failed",
			slog.String("key", c.key),
			slog.String("err", err.Error()),
		)
		return nil, err
	}
	c.logger.Info("содержимое res", slog.Any("res", res))

	// res[0] — имя ключа, res[1] — payload
	var job ImportJob
	if err := json.Unmarshal([]byte(res[1]), &job); err != nil {
		return nil, err // битое сообщение — логируем и идём дальше
	}
	return &job, nil
}
