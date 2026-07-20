package telegram

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"strings"
	"testing"

	"github.com/go-telegram/bot/models"
)

func TestHandlerLogsSuccessfulOutcome(t *testing.T) {
	dispatcher, _ := newTestDispatcher(t)
	update := &models.Update{ID: 11, Message: &models.Message{Text: "hello"}}

	record := runHandlerAndReadLog(t, dispatcher, update)
	if record["outcome"] != "success" {
		t.Errorf("outcome = %v, want success", record["outcome"])
	}
	if _, exists := record["error"]; exists {
		t.Errorf("error = %v, want field omitted", record["error"])
	}
}

func TestHandlerLogsFailedOutcome(t *testing.T) {
	dispatcher, repository := newTestDispatcher(t)
	repository.err = errors.New("database unavailable")

	record := runHandlerAndReadLog(t, dispatcher, commandUpdate("/start"))
	if record["outcome"] != "error" {
		t.Errorf("outcome = %v, want error", record["outcome"])
	}
	loggedError, ok := record["error"].(string)
	if !ok || !strings.Contains(loggedError, "save user") {
		t.Errorf("error = %v, want persistence error", record["error"])
	}
	if record["level"] != "ERROR" {
		t.Errorf("level = %v, want ERROR", record["level"])
	}
}

func runHandlerAndReadLog(t *testing.T, dispatcher *Dispatcher, update *models.Update) map[string]any {
	t.Helper()

	var output bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&output, nil))
	Handler(dispatcher, logger)(context.Background(), nil, update)

	lines := strings.Split(strings.TrimSpace(output.String()), "\n")
	if len(lines) != 1 {
		t.Fatalf("log lines = %d, want 1; output = %q", len(lines), output.String())
	}
	var record map[string]any
	if err := json.Unmarshal([]byte(lines[0]), &record); err != nil {
		t.Fatalf("decode log: %v", err)
	}
	if record["msg"] != "update handled" {
		t.Errorf("msg = %v, want update handled", record["msg"])
	}
	return record
}
