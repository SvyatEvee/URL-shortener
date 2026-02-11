package main

import (
	"URLshortener/internal/config"
	"URLshortener/internal/http-server/handlers/redirect"
	deletee "URLshortener/internal/http-server/handlers/url/delete"
	"URLshortener/internal/http-server/handlers/url/deleteUserData"
	"URLshortener/internal/http-server/handlers/url/getUsersAliases"
	"URLshortener/internal/http-server/handlers/url/save"
	"URLshortener/internal/http-server/handlers/url/update"
	"URLshortener/internal/http-server/middleware/authorization"
	"URLshortener/internal/http-server/middleware/logger"
	jwtlib "URLshortener/internal/jwt"
	"URLshortener/internal/lib/logger/sl"
	"URLshortener/internal/storage/sql"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"log/slog"
	"net/http"
	"os"
	"time"
)

const (
	envLocal = "local"
	envDev   = "dev"
	envProd  = "prod"
)

func main() {

	cfg := config.MustLoad()

	log := setupLogger(cfg.Env)
	log.Info(
		"starting url-shortener",
		slog.String("env", cfg.Env),
	)
	log.Debug("debug messages are enabled")

	storage, err := sql.New(cfg.DBDriver, cfg.ConnString)
	if err != nil {
		log.Error("failed to init storage", sl.Err(err))
		os.Exit(1) // можно return но так непонятно что была ошибка
	}

	router := chi.NewRouter()

	router.Use(middleware.RequestID)
	router.Use(logger.New(log))
	router.Use(middleware.Recoverer)
	router.Use(middleware.URLFormat)

	tokenValidator := jwtlib.New(time.Hour, time.Hour, cfg.Secret)

	router.Use(authorization.New(log, tokenValidator))

	router.Route("/", func(r chi.Router) {
		r.Post("/", save.New(log, storage))
		r.Patch("/", update.New(log, storage))
		r.Delete("/", deletee.New(log, storage))
		r.Delete("/admin", deleteUserData.New(log, storage))
		r.Get("/{alias}", redirect.New(log, storage))
		r.Get("/urls", getUsersAliases.New(log, storage))
	})

	log.Info("starting server :", slog.String("address", cfg.Address))

	srv := &http.Server{
		Addr:         cfg.Address,
		Handler:      router,
		ReadTimeout:  cfg.HTTPServer.Timeout,
		WriteTimeout: cfg.HTTPServer.Timeout,
		IdleTimeout:  cfg.HTTPServer.IdleTimeout,
	}

	if err := srv.ListenAndServe(); err != nil {
		log.Error("failed to start server")
	}

	log.Error("server stopped")
}

func setupLogger(env string) *slog.Logger {
	var log *slog.Logger
	switch env {
	case envLocal:

		log = slog.New(
			slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}),
		)
	case envDev:
		log = slog.New(
			slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}),
		)
	case envProd:
		log = slog.New(
			slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}),
		)
	}
	return log

}
