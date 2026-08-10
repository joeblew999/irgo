package handlers_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"docs-templ/app"
	"docs-templ/clidoc"
	"docs-templ/content"

	irgotest "github.com/stukennedy/irgo/pkg/testing"
)

func client(t *testing.T) *irgotest.Client {
	t.Helper()
	return irgotest.NewClient(app.NewRouter().Handler())
}

func TestLandingPage(t *testing.T) {
	resp := client(t).Get("/")
	resp.AssertOK(t)
	resp.AssertContains(t, "Native Apps with")
	resp.AssertContains(t, "One Codebase. Every Platform.")
	resp.AssertContains(t, "arch-diagram") // the architecture SVG
	resp.AssertContains(t, "Platform Comparison")
}

func TestDocsIndex(t *testing.T) {
	resp := client(t).Get("/docs")
	resp.AssertOK(t)
	resp.AssertContains(t, "Getting started")
	resp.AssertContains(t, "CLI reference")
}

func TestDemoPage(t *testing.T) {
	resp := client(t).Get("/demo")
	resp.AssertOK(t)
	resp.AssertContains(t, "Server-Driven")
	resp.AssertContains(t, "DOM Morphing")
	// The counter is a real Datastar round trip, so the page must wire the
	// buttons to the handlers rather than to a script.
	resp.AssertContains(t, "@post('/api/demo/increment')")
	resp.AssertNotContains(t, "incrementCounter")
}

// TestDemoCounterIsReal exercises the handler the demo page advertises. The
// TSX version of this page faked the exchange in JavaScript; the point of the
// port is that it no longer does.
func TestDemoCounterIsReal(t *testing.T) {
	resp := client(t).Datastar().PostJSON("/api/demo/increment",
		`{"count": 41, "log": []}`)
	resp.AssertOK(t)
	resp.AssertSSEEvent(t, "datastar-patch-elements")
	resp.AssertContains(t, "42")
	resp.AssertContains(t, "POST /api/demo/increment")
}

// TestDemoCounterFloorsAtZero — the old page clamped, so decrementing from
// empty should visibly do nothing rather than go negative.
func TestDemoCounterFloorsAtZero(t *testing.T) {
	resp := client(t).Datastar().PostJSON("/api/demo/decrement",
		`{"count": 0, "log": []}`)
	resp.AssertOK(t)
	resp.AssertNotContains(t, "-1")
}

// TestLLMsTxtIsServed guards the link the Introduction page carries. It was a
// 404 for as long as the ported page linked to a file the site did not have.
func TestLLMsTxtIsServed(t *testing.T) {
	resp := client(t).Get("/llms.txt")
	resp.AssertOK(t)
	resp.AssertHeader(t, "Content-Type", "text/plain; charset=utf-8")
	resp.AssertContains(t, "Irgo Framework")
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

// TestProsePagesRenderTheirStructure checks the renderer against the data.
//
// content's tests prove a page holds its lists, tables and callouts; this
// proves the renderer emits them. The first port had neither, so every list
// and table in the source became nothing on the page and no test noticed —
// the pages still returned 200 and still read like sentences.
func TestProsePagesRenderTheirStructure(t *testing.T) {
	c := client(t)
	for _, p := range content.Pages {
		resp := c.Get("/docs/" + p.Slug)
		resp.AssertOK(t)
		body := string(resp.Body)

		want := map[string]int{}
		for _, b := range p.Blocks {
			switch b.Kind {
			case content.KindList:
				if b.Ordered {
					want["<ol>"]++
				} else {
					want["<ul>"]++
				}
			case content.KindTable:
				want["comparison-table"]++
			case content.KindNote:
				want[`<aside class="note`]++
			case content.KindCode:
				if b.File != "" {
					want["<figcaption>"]++
				}
			}
		}
		for frag, n := range want {
			if got := strings.Count(body, frag); got != n {
				t.Errorf("%s: %d %q on the page, %d in the content", p.Slug, got, frag, n)
			}
		}
	}
}

// TestNoPortingArtifactReachesThePage is the same guard as content's, one
// level down: whatever the data holds, the reader must not be shown JSX
// spacing or a JavaScript-escaped backtick.
func TestNoPortingArtifactReachesThePage(t *testing.T) {
	c := client(t)
	for _, p := range content.Pages {
		body := string(c.Get("/docs/" + p.Slug).Body)
		if strings.Contains(body, `{" "}`) {
			t.Errorf("%s: leftover JSX spacing reached the page", p.Slug)
		}
		if strings.Contains(body, "\\`") {
			t.Errorf("%s: JS-escaped backtick reached the page", p.Slug)
		}
	}
}

// TestNoTemplateVendorsDatastar — the framework serves the client at
// /_irgo/datastar.js, matched to the Go library it talks to. A project that
// still points at its own vendored copy is running whatever it downloaded the
// day it was scaffolded, and `irgo project upgrade` deletes that file, so the
// stale tag becomes a 404 rather than an old client.
func TestNoTemplateVendorsDatastar(t *testing.T) {
	files, err := filepath.Glob("../templates/*.templ")
	if err != nil || len(files) == 0 {
		t.Fatalf("no templates found: %v", err)
	}
	for _, f := range files {
		data, err := os.ReadFile(f)
		if err != nil {
			t.Fatal(err)
		}
		if strings.Contains(string(data), "/static/js/datastar.js") {
			t.Errorf("%s loads a vendored Datastar; use /_irgo/datastar.js", filepath.Base(f))
		}
	}
}

// TestDatastarIsServed — the tag has to resolve. It pointed at a file the
// upgrade had just deleted, and nothing noticed, because a missing script is
// a page that renders and does nothing.
func TestDatastarIsServed(t *testing.T) {
	resp := client(t).Get("/_irgo/datastar.js")
	resp.AssertOK(t)
	if len(resp.Body) < 1000 {
		t.Errorf("the Datastar client is %d bytes — that is not the bundle", len(resp.Body))
	}
	if strings.Contains(string(resp.Body), "sourceMappingURL") {
		t.Error("the served client references a source map that is not shipped")
	}
}
