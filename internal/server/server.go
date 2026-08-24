// Package server wires the ntx-server process together: it opens the
// SQLite database, runs migrations, and serves RPC requests over a
// transport.Listener by dispatching them through a router.
package server

import (
	"context"
	"database/sql"
	"fmt"
	"ntx/internal/db"
	"ntx/internal/db/migrations"
	"ntx/internal/notification"
	"ntx/internal/shared/config"
	"ntx/internal/shared/dto"
	"ntx/internal/shared/frame"
	"ntx/internal/shared/log"
	"ntx/internal/shared/rpc"
	"ntx/internal/shared/transport"
	"ntx/internal/shared/transport/unixsock"
	"os"
	"path/filepath"
)

// Config holds ntx-server's runtime configuration.
type Config struct {
	SocketPath string
	DBPath     string
	LogLevel   log.Level
}

// LoadConfig reads configuration from the environment, applying defaults
// for unset values.
func LoadConfig() Config {
	cfg := Config{
		SocketPath: os.Getenv(config.SocketPathEnvName),
		DBPath:     os.Getenv(config.DBPathEnvName),
		LogLevel:   log.Level(os.Getenv(config.ServerLogLevelEnvName)),
	}

	if cfg.SocketPath == "" {
		cfg.SocketPath = config.DefaultSocketPath
	}

	if cfg.DBPath == "" {
		home, err := os.UserHomeDir()
		if err == nil {
			cfg.DBPath = filepath.Join(home, config.DefaultDBPath)
		}
	}

	return cfg
}

// Run opens the database, applies pending migrations, binds the Unix
// socket listener, and serves requests until ctx is canceled.
func Run(ctx context.Context, cfg Config) error {
	logger := log.NewStdout(log.StdoutParams{Level: cfg.LogLevel, Service: "ntx-server"})

	logger.Debug("Config initialized.", cfg)

	if err := os.MkdirAll(filepath.Dir(cfg.DBPath), 0o700); err != nil {
		return fmt.Errorf("server: create db directory: %w", err)
	}

	sqlDB, err := db.Open(cfg.DBPath)
	if err != nil {
		return fmt.Errorf("server: open db: %w", err)
	}
	defer sqlDB.Close() //nolint:errcheck // best-effort cleanup on shutdown

	migrator, err := db.New(sqlDB, migrations.All)
	if err != nil {
		return fmt.Errorf("server: init migrator: %w", err)
	}

	if err := migrator.UpAllMigrations(); err != nil {
		return fmt.Errorf("server: run migrations: %w", err)
	}

	ln, err := unixsock.Listen(cfg.SocketPath)
	if err != nil {
		return fmt.Errorf("server: listen: %w", err)
	}
	defer func() {
		_ = ln.Close()
		_ = os.Remove(cfg.SocketPath)
	}()

	go func() {
		<-ctx.Done()
		_ = ln.Close()
	}()

	return serve(ctx, ln, newRouter(sqlDB), logger)
}

// newRouter builds the rpc.Router with all RPC methods ntx-server handles.
func newRouter(sqlDB *sql.DB) *rpc.Router {
	rt := rpc.NewRouter()

	rt.Register(
		dto.CreateNotificationMethod,
		notification.NewCreateNotificationHandler(sqlDB).Handle,
	)
	rt.Register(dto.GetNotificationMethod, notification.NewGetNotificationHandler(sqlDB).Handle)

	return rt
}

func serve(ctx context.Context, ln transport.Listener, rt *rpc.Router, logger *log.Logger) error {
	for {
		conn, err := ln.Accept()
		if err != nil {
			if ctx.Err() != nil {
				return nil
			}

			return fmt.Errorf("server: accept: %w", err)
		}

		go handleConn(ctx, conn, rt, logger)
	}
}

func handleConn(ctx context.Context, conn transport.Conn, rt *rpc.Router, logger *log.Logger) {
	defer conn.Close() //nolint:errcheck // best-effort cleanup on disconnect

	for {
		rawReq, err := frame.Read(conn)
		if err != nil {
			return
		}

		rawResp, err := rt.Handle(ctx, rawReq)
		if err != nil {
			logger.ErrorContext(ctx, "Handle request error.", log.Payload{"error": err})

			return
		}

		if err := frame.Write(conn, rawResp); err != nil {
			logger.ErrorContext(ctx, "Write response error.", log.Payload{"error": err})

			return
		}
	}
}
