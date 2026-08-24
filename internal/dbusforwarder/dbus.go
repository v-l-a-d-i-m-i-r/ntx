// Package dbusforwarder watches desktop notifications on the D-Bus session
// bus, without registering as (and thus without conflicting with) the
// org.freedesktop.Notifications service owned by the system's notification
// daemon (dunst, mako, etc.), and forwards them to ntx-server over its RPC
// socket.
package dbusforwarder

import (
	"context"
	"fmt"
	"ntx/internal/shared/log"
	"time"

	"github.com/godbus/dbus/v5"
)

const (
	notificationsInterface = "org.freedesktop.Notifications"
	notificationsPath      = "/org/freedesktop/Notifications"
	notifyMember           = "Notify"

	// pixmapSignature is the D-Bus struct signature used by notification
	// hints that carry a raw icon/image (width, height, rowstride,
	// has-alpha, bits-per-sample, channels, pixel bytes).
	pixmapSignature = "(iiibiiay)"
)

// Notification is a decoded org.freedesktop.Notifications.Notify call.
type Notification struct {
	Timestamp     time.Time      `json:"timestamp"`
	AppName       string         `json:"app_name"`
	ReplacesID    uint32         `json:"replaces_id"`
	AppIcon       string         `json:"app_icon"`
	Summary       string         `json:"summary"`
	Body          string         `json:"body"`
	Actions       []string       `json:"actions"`
	Hints         map[string]any `json:"hints"`
	ExpireTimeout int32          `json:"expire_timeout"`
}

// Watch connects to the D-Bus session bus and monitors calls to
// org.freedesktop.Notifications.Notify, without owning that service name
// itself. It returns a channel of decoded notifications and a channel that
// receives at most one error before both channels are closed. Both channels
// are closed when ctx is canceled or the connection is lost.
func Watch(ctx context.Context, logger *log.Logger) (<-chan Notification, <-chan error, error) {
	conn, err := dbus.ConnectSessionBus()
	if err != nil {
		return nil, nil, fmt.Errorf("connect session bus: %w", err)
	}

	rules := []string{
		fmt.Sprintf("type='method_call',member='%s',path='%s',interface='%s'",
			notifyMember, notificationsPath, notificationsInterface),
	}

	if call := conn.BusObject().
		Call("org.freedesktop.DBus.Monitoring.BecomeMonitor", 0, rules, uint(0)); call.Err != nil {
		_ = conn.Close()

		return nil, nil, fmt.Errorf("become monitor: %w", call.Err)
	}

	msgs := make(chan *dbus.Message, 16)
	conn.Eavesdrop(msgs)

	notifications := make(chan Notification)
	errs := make(chan error, 1)

	go watchLoop(ctx, conn, msgs, notifications, errs, logger)

	return notifications, errs, nil
}

func watchLoop(
	ctx context.Context,
	conn *dbus.Conn,
	msgs chan *dbus.Message,
	notifications chan<- Notification,
	errs chan<- error,
	logger *log.Logger,
) {
	defer close(notifications)
	defer close(errs)
	defer func() { _ = conn.Close() }()

	for {
		select {
		case <-ctx.Done():
			return
		case msg, ok := <-msgs:
			if !ok {
				err := fmt.Errorf("dbus connection closed")

				logger.ErrorContext(ctx, "Dbus connection closed.", log.Payload{"error": err})

				errs <- err

				return
			}

			n, err := decodeNotify(msg.Body)
			if err != nil {
				logger.WarnContext(
					ctx,
					"Skipping malformed notification.",
					log.Payload{"error": err, "notification": msg.Body},
				)

				continue
			}

			select {
			case notifications <- n:
			case <-ctx.Done():
				return
			}
		}
	}
}

func decodeNotify(body []any) (Notification, error) {
	if len(body) != 8 {
		return Notification{}, fmt.Errorf("unexpected arg count %d, want 8", len(body))
	}

	appName, ok := body[0].(string)
	if !ok {
		return Notification{}, fmt.Errorf("arg 0 (app_name): not a string")
	}

	replacesID, ok := body[1].(uint32)
	if !ok {
		return Notification{}, fmt.Errorf("arg 1 (replaces_id): not a uint32")
	}

	appIcon, ok := body[2].(string)
	if !ok {
		return Notification{}, fmt.Errorf("arg 2 (app_icon): not a string")
	}

	summary, ok := body[3].(string)
	if !ok {
		return Notification{}, fmt.Errorf("arg 3 (summary): not a string")
	}

	notifyBody, ok := body[4].(string)
	if !ok {
		return Notification{}, fmt.Errorf("arg 4 (body): not a string")
	}

	actions, ok := body[5].([]string)
	if !ok {
		return Notification{}, fmt.Errorf("arg 5 (actions): not a []string")
	}

	rawHints, ok := body[6].(map[string]dbus.Variant)
	if !ok {
		return Notification{}, fmt.Errorf("arg 6 (hints): not a map[string]dbus.Variant")
	}

	expireTimeout, ok := body[7].(int32)
	if !ok {
		return Notification{}, fmt.Errorf("arg 7 (expire_timeout): not an int32")
	}

	return Notification{
		Timestamp:     time.Now(),
		AppName:       appName,
		ReplacesID:    replacesID,
		AppIcon:       appIcon,
		Summary:       summary,
		Body:          notifyBody,
		Actions:       actions,
		Hints:         decodeHints(rawHints),
		ExpireTimeout: expireTimeout,
	}, nil
}

func decodeHints(rawHints map[string]dbus.Variant) map[string]any {
	hints := make(map[string]any, len(rawHints))

	for key, variant := range rawHints {
		if variant.Signature().String() == pixmapSignature {
			hints[key] = decodePixmap(variant)

			continue
		}

		hints[key] = variant.Value()
	}

	return hints
}

func decodePixmap(variant dbus.Variant) any {
	fields, ok := variant.Value().([]any)
	if !ok || len(fields) != 7 {
		return variant.Value()
	}

	width, ok := fields[0].(int32)
	if !ok {
		return variant.Value()
	}

	height, ok := fields[1].(int32)
	if !ok {
		return variant.Value()
	}

	rowstride, ok := fields[2].(int32)
	if !ok {
		return variant.Value()
	}

	hasAlpha, ok := fields[3].(bool)
	if !ok {
		return variant.Value()
	}

	bitsPerSample, ok := fields[4].(int32)
	if !ok {
		return variant.Value()
	}

	channels, ok := fields[5].(int32)
	if !ok {
		return variant.Value()
	}

	pixels, ok := fields[6].([]byte)
	if !ok {
		return variant.Value()
	}

	return RawPixmap{
		Width:         width,
		Height:        height,
		Rowstride:     rowstride,
		HasAlpha:      hasAlpha,
		BitsPerSample: bitsPerSample,
		Channels:      channels,
		Pixels:        pixels,
	}
}
