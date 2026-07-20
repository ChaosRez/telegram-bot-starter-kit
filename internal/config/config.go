// Package config loads and validates application configuration.
package config

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/go-telegram/bot/models"
	"gopkg.in/yaml.v3"
)

const defaultMongoURI = "mongodb://localhost:27017"

var allowedTelegramUpdates = map[string]struct{}{
	models.AllowedUpdateMessage:                 {},
	models.AllowedUpdateEditedMessage:           {},
	models.AllowedUpdateChannelPost:             {},
	models.AllowedUpdateEditedChannelPost:       {},
	models.AllowedUpdateBusinessConnection:      {},
	models.AllowedUpdateBusinessMessage:         {},
	models.AllowedUpdateEditedBusinessMessage:   {},
	models.AllowedUpdateDeletedBusinessMessages: {},
	models.AllowedUpdateMessageReaction:         {},
	models.AllowedUpdateMessageReactionCount:    {},
	models.AllowedUpdateInlineQuery:             {},
	models.AllowedUpdateChosenInlineResult:      {},
	models.AllowedUpdateCallbackQuery:           {},
	models.AllowedUpdateShippingQuery:           {},
	models.AllowedUpdatePreCheckoutQuery:        {},
	models.AllowedUpdatePurchasedPaidMedia:      {},
	models.AllowedUpdatePoll:                    {},
	models.AllowedUpdatePollAnswer:              {},
	models.AllowedUpdateMyChatMember:            {},
	models.AllowedUpdateChatMember:              {},
	models.AllowedUpdateChatJoinRequest:         {},
	models.AllowedUpdateChatBoost:               {},
	models.AllowedUpdateRemovedChatBoost:        {},
	models.AllowedUpdateManagedBot:              {},
	models.AllowedUpdateGuestMessage:            {},
}

// Duration is a YAML-friendly time.Duration.
type Duration struct {
	time.Duration
}

// UnmarshalYAML parses durations such as "5s" or "1m".
func (d *Duration) UnmarshalYAML(node *yaml.Node) error {
	value, err := time.ParseDuration(node.Value)
	if err != nil {
		return fmt.Errorf("parse duration %q: %w", node.Value, err)
	}
	d.Duration = value
	return nil
}

// Config contains all runtime configuration.
type Config struct {
	Telegram TelegramConfig `yaml:"telegram"`
	MongoDB  MongoDBConfig  `yaml:"mongodb"`
	Logging  LoggingConfig  `yaml:"logging"`
}

// TelegramConfig configures long polling and Telegram API behavior.
type TelegramConfig struct {
	Token          string   `yaml:"-"`
	PollTimeout    Duration `yaml:"poll_timeout"`
	Workers        int      `yaml:"workers"`
	AllowedUpdates []string `yaml:"allowed_updates"`
	SyncCommands   bool     `yaml:"sync_commands"`
}

// MongoDBConfig configures the MongoDB client and collections.
type MongoDBConfig struct {
	URI              string   `yaml:"-"`
	Database         string   `yaml:"database"`
	UsersCollection  string   `yaml:"users_collection"`
	ConnectTimeout   Duration `yaml:"connect_timeout"`
	OperationTimeout Duration `yaml:"operation_timeout"`
}

// LoggingConfig configures the standard library structured logger.
type LoggingConfig struct {
	Level  string `yaml:"level"`
	Format string `yaml:"format"`
}

// Load reads non-secret settings from YAML and deployment-specific values from
// the environment. Unknown YAML fields are rejected to catch configuration
// typos at startup.
func Load(path string) (Config, error) {
	cfg := defaults()

	file, err := os.Open(path)
	if err != nil {
		return Config{}, fmt.Errorf("open config file: %w", err)
	}
	defer file.Close()

	decoder := yaml.NewDecoder(file)
	decoder.KnownFields(true)
	if err := decoder.Decode(&cfg); err != nil {
		return Config{}, fmt.Errorf("decode config file: %w", err)
	}

	if err := applyEnvironment(&cfg, os.LookupEnv); err != nil {
		return Config{}, err
	}
	if err := cfg.Validate(); err != nil {
		return Config{}, fmt.Errorf("validate config: %w", err)
	}

	return cfg, nil
}

func defaults() Config {
	return Config{
		Telegram: TelegramConfig{
			PollTimeout:    Duration{Duration: 30 * time.Second},
			Workers:        1,
			AllowedUpdates: []string{"message", "callback_query"},
			SyncCommands:   true,
		},
		MongoDB: MongoDBConfig{
			URI:              defaultMongoURI,
			Database:         "telegram_bot",
			UsersCollection:  "users",
			ConnectTimeout:   Duration{Duration: 10 * time.Second},
			OperationTimeout: Duration{Duration: 5 * time.Second},
		},
		Logging: LoggingConfig{
			Level:  "info",
			Format: "json",
		},
	}
}

type environmentLookup func(string) (string, bool)

func applyEnvironment(cfg *Config, lookup environmentLookup) error {
	if value, ok := nonEmptyEnvironment(lookup, "TELEGRAM_BOT_TOKEN"); ok {
		cfg.Telegram.Token = value
	}
	if value, ok := nonEmptyEnvironment(lookup, "MONGODB_URI"); ok {
		cfg.MongoDB.URI = value
	}
	if value, ok := nonEmptyEnvironment(lookup, "MONGODB_DATABASE"); ok {
		cfg.MongoDB.Database = value
	}
	if value, ok := nonEmptyEnvironment(lookup, "LOG_LEVEL"); ok {
		cfg.Logging.Level = strings.ToLower(value)
	}
	if value, ok := nonEmptyEnvironment(lookup, "TELEGRAM_WORKERS"); ok {
		workers, err := strconv.Atoi(value)
		if err != nil {
			return fmt.Errorf("parse TELEGRAM_WORKERS: %w", err)
		}
		cfg.Telegram.Workers = workers
	}
	return nil
}

func nonEmptyEnvironment(lookup environmentLookup, name string) (string, bool) {
	value, ok := lookup(name)
	value = strings.TrimSpace(value)
	return value, ok && value != ""
}

// Validate checks configuration before external connections are opened.
func (cfg Config) Validate() error {
	var problems []error

	if strings.TrimSpace(cfg.Telegram.Token) == "" {
		problems = append(problems, errors.New("TELEGRAM_BOT_TOKEN is required"))
	}
	if cfg.Telegram.PollTimeout.Duration < 2*time.Second || cfg.Telegram.PollTimeout.Duration > 60*time.Second {
		problems = append(problems, errors.New("telegram.poll_timeout must be between 2s and 60s"))
	}
	if cfg.Telegram.Workers < 1 || cfg.Telegram.Workers > 64 {
		problems = append(problems, errors.New("telegram.workers must be between 1 and 64"))
	}
	if len(cfg.Telegram.AllowedUpdates) == 0 {
		problems = append(problems, errors.New("telegram.allowed_updates cannot be empty"))
	}
	seenUpdates := make(map[string]struct{}, len(cfg.Telegram.AllowedUpdates))
	for index, updateType := range cfg.Telegram.AllowedUpdates {
		if _, ok := allowedTelegramUpdates[updateType]; !ok {
			problems = append(problems, fmt.Errorf(
				"telegram.allowed_updates[%d] %q is not supported",
				index,
				updateType,
			))
			continue
		}
		if _, duplicate := seenUpdates[updateType]; duplicate {
			problems = append(problems, fmt.Errorf(
				"telegram.allowed_updates contains duplicate %q",
				updateType,
			))
			continue
		}
		seenUpdates[updateType] = struct{}{}
	}
	if strings.TrimSpace(cfg.MongoDB.URI) == "" {
		problems = append(problems, errors.New("MONGODB_URI cannot be empty"))
	}
	if strings.TrimSpace(cfg.MongoDB.Database) == "" {
		problems = append(problems, errors.New("mongodb.database cannot be empty"))
	}
	if strings.TrimSpace(cfg.MongoDB.UsersCollection) == "" {
		problems = append(problems, errors.New("mongodb.users_collection cannot be empty"))
	}
	if cfg.MongoDB.ConnectTimeout.Duration <= 0 {
		problems = append(problems, errors.New("mongodb.connect_timeout must be positive"))
	}
	if cfg.MongoDB.OperationTimeout.Duration <= 0 {
		problems = append(problems, errors.New("mongodb.operation_timeout must be positive"))
	}
	if !oneOf(strings.ToLower(cfg.Logging.Level), "debug", "info", "warn", "error") {
		problems = append(problems, errors.New("logging.level must be debug, info, warn, or error"))
	}
	if !oneOf(strings.ToLower(cfg.Logging.Format), "json", "text") {
		problems = append(problems, errors.New("logging.format must be json or text"))
	}

	return errors.Join(problems...)
}

func oneOf(value string, choices ...string) bool {
	for _, choice := range choices {
		if value == choice {
			return true
		}
	}
	return false
}
