package db

import (
	"context"
	"time"

	"goserver/internal/config"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type Client struct {
	client *mongo.Client
	db     *mongo.Database
}

func Connect(ctx context.Context, cfg config.Config) (*Client, error) {
	connectCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	client, err := mongo.Connect(connectCtx, options.Client().ApplyURI(cfg.MongoDB.URI))
	if err != nil {
		return nil, err
	}

	return &Client{
		client: client,
		db:     client.Database(cfg.MongoDB.DBName),
	}, nil
}

func (client *Client) Database() *mongo.Database {
	return client.db
}

func (client *Client) Close(ctx context.Context) error {
	if client == nil || client.client == nil {
		return nil
	}

	closeCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	return client.client.Disconnect(closeCtx)
}
