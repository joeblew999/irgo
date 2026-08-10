// Conflict markers must never be committed.
//
// The workflow here merges constantly — every branch is cut from main and
// lands on integration, and the union conflicts that produces are routine and
// mostly trivial to resolve. Which is the problem: a resolution that is
// obviously correct gets committed without a second look.
//
// It happened while adding the i18n docs. A stash popped with a conflict, the
// markers went into the file, the file was committed, and it was pushed. Every
// test passed, because a Markdown file containing `<<<<<<< Updated upstream`
// is still perfectly valid Markdown. It was found by reading the file for an
// unrelated reason.
//
// Go source would at least fail to build. Markdown, YAML, JSON, .arb catalogs
// and Gradle files all accept them in silence, and those are most of what a
// merge touches.
package main

import (
	"bytes"
	"os"
	"os/exec"
	"strings"
	"testing"
)

// A file must have BOTH the opening and the closing marker at the start of a
// line before this calls it conflicted.
//
// The separator git writes between the two halves is a bare `=======`, and
// that is also how Markdown underlines a heading:
//
//	Heading
//	=======
//
// Matching it alone failed on ordinary documentation. Matching either angle
// marker alone failed on prose describing a conflict — including this file.
// Requiring the pair is what distinguishes a real conflict from writing about
// one, because git always writes all three or none.
var (
	conflictOpen  = []byte("<<<<<<< ")
	conflictClose = []byte(">>>>>>> ")
)

// atLineStart reports whether marker begins a line anywhere in body.
func atLineStart(body, marker []byte) bool {
	return bytes.HasPrefix(body, marker) ||
		bytes.Contains(body, append([]byte("\n"), marker...))
}

func TestNoConflictMarkersAreTracked(t *testing.T) {
	// Tracked files only. The working tree holds build output, module caches
	// and whatever a developer left lying about, none of which is this test's
	// business — and a vendored fixture could legitimately contain markers.
	out, err := exec.Command("git", "-C", "../..", "ls-files", "-z").Output()
	if err != nil {
		t.Skipf("git unavailable: %v", err)
	}

	var bad []string
	for _, name := range strings.Split(string(out), "\x00") {
		if name == "" || strings.HasSuffix(name, "conflict_test.go") {
			continue // this file describes the markers, so it contains them
		}
		// Read through os rather than grep: the go test cache tracks files the
		// test opens, so a marker committed later actually re-runs this instead
		// of replaying a cached pass. It is also one less tool to depend on,
		// and git grep's regexp dialect differs between platforms.
		body, err := os.ReadFile("../../" + name)
		if err != nil {
			continue // deleted but still indexed, or a submodule
		}
		if atLineStart(body, conflictOpen) && atLineStart(body, conflictClose) {
			bad = append(bad, name)
		}
	}

	if len(bad) > 0 {
		t.Errorf("these tracked files contain merge conflict markers:\n  %s\n\n"+
			"A conflict was resolved by committing the conflict. Nothing else "+
			"catches this:\nMarkdown, YAML and JSON all accept them in silence.",
			strings.Join(bad, "\n  "))
	}
}
