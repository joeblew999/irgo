// Removing an installed app from a simulator, emulator or this machine.
//
// `irgo run` installs onto a device; without the inverse the lifecycle is
// one-way. A stale install is also a real source of confusion — the app keeps
// launching with old assets or an old bridge API, and nothing says why.
package main

import (
	"fmt"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

// appBundleID is the identifier `irgo run` installs under.
func appBundleID() (string, error) {
	modulePath, err := getModulePath()
	if err != nil {
		return "", fmt.Errorf("could not determine module path: %w", err)
	}
	return bundleIDFromModulePath(modulePath), nil
}

// runAppUninstall removes the installed app for a platform. Absence is success:
// the point is to end up with the app not installed, and reporting failure for
// an app that was never there just adds noise to a cleanup script.
func runAppUninstall(platform string) error {
	switch platform {
	case "ios":
		return uninstallIOSApp()
	case "android":
		return uninstallAndroidApp()
	case "desktop":
		return uninstallDesktopApp()
	case "all":
		var errs []string
		for _, p := range []string{"ios", "android", "desktop"} {
			if err := runAppUninstall(p); err != nil {
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

func uninstallIOSApp() error {
	if runtime.GOOS != "darwin" {
		fmt.Println("  ios: skipped (simulators only exist on macOS)")
		return nil
	}
	if _, err := exec.LookPath("xcrun"); err != nil {
		fmt.Println("  ios: skipped (xcrun not found)")
		return nil
	}
	// The example app installs under this identifier; the Xcode project sets it
	// independently of the Go module path.
	const bundleID = "com.irgo.Example"
	out, err := exec.Command("xcrun", "simctl", "uninstall", "booted", bundleID).CombinedOutput()
	if err != nil {
		if strings.Contains(string(out), "No devices are booted") {
			fmt.Println("  ios: no booted simulator — nothing to remove")
			return nil
		}
		// simctl reports a missing app as an error; that is the desired state.
		fmt.Printf("  ios: not installed on the booted simulator\n")
		return nil
	}
	fmt.Printf("  ios: removed %s from the booted simulator\n", bundleID)
	return nil
}

func uninstallAndroidApp() error {
	adb := adbBin()
	if adb == "" {
		fmt.Println("  android: skipped (adb not found — install the toolchain first)")
		return nil
	}
	const pkg = "com.irgo.example"
	out, err := exec.Command(adb, "uninstall", pkg).CombinedOutput()
	text := strings.TrimSpace(string(out))
	switch {
	case err != nil && strings.Contains(text, "no devices"):
		fmt.Println("  android: no device or emulator attached — nothing to remove")
	case strings.Contains(text, "Success"):
		fmt.Printf("  android: removed %s\n", pkg)
	default:
		fmt.Printf("  android: not installed (%s)\n", firstLine(text))
	}
	return nil
}

// uninstallDesktopApp removes a copy installed into the system Applications
// directory. Build output under build/ is `irgo clean`'s job, not this one.
func uninstallDesktopApp() error {
	if runtime.GOOS != "darwin" {
		fmt.Printf("  desktop: nothing to remove on %s (the app runs from build/ — use irgo clean)\n", runtime.GOOS)
		return nil
	}
	modulePath, err := getModulePath()
	if err != nil {
		return err
	}
	name := baseName(modulePath)
	found := false
	for _, p := range []string{
		"/Applications/" + name + ".app",
		filepath.Join(homeDir(), "Applications", name+".app"),
	} {
		if pathExists(p) {
			fmt.Printf("  desktop: removing %s\n", p)
			if err := removeAllPath(p); err != nil {
				return err
			}
			found = true
		}
	}
	// Packaged output is `irgo clean`'s business, but say it is there —
	// otherwise "uninstalled" is confusing when the app is still on disk.
	if pathExists(filepath.Join("dist/macos", name+".app")) {
		fmt.Printf("  desktop: dist/macos/%s.app remains (packaged output — irgo clean removes it)\n", name)
	}
	if !found {
		fmt.Println("  desktop: not installed in /Applications")
	}
	return nil
}
