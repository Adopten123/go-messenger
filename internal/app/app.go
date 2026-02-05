package app

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/Adopten123/go-messenger/internal/config"
	"github.com/Adopten123/go-messenger/internal/handler"
	"github.com/Adopten123/go-messenger/internal/repo/pgdb"
	"github.com/Adopten123/go-messenger/internal/service"
	"github.com/Adopten123/go-messenger/internal/ws"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/redis/go-redis/v9"
)

type App struct {
	log        *slog.Logger
	cfg        *config.Config
	httpServer *http.Server
	pool       *pgxpool.Pool
	redis      *redis.Client
}

func New(log *slog.Logger, cfg *config.Config) *App {
	// 1. Init Database
	pool, err := pgxpool.New(context.Background(), cfg.Database.DSN)
	if err != nil {
		panic(fmt.Errorf("failed to connect to database: %w", err))
	}
	if err := pool.Ping(context.Background()); err != nil {
		panic(fmt.Errorf("failed to ping database: %w", err))
	}
	log.Info("connected to database")

	// 2. Init Redis
	rdb := redis.NewClient(&redis.Options{
		Addr: cfg.Redis.Address,
	})
	if err := rdb.Ping(context.Background()).Err(); err != nil {
		panic(fmt.Errorf("failed to connect to redis: %w", err))
	}
	log.Info("connected to redis")

	// 3. Init MinIO
	minioClient, err := minio.New(cfg.MinIO.Endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(cfg.MinIO.AccessKeyID, cfg.MinIO.SecretAccessKey, ""),
		Secure: cfg.MinIO.UseSSL,
	})
	if err != nil {
		panic(fmt.Errorf("failed to connect to minio: %w", err))
	}
	log.Info("connected to minio")

	// 4. Init Layers
	repo := pgdb.New(pool)
	fileService := service.NewFileService(minioClient, cfg.MinIO.Bucket, cfg.MinIO.Endpoint)

	userService := service.NewUserService(repo, cfg.TokenSecret)
	userHandler := handler.NewUserHandler(userService, cfg.TokenSecret, rdb, fileService)

	chatService := service.NewChatService(repo, pool)
	chatHandler := handler.NewChatHandler(chatService, userService)

	hub := ws.NewHub(repo, rdb)
	go hub.Run()

	wsHandler := ws.NewWSHandler(hub)

	// 5. Router
	router := chi.NewRouter()

	router.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"http://localhost:3000", "http://localhost:8082"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token"},
		ExposedHeaders:   []string{"Link"},
		AllowCredentials: true,
		MaxAge:           300,
	}))

	router.Use(middleware.RequestID)
	router.Use(middleware.Logger)
	router.Use(middleware.Recoverer)

	router.Route("/api", func(r chi.Router) {
		r.Post("/users/register", userHandler.Register)
		r.Post("/users/login", userHandler.Login)

		r.Group(func(r chi.Router) {
			r.Use(userHandler.AuthMiddleware)

			r.Get("/users/me", userHandler.GetMe)
			r.Post("/users/me/avatar", userHandler.UploadAvatar)
			r.Get("/users/{user_id}/status", userHandler.GetOnlineStatus)

			r.Post("/chats", chatHandler.CreateChat)
			r.Get("/chats/{chat_id}/messages", chatHandler.GetMessages)

			r.Get("/ws", wsHandler.HandleWS)
		})
	})

	workDir, _ := os.Getwd()
	filesDir := http.Dir(filepath.Join(workDir, "static"))
	FileServer(router, "/", filesDir)

	return &App{
		log:   log,
		cfg:   cfg,
		pool:  pool,
		redis: rdb,
		httpServer: &http.Server{
			Addr:    cfg.HTTPServer.Address,
			Handler: router,
		},
	}
}

func (a *App) Run() {
	const op = "app.Run"

	go func() {
		a.log.Info("server starting", slog.String("address", a.cfg.HTTPServer.Address))
		if err := a.httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			a.log.Error("failed to start server", slog.String("op", op), slog.String("error", err.Error()))
			os.Exit(1)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	sign := <-stop
	a.log.Info("stopping application", slog.String("signal", sign.String()))

	a.Stop()
}

func (a *App) Stop() {
	const op = "app.Stop"
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// 1. Stop HTTP-server
	if err := a.httpServer.Shutdown(ctx); err != nil {
		a.log.Error("failed to stop server gracefully", slog.String("op", op), slog.String("error", err.Error()))
	}

	// 2. Close DB
	a.log.Info("closing database connection")
	a.pool.Close()

	// 3. Close Redis
	a.log.Info("closing redis connection")
	if err := a.redis.Close(); err != nil {
		a.log.Error("failed to close redis", slog.String("op", op), slog.String("error", err.Error()))
	}

	a.log.Info("application stopped")
}

func FileServer(r chi.Router, path string, root http.FileSystem) {
	if strings.ContainsAny(path, "{}*") {
		panic("FileServer does not permit any URL parameters.")
	}

	if path != "/" && path[len(path)-1] != '/' {
		r.Get(path, http.RedirectHandler(path+"/", 301).ServeHTTP)
		path += "/"
	}
	path += "*"

	r.Get(path, func(w http.ResponseWriter, r *http.Request) {
		rctx := chi.RouteContext(r.Context())
		pathPrefix := strings.TrimSuffix(rctx.RoutePattern(), "/*")
		fs := http.StripPrefix(pathPrefix, http.FileServer(root))
		fs.ServeHTTP(w, r)
	})
}
