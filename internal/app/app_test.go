package app

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/go-telegram/bot/models"
)

func TestEnsurePollingAvailable(t *testing.T) {
	telegramAPI := &fakeWebhookInfoAPI{info: &models.WebhookInfo{}}

	if err := ensurePollingAvailable(context.Background(), telegramAPI); err != nil {
		t.Fatalf("ensurePollingAvailable() error = %v", err)
	}
}

func TestEnsurePollingAvailableRejectsConfiguredWebhook(t *testing.T) {
	const webhookURL = "https://example.com/bot/secret"
	telegramAPI := &fakeWebhookInfoAPI{info: &models.WebhookInfo{URL: webhookURL}}

	err := ensurePollingAvailable(context.Background(), telegramAPI)
	if err == nil || !strings.Contains(err.Error(), "webhook is configured") {
		t.Fatalf("ensurePollingAvailable() error = %v, want configured-webhook error", err)
	}
	if strings.Contains(err.Error(), webhookURL) {
		t.Fatalf("ensurePollingAvailable() error exposes webhook URL: %v", err)
	}
}

func TestEnsurePollingAvailableWrapsTelegramError(t *testing.T) {
	telegramAPI := &fakeWebhookInfoAPI{err: errors.New("network unavailable")}

	err := ensurePollingAvailable(context.Background(), telegramAPI)
	if err == nil || !strings.Contains(err.Error(), "inspect Telegram webhook") {
		t.Fatalf("ensurePollingAvailable() error = %v, want Telegram error", err)
	}
}

type fakeWebhookInfoAPI struct {
	info *models.WebhookInfo
	err  error
}

func (api *fakeWebhookInfoAPI) GetWebhookInfo(context.Context) (*models.WebhookInfo, error) {
	return api.info, api.err
}
