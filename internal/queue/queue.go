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

	// logger.Info("", "cfg.RedisAddr", cfg.RedisAddr)
	// logger.Info("", "cfg.RedisPassword", cfg.RedisPassword)
	// logger.Info("", "cfg.RedisDB", cfg.RedisDB)

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
	slog.Info("redis ping OK!", "host", redisAddr)

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
	res, err := c.Client.BLPop(ctx, 1*time.Second, c.key).Result()
	if err != nil {
		switch {
		case ctx.Err() != nil:
			slog.Info("Контекст отменён снаружи — graceful shutdown")
			// Контекст отменён снаружи — graceful shutdown
			return nil, nil

		case errors.Is(err, redis.Nil):
			slog.Info("Таймаут BLPOP истёк, очередь пуста — это нормально.")
			// Таймаут BLPOP истёк, очередь пуста — это нормально.
			// Воркер просто продолжает цикл.
			return nil, nil

		default:
			slog.Info("Реальная ошибка: сеть, Redis недоступен и т.п.")
			// Реальная ошибка: сеть, Redis недоступен и т.п.

			return nil, err
		}
	}

	err = c.RestoreRaw(ctx, res[1])
	if err != nil {
		c.logger.Error("LPush failed",
			slog.String("err", err.Error()),
		)
	}
	// c.logger.Info("содержимое res", slog.Any("res", res))

	// res[0] — имя ключа, res[1] — payload
	var job ImportJob
	if err := json.Unmarshal([]byte(res[1]), &job); err != nil {
		return nil, err // битое сообщение — логируем и идём дальше
	}
	return &job, nil
}

// RestoreRaw возвращает оригинальный payload в очередь
func (c *Client) RestoreRaw(ctx context.Context, payload string) error {
	return c.Client.LPush(ctx, c.key, payload).Err()
}
