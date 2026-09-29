package database

import (
	"context"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

func NewMongoClient(ctx context.Context, uri string) (*mongo.Client, error) {
	clientOptions := options.Client().
		ApplyURI(uri).
		SetConnectTimeout(10 * time.Second).
		SetTimeout(5 * time.Second)

	client, err := mongo.Connect(clientOptions)

	if err != nil {
		return nil, fmt.Errorf("connect to mongodb: %w", err)
	}

	if err := client.Ping(ctx, nil); err != nil {
		_ = client.Disconnect(ctx)
		return nil, fmt.Errorf("ping mongodb: %w", err)
	}

	return client, nil
}
