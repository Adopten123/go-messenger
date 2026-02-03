package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"

	"github.com/Adopten123/go-messenger/internal/handler"
	"github.com/Adopten123/go-messenger/internal/service"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Adopten123/go-messenger/internal/config"
	"github.com/Adopten123/go-messenger/internal/repo/pgdb"
)

func main() {
	// 1. Init config
	os.Setenv("CONFIG_PATH", "./config/local.yaml") // TODO: hardcode for dev

	cfg := config.MustLoad()

	// 2. Init logger
	log := setupLogger(cfg.Env)
	log.Info("starting application", slog.String("env", cfg.Env))

	// 3. Connection to DB
	pool, err := pgxpool.New(context.Background(), cfg.Database.DSN)
	if err != nil {
		log.Error("failed to connect to database", slog.String("error", err.Error()))
		os.Exit(1)
	}
	defer pool.Close()

	if err := pool.Ping(context.Background()); err != nil {
		log.Error("failed to ping database", slog.String("error", err.Error()))
		os.Exit(1)
	}
	log.Info("connected to database")

	// 4. Init layers
	repo := pgdb.New(pool)

	userService := service.NewUserService(repo, cfg.TokenSecret)
	userHandler := handler.NewUserHandler(userService, cfg.TokenSecret)

	// 5. Init router

	router := chi.NewRouter()
	router.Use(middleware.RequestID)
	router.Use(middleware.Logger)
	router.Use(middleware.Recoverer)

	router.Route("/api", func(r chi.Router) {
		r.Post("/users/register", userHandler.Register)
		r.Post("/users/login", userHandler.Login)

		r.Group(func(r chi.Router) {
			r.Use(userHandler.AuthMiddleware)
			r.Get("/users/me", userHandler.GetMe)

			// TODO: Add chats
		})
	})

	router.Get("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("Welcome to GO-Messenger!"))
	})

	// 6. Starting Server
	log.Info("server starting", slog.String("address", cfg.HTTPServer.Address))

	srv := &http.Server{
		Addr:         cfg.HTTPServer.Address,
		Handler:      router,
		ReadTimeout:  cfg.HTTPServer.Timeout,
		WriteTimeout: cfg.HTTPServer.Timeout,
		IdleTimeout:  cfg.HTTPServer.IdleTimeout,
	}

	if err := srv.ListenAndServe(); err != nil {
		log.Error("failed to start server", slog.String("error", err.Error()))
	}
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
