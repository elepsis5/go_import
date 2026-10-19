package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/redis/go-redis/v9"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
)

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// --- MongoDB ---
	mongoURI := envOr("MONGO_URI", "mongodb://root:8d3wjRY8@172.17.0.1:27018")
	mClient, err := mongo.Connect(ctx, options.Client().ApplyURI(mongoURI))
	if err != nil {
		log.Fatalf("mongo connect: %v", err)
	}
	defer mClient.Disconnect(context.Background())

	if err := mClient.Ping(ctx, readpref.Primary()); err != nil {
		log.Fatalf("mongo ping FAILED: %v", err)
	}
	log.Printf("mongo ping OK (%s)", mongoURI)

	// бонус: список коллекций — сразу видно, что смотрим в ту базу
	dbName := envOr("MONGO_DB", "marketplace")
	colls, err := mClient.Database(dbName).ListCollectionNames(ctx, map[string]interface{}{})
	if err != nil {
		log.Fatalf("list collections: %v", err)
	}
	log.Printf("db %q collections: %v", dbName, colls)

	// --- Redis ---
	redisAddr := envOr("REDIS_ADDR", "redis:6379")
	rdb := redis.NewClient(&redis.Options{
		Addr:     redisAddr,
		Password: os.Getenv("REDIS_PASSWORD"),
	})
	if err := rdb.Ping(ctx).Err(); err != nil {
		log.Fatalf("redis ping FAILED: %v", err)
	}
	log.Printf("redis ping OK (%s)", redisAddr)

	fmt.Println("ALL CONNECTIONS OK")
}

func envOr(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
