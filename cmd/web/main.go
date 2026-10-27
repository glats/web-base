package main

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/joho/godotenv"
	"github.com/rs/zerolog/log"

	"app.ddd/internal/adapter/config"
	"app.ddd/internal/adapter/sqlite"
	"app.ddd/internal/common/logger"
	"app.ddd/internal/port/web"
	"app.ddd/internal/service"
)

const shutdownTimeout = 30 * time.Second

func main() {
	if err := godotenv.Load(".env"); err != nil && !errors.Is(err, os.ErrNotExist) {
		log.Fatal().Err(err).Msg("load .env")
	}

	cfg := config.Get()
	if err := logger.Configure(cfg.Log.Level, cfg.Log.Pretty, os.Stderr); err != nil {
		log.Fatal().Err(err).Str("level", cfg.Log.Level).Msg("parse LOG_LEVEL")
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)

	err := run(ctx, cfg, func(addr net.Addr) { log.Info().Str("addr", addr.String()).Msg("listening") })
	stop()
	if err != nil {
		log.Fatal().Err(err).Msg("web server stopped")
	}
}

func run(ctx context.Context, cfg config.Config, ready func(net.Addr)) error {
	if err := sqlite.Migrate(cfg.Database.Path); err != nil {
		return err
	}

	db, err := sqlite.Open(cfg.Database.Path)
	if err != nil {
		return err
	}
	defer func() { _ = db.Close() }()

	if cfg.Seed.Enabled {
		if err := sqlite.Seed(ctx, db); err != nil {
			return err
		}
	}

	application := service.NewApp(sqlite.NewUserRepository(db))

	listener, err := net.Listen("tcp", cfg.HTTP.Addr)
	if err != nil {
		return fmt.Errorf("listen on %q: %w", cfg.HTTP.Addr, err)
	}

	server := &http.Server{
		Handler:           web.NewHandler(application),
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       2 * time.Minute,
	}

	served := make(chan error, 1)
	go func() { served <- server.Serve(listener) }()
	if ready != nil {
		ready(listener.Addr())
	}

	select {
	case err := <-served:
		return fmt.Errorf("serve: %w", err)
	case <-ctx.Done():
	}

	shutdownCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), shutdownTimeout)
	defer cancel()
	if err := server.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("shutdown: %w", err)
	}
	if err := <-served; !errors.Is(err, http.ErrServerClosed) {
		return fmt.Errorf("serve: %w", err)
	}

	return nil
}
