// Package app wires infrastructure and starts the polling lifecycle.
package app

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"

	"github.com/rezamalek/telegram-bot-starter-kit/internal/config"
	"github.com/rezamalek/telegram-bot-starter-kit/internal/database"
	"github.com/rezamalek/telegram-bot-starter-kit/internal/localization"
	telegramadapter "github.com/rezamalek/telegram-bot-starter-kit/internal/telegram"
	"github.com/rezamalek/telegram-bot-starter-kit/internal/users"
)

const (
	shutdownTimeout        = 10 * time.Second
	telegramRequestTimeout = 10 * time.Second
)

type webhookInfoAPI interface {
	GetWebhookInfo(context.Context) (*models.WebhookInfo, error)
}

// Run starts MongoDB, builds the Telegram bot, and blocks while long polling is
// active. Cancel ctx to stop polling and close the database connection.
func Run(ctx context.Context, cfg config.Config, logger *slog.Logger) (runErr error) {
	mongoClient, err := database.Connect(
		ctx,
		cfg.MongoDB.URI,
		cfg.MongoDB.ConnectTimeout.Duration,
		cfg.MongoDB.OperationTimeout.Duration,
	)
	if err != nil {
		return err
	}
	defer func() {
		shutdownContext, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
		defer cancel()
		if err := mongoClient.Disconnect(shutdownContext); err != nil && runErr == nil {
			runErr = fmt.Errorf("disconnect MongoDB: %w", err)
		}
	}()

	userRepository := users.NewMongoRepository(
		mongoClient.Database(cfg.MongoDB.Database),
		cfg.MongoDB.UsersCollection,
		cfg.MongoDB.OperationTimeout.Duration,
	)
	if err := userRepository.EnsureIndexes(ctx); err != nil {
		return err
	}

	localizer, err := localization.New()
	if err != nil {
		return fmt.Errorf("load localizations: %w", err)
	}
	dispatcher := telegramadapter.NewDispatcher(userRepository, localizer)

	pollTimeout := cfg.Telegram.PollTimeout.Duration
	httpClient := &http.Client{Timeout: pollTimeout + 10*time.Second}
	telegramBot, err := bot.New(
		cfg.Telegram.Token,
		bot.WithHTTPClient(pollTimeout, httpClient),
		bot.WithAllowedUpdates(bot.AllowedUpdates(cfg.Telegram.AllowedUpdates)),
		bot.WithWorkers(cfg.Telegram.Workers),
		bot.WithNotAsyncHandlers(),
		bot.WithDefaultHandler(telegramadapter.Handler(dispatcher, logger)),
		bot.WithMiddlewares(
			telegramadapter.Recovery(logger),
		),
		bot.WithErrorsHandler(func(err error) {
			logger.Error("Telegram polling error", "error", err)
		}),
	)
	if err != nil {
		return fmt.Errorf("create Telegram bot: %w", err)
	}
	if err := ensurePollingAvailable(ctx, telegramBot); err != nil {
		return err
	}

	if cfg.Telegram.SyncCommands {
		syncCommands(ctx, telegramBot, logger)
	}

	logger.Info("bot started",
		"transport", "long_polling",
		"workers", cfg.Telegram.Workers,
		"database", cfg.MongoDB.Database,
	)
	telegramBot.Start(ctx)
	logger.Info("polling stopped")

	return nil
}

func ensurePollingAvailable(ctx context.Context, telegramAPI webhookInfoAPI) error {
	requestContext, cancel := context.WithTimeout(ctx, telegramRequestTimeout)
	defer cancel()

	webhook, err := telegramAPI.GetWebhookInfo(requestContext)
	if err != nil {
		return fmt.Errorf("inspect Telegram webhook: %w", err)
	}
	if webhook != nil && webhook.URL != "" {
		return errors.New("Telegram webhook is configured; delete it before starting long polling")
	}
	return nil
}

func syncCommands(ctx context.Context, telegramBot *bot.Bot, logger *slog.Logger) {
	requestContext, cancel := context.WithTimeout(ctx, telegramRequestTimeout)
	defer cancel()

	_, err := telegramBot.SetMyCommands(requestContext, &bot.SetMyCommandsParams{
		Commands: []models.BotCommand{
			{Command: "start", Description: "Start the bot"},
			{Command: "help", Description: "Show available commands"},
			{Command: "stop", Description: "Say goodbye"},
		},
	})
	if err != nil {
		logger.Warn("could not synchronize Telegram commands", "error", err)
	}
}
