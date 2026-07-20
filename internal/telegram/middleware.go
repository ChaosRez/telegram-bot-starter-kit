package telegram

import (
	"context"
	"log/slog"
	"runtime/debug"
	"time"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)

// Handler adapts the error-returning dispatcher to the Telegram library.
func Handler(dispatcher *Dispatcher, logger *slog.Logger) bot.HandlerFunc {
	return func(ctx context.Context, telegramBot *bot.Bot, update *models.Update) {
		started := time.Now()
		attributes := []any{
			"update_id", update.ID,
			"update_type", updateType(update),
		}
		if err := dispatcher.Handle(ctx, telegramBot, update); err != nil {
			logger.ErrorContext(ctx, "update handled",
				append(attributes,
					"outcome", "error",
					"duration_ms", time.Since(started).Milliseconds(),
					"error", err,
				)...,
			)
			return
		}
		logger.InfoContext(ctx, "update handled",
			append(attributes,
				"outcome", "success",
				"duration_ms", time.Since(started).Milliseconds(),
			)...,
		)
	}
}

// Recovery prevents a malformed update or application panic from terminating
// the polling process.
func Recovery(logger *slog.Logger) bot.Middleware {
	return func(next bot.HandlerFunc) bot.HandlerFunc {
		return func(ctx context.Context, telegramBot *bot.Bot, update *models.Update) {
			defer func() {
				if recovered := recover(); recovered != nil {
					logger.ErrorContext(ctx, "panic while handling update",
						"update_id", update.ID,
						"panic", recovered,
						"stack", string(debug.Stack()),
					)
				}
			}()
			next(ctx, telegramBot, update)
		}
	}
}

func updateType(update *models.Update) string {
	switch {
	case update.Message != nil:
		return "message"
	case update.EditedMessage != nil:
		return "edited_message"
	case update.CallbackQuery != nil:
		return "callback_query"
	case update.InlineQuery != nil:
		return "inline_query"
	case update.MyChatMember != nil:
		return "my_chat_member"
	default:
		return "other"
	}
}
