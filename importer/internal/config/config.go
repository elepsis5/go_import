package config

import (
	"fmt"
	"os"
)

type Config struct {
	MongoURI      string
	MongoDB       string
	RedisAddr     string
	RedisPassword string
	RedisDB       int
	QueueKey      string
	Workers       int // сколько горутин обрабатывают очередь
}

func FromEnv() Config {
	return Config{
		MongoURI:      envOr("MONGO_URI", "mongodb://root:example@localhost:27017"),
		MongoDB:       envOr("MONGO_DB", "marketplace"),
		RedisAddr:     envOr("REDIS_ADDR", "localhost:6379"),
		RedisPassword: os.Getenv("REDIS_PASSWORD"),
		RedisDB:       0,
		QueueKey:      envOr("GO_IMPORT_QUEUE", "go:import:jobs"),
		Workers:       4,
	}
}

func envOr(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
