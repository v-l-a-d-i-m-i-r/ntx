package dbusforwarder

import (
	"bytes"
	"encoding/base64"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"testing"
)

func makeRawPixmap(w, h int) RawPixmap {
	pixels := make([]byte, w*h*4)
	for i := range pixels {
		pixels[i] = 0xAB
	}

	return RawPixmap{
		Width:         int32(w),
		Height:        int32(h),
		Rowstride:     int32(w * 4),
		HasAlpha:      true,
		BitsPerSample: 8,
		Channels:      4,
		Pixels:        pixels,
	}
}

func TestResolvePrefersImageDataHintOverAppIcon(t *testing.T) {
	t.Parallel()

	hints := map[string]any{
		"image-data": makeRawPixmap(2, 2),
		"other":      "keep-me",
	}

	got, cleaned := Resolve("some-app-icon-name", hints)

	if got == nil {
		t.Fatal("Resolve: icon = nil, want resolved icon from image-data hint")
	}

	if got.Mime != "image/png" {
		t.Errorf("Mime = %q, want image/png", got.Mime)
	}

	decoded, err := base64.StdEncoding.DecodeString(got.Base64)
	if err != nil {
		t.Fatalf("decode base64: %v", err)
	}

	if _, err := png.Decode(bytes.NewReader(decoded)); err != nil {
		t.Errorf("resolved icon is not valid PNG: %v", err)
	}

	if _, ok := cleaned["image-data"]; ok {
		t.Error("cleaned hints still contain image-data")
	}

	if cleaned["other"] != "keep-me" {
		t.Error("cleaned hints lost unrelated key")
	}
}

func TestResolveFallsBackToAppIconPath(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	iconPath := filepath.Join(dir, "myicon.png")
	writeTestPNG(t, iconPath, 3, 3)

	got, _ := Resolve(iconPath, map[string]any{})

	if got == nil {
		t.Fatal("Resolve: icon = nil, want resolved icon from app_icon path")
	}

	if got.Mime != "image/png" {
		t.Errorf("Mime = %q, want image/png", got.Mime)
	}
}

func TestResolveNoSourcesReturnsNil(t *testing.T) {
	t.Parallel()

	got, cleaned := Resolve("nonexistent-icon-name-xyz", map[string]any{})

	if got != nil {
		t.Errorf("Resolve: icon = %+v, want nil", got)
	}

	if len(cleaned) != 0 {
		t.Errorf("cleaned hints = %+v, want empty", cleaned)
	}
}

func TestResolveStripsAllImageKeysEvenOnFailure(t *testing.T) {
	t.Parallel()

	hints := map[string]any{
		"image-data": "not-a-pixmap",
		"image_path": "/nonexistent/path.png",
		"icon_data":  "also-not-a-pixmap",
	}

	_, cleaned := Resolve("", hints)

	for _, key := range []string{"image-data", "image_path", "icon_data"} {
		if _, ok := cleaned[key]; ok {
			t.Errorf("cleaned hints still contain %q", key)
		}
	}
}

func writeTestPNG(t *testing.T, path string, w, h int) {
	t.Helper()

	img := image.NewRGBA(image.Rect(0, 0, w, h))
	img.Set(0, 0, color.RGBA{R: 1, G: 2, B: 3, A: 255})

	f, err := os.Create(path) //nolint:gosec // test fixture path from t.TempDir
	if err != nil {
		t.Fatalf("create test png: %v", err)
	}
	defer f.Close() //nolint:errcheck // test fixture cleanup

	if err := png.Encode(f, img); err != nil {
		t.Fatalf("encode test png: %v", err)
	}
}
