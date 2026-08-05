package main

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"hash/fnv"
	"image"
	"image/color"
	"image/png"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

// This file adds the single-source app icon (appicon.png, static/icon.png, or
// [common] icon in irgo.package.toml — see findAppIcon) to BUILD outputs, so
// `irgo build/run` shows a real icon on each platform, not just `irgo package`.

// writeICO writes a Windows .ico containing 16/32/48/256 px PNG-embedded
// images scaled from src (PNG-in-ICO, supported since Vista).
func writeICO(path string, src image.Image) error {
	sizes := []int{16, 32, 48, 256}
	type entry struct {
		size int
		png  []byte
	}
	var entries []entry
	for _, s := range sizes {
		dst := image.NewRGBA(image.Rect(0, 0, s, s))
		scaleNearest(dst, src)
		var buf bytes.Buffer
		if err := png.Encode(&buf, dst); err != nil {
			return err
		}
		entries = append(entries, entry{s, buf.Bytes()})
	}

	var out bytes.Buffer
	// ICONDIR
	if err := binary.Write(&out, binary.LittleEndian, uint16(0)); err != nil { // reserved
		return err
	}
	if err := binary.Write(&out, binary.LittleEndian, uint16(1)); err != nil { // type: icon
		return err
	}
	if err := binary.Write(&out, binary.LittleEndian, uint16(len(entries))); err != nil {
		return err
	}
	offset := 6 + 16*len(entries)
	for _, e := range entries {
		w, h := e.size, e.size
		if w >= 256 {
			w, h = 0, 0 // 0 means 256 in the ICO header
		}
		if err := binary.Write(&out, binary.LittleEndian, uint8(w)); err != nil {
			return err
		}
		if err := binary.Write(&out, binary.LittleEndian, uint8(h)); err != nil {
			return err
		}
		if err := binary.Write(&out, binary.LittleEndian, uint8(0)); err != nil { // palette
			return err
		}
		if err := binary.Write(&out, binary.LittleEndian, uint8(0)); err != nil { // reserved
			return err
		}
		if err := binary.Write(&out, binary.LittleEndian, uint16(1)); err != nil { // planes
			return err
		}
		if err := binary.Write(&out, binary.LittleEndian, uint16(32)); err != nil { // bpp
			return err
		}
		if err := binary.Write(&out, binary.LittleEndian, uint32(len(e.png))); err != nil {
			return err
		}
		if err := binary.Write(&out, binary.LittleEndian, uint32(offset)); err != nil {
			return err
		}
		offset += len(e.png)
	}
	for _, e := range entries {
		if _, err := out.Write(e.png); err != nil {
			return err
		}
	}
	return os.WriteFile(path, out.Bytes(), 0o644)
}

// embedWindowsIcon generates an .ico + .rc from the source icon and compiles
// them with windres (from the mingw-w64 toolchain we already require for
// Windows builds) into a .syso next to the main package, so the next
// `go build` links the icon into the .exe. Returns a cleanup func that must
// run after the build.
func embedWindowsIcon(iconPath string) (func(), error) {
	src, err := decodeIcon(iconPath)
	if err != nil {
		return nil, fmt.Errorf("decoding icon %s: %w", iconPath, err)
	}
	tmp, err := os.MkdirTemp("", "irgo-ico-")
	if err != nil {
		return nil, err
	}
	icoPath := filepath.Join(tmp, "app.ico")
	if err := writeICO(icoPath, src); err != nil {
		return nil, err
	}
	rcPath := filepath.Join(tmp, "app.rc")
	if err := os.WriteFile(rcPath, []byte("1 ICON \"app.ico\"\n"), 0o644); err != nil {
		return nil, err
	}

	windres := "windres"
	if runtime.GOOS == "darwin" {
		windres = "x86_64-w64-mingw32-windres"
	}
	if _, err := exec.LookPath(windres); err != nil {
		return nil, fmt.Errorf("windres not found (install mingw-w64 for Windows cross-builds)")
	}

	// .syso in the package dir gets linked automatically for the target
	// GOOS/GOARCH (we always target windows/amd64).
	syso := "rsrc_windows_amd64.syso"
	cmd := exec.Command(windres, "-i", rcPath, "-o", syso)
	cmd.Dir = tmp
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("windres failed: %w", err)
	}
	dest := filepath.Join(syso)
	if err := copyFile(filepath.Join(tmp, syso), dest); err != nil {
		return nil, err
	}
	fmt.Printf("  embedded Windows icon from %s\n", iconPath)
	return func() { os.Remove(dest) }, nil
}

// ---------------------------------------------------------------------------
// single-source-icon pipeline — one PNG → every store's required assets
// ---------------------------------------------------------------------------

type iconVariant struct {
	path string
	w, h int
}

// scaleNearest draws src scaled into dst with nearest-neighbor (stdlib only).
func scaleNearest(dst *image.RGBA, src image.Image) {
	b := dst.Bounds()
	sb := src.Bounds()
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			sx := sb.Min.X + (x-b.Min.X)*(sb.Dx())/b.Dx()
			sy := sb.Min.Y + (y-b.Min.Y)*(sb.Dy())/b.Dy()
			dst.Set(x, y, src.At(sx, sy))
		}
	}
}

// findAppIcon resolves the source icon: explicit flag > [common] icon in
// irgo.package.toml > appicon.png > static/icon.png. Returns "" when absent
// (callers fall back to placeholders or template defaults).
func findAppIcon(explicit string) string {
	if explicit != "" {
		return expandHome(explicit)
	}
	cfg := parsePackageConfig()
	if cfg.Icon != "" {
		return expandHome(cfg.Icon)
	}
	for _, p := range []string{"appicon.png", filepath.Join("static", "icon.png")} {
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}
	return ""
}

// decodeIcon loads a PNG source icon.
func decodeIcon(path string) (image.Image, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return png.Decode(f)
}

// writeIconVariant scales the source into a sized PNG at outDir.
func writeIconVariant(outDir string, src image.Image, v iconVariant) error {
	dst := image.NewRGBA(image.Rect(0, 0, v.w, v.h))
	scaleNearest(dst, src)
	f, err := os.Create(filepath.Join(outDir, v.path))
	if err != nil {
		return err
	}
	defer f.Close()
	return png.Encode(f, dst)
}

// generateAndroidIcons writes the Android launcher mipmaps from a source icon
// into the Example app's res/ directory (replacing the template defaults).
func generateAndroidIcons(iconPath, resDir string) error {
	src, err := decodeIcon(iconPath)
	if err != nil {
		return fmt.Errorf("decoding icon %s: %w", iconPath, err)
	}
	variants := []iconVariant{
		{filepath.Join("mipmap-mdpi", "ic_launcher.png"), 48, 48},
		{filepath.Join("mipmap-hdpi", "ic_launcher.png"), 72, 72},
		{filepath.Join("mipmap-xhdpi", "ic_launcher.png"), 96, 96},
		{filepath.Join("mipmap-xxhdpi", "ic_launcher.png"), 144, 144},
		{filepath.Join("mipmap-xxxhdpi", "ic_launcher.png"), 192, 192},
	}
	for _, v := range variants {
		if err := os.MkdirAll(filepath.Join(resDir, filepath.Dir(v.path)), 0o755); err != nil {
			return err
		}
		if err := writeIconVariant(filepath.Join(resDir, filepath.Dir(v.path)), src, iconVariant{filepath.Base(v.path), v.w, v.h}); err != nil {
			return err
		}
	}
	fmt.Printf("  wrote Android launcher icons from %s\n", iconPath)
	return nil
}

// generateICNS builds a .icns (via iconutil) from a source icon and installs it
// into the .app bundle, pointing CFBundleIconFile at it.
func generateICNS(iconPath, appDir string) error {
	src, err := decodeIcon(iconPath)
	if err != nil {
		return fmt.Errorf("decoding icon %s: %w", iconPath, err)
	}
	tmp, err := os.MkdirTemp("", "irgo-icns-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmp)

	iconset := filepath.Join(tmp, "icon.iconset")
	if err := os.MkdirAll(iconset, 0o755); err != nil {
		return err
	}
	variants := []iconVariant{
		{"icon_16x16.png", 16, 16},
		{"icon_16x16@2x.png", 32, 32},
		{"icon_32x32.png", 32, 32},
		{"icon_32x32@2x.png", 64, 64},
		{"icon_128x128.png", 128, 128},
		{"icon_128x128@2x.png", 256, 256},
		{"icon_256x256.png", 256, 256},
		{"icon_256x256@2x.png", 512, 512},
		{"icon_512x512.png", 512, 512},
		{"icon_512x512@2x.png", 1024, 1024},
	}
	for _, v := range variants {
		if err := writeIconVariant(iconset, src, v); err != nil {
			return err
		}
	}
	icnsPath := filepath.Join(tmp, "icon.icns")
	if err := runCommand("iconutil", "-c", "icns", iconset, "-o", icnsPath); err != nil {
		return fmt.Errorf("iconutil failed: %w", err)
	}

	resDir := filepath.Join(appDir, "Contents", "Resources")
	if err := os.MkdirAll(resDir, 0o755); err != nil {
		return err
	}
	if err := copyFile(icnsPath, filepath.Join(resDir, "icon.icns")); err != nil {
		return err
	}

	// Point CFBundleIconFile at it (insert into the root <dict> of Info.plist).
	infoPath := filepath.Join(appDir, "Contents", "Info.plist")
	if data, err := os.ReadFile(infoPath); err == nil {
		iconKeys := "<key>CFBundleIconFile</key>\n\t<string>icon.icns</string>\n\t"
		if !strings.Contains(string(data), "CFBundleIconFile") {
			updated := strings.Replace(string(data), "<dict>", "<dict>\n\t"+iconKeys, 1)
			_ = os.WriteFile(infoPath, []byte(updated), 0o644)
		}
	}
	fmt.Printf("  wrote macOS app icon from %s\n", iconPath)
	return nil
}

// writeMSIXAssets writes the MSIX visual assets. With an icon it scales the
// given PNG (stdlib nearest-neighbor); without it, solid-color placeholders.
func writeMSIXAssets(assetsDir, iconPath, appName string) error {
	specs := []struct {
		name string
		w, h int
	}{
		{"StoreLogo.png", 50, 50},
		{"Square44x44Logo.png", 44, 44},
		{"Square150x150Logo.png", 150, 150},
		{"Square310x310Logo.png", 310, 310},
		{"Wide310x150Logo.png", 310, 150},
	}

	var src image.Image
	if iconPath != "" {
		f, err := os.Open(iconPath)
		if err != nil {
			return fmt.Errorf("opening icon %s: %w", iconPath, err)
		}
		src, err = png.Decode(f)
		f.Close()
		if err != nil {
			return fmt.Errorf("decoding icon %s (must be PNG): %w", iconPath, err)
		}
	}

	// Deterministic placeholder color from the app name.
	var c color.RGBA = color.RGBA{0x1F, 0x4E, 0x79, 0xFF}
	if src == nil {
		h := fnv.New32a()
		h.Write([]byte(appName))
		sum := h.Sum32()
		c = color.RGBA{
			R: uint8(30 + sum%200),
			G: uint8(40 + (sum/200)%180),
			B: uint8(50 + (sum/40000)%160),
			A: 255,
		}
	}

	for _, s := range specs {
		dst := image.NewRGBA(image.Rect(0, 0, s.w, s.h))
		if src != nil {
			scaleNearest(dst, src)
		} else {
			for x := 0; x < s.w; x++ {
				for y := 0; y < s.h; y++ {
					dst.Set(x, y, c)
				}
			}
		}
		f, err := os.Create(filepath.Join(assetsDir, s.name))
		if err != nil {
			return err
		}
		err = png.Encode(f, dst)
		f.Close()
		if err != nil {
			return err
		}
	}
	return nil
}

// generateIOSIcons writes the app icon into the Xcode asset catalog. Xcode 14+
// accepts a single 1024x1024 universal image and derives every other size, so
// one source PNG covers the whole set.
//
// The project sets ASSETCATALOG_COMPILER_APPICON_NAME = AppIcon, so without
// this the build looks for an icon, finds nothing, and ships the blank default.
func generateIOSIcons(iconPath, iosProjectPath string) error {
	src, err := decodeIcon(iconPath)
	if err != nil {
		return err
	}
	set := filepath.Join(iosProjectPath, "Example", "Assets.xcassets", "AppIcon.appiconset")
	if err := os.MkdirAll(set, 0o755); err != nil {
		return err
	}
	if err := writeIconVariant(set, src, iconVariant{path: "icon-1024.png", w: 1024, h: 1024}); err != nil {
		return err
	}
	contents := `{
  "images" : [
    {
      "filename" : "icon-1024.png",
      "idiom" : "universal",
      "platform" : "ios",
      "size" : "1024x1024"
    }
  ],
  "info" : {
    "author" : "xcode",
    "version" : 1
  }
}
`
	return os.WriteFile(filepath.Join(set, "Contents.json"), []byte(contents), 0o644)
}
