// Package users defines the user domain and its persistence boundary.
package users

import (
	"context"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

// Profile contains the Telegram user fields intentionally retained by the bot.
// Telegram IDs require 64-bit storage.
type Profile struct {
	TelegramID   int64
	Username     string
	FirstName    string
	LastName     string
	LanguageCode string
}

// Repository is the persistence contract consumed by Telegram handlers.
type Repository interface {
	Upsert(context.Context, Profile) error
}

// MongoRepository persists users in MongoDB.
type MongoRepository struct {
	collection       *mongo.Collection
	operationTimeout time.Duration
}

// NewMongoRepository creates a MongoDB-backed user repository.
func NewMongoRepository(
	database *mongo.Database,
	collectionName string,
	operationTimeout time.Duration,
) *MongoRepository {
	return &MongoRepository{
		collection:       database.Collection(collectionName),
		operationTimeout: operationTimeout,
	}
}

// EnsureIndexes creates the unique Telegram user ID index required for safe
// concurrent upserts.
func (repository *MongoRepository) EnsureIndexes(ctx context.Context) error {
	operationContext, cancel := context.WithTimeout(ctx, repository.operationTimeout)
	defer cancel()

	_, err := repository.collection.Indexes().CreateOne(operationContext, mongo.IndexModel{
		Keys: bson.D{{Key: "telegram_id", Value: 1}},
		Options: options.Index().
			SetName("users_telegram_id_unique").
			SetUnique(true),
	})
	if err != nil {
		return fmt.Errorf("create users index: %w", err)
	}
	return nil
}

// Upsert creates a user on first contact and refreshes mutable Telegram fields
// on subsequent interactions.
func (repository *MongoRepository) Upsert(ctx context.Context, profile Profile) error {
	operationContext, cancel := context.WithTimeout(ctx, repository.operationTimeout)
	defer cancel()

	now := time.Now().UTC()
	filter := bson.D{{Key: "telegram_id", Value: profile.TelegramID}}
	update := bson.D{
		{Key: "$set", Value: bson.D{
			{Key: "username", Value: profile.Username},
			{Key: "first_name", Value: profile.FirstName},
			{Key: "last_name", Value: profile.LastName},
			{Key: "language_code", Value: profile.LanguageCode},
			{Key: "last_seen_at", Value: now},
			{Key: "updated_at", Value: now},
		}},
		{Key: "$setOnInsert", Value: bson.D{
			{Key: "telegram_id", Value: profile.TelegramID},
			{Key: "created_at", Value: now},
		}},
	}

	if _, err := repository.collection.UpdateOne(
		operationContext,
		filter,
		update,
		options.UpdateOne().SetUpsert(true),
	); err != nil {
		return fmt.Errorf("upsert Telegram user: %w", err)
	}

	return nil
}
