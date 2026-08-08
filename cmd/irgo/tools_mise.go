// mise, asked first when irgo has to install something.
//
// irgo downloaded 176 MB of Node and 309 MB of JDK onto a machine whose mise
// already had both. That is the disruptive way round: a second copy of a
// toolchain, in a directory only irgo knows about, that the developer's own
// version manager cannot see or clean up.
//
// So when a tool is missing and mise can provide it, mise does. irgo's own
// downloader stays as the fallback, because mise cannot reach everything —
// templ and gomobile are `go install`, tailwindcss is not in the registry, and
// the Android NDK needs the exact pin irgo holds rather than whatever a plugin
// resolves to.
//
// Two rules:
//
//   - Never install a second mise. A host that has one keeps it — PATH, then
//     mise's own install location, then irgo's, before anything is fetched.
//     When a tool's install path goes through mise and the host has none, one
//     static binary is downloaded: at that point mise is not optional, it is
//     what stands between the developer and a working build.
//   - Never touch the user's config. `mise use -g` rewrites config.toml.
//     `mise install` plus `mise where` gets the same binary and leaves it alone.
package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

// pinMise is bootstrapped only when a tool needs mise and the host has none.
// irgo never upgrades a mise it did not install.
const pinMise = "v2026.8.2"

// ensureMise finds mise, or installs one.
//
// Called when a tool's install path goes through mise: at that point mise is
// not optional, it is the thing standing between the developer and a working
// build. Downloading one static binary is a smaller imposition than failing.
//
// A host that already has mise keeps it — three checks before anything is
// fetched, so nobody ends up with two.
func ensureMise() (string, error) {
	if p, ok := miseCmd(); ok {
		return p, nil
	}
	bin := filepath.Join(irgoBinDir(), "mise")
	if runtime.GOOS == "windows" {
		bin += ".exe"
	}
	asset, err := miseAsset()
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(filepath.Dir(bin), 0o755); err != nil {
		return "", err
	}
	fmt.Printf("mise not found — downloading %s...\n", pinMise)

	// To .part then rename, so an interrupted download cannot leave something
	// that later looks installed. Same as Tailwind's.
	tmp := bin + ".part"
	if err := downloadFile("https://github.com/jdx/mise/releases/download/"+pinMise+"/"+asset, tmp); err != nil {
		os.Remove(tmp)
		return "", fmt.Errorf("downloading mise: %w", err)
	}
	if err := os.Chmod(tmp, 0o755); err != nil {
		os.Remove(tmp)
		return "", err
	}
	if err := os.Rename(tmp, bin); err != nil {
		os.Remove(tmp)
		return "", err
	}
	markToolInstalled("mise")
	return bin, nil
}

// miseAsset maps the host to a release asset. mise publishes bare binaries
// beside the archives, so there is nothing to extract.
func miseAsset() (string, error) {
	var host, arch string
	switch runtime.GOOS {
	case "darwin":
		host = "macos"
	case "linux":
		host = "linux"
	case "windows":
		return "mise-" + pinMise + "-windows-x64.exe", nil
	default:
		return "", fmt.Errorf("no mise build for %s", runtime.GOOS)
	}
	switch runtime.GOARCH {
	case "arm64":
		arch = "arm64"
	case "amd64":
		arch = "x64"
	default:
		return "", fmt.Errorf("no mise build for %s/%s", runtime.GOOS, runtime.GOARCH)
	}
	return fmt.Sprintf("mise-%s-%s-%s", pinMise, host, arch), nil
}

// miseCmd returns mise if this machine already has it, without installing.
func miseCmd() (string, bool) {
	if p, err := exec.LookPath("mise"); err == nil {
		return p, true
	}
	// mise's own installer puts it here, and it is not always on PATH.
	p := filepath.Join(homeDir(), ".local", "bin", "mise")
	if pathExists(p) {
		return p, true
	}
	return "", false
}

// miseTool returns a tool's binary from mise, installing it if mise can.
//
// spec is what mise is asked for, with irgo's pin in it — "node@22.14.0",
// "aqua:tailwindlabs/tailwindcss@4.3.3". Returns "" when mise cannot provide
// it, which is the caller's signal to fall back to irgo's own download.
func miseTool(spec, binary string) string {
	// Installing, not looking: mise is required here, so fetch one if needed.
	mise, err := ensureMise()
	if err != nil {
		return ""
	}
	// Already installed: no network, no output, just a path.
	if p := miseWhere(mise, spec, binary); p != "" {
		return p
	}
	if err := exec.Command(mise, "install", spec).Run(); err != nil {
		return ""
	}
	return miseWhere(mise, spec, binary)
}

// miseWhere asks where a version lives and finds the binary inside it.
//
// `mise where` reports an install directory without consulting or changing
// which version is active, so this works whether or not the tool is in the
// developer's config.
//
// The binary is searched for rather than assumed. Layouts differ — node uses
// bin/, an aqua tool sits at the root, some nest a directory deep — and
// hardcoding bin/ meant tailwindcss silently fell through to irgo's own
// download while mise had it all along.
func miseWhere(mise, spec, binary string) string {
	out, err := exec.Command(mise, "where", spec).Output()
	if err != nil {
		return ""
	}
	dir := strings.TrimSpace(string(out))
	if dir == "" {
		return ""
	}
	if runtime.GOOS == "windows" && !strings.HasSuffix(binary, ".exe") {
		binary += ".exe"
	}
	for _, p := range []string{
		filepath.Join(dir, binary),
		filepath.Join(dir, "bin", binary),
	} {
		if pathExists(p) {
			return p
		}
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return ""
	}
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		for _, p := range []string{
			filepath.Join(dir, e.Name(), binary),
			filepath.Join(dir, e.Name(), "bin", binary),
		} {
			if pathExists(p) {
				return p
			}
		}
	}
	return ""
}

// installedViaMise reports whether the exact version irgo pins is installed.
//
// No bookkeeping: the pin is the identity. irgo only ever asks for
// node@22.14.0, so that is the only node it can remove, and irgo's own source
// says which one that is. A file recording "irgo installed this" only added
// the distinction between installing a version and finding it already there —
// and tools remove prints what it will delete and refuses without
// confirmation, so a person sees node@22.14.0 before anything happens.
func installedViaMise(spec string) bool {
	mise, ok := miseCmd()
	if !ok {
		return false
	}
	return exec.Command(mise, "where", spec).Run() == nil
}
