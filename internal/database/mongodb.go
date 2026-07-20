// Package database owns MongoDB client lifecycle.
package database

import (
	"context"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

// Connect constructs a MongoDB client and verifies that a server is reachable.
func Connect(
	ctx context.Context,
	uri string,
	connectTimeout time.Duration,
	operationTimeout time.Duration,
) (*mongo.Client, error) {
	serverAPI := options.ServerAPI(options.ServerAPIVersion1)
	clientOptions := options.Client().
		ApplyURI(uri).
		SetAppName("telegram-bot-starter-kit").
		SetServerAPIOptions(serverAPI).
		SetTimeout(operationTimeout)

	client, err := mongo.Connect(clientOptions)
	if err != nil {
		return nil, fmt.Errorf("create MongoDB client: %w", err)
	}

	pingContext, cancel := context.WithTimeout(ctx, connectTimeout)
	defer cancel()

	if err := client.Database("admin").RunCommand(
		pingContext,
		bson.D{{Key: "ping", Value: 1}},
	).Err(); err != nil {
		disconnectContext, disconnectCancel := context.WithTimeout(context.Background(), connectTimeout)
		defer disconnectCancel()
		_ = client.Disconnect(disconnectContext)
		return nil, fmt.Errorf("ping MongoDB: %w", err)
	}

	return client, nil
}
