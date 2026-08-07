package main

import (
	"io"
	"os"
	"regexp"
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

// TestUsageIndexListsEveryCommand keeps the front page honest.
//
// The index is prose while the commands come from a table, so it drifts: `app
// deploy cloudflare` shipped working and undocumented, and `app build` went on
// listing targets that no longer matched what it accepted. The detail pages
// were right both times — it was the first screen, which is the only one most
// people read, that was wrong.
func TestUsageIndexListsEveryCommand(t *testing.T) {
	out := captureStdout(t, printUsage)
	for noun, verbs := range nounVerbs {
		for _, verb := range verbs {
			if !strings.Contains(out, noun+" "+verb) {
				t.Errorf("irgo help does not mention %q on its front page", noun+" "+verb)
			}
		}
	}
}

// TestEveryDeclaredBuildTargetIsDispatched checks the list against the code
// that has to honour it.
//
// Comparing the list to the help would prove nothing — the help is rendered
// from the list, so the two agree by construction and the test passes for a
// target that does not exist. What can actually be wrong is the list promising
// a target runBuild has no case for, which fails at run time with "unknown
// target" after the help said it was supported.
func TestEveryDeclaredBuildTargetIsDispatched(t *testing.T) {
	// Two files and two shapes, because desktop is dispatched by an if in
	// runAppBuild while the rest are cases in runBuild. That inconsistency is
	// why this reads the source rather than the switch: there is no single
	// switch to read.
	var body string
	for _, f := range []string{"cmd_app_build.go", "cmd_app.go"} {
		src, err := os.ReadFile(f)
		if err != nil {
			t.Fatal(err)
		}
		body += string(src)
	}
	for _, target := range buildTargets {
		if strings.Contains(body, `case "`+target+`"`) || strings.Contains(body, `== "`+target+`"`) {
			continue
		}
		t.Errorf("buildTargets promises %q but nothing dispatches it — "+
			"the help would advertise a target that fails at run time", target)
	}
}

// TestEveryVerbIsDeclared — the index, the detail page and the generated
// README all read commands.go, so a verb missing from it renders as three
// blanks and errors in none of them.
func TestEveryVerbIsDeclared(t *testing.T) {
	for noun, verbs := range nounVerbs {
		for _, verb := range verbs {
			c, ok := commands[noun+" "+verb]
			if !ok {
				t.Errorf("%s %s is dispatched but has no entry in commands.go — "+
					"no help, no index line, no README row", noun, verb)
				continue
			}
			if c.summary == "" {
				t.Errorf("%s %s has no summary — it renders as a blank row", noun, verb)
			}
			if len(c.usage) == 0 {
				t.Errorf("%s %s has no usage lines", noun, verb)
			}
		}
	}
}

// TestEveryFlagIsDocumented finds flags nobody can discover.
//
// An audit once turned up nine, including every macOS and Windows signing
// override — the way CI supplies secrets without writing them to disk. Being
// undocumented made the documented path look like the only one.
func TestEveryFlagIsDocumented(t *testing.T) {
	// Flags irgo passes to other programs rather than accepts itself.
	passedThrough := map[string]bool{
		"--minify": true, // tailwind
		"--depth":  true, // git clone
		"--exists": true, // pkg-config
		"--yes":    true, // npx
		"--help":   true, // handled before dispatch
	}

	help := captureStdout(t, printUsage)
	for noun, verbs := range nounVerbs {
		help += captureStdout(t, func() { printCommandHelp(noun, "") })
		for _, verb := range verbs {
			help += captureStdout(t, func() { printCommandHelp(noun, verb) })
		}
	}

	entries, err := os.ReadDir(".")
	if err != nil {
		t.Fatal(err)
	}
	flag := regexp.MustCompile(`"(--[a-z][a-z-]+)"`)
	seen := map[string]bool{}
	for _, e := range entries {
		name := e.Name()
		if !strings.HasPrefix(name, "cmd_") || !strings.HasSuffix(name, ".go") ||
			strings.HasSuffix(name, "_test.go") {
			continue
		}
		src, err := os.ReadFile(name)
		if err != nil {
			t.Fatal(err)
		}
		for _, m := range flag.FindAllStringSubmatch(string(src), -1) {
			f := m[1]
			if passedThrough[f] || seen[f] {
				continue
			}
			seen[f] = true
			if !strings.Contains(help, f) {
				t.Errorf("%s accepts %s and no help mentions it", name, f)
			}
		}
	}
}

// TestDeclaredFlagsExist is the other direction: help.go can now promise a
// flag the CLI never reads, which is worse than an undocumented one — someone
// passes it, nothing happens, and nothing says so.
func TestDeclaredFlagsExist(t *testing.T) {
	var src string
	entries, err := os.ReadDir(".")
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range entries {
		n := e.Name()
		if strings.HasSuffix(n, ".go") && !strings.HasSuffix(n, "_test.go") && n != "commands.go" {
			b, err := os.ReadFile(n)
			if err != nil {
				t.Fatal(err)
			}
			src += string(b)
		}
	}
	name := regexp.MustCompile(`^(--[a-z][a-z-]*)`)
	for key, c := range commands {
		for _, f := range c.flags {
			m := name.FindStringSubmatch(f[0])
			if m == nil {
				t.Errorf("%s: flag spec %q does not start with a --flag", key, f[0])
				continue
			}
			if !strings.Contains(src, `"`+m[1]+`"`) {
				t.Errorf("%s documents %s but no code reads it — it would be accepted and ignored",
					key, m[1])
			}
		}
	}
}
