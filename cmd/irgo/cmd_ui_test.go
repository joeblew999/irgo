package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestSameSizeEditIsCopied — the assets are compared by content, not size.
//
// An upgrade that leaves a file the same length (a colour changed, a selector
// renamed) would be skipped by a size comparison, and the project would keep
// serving the old asset against new templ wrappers with nothing to show for
// it. Modification time is no better: the module cache stamps its files when
// it extracts them.
func TestSameSizeEditIsCopied(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "src.css")
	dst := filepath.Join(dir, "dst.css")

	if err := os.WriteFile(src, []byte(":root{--a:#111}"), 0o644); err != nil {
		t.Fatal(err)
	}
	// Same length, different content — the case a size check misses.
	if err := os.WriteFile(dst, []byte(":root{--a:#222}"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := copyMorpheusFile(src, dst); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(dst)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != ":root{--a:#111}" {
		t.Errorf("a same-size change was not copied: %q", got)
	}
}

// TestCopyIsWritableAfterwards — the module cache is read-only, and copying
// its mode through would give a project files the next build cannot overwrite.
func TestCopyIsWritableAfterwards(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "src.css")
	if err := os.WriteFile(src, []byte("x"), 0o444); err != nil {
		t.Fatal(err)
	}
	dst := filepath.Join(dir, "out", "dst.css")
	if err := copyMorpheusFile(src, dst); err != nil {
		t.Fatal(err)
	}
	fi, err := os.Stat(dst)
	if err != nil {
		t.Fatal(err)
	}
	if fi.Mode().Perm()&0o200 == 0 {
		t.Errorf("copy is %o — the next build could not rewrite it", fi.Mode().Perm())
	}
}

// TestOrphanedLinksAreReported — a layout that links assets nothing refreshes.
//
// `go mod tidy` keeps the module only while something imports it. Remove the
// last component from a page and the next tidy prunes it, but the layout still
// links three files. It takes two steps to bite: the first tidy keeps the
// module because the generated _templ.go still imports it, and only after a
// build regenerates that file does the next tidy drop it. By then the edit is
// in a different session and nothing connects the two.
func TestOrphanedLinksAreReported(t *testing.T) {
	inTempProject(t, map[string]string{
		"templates/layout.templ": `templ Layout() {
	<link rel="stylesheet" href="/static/css/morpheus/morpheus.css"/>
}`,
	})

	out := captureStdout(t, func() {
		if err := morpheusOrphanedLinks(); err != nil {
			t.Fatal(err)
		}
	})
	if !strings.Contains(out, "not a dependency") {
		t.Errorf("a layout linking Morpheus assets with no module said nothing:\n%s", out)
	}
	if !strings.Contains(out, "go get") {
		t.Errorf("the report does not say how to fix it:\n%s", out)
	}
}

// TestNoLinksNoNoise — a project that never used the kit must hear nothing. A
// note that appears for everyone is a note everyone learns to skip.
func TestNoLinksNoNoise(t *testing.T) {
	inTempProject(t, map[string]string{
		"templates/layout.templ": "templ Layout() {\n\t<title>x</title>\n}",
	})

	if out := captureStdout(t, func() { morpheusOrphanedLinks() }); out != "" {
		t.Errorf("said something to a project that does not use Morpheus:\n%s", out)
	}
}

func inTempProject(t *testing.T, files map[string]string) {
	t.Helper()
	dir := t.TempDir()
	for name, body := range files {
		p := filepath.Join(dir, name)
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	prev, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.Chdir(prev) })
}

// TestSecondHeadIsNotCoveredByTheFirst — a scaffolded project has two <head>
// blocks, Layout and FullscreenPage, and the home page uses the second.
//
// This is the case that made the earlier check worse than none: it asked
// whether *any* template mentioned the kit, so adding the links to Layout
// silenced it while the page that actually renders stayed unstyled. There is
// no error at that point — the markup is right, the assets are served, the
// component is simply inert — so the check going quiet is the whole failure.
func TestSecondHeadIsNotCoveredByTheFirst(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "templates"), 0o755); err != nil {
		t.Fatal(err)
	}
	layout := `package templates

templ Layout(title string) {
	<html>
		<head>
			<link rel="stylesheet" href="/static/css/morpheus/morpheus.css"/>
		</head>
		<body>{ children... }</body>
	</html>
}

templ FullscreenPage(title string) {
	<html>
		<head>
			<title>{ title }</title>
		</head>
		<body>{ children... }</body>
	</html>
}
`
	if err := os.WriteFile(filepath.Join(dir, "templates", "layout.templ"), []byte(layout), 0o644); err != nil {
		t.Fatal(err)
	}

	wd, _ := os.Getwd()
	defer os.Chdir(wd)
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}

	missing, err := headsWithoutMorpheus()
	if err != nil {
		t.Fatal(err)
	}
	if len(missing) != 1 {
		t.Fatalf("want exactly FullscreenPage reported, got %v", missing)
	}
	if !strings.HasSuffix(missing[0], ":FullscreenPage") {
		t.Errorf("reported the wrong block: %q", missing[0])
	}
}

// TestLinkedHeadsAreSilent — a project that has done the wiring must not be
// nagged on every build, or the note stops being read.
func TestLinkedHeadsAreSilent(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "templates"), 0o755); err != nil {
		t.Fatal(err)
	}
	layout := `package templates

templ Layout(title string) {
	<head>
		<link rel="stylesheet" href="/static/css/morpheus/morpheus.css"/>
		<script type="module" src="/static/js/morpheus.js"></script>
	</head>
}
`
	if err := os.WriteFile(filepath.Join(dir, "templates", "layout.templ"), []byte(layout), 0o644); err != nil {
		t.Fatal(err)
	}

	wd, _ := os.Getwd()
	defer os.Chdir(wd)
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}

	missing, err := headsWithoutMorpheus()
	if err != nil {
		t.Fatal(err)
	}
	if len(missing) != 0 {
		t.Errorf("nagged a project that is already wired up: %v", missing)
	}
}
