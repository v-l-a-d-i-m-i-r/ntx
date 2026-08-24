package dbusforwarder

import (
	"context"
	"fmt"
	"ntx/internal/shared/config"
	"ntx/internal/shared/dto"
	"ntx/internal/shared/log"
	"ntx/internal/shared/rpc"
	"os"
	"time"
)

// Config holds ntx-dbus-forwarder's runtime configuration.
type Config struct {
	SocketPath string
	LogLevel   log.Level
}

// LoadConfig reads configuration from the environment, applying defaults
// for unset values.
func LoadConfig() Config {
	cfg := Config{
		SocketPath: os.Getenv(config.SocketPathEnvName),
		LogLevel:   log.Level(os.Getenv(config.DbusForwarderLogLevelEnvName)),
	}

	if cfg.SocketPath == "" {
		cfg.SocketPath = config.DefaultSocketPath
	}

	return cfg
}

// Run watches D-Bus notifications and forwards each one to ntx-server until
// ctx is canceled or the D-Bus connection is lost.
func Run(ctx context.Context, cfg Config) error {
	logger := log.NewStdout(log.StdoutParams{Level: cfg.LogLevel, Service: "ntx-dbus-forwarder"})

	logger.Debug("Config initialized.", cfg)

	notifications, errs, err := Watch(ctx, logger)
	if err != nil {
		return fmt.Errorf("dbusforwarder: watch: %w", err)
	}

	client := rpc.NewClient(cfg.SocketPath)

	for n := range notifications {
		err := send(client, buildParams(n))
		if err != nil {
			logger.ErrorContext(ctx, "Send notification error.", log.Payload{
				"error":        err,
				"notification": n,
			})
		}
		if err == nil {
			logger.DebugContext(ctx, "Notification sent sucessfully.", log.Payload{
				"notification": n,
			})
		}
	}

	if err := <-errs; err != nil && ctx.Err() == nil {
		return err
	}

	return nil
}

// send sends a createNotification request built from params.
func send(client *rpc.Client, params dto.CreateNotificationParams) error {
	resp, err := client.Call(dto.CreateNotificationMethod, params)
	if err != nil {
		return err
	}

	if resp.Error != nil {
		return fmt.Errorf("server: %s", resp.Error.Message)
	}

	return nil
}

// buildParams resolves the notification's icon and builds the
// createNotification request params.
func buildParams(n Notification) dto.CreateNotificationParams {
	resolvedIcon, cleanedHints := Resolve(n.AppIcon, n.Hints)

	params := dto.CreateNotificationParams{
		Timestamp:     n.Timestamp.Format(time.RFC3339),
		AppName:       n.AppName,
		ReplacesID:    n.ReplacesID,
		Summary:       n.Summary,
		Body:          n.Body,
		Actions:       n.Actions,
		Hints:         cleanedHints,
		ExpireTimeout: n.ExpireTimeout,
	}

	if resolvedIcon != nil {
		params.IconBase64 = resolvedIcon.Base64
		params.IconMime = resolvedIcon.Mime
	}

	return params
}
