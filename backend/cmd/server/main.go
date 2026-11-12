package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"

	"github.com/plantdoc/backend/internal/ai"
	"github.com/plantdoc/backend/internal/config"
	"github.com/plantdoc/backend/internal/handler"
	"github.com/plantdoc/backend/internal/repository"
	"github.com/plantdoc/backend/internal/service"
	"github.com/plantdoc/backend/internal/storage"
)

func main() {
	_ = godotenv.Load() // .env is optional

	cfg, err := config.Load()
	if err != nil {
		slog.Error("config", "err", err)
		os.Exit(1)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	db, err := pgxpool.New(ctx, cfg.DatabaseURL)
	if err != nil {
		slog.Error("db connect", "err", err)
		os.Exit(1)
	}
	defer db.Close()
	if err := db.Ping(ctx); err != nil {
		slog.Error("db ping", "err", err)
		os.Exit(1)
	}

	uploader, err := storage.New(cfg)
	if err != nil {
		slog.Error("storage", "err", err)
		os.Exit(1)
	}

	svc := service.NewDiagnosisService(
		ai.NewGemini(cfg.GeminiAPIKey, append([]string{cfg.GeminiModel}, cfg.GeminiFallbackModels...)),
		repository.NewConsultationRepo(db),
		uploader,
	)
	h := handler.New(svc, cfg.MaxUploadMB)

	srv := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           handler.Logger(handler.CORS(cfg.AllowedOrigins, h.Routes())),
		ReadHeaderTimeout: 10 * time.Second,
	}

	go func() {
		slog.Info("server listening", "port", cfg.Port, "storage", cfg.StorageProvider, "model", cfg.GeminiModel)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			slog.Error("server", "err", err)
			stop()
		}
	}()

	<-ctx.Done()
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_ = srv.Shutdown(shutdownCtx)
	slog.Info("server stopped")
}
