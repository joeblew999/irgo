package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestForkPinSurvivesLocalPin — `pin local` must not touch a replace that
// names a module and version.
//
// This repository's own demo pinned a fork:
//
//	replace github.com/stukennedy/irgo => github.com/joeblew999/irgo v0.4.0-...
//
// An earlier `pin local` overwrote it with an absolute path, and switching
// back produced a project pinned to the published upstream instead of the fork
// it had deliberately chosen — a silent change of which framework it builds
// against, visible only as a go.mod diff nobody was looking for.
func TestForkPinSurvivesLocalPin(t *testing.T) {
	dir := t.TempDir()
	gomod := "module demo\n\ngo 1.24\n\n" +
		"replace " + upstreamModule + " => github.com/someone/irgo v0.4.0-fork\n"
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte(gomod), 0o644); err != nil {
		t.Fatal(err)
	}

	wd, _ := os.Getwd()
	defer os.Chdir(wd)
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}

	if err := dropCommittedLocalReplace(); err != nil {
		t.Fatal(err)
	}

	after, err := os.ReadFile("go.mod")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(after), "github.com/someone/irgo v0.4.0-fork") {
		t.Errorf("the fork pin was removed — a local pin must not change which "+
			"module the project builds against.\ngo.mod is now:\n%s", after)
	}
}

// TestLocalPathReplaceIsRemoved — the converse: a local-path replace written
// by an older irgo is machine-specific and must go, since it is exactly what
// breaks a teammate's clone and CI.
func TestLocalPathReplaceIsRemoved(t *testing.T) {
	dir := t.TempDir()
	gomod := "module demo\n\ngo 1.24\n\n" +
		"replace " + upstreamModule + " => /Users/someone/checkout\n"
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte(gomod), 0o644); err != nil {
		t.Fatal(err)
	}

	wd, _ := os.Getwd()
	defer os.Chdir(wd)
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}

	if err := dropCommittedLocalReplace(); err != nil {
		t.Fatal(err)
	}

	after, _ := os.ReadFile("go.mod")
	if strings.Contains(string(after), "/Users/someone/checkout") {
		t.Errorf("a machine-specific replace survived in go.mod:\n%s", after)
	}
}

// TestReleaseKeepsAForkPin — "release" means stop using your local checkout,
// not abandon the fork the project depends on.
//
// A project pinned to a fork and then told `pin release` would have had its
// replace removed and been dropped back to upstream, which has none of what it
// depends on. The only symptom is code that stops compiling, and nothing in
// the message would say why.
func TestReleaseKeepsAForkPin(t *testing.T) {
	dir := t.TempDir()
	gomod := "module demo\n\ngo 1.24\n\n" +
		"replace " + upstreamModule + " => github.com/someone/irgo v0.6.6\n"
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte(gomod), 0o644); err != nil {
		t.Fatal(err)
	}

	wd, _ := os.Getwd()
	defer os.Chdir(wd)
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}

	if got := forkPin(); got != "github.com/someone/irgo v0.6.6" {
		t.Fatalf("forkPin did not see the fork: %q", got)
	}
}

// TestLocalPathIsNotAForkPin — a checkout is what release is meant to undo.
func TestLocalPathIsNotAForkPin(t *testing.T) {
	dir := t.TempDir()
	gomod := "module demo\n\ngo 1.24\n\n" +
		"replace " + upstreamModule + " => /Users/someone/checkout\n"
	os.WriteFile(filepath.Join(dir, "go.mod"), []byte(gomod), 0o644)

	wd, _ := os.Getwd()
	defer os.Chdir(wd)
	os.Chdir(dir)

	if got := forkPin(); got != "" {
		t.Errorf("treated a local checkout as a fork pin: %q", got)
	}
}
