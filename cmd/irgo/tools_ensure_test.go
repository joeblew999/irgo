package main

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// TestEveryToolCanBeObtainedOrSaysWhyNot — the whole ensure idea is that a
// call needing a tool gets one. A row with no path and no way to get it is a
// dead end someone hits mid-build with nothing to do about it.
//
// go is the exception on purpose: it compiles the CLI, so it is there before
// irgo runs at all. CI installs it the same way.
func TestEveryToolCanBeObtainedOrSaysWhyNot(t *testing.T) {
	provided := map[string]string{
		"go":         "compiles irgo itself, so it is a prerequisite",
		"git":        "ships with the platform's developer tools",
		"mise":       "optional; irgo uses what it finds",
		"xcrun":      "part of Xcode, which only Apple can install",
		"xcodebuild": "part of Xcode",
		"codesign":   "part of Xcode",
		"security":   "part of macOS",
		"clang":      "part of the Xcode command line tools",
		"powershell": "part of Windows",
	}
	for _, row := range toolLocators() {
		if row.ensure != nil {
			continue
		}
		if _, ok := provided[row.name]; !ok {
			t.Errorf("%s has no ensure function and no recorded reason — "+
				"either give it one, or record why irgo cannot provide it", row.name)
		}
	}
}

// TestInstallableToolsAreReportedByDoctor — install and doctor read the same
// table, so a tool cannot be installable without being visible.
func TestInstallableToolsAreReportedByDoctor(t *testing.T) {
	rows := toolLocators()
	if len(rows) < 15 {
		t.Fatalf("doctor reports %d tools — it used to be five, and the point was to cover them all", len(rows))
	}
	var installable int
	for _, r := range rows {
		if r.ensure != nil {
			installable++
		}
	}
	if installable == 0 {
		t.Fatal("no tool can be installed, which cannot be right")
	}
	t.Logf("%d tools reported, %d of them irgo can install", len(rows), installable)
}

// TestEveryToolSaysWhereItComesFrom — doctor exists so nobody has to read the
// source to learn how a tool arrives. A blank column sends them back to it.
func TestEveryToolSaysWhereItComesFrom(t *testing.T) {
	for _, row := range toolLocators() {
		if row.how == "" {
			t.Errorf("%s does not say where it comes from", row.name)
		}
	}
}

// TestRemovalCoversWhatIrgoInstalled — install and remove read the same table,
// so anything irgo put on the machine has to be offered back. The first
// version of this attached removal inside doctor's printer, so `tools remove`
// found nothing removable at all and would have left every download behind.
func TestRemovalCoversWhatIrgoInstalled(t *testing.T) {
	var removable int
	for _, row := range toolLocators() {
		if row.remove != nil {
			removable++
		}
		// A row irgo can install, that is present, and that lives somewhere
		// irgo owns, must be removable.
		if row.ensure != nil && row.path != "" &&
			strings.HasPrefix(row.path, irgoHome()) && row.remove == nil {
			t.Errorf("%s is in %s but nothing offers to remove it", row.name, irgoHome())
		}
	}
	if removable == 0 {
		t.Error("nothing at all is removable, which cannot be right on a machine irgo has built on")
	}
	t.Logf("%d tools removable", removable)
}

// TestABrokenShimIsNotReportedAsPresent — a version manager's shim is a real
// file that fails with "no version is set" when nothing selects one. doctor
// named the sops shim as a usable path, which is a build failure waiting in a
// column that claims everything is fine.
func TestABrokenShimIsNotReportedAsPresent(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("shell stub is POSIX")
	}
	dir := filepath.Join(t.TempDir(), "shims")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	broken := filepath.Join(dir, "sometool")
	if err := os.WriteFile(broken,
		[]byte("#!/bin/sh\necho 'mise ERROR No version is set for shim: sometool' >&2\nexit 1\n"),
		0o755); err != nil {
		t.Fatal(err)
	}
	if runs(broken) {
		t.Error("a shim with no version selected was reported as usable")
	}

	working := filepath.Join(dir, "goodtool")
	if err := os.WriteFile(working, []byte("#!/bin/sh\necho v1.2.3\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	if !runs(working) {
		t.Error("a working shim was rejected")
	}

	// Everything outside a shims directory is trusted: go, codesign and
	// security do not take --version, and testing them called three working
	// tools absent.
	if !runs("/usr/bin/codesign") {
		t.Error("a real binary that does not take --version was rejected")
	}
}

// TestVersionParsingRejectsJunk — "xcrun version 72." read as 72.
func TestVersionParsingRejectsJunk(t *testing.T) {
	for _, bad := range []string{"72.", "", "abc", "/usr/bin/x", "1", "2026-08-07"} {
		if versionShaped(bad) {
			t.Errorf("%q read as a version", bad)
		}
	}
	for _, good := range []string{"4.3.3", "22.14.0", "0.3.977", "1.0.41", "2026.8.1"} {
		if !versionShaped(good) {
			t.Errorf("%q did not read as a version", good)
		}
	}
}
