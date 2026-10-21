package mongo

import (
	"context"

	"test/internal/config"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
)

// Client — обёртка над клиентом драйвера.
type Client struct {
	Client *mongo.Client
	DB     *mongo.Database
}

// Connect подключается к MongoDB по настройкам из конфига.
func Connect(ctx context.Context, cfg config.Config) (*Client, error) {
	opts := options.Client().
		ApplyURI(cfg.MongoURI).
		SetConnectTimeout(5 * 1000000000) // 5s, если захочешь вынести в config — дам ниже

	client, err := mongo.Connect(ctx, opts)
	if err != nil {
		return nil, err
	}

	if err := client.Ping(ctx, readpref.Primary()); err != nil {
		return nil, err
	}

	return &Client{
		Client: client,
		DB:     client.Database(cfg.MongoDB),
	}, nil
}

// Disconnect закрывает соединение.
func (c *Client) Disconnect(ctx context.Context) error {
	return c.Client.Disconnect(ctx)
}
