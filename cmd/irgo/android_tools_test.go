package main

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestAndroidHome(t *testing.T) {
	t.Setenv("ANDROID_HOME", "/custom/sdk")
	if got := androidHome(); got != "/custom/sdk" {
		t.Fatalf("expected ANDROID_HOME to win, got %q", got)
	}
	t.Setenv("ANDROID_HOME", "")
	got := androidHome()
	if got == "" || got == "/custom/sdk" {
		t.Fatalf("expected a per-platform default, got %q", got)
	}
	if runtime.GOOS == "darwin" && got != filepath.Join(homeDir(), "Library", "Android", "sdk") {
		t.Fatalf("unexpected darwin default: %q", got)
	}
}

// goBin must fall back to $GOROOT/bin/go when PATH cannot resolve "go" — the
// Windows git-bash case where the native Path var is stripped.
func TestGoBinGOROOTFallback(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("PATH/exec semantics differ on Windows")
	}
	gr := t.TempDir()
	bin := filepath.Join(gr, "bin")
	if err := os.MkdirAll(bin, 0o755); err != nil {
		t.Fatal(err)
	}
	goPath := filepath.Join(bin, "go")
	if err := os.WriteFile(goPath, []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", t.TempDir()) // empty dir: LookPath("go") fails
	t.Setenv("GOROOT", gr)
	if got := goBin(); got != goPath {
		t.Fatalf("expected GOROOT fallback %s, got %q", goPath, got)
	}
}

func TestTemplVersionFromGoMod(t *testing.T) {
	dir := t.TempDir()

	t.Run("single-line require", func(t *testing.T) {
		t.Chdir(dir)
		if err := os.WriteFile(filepath.Join(dir, "go.mod"),
			[]byte("module m\n\ngo 1.24\n\nrequire github.com/a-h/templ v0.3.977\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		if v := templVersionFromGoMod(); v != "0.3.977" {
			t.Fatalf("expected 0.3.977, got %q", v)
		}
	})

	t.Run("block require", func(t *testing.T) {
		t.Chdir(dir)
		content := "module m\n\ngo 1.24\n\nrequire (\n\tgithub.com/a-h/templ v0.4.0\n\tgithub.com/other v1.0.0\n)\n"
		if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
		if v := templVersionFromGoMod(); v != "0.4.0" {
			t.Fatalf("expected 0.4.0, got %q", v)
		}
	})

	t.Run("no templ", func(t *testing.T) {
		t.Chdir(dir)
		if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module m\n\ngo 1.24\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		if v := templVersionFromGoMod(); v != "" {
			t.Fatalf("expected empty, got %q", v)
		}
	})
}

func TestAvdHomeDir(t *testing.T) {
	t.Setenv("ANDROID_AVD_HOME", "/custom/avd")
	if got := avdHomeDir(); got != "/custom/avd" {
		t.Fatalf("expected ANDROID_AVD_HOME to win, got %q", got)
	}
	t.Setenv("ANDROID_AVD_HOME", "")
	got := avdHomeDir()
	if got == "" || got == "/custom/avd" {
		t.Fatalf("expected a default, got %q", got)
	}
	if got != filepath.Join(homeDir(), ".android", "avd") {
		t.Fatalf("unexpected default: %q", got)
	}
}
