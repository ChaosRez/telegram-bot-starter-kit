// Package telegram adapts Telegram updates to application handlers.
package telegram

import (
	"context"
	"fmt"
	"strings"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"

	"github.com/rezamalek/telegram-bot-starter-kit/internal/localization"
	"github.com/rezamalek/telegram-bot-starter-kit/internal/users"
)

const aboutCallback = "about"

// API is the Telegram behavior used by handlers. Keeping this interface small
// makes handlers testable without calling Telegram.
type API interface {
	SendMessage(context.Context, *bot.SendMessageParams) (*models.Message, error)
	AnswerCallbackQuery(context.Context, *bot.AnswerCallbackQueryParams) (bool, error)
}

// Dispatcher routes supported updates to focused handlers.
type Dispatcher struct {
	users     users.Repository
	localizer *localization.Localizer
}

// NewDispatcher constructs an update dispatcher.
func NewDispatcher(repository users.Repository, localizer *localization.Localizer) *Dispatcher {
	return &Dispatcher{users: repository, localizer: localizer}
}

// Handle processes one Telegram update.
func (dispatcher *Dispatcher) Handle(ctx context.Context, api API, update *models.Update) error {
	if update.CallbackQuery != nil {
		if err := dispatcher.handleCallback(ctx, api, update.CallbackQuery); err != nil {
			return err
		}
		if err := dispatcher.persistUser(ctx, &update.CallbackQuery.From); err != nil {
			return err
		}
		return nil
	}
	if update.Message == nil {
		return nil
	}
	if err := dispatcher.persistUser(ctx, update.Message.From); err != nil {
		return err
	}

	command, ok := commandFromMessage(update.Message)
	if !ok {
		return nil
	}

	switch command {
	case "start":
		return dispatcher.start(ctx, api, update.Message)
	case "help":
		return dispatcher.sendLocalized(ctx, api, update.Message, "help_message", nil)
	case "stop":
		return dispatcher.sendLocalized(ctx, api, update.Message, "farewell_message", nil)
	default:
		return dispatcher.sendLocalized(ctx, api, update.Message, "unknown_command", nil)
	}
}

func (dispatcher *Dispatcher) start(ctx context.Context, api API, message *models.Message) error {
	locale := ""
	if message.From != nil {
		locale = message.From.LanguageCode
	}

	markup := models.InlineKeyboardMarkup{InlineKeyboard: [][]models.InlineKeyboardButton{{
		{
			Text:         dispatcher.localizer.Text(locale, "about_button"),
			CallbackData: aboutCallback,
		},
	}}}
	return dispatcher.sendLocalized(ctx, api, message, "greeting_message", markup)
}

func (dispatcher *Dispatcher) handleCallback(
	ctx context.Context,
	api API,
	query *models.CallbackQuery,
) error {
	key := "unknown_callback"
	if query.Data == aboutCallback {
		key = "about_message"
	}

	_, err := api.AnswerCallbackQuery(ctx, &bot.AnswerCallbackQueryParams{
		CallbackQueryID: query.ID,
		Text:            dispatcher.localizer.Text(query.From.LanguageCode, key),
	})
	if err != nil {
		return fmt.Errorf("answer callback query: %w", err)
	}
	return nil
}

func (dispatcher *Dispatcher) persistUser(ctx context.Context, user *models.User) error {
	if user == nil {
		return nil
	}
	if err := dispatcher.users.Upsert(ctx, users.Profile{
		TelegramID:   user.ID,
		Username:     user.Username,
		FirstName:    user.FirstName,
		LastName:     user.LastName,
		LanguageCode: user.LanguageCode,
	}); err != nil {
		return fmt.Errorf("save user: %w", err)
	}
	return nil
}

func (dispatcher *Dispatcher) sendLocalized(
	ctx context.Context,
	api API,
	message *models.Message,
	key string,
	replyMarkup models.ReplyMarkup,
) error {
	locale := ""
	if message.From != nil {
		locale = message.From.LanguageCode
	}

	_, err := api.SendMessage(ctx, &bot.SendMessageParams{
		ChatID:      message.Chat.ID,
		Text:        dispatcher.localizer.Text(locale, key),
		ReplyMarkup: replyMarkup,
	})
	if err != nil {
		return fmt.Errorf("send %s: %w", key, err)
	}
	return nil
}

// commandFromMessage reads Telegram's bot_command entity instead of relying on
// a prefix regex. Telegram commands are ASCII, so their UTF-16 entity length is
// also a safe byte boundary.
func commandFromMessage(message *models.Message) (string, bool) {
	for _, entity := range message.Entities {
		if entity.Type != models.MessageEntityTypeBotCommand || entity.Offset != 0 {
			continue
		}
		if entity.Length < 2 || entity.Length > len(message.Text) || message.Text[0] != '/' {
			return "", false
		}

		command := message.Text[1:entity.Length]
		if separator := strings.IndexByte(command, '@'); separator >= 0 {
			command = command[:separator]
		}
		command = strings.ToLower(strings.TrimSpace(command))
		return command, command != ""
	}
	return "", false
}
