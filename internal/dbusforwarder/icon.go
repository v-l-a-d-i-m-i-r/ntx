package dbusforwarder

import (
	"bytes"
	"encoding/base64"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

// hintKeys lists the notification hint keys that may carry image data, in
// resolution priority order (highest first). icon_data is the deprecated
// alias for image-data.
var hintKeys = []string{"image-data", "image_path", "icon_data"}

// Icon is a fully resolved, self-contained notification icon.
type Icon struct {
	Base64 string
	Mime   string
}

// RawPixmap is the decoded form of a D-Bus notification hint carrying a raw
// icon/image (the "(iiibiiay)" struct signature): width, height, rowstride,
// has-alpha, bits-per-sample, channels, and raw pixel bytes.
type RawPixmap struct {
	Width         int32
	Height        int32
	Rowstride     int32
	HasAlpha      bool
	BitsPerSample int32
	Channels      int32
	Pixels        []byte
}

// Resolve picks a single icon from hints (in priority order: image-data,
// image_path, icon_data) or, failing that, from appIcon. It returns the
// resolved icon (nil if none could be resolved) and hints with the
// image-bearing keys removed, since their data has been extracted into the
// returned Icon.
func Resolve(appIcon string, hints map[string]any) (*Icon, map[string]any) {
	cleaned := make(map[string]any, len(hints))

	for k, v := range hints {
		cleaned[k] = v
	}

	var resolved *Icon

	for _, key := range hintKeys {
		v, ok := cleaned[key]
		delete(cleaned, key)

		if ok && resolved == nil {
			resolved = resolveHintValue(v)
		}
	}

	if resolved == nil {
		resolved = resolveAppIcon(appIcon)
	}

	return resolved, cleaned
}

func resolveHintValue(v any) *Icon {
	switch val := v.(type) {
	case RawPixmap:
		return pixmapToIcon(val)
	case string:
		return fileToIcon(val)
	default:
		return nil
	}
}

func pixmapToIcon(p RawPixmap) *Icon {
	const supportedBitsPerSample = 8

	if p.BitsPerSample != supportedBitsPerSample {
		return nil
	}

	if p.Channels != 3 && p.Channels != 4 {
		return nil
	}

	if p.Width <= 0 || p.Height <= 0 {
		return nil
	}

	img := image.NewRGBA(image.Rect(0, 0, int(p.Width), int(p.Height)))

	for y := range int(p.Height) {
		rowStart := y * int(p.Rowstride)

		for x := range int(p.Width) {
			pixStart := rowStart + x*int(p.Channels)
			if pixStart+int(p.Channels) > len(p.Pixels) {
				continue
			}

			a := byte(255)
			if p.Channels == 4 {
				a = p.Pixels[pixStart+3]
			}

			img.SetRGBA(x, y, color.RGBA{
				R: p.Pixels[pixStart],
				G: p.Pixels[pixStart+1],
				B: p.Pixels[pixStart+2],
				A: a,
			})
		}
	}

	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		return nil
	}

	return &Icon{Base64: base64.StdEncoding.EncodeToString(buf.Bytes()), Mime: "image/png"}
}

func fileToIcon(path string) *Icon {
	path = strings.TrimPrefix(path, "file://")

	data, err := os.ReadFile(
		path,
	) //nolint:gosec // path originates from a locally captured D-Bus notification
	if err != nil {
		return nil
	}

	return &Icon{
		Base64: base64.StdEncoding.EncodeToString(data),
		Mime:   mimeForExt(filepath.Ext(path)),
	}
}

func mimeForExt(ext string) string {
	switch strings.ToLower(ext) {
	case ".svg":
		return "image/svg+xml"
	case ".xpm":
		return "image/x-xpixmap"
	default:
		return "image/png"
	}
}

func resolveAppIcon(appIcon string) *Icon {
	if appIcon == "" {
		return nil
	}

	if strings.HasPrefix(appIcon, "file://") || strings.Contains(appIcon, "/") {
		return fileToIcon(appIcon)
	}

	path := findIconByName(appIcon)
	if path == "" {
		return nil
	}

	return fileToIcon(path)
}

var hicolorSizeRe = regexp.MustCompile(`^(\d+)x\d+$`)

// findIconByName performs a simplified, non-spec-compliant lookup of an
// icon by freedesktop icon name: it globs a fixed set of standard
// directories rather than parsing index.theme files or resolving theme
// inheritance.
func findIconByName(name string) string {
	homeDir, _ := os.UserHomeDir()

	pixmapDirs := []string{"/usr/share/pixmaps"}
	hicolorDirs := []string{"/usr/share/icons/hicolor"}

	if homeDir != "" {
		pixmapDirs = append(pixmapDirs, filepath.Join(homeDir, ".local/share/pixmaps"))
		hicolorDirs = append(hicolorDirs, filepath.Join(homeDir, ".local/share/icons/hicolor"))
	}

	if best := findLargestHicolorPNG(hicolorDirs, name); best != "" {
		return best
	}

	for _, dir := range pixmapDirs {
		for _, ext := range []string{".png", ".svg", ".xpm"} {
			path := filepath.Join(dir, name+ext)
			if fileExists(path) {
				return path
			}
		}
	}

	for _, dir := range hicolorDirs {
		for _, ext := range []string{".svg", ".png"} {
			path := filepath.Join(dir, "scalable", "apps", name+ext)
			if fileExists(path) {
				return path
			}
		}
	}

	return ""
}

func findLargestHicolorPNG(hicolorDirs []string, name string) string {
	type candidate struct {
		size int
		path string
	}

	var candidates []candidate

	for _, dir := range hicolorDirs {
		entries, err := os.ReadDir(dir)
		if err != nil {
			continue
		}

		for _, entry := range entries {
			m := hicolorSizeRe.FindStringSubmatch(entry.Name())
			if m == nil {
				continue
			}

			size, err := strconv.Atoi(m[1])
			if err != nil {
				continue
			}

			path := filepath.Join(dir, entry.Name(), "apps", name+".png")
			if fileExists(path) {
				candidates = append(candidates, candidate{size: size, path: path})
			}
		}
	}

	if len(candidates) == 0 {
		return ""
	}

	sort.Slice(candidates, func(i, j int) bool { return candidates[i].size > candidates[j].size })

	return candidates[0].path
}

func fileExists(path string) bool {
	info, err := os.Stat(path)

	return err == nil && !info.IsDir()
}
