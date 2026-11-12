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

	"rest-api-scheduler-task/internal/api"
	"rest-api-scheduler-task/internal/auth"
	"rest-api-scheduler-task/internal/config"
	"rest-api-scheduler-task/internal/db"

	"github.com/joho/godotenv"
)

func main() {
	if err := godotenv.Load(); err != nil {
		slog.Warn("no . env file, using system env", "err", err)
	}
	cfg, err := config.Load()
	if err != nil {
		slog.Error("config error", "err", err)
		os.Exit(1)
	}

	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: config.ParseLevel(cfg.LogLevel),
	}))
	slog.SetDefault(logger)

	store, err := db.New(context.Background(), cfg.DB, logger)
	if err != nil {
		logger.Error("store init failed", "err", err)
		os.Exit(1)
	}
	defer store.Close()

	// check schema
	ctxInit, cancelInit := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancelInit()

	initialized, err := store.IsInitialized(ctxInit)
	if err != nil {
		logger.Error("check init failed", "err", err)
		os.Exit(1)
	}

	// create schema
	if !initialized {
		logger.Info("scheduler table not found, initializing schema")
		if err := store.Init(ctxInit); err != nil {
			logger.Info("init schema failed", "err", err)
			os.Exit(1)
		} else {
			logger.Info("schema alredy initialized")
		}
	}

	authMgr := auth.NewManager(cfg.JWTSecret, cfg.Password, 24*time.Hour, logger)

	handler := api.NewRouter(logger, cfg.StaticDir, store, authMgr)

	srv := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           handler,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       cfg.ReadTimeout,
		WriteTimeout:      cfg.WriteTimeout,
		IdleTimeout:       60 * time.Second,
	}

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		logger.Info("starting server",
			"addr", srv.Addr,
			"static", cfg.StaticDir,
			"log_level", cfg.LogLevel,
		)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Error("server failed", "err", err)
			os.Exit(1)
		}
	}()

	sig := <-quit

	logger.Info("shutdown signal received", "signal", sig.String())

	ctx, cancel := context.WithTimeout(context.Background(), cfg.ShutdownTimeout)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		logger.Error("graceful shutdown failed", "err", err)
		if cerr := srv.Close(); cerr != nil {
			logger.Error("forced close failed", "err", cerr)
		}
		os.Exit(1)
	}
	logger.Info("server stopped gracefully")
}
