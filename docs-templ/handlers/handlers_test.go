package handlers_test

import (
	"strings"
	"testing"

	"docs-templ/app"
	"docs-templ/clidoc"

	irgotest "github.com/stukennedy/irgo/pkg/testing"
)

func client(t *testing.T) *irgotest.Client {
	t.Helper()
	return irgotest.NewClient(app.NewRouter().Handler())
}

func TestHomePage(t *testing.T) {
	resp := client(t).Get("/")
	resp.AssertOK(t)
	resp.AssertContains(t, "Getting started")
	resp.AssertContains(t, "CLI reference")
}

func TestGettingStarted(t *testing.T) {
	resp := client(t).Get("/docs/getting-started")
	resp.AssertOK(t)
	resp.AssertContains(t, "irgo project new myapp")
}

// TestCLIPageRendersEveryCommand is the page's whole justification: it is
// generated, so every command the binary declares has to appear on it. A page
// that renders but silently drops half the CLI would look fine.
func TestCLIPageRendersEveryCommand(t *testing.T) {
	resp := client(t).Get("/docs/cli")
	resp.AssertOK(t)

	cmds := clidoc.Commands()
	if len(cmds) == 0 {
		t.Fatal("no commands embedded — cli.json is empty or unparsed")
	}
	for _, c := range cmds {
		resp.AssertContains(t, "irgo "+c.Name())
	}
	t.Logf("%d commands rendered", len(cmds))
}

// TestCLIPageShowsFlags — the flags are the part the old hand-written site got
// most wrong, so check the page carries them rather than only the names.
func TestCLIPageShowsFlags(t *testing.T) {
	resp := client(t).Get("/docs/cli")
	resp.AssertOK(t)

	var withFlags, shown int
	for _, c := range clidoc.Commands() {
		for _, f := range c.Flags {
			withFlags++
			spec, _, _ := strings.Cut(f.Spec, " ")
			spec = strings.TrimSuffix(spec, ",")
			if strings.Contains(string(resp.Body), spec) {
				shown++
			}
		}
	}
	if withFlags == 0 {
		t.Fatal("no flags declared — the CLI dump looks wrong")
	}
	if shown != withFlags {
		t.Errorf("%d of %d declared flags reach the page", shown, withFlags)
	}
}
