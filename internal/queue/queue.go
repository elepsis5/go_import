package queue

import (
	"context"
	"encoding/json"
	"log"
	"log/slog"
	"os"
	"time"

	"test/internal/config"

	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
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
	ctx, cancel := context.WithCancel(context.Background())
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
		Addr:     os.Getenv("REDIS_ADDR"),
		Password: os.Getenv("REDIS_PASSWORD"),
		DB:       0,
	})
	if err := rdb.Ping(ctx).Err(); err != nil {
		log.Fatalf("redis ping FAILED:", "error", err)
	}
	logger.Info("redis ping OK!", "host", redisAddr)

	return &Client{
		Client: rdb,
		key:    cfg.QueueKey,
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
	logger, err := zap.NewDevelopment()

	res, err := c.Client.BLPop(ctx, 5*time.Second, c.key).Result()
	if ctx.Err() != nil {
		// Для BLPop таймаут — это нормальное поведение, логируем как «тихий таймаут»
		logger.Info("тихий таймаут BLPop",
			zap.String("key", c.key),
			zap.Duration("timeout", 5*time.Second),
			zap.Error(err),
		)
		return nil, nil // считаем это не ошибкой, а нормальным завершением цикла воркера
	}
	logger.Info("содержимое res", zap.Any("res", res))

	// res[0] — имя ключа, res[1] — payload
	var job ImportJob
	if err := json.Unmarshal([]byte(res[1]), &job); err != nil {
		return nil, err // битое сообщение — логируем и идём дальше
	}
	return &job, nil
}
