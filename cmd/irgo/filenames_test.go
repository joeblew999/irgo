package main

import (
	"os"
	"strings"
	"testing"
)

// TestNoFilenameEndsInGOOS guards a Go rule that fails silently.
//
// A file whose name ends in _<GOOS>.go is compiled only for that GOOS, so
// cmd_ios.go builds on nothing anyone runs and simply vanishes from the
// package — no error, no warning, just undefined symbols somewhere else. It
// has happened twice here: build_ios.go and build_android.go, then cmd_ios.go
// while renaming files to match the CLI.
//
// The naming convention puts the platform first (ios_build.go) partly for
// readability and mostly to avoid this. A comment did not prevent a repeat;
// a failing test does.
//
// Scoped to this package deliberately. Elsewhere the suffix is a feature —
// desktop/menu_darwin.go is meant to compile only on macOS. What protects
// platform-specific code outside the CLI is that every target is built in CI,
// so a file that vanished from one would fail to compile rather than fail to
// exist.
func TestNoFilenameEndsInGOOS(t *testing.T) {
	// The full set Go recognises, so a file named for a platform irgo does not
	// target today still fails rather than waiting to surprise someone.
	goos := []string{
		"aix", "android", "darwin", "dragonfly", "freebsd", "hurd", "illumos",
		"ios", "js", "linux", "nacl", "netbsd", "openbsd", "plan9", "solaris",
		"wasip1", "windows", "zos",
	}
	entries, err := os.ReadDir(".")
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range entries {
		name := e.Name()
		if e.IsDir() || !strings.HasSuffix(name, ".go") {
			continue
		}
		base := strings.TrimSuffix(name, ".go")
		base = strings.TrimSuffix(base, "_test")
		for _, os_ := range goos {
			if strings.HasSuffix(base, "_"+os_) {
				t.Errorf("%s is compiled only for GOOS=%s and is excluded from every "+
					"other build. Put the platform first instead: %s_<what>.go",
					name, os_, os_)
			}
		}
	}
}
