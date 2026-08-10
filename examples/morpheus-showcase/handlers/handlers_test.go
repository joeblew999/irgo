package handlers_test

import (
	"strings"
	"testing"

	site "github.com/romshark/morpheus/showcase"
	irgotest "github.com/stukennedy/irgo/pkg/testing"

	"morpheus-showcase/app"
)

// What this example claims is that Morpheus's own showcase runs on irgo
// unchanged, and that it does not fall behind the version it is built against.
// Both are claims about every page, not about a page — so the tests enumerate
// rather than sample.
//
// The scaffolded tests that were here asserted a Datastar endpoint at
// /api/init and the word "Irgo" on the home page. Neither exists here: this
// serves someone else's site. They failed, which is the correct outcome for a
// test kept past the thing it described.

// TestEveryPageServes — the showcase enumerates itself, so this asks the same
// list the router was built from whether each entry actually renders.
//
// This is the failure this example exists to catch. A page added upstream
// arrives on the next `go get -u` with no code change here, which is the point
// — and also means nothing has looked at it. One that panics or renders empty
// would otherwise be found by a person clicking through 65 pages.
func TestEveryPageServes(t *testing.T) {
	client := irgotest.NewClient(app.NewRouter().Handler())

	pages := site.Pages("v0.1.0")
	if len(pages) == 0 {
		t.Fatal("the showcase enumerates no pages — Pages() is the source the " +
			"router is built from, so this example is serving nothing")
	}

	for _, p := range pages {
		t.Run(p.Path, func(t *testing.T) {
			resp := client.Get(p.Path)
			resp.AssertOK(t)
			resp.AssertHTML(t)
		})
	}

	t.Logf("%d pages served", len(pages))
}

// TestPagesAreMorpheus — that the pages render Morpheus's own markup rather
// than an error page that happens to return 200.
func TestPagesAreMorpheus(t *testing.T) {
	client := irgotest.NewClient(app.NewRouter().Handler())

	resp := client.Get("/")
	resp.AssertOK(t)
	// A custom element tag: the showcase is built from them, so markup without
	// one is not the showcase.
	resp.AssertContains(t, "<neo-")
}

// TestAssetsAreLinked — the showcase's stylesheets and bundle come out of the
// Go module via sync.go, and a page that links one irgo did not copy renders
// unstyled with nothing to say why.
func TestAssetsAreLinked(t *testing.T) {
	client := irgotest.NewClient(app.NewRouter().Handler())

	resp := client.Get("/")
	resp.AssertOK(t)

	body := string(resp.Body)
	for _, want := range []string{"morpheus.css", ".js"} {
		if !strings.Contains(body, want) {
			t.Errorf("the home page links no %s — run `go generate ./...` to "+
				"copy the showcase's assets out of the module", want)
		}
	}
}
