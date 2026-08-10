package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestHyphenatedDatastarIsFound — the mistake this exists for.
//
// `data-attr-class` renders, and binds nothing. A handler test passes, the
// console is empty, and the only symptom is a class that never appears.
func TestHyphenatedDatastarIsFound(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "templates"), 0o755); err != nil {
		t.Fatal(err)
	}
	body := `templ Page() {
	<main data-signals-theme="'default'">
		<div data-attr-class="'theme-' + $theme"></div>
	</main>
}
`
	if err := os.WriteFile(filepath.Join(dir, "templates", "p.templ"), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}

	wd, _ := os.Getwd()
	defer os.Chdir(wd)
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}

	out := captureStdout(t, func() { _ = checkDatastarSyntax() })
	for _, want := range []string{"data-attr-class", "data-attr:class",
		"data-signals-theme", "data-signals:theme"} {
		if !strings.Contains(out, want) {
			t.Errorf("did not report %q:\n%s", want, out)
		}
	}
}

// TestCorrectDatastarIsSilent — a project that got it right must not be
// nagged, or the note stops being read.
func TestCorrectDatastarIsSilent(t *testing.T) {
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, "templates"), 0o755)
	body := `templ Page() {
	<main data-signals:theme="'default'" data-text="$theme">
		<div data-attr:class="'theme-' + $theme" data-on:click="$theme = 'x'"></div>
		<div data-on-load="@get('/x')" data-show="$open"></div>
	</main>
}
`
	os.WriteFile(filepath.Join(dir, "templates", "p.templ"), []byte(body), 0o644)

	wd, _ := os.Getwd()
	defer os.Chdir(wd)
	os.Chdir(dir)

	if out := captureStdout(t, func() { _ = checkDatastarSyntax() }); out != "" {
		t.Errorf("nagged a project that is correct:\n%s", out)
	}
}
