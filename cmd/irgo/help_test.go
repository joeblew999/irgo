package main

import (
	"io"
	"os"
	"strings"
	"testing"
)

// captureStdout runs f and returns what it printed.
func captureStdout(t *testing.T, f func()) string {
	t.Helper()
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	orig := os.Stdout
	os.Stdout = w
	defer func() { os.Stdout = orig }()

	f()
	w.Close()
	out, err := io.ReadAll(r)
	if err != nil {
		t.Fatal(err)
	}
	return string(out)
}

// TestEveryCommandHasHelp fails when a command exists with nothing to say about
// it. Help drifted from the CLI before precisely because nothing connected the
// two: commands were renamed and the help kept describing the old ones, down to
// files that no longer shipped.
func TestEveryCommandHasHelp(t *testing.T) {
	for noun, verbs := range nounVerbs {
		for _, verb := range verbs {
			out := captureStdout(t, func() { printCommandHelp(noun, verb) })
			if strings.Contains(out, "No help for") {
				t.Errorf("irgo %s %s has no help", noun, verb)
				continue
			}
			// The first line names the command, so a topic wired to the wrong
			// key is caught too rather than silently answering for its
			// neighbour.
			want := "irgo " + noun + " " + verb
			if first, _, _ := strings.Cut(out, "\n"); !strings.HasPrefix(first, want) {
				t.Errorf("irgo %s %s: help starts with %q, want it to start with %q",
					noun, verb, first, want)
			}
		}
	}
}

// TestHelpForANounListsItsVerbs checks the middle level: `irgo help app` should
// answer with app's verbs rather than falling through to the whole CLI.
func TestHelpForANounListsItsVerbs(t *testing.T) {
	for noun, verbs := range nounVerbs {
		out := captureStdout(t, func() { printCommandHelp(noun, "") })
		for _, verb := range verbs {
			if !strings.Contains(out, "irgo "+noun+" "+verb) {
				t.Errorf("irgo help %s does not mention %s", noun, verb)
			}
		}
	}
}
