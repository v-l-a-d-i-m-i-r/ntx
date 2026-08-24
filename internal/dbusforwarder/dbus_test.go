package dbusforwarder

import (
	"testing"

	"github.com/godbus/dbus/v5"
)

func TestDecodeNotifyOK(t *testing.T) {
	t.Parallel()

	body := []any{
		"Firefox",
		uint32(0),
		"firefox",
		"New message",
		"You have 3 unread emails",
		[]string{"default", "Open"},
		map[string]dbus.Variant{
			"urgency": dbus.MakeVariant(byte(1)),
		},
		int32(-1),
	}

	n, err := decodeNotify(body)
	if err != nil {
		t.Fatalf("decodeNotify: unexpected error: %v", err)
	}

	if n.AppName != "Firefox" {
		t.Errorf("AppName = %q, want %q", n.AppName, "Firefox")
	}

	if n.Summary != "New message" {
		t.Errorf("Summary = %q, want %q", n.Summary, "New message")
	}

	if n.Body != "You have 3 unread emails" {
		t.Errorf("Body = %q, want %q", n.Body, "You have 3 unread emails")
	}

	if n.ExpireTimeout != -1 {
		t.Errorf("ExpireTimeout = %d, want -1", n.ExpireTimeout)
	}

	if got, ok := n.Hints["urgency"].(byte); !ok || got != 1 {
		t.Errorf("Hints[urgency] = %v, want byte(1)", n.Hints["urgency"])
	}
}

func TestDecodeNotifyWrongArgCount(t *testing.T) {
	t.Parallel()

	_, err := decodeNotify([]any{"only-one-arg"})
	if err == nil {
		t.Fatal("decodeNotify: expected error for wrong arg count, got nil")
	}
}

func TestDecodeNotifyWrongArgType(t *testing.T) {
	t.Parallel()

	body := []any{
		123, // app_name should be a string
		uint32(0),
		"",
		"",
		"",
		[]string{},
		map[string]dbus.Variant{},
		int32(0),
	}

	_, err := decodeNotify(body)
	if err == nil {
		t.Fatal("decodeNotify: expected error for wrong arg type, got nil")
	}
}

func TestDecodeHintsPixmapImage(t *testing.T) {
	t.Parallel()

	pixels := make([]byte, 64*64*4)
	pixmapFields := []any{int32(64), int32(64), int32(256), true, int32(8), int32(4), pixels}

	rawHints := map[string]dbus.Variant{
		"image-data": dbus.MakeVariantWithSignature(
			pixmapFields,
			dbus.ParseSignatureMust(pixmapSignature),
		),
	}

	hints := decodeHints(rawHints)

	pixmap, ok := hints["image-data"].(RawPixmap)
	if !ok {
		t.Fatalf("hints[image-data] = %#v, want RawPixmap", hints["image-data"])
	}

	if pixmap.Width != 64 || pixmap.Height != 64 {
		t.Errorf("pixmap dimensions = %dx%d, want 64x64", pixmap.Width, pixmap.Height)
	}

	if len(pixmap.Pixels) != len(pixels) {
		t.Errorf("len(pixmap.Pixels) = %d, want %d", len(pixmap.Pixels), len(pixels))
	}
}

func TestDecodeHintsScalarPassthrough(t *testing.T) {
	t.Parallel()

	rawHints := map[string]dbus.Variant{
		"desktop-entry": dbus.MakeVariant("firefox"),
	}

	hints := decodeHints(rawHints)

	if hints["desktop-entry"] != "firefox" {
		t.Errorf("hints[desktop-entry] = %v, want %q", hints["desktop-entry"], "firefox")
	}
}
