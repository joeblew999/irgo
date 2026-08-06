// Installing an already-built app.
//
// `irgo run` builds, installs and launches in one step, which is right while
// developing and wrong when you want to check the artifact itself: whether the
// signed bundle installs, whether the packaged app behaves like it will for a
// user. Doing that meant copying files by hand, which is the fiddling the CLI
// exists to remove — and it left `app` with a remove and no install.
//
// Nothing is built here. Whatever exists is installed, and the packaged
// artifact wins over the development one, because that is the thing that
// actually ships.
package main

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

// runAppInstall installs the built app for a platform.
func runAppInstall(platform string) error {
	switch platform {
	case "ios":
		return installIOSApp()
	case "android":
		return installAndroidApp()
	case "desktop":
		return installDesktopApp()
	case "all":
		var errs []string
		for _, p := range []string{"ios", "android", "desktop"} {
			if err := runAppInstall(p); err != nil {
				errs = append(errs, fmt.Sprintf("%s: %v", p, err))
			}
		}
		if len(errs) > 0 {
			return fmt.Errorf("%s", strings.Join(errs, "; "))
		}
		return nil
	default:
		return fmt.Errorf("unknown platform: %s (use ios, android, desktop, or all)", platform)
	}
}

// iosSimulatorApp is where `irgo build ios --sim` leaves its bundle.
const iosSimulatorApp = "build/ios/DerivedData/Build/Products/Debug-iphonesimulator/Example.app"

func installIOSApp() error {
	if err := requireMacOS("iOS install"); err != nil {
		return err
	}
	if !pathExists(iosSimulatorApp) {
		return fmt.Errorf("no Simulator app at %s — build it first: irgo build ios --sim", iosSimulatorApp)
	}
	fmt.Printf("  ios: installing %s on the booted simulator\n", iosSimulatorApp)
	if err := runCommand("xcrun", "simctl", "install", "booted", iosSimulatorApp); err != nil {
		return fmt.Errorf("simctl install failed (is a simulator booted? try: irgo run ios): %w", err)
	}
	fmt.Printf("  ios: installed %s\n", iosBundleID)
	return nil
}

func installAndroidApp() error {
	adb := adbBin()
	if adb == "" {
		return fmt.Errorf("adb not found — install the toolchain first: irgo tools install android")
	}
	apk := filepath.Join("android/Example", "app/build/outputs/apk/debug/app-debug.apk")
	if !pathExists(apk) {
		return fmt.Errorf("no APK at %s — build it first: irgo run android", apk)
	}
	fmt.Printf("  android: installing %s\n", apk)
	// -r replaces an existing install rather than failing on a conflict.
	if err := runCommand(adb, "install", "-r", apk); err != nil {
		return fmt.Errorf("adb install failed (is a device or emulator attached?): %w", err)
	}
	return nil
}

// installDesktopApp puts the app where a user would have it, so it can be
// launched and uninstalled like a real install rather than run from build/.
func installDesktopApp() error {
	modulePath, err := getModulePath()
	if err != nil {
		return err
	}
	name := filepath.Base(modulePath)

	if runtime.GOOS != "darwin" {
		return fmt.Errorf("desktop install is macOS-only; on %s run the binary from build/desktop/%s",
			runtime.GOOS, runtime.GOOS)
	}

	// The packaged bundle is signed and entitled — the one that ships — so it
	// is preferred when present.
	src := filepath.Join("dist/macos", name+".app")
	origin := "packaged"
	if !pathExists(src) {
		src = filepath.Join("build/desktop/macos", name+".app")
		origin = "development"
	}
	if !pathExists(src) {
		return fmt.Errorf("no macOS app found — build one first: irgo build desktop (or irgo package macos)")
	}

	dst := "/Applications/" + name + ".app"
	if pathExists(dst) {
		// Replacing rather than merging: a stale file left inside an old
		// bundle is exactly the sort of thing that produces a confusing
		// runtime failure.
		if err := os.RemoveAll(dst); err != nil {
			return fmt.Errorf("removing the existing %s: %w", dst, err)
		}
	}
	fmt.Printf("  desktop: installing the %s build to %s\n", origin, dst)
	if err := copyDir(src, dst); err != nil {
		return fmt.Errorf("installing to /Applications (permission?): %w", err)
	}
	fmt.Printf("  desktop: installed %s\n", dst)
	fmt.Println("  run it:  open " + dst)
	return nil
}
