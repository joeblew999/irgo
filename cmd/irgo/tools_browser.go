// The browser irgo's browser tests run in.
//
// pkg/testing answers what a handler returned. It cannot see a component that
// never upgraded, a stylesheet that 404s, or a signal that never updated —
// every one of which renders correct HTML. Those need a browser.
//
// Which meant a manual install, so the tests skipped, so nobody ran them, so
// the failures they exist to catch went undiagnosed. That is the same trap as
// asking someone to install a JDK before an Android build: irgo downloads the
// JDK, and it downloads this.
//
// Playwright rather than a CDP library, deliberately. It drives webkit as well
// as chromium, and webkit is Safari's engine — which is the iOS WebView irgo
// ships to. Trading that away to save a few dependencies would mean the one
// target irgo cannot test is the one it cannot check any other way.
package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

// pinPlaywright is the driver version. It must match what pkg/browsertest
// imports: the driver and the library speak a versioned protocol, and a
// mismatch fails at run time with a message about neither.
const pinPlaywright = "v0.6100.0"

// playwrightPackage is the CLI that installs browsers.
const playwrightPackage = "github.com/mxschmitt/playwright-go/cmd/playwright"

// browserEngines are what `irgo tools install` provisions.
//
// Chromium only by default. Firefox and webkit are a few hundred megabytes
// each, and are worth fetching when there is a question they answer — a Safari
// rendering difference, say — rather than on every machine that runs a test.
var browserEngines = []string{"chromium"}

// ensureBrowsers installs the browsers Playwright needs, if they are missing.
//
// Idempotent and quiet when they are already there: Playwright checks its own
// cache, so this costs a process start on the common path.
func ensureBrowsers(engines ...string) error {
	if len(engines) == 0 {
		engines = browserEngines
	}
	if browsersInstalled(engines) {
		return nil
	}

	fmt.Printf("Installing %s for browser tests...\n", strings.Join(engines, ", "))

	args := []string{"run", playwrightPackage + "@" + pinPlaywright, "install"}
	// --with-deps installs the system libraries a browser needs. Linux CI
	// images have none of them; elsewhere it does nothing.
	if runtime.GOOS == "linux" {
		args = append(args, "--with-deps")
	}
	args = append(args, engines...)

	cmd := exec.Command(goBin(), args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("installing browsers: %w\n"+
			"  browser tests will be skipped until this succeeds", err)
	}
	return nil
}

// browsersInstalled reports whether Playwright's cache already holds them.
//
// Reading the cache rather than shelling out: `playwright install` is
// idempotent but starts a Node driver to find that out, and this runs before
// every test invocation.
func browsersInstalled(engines []string) bool {
	dir := playwrightCacheDir()
	if dir == "" {
		return false
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return false
	}
	for _, engine := range engines {
		found := false
		for _, e := range entries {
			// Directories are named chromium-1234, webkit-2345, and chromium
			// may install as chromium_headless_shell-1234.
			if strings.HasPrefix(e.Name(), engine+"-") ||
				strings.HasPrefix(e.Name(), engine+"_") {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	return true
}

// playwrightCacheDir is where Playwright keeps its browsers, per platform.
func playwrightCacheDir() string {
	if v := os.Getenv("PLAYWRIGHT_BROWSERS_PATH"); v != "" {
		return v
	}
	home := homeDir()
	if home == "" {
		return ""
	}
	switch runtime.GOOS {
	case "darwin":
		return filepath.Join(home, "Library", "Caches", "ms-playwright")
	case "windows":
		return filepath.Join(home, "AppData", "Local", "ms-playwright")
	default:
		return filepath.Join(home, ".cache", "ms-playwright")
	}
}

// hasBrowserTests reports whether this project has any, so a project without
// them never downloads a browser.
//
// By import rather than by filename: a test that imports the harness is a
// browser test whatever it is called, and one that does not is not.
func hasBrowserTests() bool {
	found := false
	filepath.WalkDir(".", func(path string, d os.DirEntry, err error) error {
		if err != nil || found {
			return nil
		}
		if d.IsDir() {
			// Not the root: its name is ".", which the dotfile rule below
			// matches — so skipping it walked nothing at all, and every
			// project looked as though it had no browser tests.
			if path == "." {
				return nil
			}
			// Vendored and generated trees are not this project's tests.
			if n := d.Name(); n == "vendor" || n == "node_modules" ||
				n == "build" || n == "tmp" || strings.HasPrefix(n, ".") {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(path, "_test.go") {
			return nil
		}
		body, err := os.ReadFile(path)
		if err == nil && strings.Contains(string(body), browsertestImport) {
			found = true
		}
		return nil
	})
	return found
}

const browsertestImport = "irgo/pkg/browsertest"

func init() {
	registerPreTestStep(func() error {
		if !hasBrowserTests() {
			return nil
		}
		if err := ensureBrowsers(); err != nil {
			// Not fatal: tests needing a browser skip without one, and failing
			// the run would stop the tests that do not need it.
			fmt.Printf("Note: %v\n", err)
		}
		return nil
	})
}

// chromiumDir is where Playwright put the browser, or "" if it is not there.
//
// A directory rather than a binary: the layout inside differs per platform,
// and doctor only needs to know whether the engine exists.
func chromiumDir() string {
	dir := playwrightCacheDir()
	if dir == "" {
		return ""
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return ""
	}
	for _, e := range entries {
		if strings.HasPrefix(e.Name(), "chromium") {
			return filepath.Join(dir, e.Name())
		}
	}
	return ""
}
