package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadAppliesEnvironmentOverrides(t *testing.T) {
	t.Setenv("TELEGRAM_BOT_TOKEN", "test-token")
	t.Setenv("MONGODB_URI", "mongodb://database.example:27017")
	t.Setenv("MONGODB_DATABASE", "overridden")
	t.Setenv("TELEGRAM_WORKERS", "4")
	t.Setenv("LOG_LEVEL", "debug")

	path := writeConfig(t, `{}`)
	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if cfg.Telegram.Token != "test-token" {
		t.Errorf("Telegram.Token = %q", cfg.Telegram.Token)
	}
	if cfg.Telegram.Workers != 4 {
		t.Errorf("Telegram.Workers = %d, want 4", cfg.Telegram.Workers)
	}
	if cfg.MongoDB.URI != "mongodb://database.example:27017" {
		t.Errorf("MongoDB.URI = %q", cfg.MongoDB.URI)
	}
	if cfg.MongoDB.Database != "overridden" {
		t.Errorf("MongoDB.Database = %q", cfg.MongoDB.Database)
	}
	if cfg.Logging.Level != "debug" {
		t.Errorf("Logging.Level = %q", cfg.Logging.Level)
	}
}

func TestLoadRejectsUnknownYAMLFields(t *testing.T) {
	t.Setenv("TELEGRAM_BOT_TOKEN", "test-token")

	_, err := Load(writeConfig(t, "telegram:\n  typo: true\n"))
	if err == nil || !strings.Contains(err.Error(), "field typo not found") {
		t.Fatalf("Load() error = %v, want unknown-field error", err)
	}
}

func TestLoadValidatesWorkerCount(t *testing.T) {
	t.Setenv("TELEGRAM_BOT_TOKEN", "test-token")

	_, err := Load(writeConfig(t, "telegram:\n  workers: 0\n"))
	if err == nil || !strings.Contains(err.Error(), "workers must be between") {
		t.Fatalf("Load() error = %v, want worker validation error", err)
	}
}

func TestLoadRejectsUnknownAllowedUpdate(t *testing.T) {
	t.Setenv("TELEGRAM_BOT_TOKEN", "test-token")

	_, err := Load(writeConfig(t, "telegram:\n  allowed_updates:\n    - message\n    - callback_queries\n"))
	if err == nil || !strings.Contains(err.Error(), `allowed_updates[1] "callback_queries" is not supported`) {
		t.Fatalf("Load() error = %v, want unsupported-update error", err)
	}
}

func TestLoadRejectsDuplicateAllowedUpdate(t *testing.T) {
	t.Setenv("TELEGRAM_BOT_TOKEN", "test-token")

	_, err := Load(writeConfig(t, "telegram:\n  allowed_updates:\n    - message\n    - message\n"))
	if err == nil || !strings.Contains(err.Error(), `allowed_updates contains duplicate "message"`) {
		t.Fatalf("Load() error = %v, want duplicate-update error", err)
	}
}

func TestLoadRequiresBotToken(t *testing.T) {
	t.Setenv("TELEGRAM_BOT_TOKEN", "")

	_, err := Load(writeConfig(t, `{}`))
	if err == nil || !strings.Contains(err.Error(), "TELEGRAM_BOT_TOKEN is required") {
		t.Fatalf("Load() error = %v, want missing-token error", err)
	}
}

func writeConfig(t *testing.T, contents string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "config.yaml")
	if err := os.WriteFile(path, []byte(contents), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}
	return path
}
