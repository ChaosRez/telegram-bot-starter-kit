package telegram

import (
	"context"
	"errors"
	"testing"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"

	"github.com/rezamalek/telegram-bot-starter-kit/internal/localization"
	"github.com/rezamalek/telegram-bot-starter-kit/internal/users"
)

func TestStartPersistsUserAndReplies(t *testing.T) {
	dispatcher, repository := newTestDispatcher(t)
	api := &fakeAPI{}
	update := commandUpdate("/start@starter_bot")

	if err := dispatcher.Handle(context.Background(), api, update); err != nil {
		t.Fatalf("Handle() error = %v", err)
	}

	if repository.profile.TelegramID != 42 {
		t.Errorf("stored TelegramID = %d", repository.profile.TelegramID)
	}
	if repository.profile.LanguageCode != "de-DE" {
		t.Errorf("stored LanguageCode = %q", repository.profile.LanguageCode)
	}
	if len(api.messages) != 1 {
		t.Fatalf("sent messages = %d, want 1", len(api.messages))
	}
	if api.messages[0].Text != "Hallo! Telegram ist großartig." {
		t.Errorf("message text = %q", api.messages[0].Text)
	}

	markup, ok := api.messages[0].ReplyMarkup.(models.InlineKeyboardMarkup)
	if !ok || len(markup.InlineKeyboard) != 1 {
		t.Fatalf("reply markup = %#v", api.messages[0].ReplyMarkup)
	}
	if markup.InlineKeyboard[0][0].CallbackData != aboutCallback {
		t.Errorf("callback data = %q", markup.InlineKeyboard[0][0].CallbackData)
	}
}

func TestUnknownCommandGetsHelpfulReply(t *testing.T) {
	dispatcher, _ := newTestDispatcher(t)
	api := &fakeAPI{}

	if err := dispatcher.Handle(context.Background(), api, commandUpdate("/wat")); err != nil {
		t.Fatalf("Handle() error = %v", err)
	}
	if len(api.messages) != 1 || api.messages[0].Text != "Diesen Befehl kenne ich nicht. Versuche /help." {
		t.Fatalf("messages = %#v", api.messages)
	}
}

func TestPlainTextIsIgnored(t *testing.T) {
	dispatcher, _ := newTestDispatcher(t)
	api := &fakeAPI{}
	update := &models.Update{Message: &models.Message{Text: "hello"}}

	if err := dispatcher.Handle(context.Background(), api, update); err != nil {
		t.Fatalf("Handle() error = %v", err)
	}
	if len(api.messages) != 0 {
		t.Fatalf("sent messages = %d, want 0", len(api.messages))
	}
}

func TestAboutCallbackIsAlwaysAnswered(t *testing.T) {
	dispatcher, _ := newTestDispatcher(t)
	api := &fakeAPI{}
	update := &models.Update{CallbackQuery: &models.CallbackQuery{
		ID:   "callback-1",
		Data: aboutCallback,
		From: models.User{LanguageCode: "de"},
	}}

	if err := dispatcher.Handle(context.Background(), api, update); err != nil {
		t.Fatalf("Handle() error = %v", err)
	}
	if len(api.answers) != 1 {
		t.Fatalf("callback answers = %d, want 1", len(api.answers))
	}
	if api.answers[0].Text != "Ein Polling-orientiertes Telegram-Bot-Starter-Kit in Go." {
		t.Errorf("callback answer = %q", api.answers[0].Text)
	}
}

func TestUnknownCallbackIsAnswered(t *testing.T) {
	dispatcher, _ := newTestDispatcher(t)
	api := &fakeAPI{}
	update := &models.Update{CallbackQuery: &models.CallbackQuery{
		ID:   "callback-2",
		Data: "outdated-action",
		From: models.User{ID: 42, LanguageCode: "en"},
	}}

	if err := dispatcher.Handle(context.Background(), api, update); err != nil {
		t.Fatalf("Handle() error = %v", err)
	}
	if len(api.answers) != 1 || api.answers[0].Text != "This action is no longer available." {
		t.Fatalf("callback answers = %#v", api.answers)
	}
}

func TestStartReturnsRepositoryErrorWithoutSending(t *testing.T) {
	dispatcher, repository := newTestDispatcher(t)
	repository.err = errors.New("database unavailable")
	api := &fakeAPI{}

	err := dispatcher.Handle(context.Background(), api, commandUpdate("/start"))
	if err == nil {
		t.Fatal("Handle() error = nil")
	}
	if len(api.messages) != 0 {
		t.Fatalf("sent messages = %d, want 0", len(api.messages))
	}
}

func newTestDispatcher(t *testing.T) (*Dispatcher, *fakeRepository) {
	t.Helper()
	localizer, err := localization.New()
	if err != nil {
		t.Fatalf("localization.New() error = %v", err)
	}
	repository := &fakeRepository{}
	return NewDispatcher(repository, localizer), repository
}

func commandUpdate(text string) *models.Update {
	return &models.Update{
		ID: 10,
		Message: &models.Message{
			Text: text,
			Entities: []models.MessageEntity{{
				Type:   models.MessageEntityTypeBotCommand,
				Offset: 0,
				Length: len(text),
			}},
			From: &models.User{
				ID:           42,
				Username:     "example",
				FirstName:    "Example",
				LanguageCode: "de-DE",
			},
			Chat: models.Chat{ID: 99},
		},
	}
}

type fakeRepository struct {
	profile users.Profile
	err     error
}

func (repository *fakeRepository) Upsert(_ context.Context, profile users.Profile) error {
	repository.profile = profile
	return repository.err
}

type fakeAPI struct {
	messages []*bot.SendMessageParams
	answers  []*bot.AnswerCallbackQueryParams
}

func (api *fakeAPI) SendMessage(
	_ context.Context,
	params *bot.SendMessageParams,
) (*models.Message, error) {
	api.messages = append(api.messages, params)
	return &models.Message{}, nil
}

func (api *fakeAPI) AnswerCallbackQuery(
	_ context.Context,
	params *bot.AnswerCallbackQueryParams,
) (bool, error) {
	api.answers = append(api.answers, params)
	return true, nil
}
