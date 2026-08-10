//go:build ignore

// Refreshes the showcase's assets from whichever Morpheus version go.mod
// resolves, and rewrites the embed list to match.
//
// Run after changing the dependency:
//
//	go get -u github.com/romshark/morpheus && go generate ./...
//
// Doing it by hand is how thirteen images went missing: the minified bundle
// was copied and the showcase's own static tree — avatars, icons, fonts, the
// favicon — was not, and the pages rendered well enough to look finished.
package main

import (
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

func main() {
	dir := moduleDir()
	if err := clearAssets("static"); err != nil {
		fail(err)
	}
	// Both trees: min/ holds the stylesheets and bundle the pages link, and
	// showcase/static holds everything else they reference.
	for _, src := range []string{
		filepath.Join(dir, "min"),
		filepath.Join(dir, "showcase", "static"),
	} {
		copyTree(src, "static", filepath.Base(src) == "min")
	}
	fmt.Println("static/ refreshed from", dir)
}

// clearAssets empties static/ of copied assets, so a file dropped upstream
// stops being served here rather than lingering forever.
//
// Everything except the two kinds of file that are not assets:
//
//   - embed.go, which embeds this very tree. Removing the whole directory took
//     it too, and the next build failed with the package missing rather than
//     an asset — a confusing way to be told a copy step ran.
//   - .gitkeep, which is what keeps a directory an embed pattern names but
//     upstream currently fills with nothing. Without it the pattern matches no
//     files and //go:embed is a compile error.
func clearAssets(root string) error {
	entries, err := os.ReadDir(root)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}
	for _, e := range entries {
		if !e.IsDir() && (strings.HasSuffix(e.Name(), ".go") || e.Name() == ".gitkeep") {
			continue
		}
		if e.IsDir() {
			if err := clearAssets(filepath.Join(root, e.Name())); err != nil {
				return err
			}
			continue
		}
		if err := os.Remove(filepath.Join(root, e.Name())); err != nil {
			return err
		}
	}
	return nil
}

func moduleDir() string {
	out, err := exec.Command("go", "list", "-m", "-f", "{{.Dir}}",
		"github.com/romshark/morpheus").Output()
	if err != nil {
		fail(fmt.Errorf("morpheus is not a dependency: %w", err))
	}
	return strings.TrimSpace(string(out))
}

// copyTree copies src into dst. min/ keeps its directory name because the
// pages link /static/min/…; the showcase's own static tree is flattened into
// static/ because they link /static/… directly.
func copyTree(src, dst string, keepDir bool) {
	base := dst
	if keepDir {
		base = filepath.Join(dst, filepath.Base(src))
	}
	err := filepath.WalkDir(src, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		rel, _ := filepath.Rel(src, path)
		out := filepath.Join(base, rel)
		if err := os.MkdirAll(filepath.Dir(out), 0o755); err != nil {
			return err
		}
		body, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		// 0644, not the source's mode: the module cache is read-only.
		return os.WriteFile(out, body, 0o644)
	})
	if err != nil {
		fail(err)
	}
}

func fail(err error) {
	fmt.Fprintln(os.Stderr, "sync:", err)
	os.Exit(1)
}
