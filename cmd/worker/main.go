package main

import (
	"context"
	"log"
	"log/slog"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"test/internal/config"
	"test/internal/importer"
	"test/internal/mongo"
	"test/internal/queue"

	// "go.mongodb.org/mongo-driver/mongo"

	"go.mongodb.org/mongo-driver/mongo/readpref"
)

func main() {
	cfg := config.FromEnv()

	// Подкоманда healthcheck: вызывается из docker-compose healthcheck.
	// Пингует mongo и redis, завершается с кодом 0 (OK) или 1 (FAIL).
	if len(os.Args) > 1 && os.Args[1] == "healthcheck" {
		os.Exit(runHealthcheck(cfg))
	}

	runWorker(cfg)
}

// runHealthcheck — быстрый прогон соединений, без запуска воркеров.
func runHealthcheck(cfg config.Config) int {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	mClient, err := mongo.Connect(ctx, cfg)
	if err != nil {
		log.Printf("healthcheck: mongo connect failed: %v", err)
		return 1
	}
	defer mClient.Disconnect(context.Background())

	if err := mClient.Client.Ping(ctx, readpref.Primary()); err != nil {
		log.Printf("healthcheck: mongo ping failed: %v", err)
		return 1
	}

	q := queue.New(cfg)
	if err := q.Client.Ping(ctx); err != nil {
		log.Printf("healthcheck: redis ping failed: %v", err)
		return 1
	}

	log.Printf("healthcheck: OK (mongo=%s db=%s redis=%s)", cfg.MongoURI, cfg.MongoDB, cfg.RedisAddr)
	return 0
}

// runWorker — основной режим: вечный цикл чтения очереди + пул горутин.
func runWorker(cfg config.Config) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level:     slog.LevelDebug,
		AddSource: true, // ← вот переключатель файла и строки
	}))
	mClient, err := mongo.Connect(ctx, cfg)
	if err != nil {
		logger.Error("mongo connect:", slog.Any("error", err))
		return
	}
	defer mClient.Disconnect(context.Background())

	// Проверяем связь сразу на старте: если mongo недоступна,
	// контейнер упадёт и docker его перезапустит — лучше, чем висеть в ошибках.
	ctxPing, cancelPing := context.WithTimeout(ctx, 5*time.Second)
	defer cancelPing()
	if err := mClient.Client.Ping(ctxPing, readpref.Primary()); err != nil {
		logger.Error("mongo ping:", "error", err)
	}

	q := queue.New(cfg)

	if err := q.Client.Ping(ctx).Err(); err != nil { // ← .Err() — вот чего не хватало
		logger.Error("redis ping error", slog.String("err", err.Error()))
	}

	imp, err := importer.New(mClient.Client.Database(cfg.MongoDB))
	if err != nil {
		logger.Error("importer not created:", "error", err)
	}
	// Graceful shutdown: ловим SIGINT (Ctrl+C) и SIGTERM (docker stop).
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)

	logger.Info("worker started", slog.String("queue", cfg.QueueKey), slog.Int("workers", cfg.Workers))

	var wg sync.WaitGroup
	for w := 0; w < cfg.Workers; w++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for {
				select {
				case <-ctx.Done():
					logger.Debug("лог ctx.Done()") ///////////////////////////////////////////////////

					return
				default:
				}
				logger.Debug("лог перед q.Consume") ///////////////////////////////////////////////////

				job, err := q.Consume(ctx)
				if err != nil {
					logger.Debug("queue error", slog.Int("worker", id), slog.String("err", err.Error()))
					time.Sleep(time.Second) // не долбим redis при сбоях
					continue
				}
				if job == nil { // таймаут BLPOP — тихо ждём дальше
					continue
				}

				logger.Debug("лог перед imp.Execute") ///////////////////////////////////////////////////

				start := time.Now()
				if err := imp.Execute(ctx, job); err != nil {
					logger.Debug(
						"worker execute failed",
						slog.Int("worker", id),
						slog.String("import", job.ImportID),
						slog.String("supplier", job.SupplierID),
					)
					continue
				}
				logger.Debug(
					"worker execute failed",
					slog.Int("worker", id),
					slog.String("import", job.ImportID),
					slog.String("supplier", job.SupplierID),
				)
				logger.Debug("[worker %d] import=%s supplier=%s done in %s (%d products)",
					id, job.ImportID, job.SupplierID, time.Since(start), len(job.ProductIDs))
			}
		}(w)
	}

	<-sig
	logger.Info("shutting down: waiting for current jobs...")

	cancel()  // горутины выходят из Consume по ctx.Done()
	wg.Wait() // дожидаемся завершения текущих Execute
	logger.Info("shutdown complete")
}
