package main

import (
	"log/slog"
	"os"

	"github.com/Adopten123/go-messenger/internal/app"
	"github.com/Adopten123/go-messenger/internal/config"
)

func main() {
	// 1. Load Config
	os.Setenv("CONFIG_PATH", "./config/local.yaml") // TODO: remove hardcode in prod
	cfg := config.MustLoad()

	// 2. Init Logger
	log := setupLogger(cfg.Env)
	log.Info("initializing application", slog.String("env", cfg.Env))

	// 3. Init & Run
	application := app.New(log, cfg)

	application.Run()
}

func setupLogger(env string) *slog.Logger {
	var log *slog.Logger

	switch env {
	case "local":
		log = slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))
	case "prod":
		log = slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	}
	return log
}
