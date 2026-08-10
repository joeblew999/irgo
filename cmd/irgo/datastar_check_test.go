package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestDatastarCheckSuggestsTheRightColon pins the case the first version got
// wrong: an argument with its own hyphen.
//
// The separator is the FIRST hyphen after the plugin name, not the last, so
// data-attr-aria-label wants data-attr:aria-label. Deriving the suggestion by
// splitting on the last hyphen produced data-attr-aria:label — advice that is
// as broken as the attribute it was correcting, and confidently printed.
func TestDatastarCheckSuggestsTheRightColon(t *testing.T) {
	for _, tc := range []struct {
		name, attr, want string
		flagged          bool
	}{
		{"plain argument", `data-attr-class="x"`, "data-attr:class", true},
		{"hyphenated argument", `data-attr-aria-label="x"`, "data-attr:aria-label", true},
		{"deeply hyphenated", `data-bind-foo-bar-baz="x"`, "data-bind:foo-bar-baz", true},
		{"data-on-load is real", `data-on-load="x"`, "", false},
		{"data-on-interval is real", `data-on-interval="x"`, "", false},
		{"correct colon form", `data-attr:class="x"`, "", false},
		{"data-text takes no argument", `data-text="$x"`, "", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			if err := os.MkdirAll(filepath.Join(dir, "templates"), 0o755); err != nil {
				t.Fatal(err)
			}
			body := "templ X() {\n\t<div " + tc.attr + "></div>\n}\n"
			if err := os.WriteFile(filepath.Join(dir, "templates", "x.templ"), []byte(body), 0o644); err != nil {
				t.Fatal(err)
			}

			out := inDir(t, dir, func() { _ = checkDatastarSyntax() })

			if !tc.flagged {
				if strings.Contains(out, "->") {
					t.Errorf("%s should not be flagged, got:\n%s", tc.attr, out)
				}
				return
			}
			if !strings.Contains(out, tc.want) {
				t.Errorf("%s: want suggestion %q, got:\n%s", tc.attr, tc.want, out)
			}
		})
	}
}

// inDir runs fn with the working directory set to dir and stdout captured,
// because checkDatastarSyntax reads ./templates and reports by printing.
func inDir(t *testing.T, dir string, fn func()) string {
	t.Helper()

	was, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := os.Chdir(was); err != nil {
			t.Fatal(err)
		}
	}()

	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	stdout := os.Stdout
	os.Stdout = w

	done := make(chan string)
	go func() {
		var sb strings.Builder
		buf := make([]byte, 4096)
		for {
			n, err := r.Read(buf)
			sb.Write(buf[:n])
			if err != nil {
				break
			}
		}
		done <- sb.String()
	}()

	fn()

	w.Close()
	os.Stdout = stdout
	return <-done
}
