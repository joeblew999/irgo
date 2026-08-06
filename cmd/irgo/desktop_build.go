package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

// runDesktop builds and runs a desktop app. With built=true it launches the
// artifact `irgo build desktop` produced instead of `go run` — that is the one
// that behaves like what ships (bundled resources, app icon, no toolchain).
func runDesktop(devMode, built bool) error {
	if built {
		return launchBuiltDesktop()
	}
	fmt.Println("Starting desktop app...")

	// Regenerate the gitignored-but-embedded assets, as the build paths do.
	if err := ensureAssets(); err != nil {
		return err
	}

	// Verify the toolchain before running (clear errors, idempotent — never
	// assume the machine is pre-configured).
	if err := ensureDesktopToolchain(runtime.GOOS); err != nil {
		return err
	}

	args := []string{"run", "-tags", "desktop", "."}
	if devMode {
		args = append(args, "--dev")
	}

	cmd := exec.Command(goBin(), args...)
	cmd.Env = append(os.Environ(), "CGO_ENABLED=1")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	return cmd.Run()
}

// ensureDesktopToolchain verifies the CGO/webview toolchain needed to build
// desktop apps for `target` on the current host, returning actionable install
// guidance when anything is missing. Checked on every build/run — the command
// never assumes a pre-configured machine (idempotent: same check each time,
// clear errors instead of an obscure `go build` failure).
func ensureDesktopToolchain(target string) error {
	switch target {
	case "darwin", "macos":
		if runtime.GOOS != "darwin" {
			return fmt.Errorf("macOS desktop builds require macOS (Xcode) — cannot cross-compile from %s.\n  Run `irgo doctor` to see everything this host can and cannot build", runtime.GOOS)
		}
		if _, err := exec.LookPath("clang"); err != nil {
			return fmt.Errorf("C compiler not found — install Xcode Command Line Tools: xcode-select --install")
		}
		if out, err := exec.Command("xcode-select", "-p").Output(); err != nil || strings.TrimSpace(string(out)) == "" {
			return fmt.Errorf("Xcode Command Line Tools not installed — run: xcode-select --install")
		}
	case "windows":
		if runtime.GOOS == "darwin" {
			// macOS cross-compiles via mingw-w64 — install it rather than
			// telling the caller to.
			if err := ensureOSPackage("mingw-w64"); err != nil {
				return err
			}
		} else if runtime.GOOS == "windows" {
			for _, tool := range []string{"gcc", "g++"} {
				if _, err := exec.LookPath(tool); err != nil {
					return fmt.Errorf("%s not found — install MSYS2 (https://www.msys2.org) with the mingw-w64-x86_64 toolchain", tool)
				}
			}
		} else {
			return fmt.Errorf("Windows desktop builds are supported from Windows or macOS (with mingw-w64), not %s.\n  Run `irgo doctor` to see everything this host can and cannot build", runtime.GOOS)
		}
	case "linux":
		if runtime.GOOS != "linux" {
			return fmt.Errorf("Linux desktop builds require Linux (GTK3 + WebKit2GTK) — cannot cross-compile from %s.\n  Run `irgo doctor` to see everything this host can and cannot build", runtime.GOOS)
		}
		if _, err := exec.LookPath("gcc"); err != nil {
			return fmt.Errorf("gcc not found — install build-essential: sudo apt install build-essential")
		}
		if err := ensureOSPackage("webkit2gtk"); err != nil {
			return err
		}
	default:
		return fmt.Errorf("unsupported desktop platform: %s", target)
	}
	return nil
}

// desktopTargetsForHost lists the desktop targets this host can actually
// produce. macOS needs Xcode and Linux needs GTK3 + WebKit2GTK, so neither
// cross-compiles; Windows is the only one that does, from macOS via mingw-w64.
func desktopTargetsForHost() []string {
	switch runtime.GOOS {
	case "darwin":
		return []string{"darwin", "windows"}
	case "linux":
		return []string{"linux"}
	case "windows":
		return []string{"windows"}
	default:
		return nil
	}
}

// buildDesktop builds desktop app for target platform
func buildDesktop(target string) error {
	if target == "" {
		target = runtime.GOOS
	}

	if target == "all" {
		targets := desktopTargetsForHost()
		if len(targets) == 0 {
			return fmt.Errorf("no desktop targets can be built from %s", runtime.GOOS)
		}
		fmt.Printf("Building desktop apps for: %s\n", strings.Join(targets, ", "))
		for _, t := range targets {
			if err := buildDesktop(t); err != nil {
				return fmt.Errorf("%s: %w", t, err)
			}
		}
		return nil
	}

	fmt.Printf("Building desktop app for %s...\n", target)

	// Verify the toolchain before building (clear errors, idempotent).
	if err := ensureDesktopToolchain(target); err != nil {
		return err
	}

	// Regenerate the gitignored-but-embedded assets (templ + CSS) first.
	if err := ensureAssets(); err != nil {
		return err
	}

	modulePath, err := getModulePath()
	if err != nil {
		return fmt.Errorf("could not determine module path: %w", err)
	}

	switch target {
	case "darwin", "macos":
		return buildDesktopMacOS(modulePath)
	case "windows":
		return buildDesktopWindows(modulePath)
	case "linux":
		return buildDesktopLinux(modulePath)
	default:
		return fmt.Errorf("unsupported desktop platform: %s (use darwin, windows, or linux)", target)
	}
}

func buildDesktopMacOS(modulePath string) error {
	appName := filepath.Base(modulePath)
	outDir := "build/desktop/macos"
	appBundle := filepath.Join(outDir, appName+".app")

	// Create .app bundle structure
	if err := os.MkdirAll(filepath.Join(appBundle, "Contents", "MacOS"), 0755); err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Join(appBundle, "Contents", "Resources"), 0755); err != nil {
		return err
	}

	// Build the binary with CGO enabled (required for webview)
	binaryPath := filepath.Join(appBundle, "Contents", "MacOS", appName)
	cmd := exec.Command(goBin(), "build", "-tags", "desktop", "-o", binaryPath, ".")
	cmd.Env = append(os.Environ(), "CGO_ENABLED=1")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("go build failed: %w", err)
	}

	// Copy static assets to Resources
	if _, err := os.Stat("static"); err == nil {
		if err := copyDir("static", filepath.Join(appBundle, "Contents", "Resources", "static")); err != nil {
			fmt.Printf("Warning: could not copy static assets: %v\n", err)
		}
	}

	// Generate Info.plist
	plistContent := generateMacOSPlist(appName, bundleIDFromModulePath(modulePath))
	plistPath := filepath.Join(appBundle, "Contents", "Info.plist")
	if err := os.WriteFile(plistPath, []byte(plistContent), 0644); err != nil {
		return fmt.Errorf("could not write Info.plist: %w", err)
	}

	// App icon: single source icon → .icns in the bundle (if present).
	if ic := findAppIcon(""); ic != "" {
		if err := generateICNS(ic, appBundle); err != nil {
			fmt.Printf("Warning: could not add app icon: %v\n", err)
		}
	}

	fmt.Printf("macOS app built: %s\n", appBundle)
	runHint("open "+appBundle, "irgo run desktop --built")
	return nil
}

func buildDesktopWindows(modulePath string) error {
	appName := filepath.Base(modulePath)
	outDir := "build/desktop/windows"

	if err := os.MkdirAll(outDir, 0755); err != nil {
		return err
	}

	binaryPath := filepath.Join(outDir, appName+".exe")

	// App icon: single source icon → .syso linked into the .exe. Only attempted
	// on native Windows — cross-compiling (mingw windres) produces a .syso that
	// Go's linker rejects, so we skip it there (the MSIX package still gets the
	// icon via its tile assets).
	if runtime.GOOS == "windows" {
		if ic := findAppIcon(""); ic != "" {
			if cleanup, err := embedWindowsIcon(ic); err != nil {
				fmt.Printf("Warning: could not embed Windows icon: %v\n", err)
			} else {
				defer cleanup()
			}
		}
	} else if findAppIcon("") != "" {
		fmt.Println("  (skipping .exe icon embed: only on native Windows; MSIX package carries the icon)")
	}

	cmd := exec.Command(goBin(), "build",
		"-tags", "desktop",
		"-ldflags", "-H windowsgui", // Hide console window
		"-o", binaryPath,
		".",
	)
	// Cross-compiling to windows/amd64 requires the mingw-w64 CC (checked in
	// ensureDesktopToolchain); without GOOS/GOARCH/CC the host toolchain would
	// be used and the link fails.
	cmd.Env = append(os.Environ(),
		"GOOS=windows",
		"GOARCH=amd64",
		"CC=x86_64-w64-mingw32-gcc",
		"CXX=x86_64-w64-mingw32-g++", // webview uses C++; without this the host clang++ is used
		"CGO_ENABLED=1",
	)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("go build failed: %w", err)
	}

	// Copy static assets
	if _, err := os.Stat("static"); err == nil {
		if err := copyDir("static", filepath.Join(outDir, "static")); err != nil {
			fmt.Printf("Warning: could not copy static assets: %v\n", err)
		}
	}

	fmt.Printf("Windows app built: %s\n", binaryPath)
	runHint(binaryPath, "irgo run desktop --built  (on Windows)")
	return nil
}

func buildDesktopLinux(modulePath string) error {
	appName := filepath.Base(modulePath)
	outDir := "build/desktop/linux"

	if err := os.MkdirAll(outDir, 0755); err != nil {
		return err
	}

	binaryPath := filepath.Join(outDir, appName)
	cmd := exec.Command(goBin(), "build",
		"-tags", "desktop",
		"-o", binaryPath,
		".",
	)
	cmd.Env = append(os.Environ(), "CGO_ENABLED=1")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("go build failed: %w", err)
	}

	// Copy static assets
	if _, err := os.Stat("static"); err == nil {
		if err := copyDir("static", filepath.Join(outDir, "static")); err != nil {
			fmt.Printf("Warning: could not copy static assets: %v\n", err)
		}
	}

	fmt.Printf("Linux app built: %s\n", binaryPath)
	runHint("./"+binaryPath, "irgo run desktop --built")
	return nil
}

// bundleIDFromModulePath converts a Go module path into a valid reverse-DNS
// style CFBundleIdentifier. A raw module path like "github.com/user/app" is
// invalid (contains slashes); it becomes "com.github.user.app". A bare module
// name like "myapp" becomes "com.irgo.myapp". Characters outside
// [A-Za-z0-9.-] are stripped.
func bundleIDFromModulePath(modulePath string) string {
	segments := strings.Split(modulePath, "/")

	var parts []string
	if len(segments) > 1 && strings.Contains(segments[0], ".") {
		// First segment is a host name (e.g. github.com) - reverse it
		host := strings.Split(segments[0], ".")
		for i, j := 0, len(host)-1; i < j; i, j = i+1, j-1 {
			host[i], host[j] = host[j], host[i]
		}
		parts = append(parts, host...)
		parts = append(parts, segments[1:]...)
	} else if len(segments) > 1 {
		parts = segments
	} else {
		// Bare module name like "myapp"
		parts = append([]string{"com", "irgo"}, segments...)
	}

	// Strip invalid characters from each part and drop empty parts
	sanitized := make([]string, 0, len(parts))
	for _, part := range parts {
		var b strings.Builder
		for _, r := range part {
			if (r >= 'A' && r <= 'Z') || (r >= 'a' && r <= 'z') ||
				(r >= '0' && r <= '9') || r == '-' {
				b.WriteRune(r)
			}
		}
		if b.Len() > 0 {
			sanitized = append(sanitized, b.String())
		}
	}

	if len(sanitized) == 0 {
		return "com.irgo.app"
	}
	return strings.Join(sanitized, ".")
}

func generateMacOSPlist(appName, bundleID string) string {
	return fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>CFBundleExecutable</key>
    <string>%s</string>
    <key>CFBundleIdentifier</key>
    <string>%s</string>
    <key>CFBundleName</key>
    <string>%s</string>
    <key>CFBundleVersion</key>
    <string>1.0.0</string>
    <key>CFBundleShortVersionString</key>
    <string>1.0.0</string>
    <key>CFBundlePackageType</key>
    <string>APPL</string>
    <key>NSHighResolutionCapable</key>
    <true/>
</dict>
</plist>`, appName, bundleID, appName)
}

func copyDir(src, dst string) error {
	return filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		relPath, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		dstPath := filepath.Join(dst, relPath)

		if info.IsDir() {
			return os.MkdirAll(dstPath, info.Mode())
		}

		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		return os.WriteFile(dstPath, data, info.Mode())
	})
}

// hasFlag checks if any of the given flags are present in args
func hasFlag(args []string, flags ...string) bool {
	for _, arg := range args {
		for _, flag := range flags {
			if arg == flag {
				return true
			}
		}
	}
	return false
}

// launchBuiltDesktop runs the artifact from build/desktop for this host.
func launchBuiltDesktop() error {
	modulePath, err := getModulePath()
	if err != nil {
		return fmt.Errorf("could not determine module path: %w", err)
	}
	app := filepath.Base(modulePath)

	var path string
	var cmd []string
	switch runtime.GOOS {
	case "darwin":
		// Prefer the packaged app when there is one: it is the artifact that
		// actually ships (signed, entitled, notarized), so running it is what
		// tells you whether shipping will work.
		path = filepath.Join("dist/macos", app+".app")
		if !pathExists(path) {
			path = filepath.Join("build/desktop/macos", app+".app")
		}
		cmd = []string{"open", path}
	case "windows":
		path = filepath.Join("build/desktop/windows", app+".exe")
		cmd = []string{path}
	case "linux":
		path = filepath.Join("build/desktop/linux", app)
		cmd = []string{"./" + path}
	default:
		return fmt.Errorf("no desktop artifact layout known for %s", runtime.GOOS)
	}

	if _, err := os.Stat(path); err != nil {
		return fmt.Errorf("no built app at %s — build it first: irgo build desktop", path)
	}
	fmt.Printf("Launching %s...\n", path)
	return runCommand(cmd[0], cmd[1:]...)
}
