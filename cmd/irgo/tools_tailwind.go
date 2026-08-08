// Tailwind without Node.
//
// Tailwind ships a standalone executable for every platform, so the stylesheet
// does not need Node, npm, bun, a package.json or a node_modules tree. Those
// were a whole second toolchain to install, pin and keep working — and the one
// most likely to be missing or half-configured on a given machine.
//
// irgo downloads the binary the same way it downloads the JDK and the Android
// SDK: on first use, into ~/.irgo, at a pinned version, and removable again.
package main

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

// pinTailwind is the version irgo downloads. Pinned rather than "latest" so a
// stylesheet does not change under a project because a release happened.
const pinTailwind = "v4.3.3"

// tailwindBin returns the path irgo keeps its Tailwind at.
func tailwindBin() string {
	name := "tailwindcss-" + pinTailwind
	if runtime.GOOS == "windows" {
		name += ".exe"
	}
	return filepath.Join(irgoBinDir(), name)
}

// tailwindPath finds an existing Tailwind without installing one.
//
// irgo's own copy, then mise's. doctor calls this, so it names the binary a
// build would use — it was reporting "absent" while builds happily used the
// one mise had, which is the gap this whole exercise kept turning up.
func tailwindPath() string {
	if fi, err := os.Stat(tailwindBin()); err == nil && fi.Mode()&0o111 != 0 {
		return tailwindBin()
	}
	if mise, ok := miseCmd(); ok {
		return miseWhere(mise, "aqua:tailwindlabs/tailwindcss@"+
			strings.TrimPrefix(pinTailwind, "v"), "tailwindcss")
	}
	return ""
}

// tailwindAsset maps the host to a release asset name.
func tailwindAsset() (string, error) {
	switch runtime.GOOS {
	case "darwin":
		if runtime.GOARCH == "arm64" {
			return "tailwindcss-macos-arm64", nil
		}
		return "tailwindcss-macos-x64", nil
	case "linux":
		if runtime.GOARCH == "arm64" {
			return "tailwindcss-linux-arm64", nil
		}
		return "tailwindcss-linux-x64", nil
	case "windows":
		return "tailwindcss-windows-x64.exe", nil
	}
	return "", fmt.Errorf("no Tailwind build for %s/%s", runtime.GOOS, runtime.GOARCH)
}

// ensureTailwind downloads the standalone CLI when it is missing, and returns
// its path. Idempotent: the version is in the filename, so an upgrade fetches
// a new file rather than silently replacing one that a build may be using.
func ensureTailwind() (string, error) {
	if bin := tailwindPath(); bin != "" {
		return bin, nil
	}
	bin := tailwindBin()

	// mise first, at irgo's pin. aqua has it, which the registry does not say
	// because that listing is truncated.
	if bin := miseTool("aqua:tailwindlabs/tailwindcss@"+strings.TrimPrefix(pinTailwind, "v"),
		"tailwindcss"); bin != "" {
		return bin, nil
	}

	asset, err := tailwindAsset()
	if err != nil {
		return "", err
	}
	url := fmt.Sprintf("https://github.com/tailwindlabs/tailwindcss/releases/download/%s/%s",
		pinTailwind, asset)

	if err := os.MkdirAll(filepath.Dir(bin), 0o755); err != nil {
		return "", err
	}
	fmt.Printf("Downloading Tailwind %s (%s)...\n", pinTailwind, asset)

	// Download to a temporary name and rename, so an interrupted download can
	// never leave a truncated binary that later looks installed.
	tmp := bin + ".part"
	if err := downloadFile(url, tmp); err != nil {
		os.Remove(tmp)
		return "", fmt.Errorf("downloading Tailwind: %w", err)
	}
	if err := os.Chmod(tmp, 0o755); err != nil {
		os.Remove(tmp)
		return "", err
	}
	if err := os.Rename(tmp, bin); err != nil {
		os.Remove(tmp)
		return "", err
	}
	markToolInstalled("tailwindcss")
	return bin, nil
}

// removeTailwindPath removes the downloaded Tailwind binaries and returns the
// last path removed, or "" when none were present.
func removeTailwindPath() string {
	dir := irgoBinDir()
	entries, err := os.ReadDir(dir)
	if err != nil {
		return ""
	}
	last := ""
	for _, e := range entries {
		if strings.HasPrefix(e.Name(), "tailwindcss") {
			p := filepath.Join(dir, e.Name())
			if os.Remove(p) == nil {
				last = p
			}
		}
	}
	_ = os.Remove(dir) // only succeeds when empty
	clearToolMarker("tailwindcss")
	return last
}

// removeTailwind is the inverse, for uninstall-tools.
func removeTailwind() bool {
	dir := irgoBinDir()
	entries, err := os.ReadDir(dir)
	if err != nil {
		return false
	}
	removed := false
	for _, e := range entries {
		if len(e.Name()) >= 11 && e.Name()[:11] == "tailwindcss" {
			if os.Remove(filepath.Join(dir, e.Name())) == nil {
				fmt.Printf("  tailwindcss: removed (%s)\n", filepath.Join(dir, e.Name()))
				removed = true
			}
		}
	}
	_ = os.Remove(dir) // only succeeds when empty
	clearToolMarker("tailwindcss")
	return removed
}
