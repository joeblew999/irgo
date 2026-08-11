package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const plistStub = `<?xml version="1.0" encoding="UTF-8"?>
<plist version="1.0">
<dict>
	<key>CFBundleName</key>
	<string>Example</string>
</dict>
</plist>
`

func TestSetPlistLocalizations(t *testing.T) {
	write := func(t *testing.T, body string) string {
		t.Helper()
		p := filepath.Join(t.TempDir(), "Info.plist")
		if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
		return p
	}

	t.Run("declares every locale", func(t *testing.T) {
		p := write(t, plistStub)
		if err := setPlistLocalizations(p, []string{"de", "en", "pt-BR"}); err != nil {
			t.Fatal(err)
		}
		got, _ := os.ReadFile(p)
		for _, want := range []string{
			"<key>CFBundleLocalizations</key>",
			"<string>de</string>", "<string>en</string>", "<string>pt-BR</string>",
		} {
			if !strings.Contains(string(got), want) {
				t.Errorf("missing %q:\n%s", want, got)
			}
		}
		// Everything that was there must survive.
		if !strings.Contains(string(got), "<string>Example</string>") {
			t.Errorf("clobbered the rest of the plist:\n%s", got)
		}
	})

	// Runs on every build. Appending a second array is valid XML and undefined
	// behaviour in a plist, so this must replace rather than accumulate.
	t.Run("re-running replaces, never appends", func(t *testing.T) {
		p := write(t, plistStub)
		for i := 0; i < 3; i++ {
			if err := setPlistLocalizations(p, []string{"de", "en"}); err != nil {
				t.Fatal(err)
			}
		}
		got, _ := os.ReadFile(p)
		if n := strings.Count(string(got), "CFBundleLocalizations"); n != 1 {
			t.Errorf("found %d arrays, want 1:\n%s", n, got)
		}
	})

	t.Run("adding a language updates it", func(t *testing.T) {
		p := write(t, plistStub)
		_ = setPlistLocalizations(p, []string{"en"})
		_ = setPlistLocalizations(p, []string{"en", "fr"})
		got, _ := os.ReadFile(p)
		if !strings.Contains(string(got), "<string>fr</string>") {
			t.Errorf("fr not added:\n%s", got)
		}
		if n := strings.Count(string(got), "CFBundleLocalizations"); n != 1 {
			t.Errorf("found %d arrays, want 1", n)
		}
	})

	// Every build calls this. Rewriting an unchanged file would dirty the
	// working tree on each one and make `project upgrade --check` noisy.
	t.Run("unchanged means untouched", func(t *testing.T) {
		p := write(t, plistStub)
		_ = setPlistLocalizations(p, []string{"de"})
		before, _ := os.Stat(p)
		_ = setPlistLocalizations(p, []string{"de"})
		after, _ := os.Stat(p)
		if !before.ModTime().Equal(after.ModTime()) {
			t.Error("rewrote a file that had not changed")
		}
	})

	t.Run("a shell that does not exist is not an error", func(t *testing.T) {
		if err := setPlistLocalizations(filepath.Join(t.TempDir(), "nope.plist"), []string{"de"}); err != nil {
			t.Errorf("got %v, want nil", err)
		}
	})
}
