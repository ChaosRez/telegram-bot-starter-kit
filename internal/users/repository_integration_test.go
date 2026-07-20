//go:build integration

package users_test

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/rezamalek/telegram-bot-starter-kit/internal/database"
	"github.com/rezamalek/telegram-bot-starter-kit/internal/users"
)

func TestMongoRepositoryUpsert(t *testing.T) {
	uri := os.Getenv("MONGODB_URI")
	if uri == "" {
		t.Skip("MONGODB_URI is required for integration tests")
	}

	ctx := context.Background()
	client, err := database.Connect(ctx, uri, 10*time.Second, 5*time.Second)
	if err != nil {
		t.Fatalf("connect MongoDB: %v", err)
	}
	t.Cleanup(func() {
		shutdownContext, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		_ = client.Disconnect(shutdownContext)
	})

	databaseName := fmt.Sprintf("telegram_bot_test_%d", time.Now().UnixNano())
	db := client.Database(databaseName)
	t.Cleanup(func() {
		cleanupContext, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		_ = db.Drop(cleanupContext)
	})

	repository := users.NewMongoRepository(db, "users", 5*time.Second)
	if err := repository.EnsureIndexes(ctx); err != nil {
		t.Fatalf("EnsureIndexes() error = %v", err)
	}

	profile := users.Profile{
		TelegramID:   5_000_000_000,
		Username:     "first_username",
		FirstName:    "First",
		LanguageCode: "en",
	}
	if err := repository.Upsert(ctx, profile); err != nil {
		t.Fatalf("first Upsert() error = %v", err)
	}
	profile.Username = "updated_username"
	if err := repository.Upsert(ctx, profile); err != nil {
		t.Fatalf("second Upsert() error = %v", err)
	}

	var stored struct {
		TelegramID int64  `bson:"telegram_id"`
		Username   string `bson:"username"`
	}
	if err := db.Collection("users").FindOne(
		ctx,
		bson.D{{Key: "telegram_id", Value: profile.TelegramID}},
	).Decode(&stored); err != nil {
		t.Fatalf("find user: %v", err)
	}
	if stored.TelegramID != profile.TelegramID {
		t.Errorf("TelegramID = %d, want %d", stored.TelegramID, profile.TelegramID)
	}
	if stored.Username != "updated_username" {
		t.Errorf("Username = %q, want updated_username", stored.Username)
	}

	count, err := db.Collection("users").CountDocuments(ctx, bson.D{})
	if err != nil {
		t.Fatalf("count users: %v", err)
	}
	if count != 1 {
		t.Errorf("user count = %d, want 1", count)
	}
}
