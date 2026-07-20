package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/rezamalek/telegram-bot-starter-kit/internal/app"
	"github.com/rezamalek/telegram-bot-starter-kit/internal/config"
	"github.com/rezamalek/telegram-bot-starter-kit/internal/logging"
)

func main() {
	os.Exit(run())
}

func run() int {
	configPath := flag.String("config", defaultConfigPath(), "path to YAML configuration")
	flag.Parse()

	cfg, err := config.Load(*configPath)
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "configuration error: %v\n", err)
		return 1
	}

	logger := logging.New(cfg.Logging.Level, cfg.Logging.Format)
	slog.SetDefault(logger)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if err := app.Run(ctx, cfg, logger); err != nil {
		logger.Error("bot stopped with an error", "error", err)
		return 1
	}
	return 0
}

func defaultConfigPath() string {
	if path := os.Getenv("CONFIG_FILE"); path != "" {
		return path
	}
	return "config/config.yaml"
}
