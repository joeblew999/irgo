package handlers_test

import (
	"strings"
	"testing"

	"github.com/gost-dom/browser"
	"github.com/gost-dom/browser/scripting/sobekengine"

	"docs-templ/app"
)

// Tests that drive a real DOM, using gost-dom — a headless browser written in
// Go that consumes an http.Handler directly, with no TCP and no Chrome.
//
// The tests in handlers_test.go assert on response bodies, which cannot tell a
// working page from one whose markup happens to contain the right strings.
// These load the page, build the DOM, and run its scripts, in about 20ms.
//
// gost-dom does not render: there is no layout and no screenshot, so it cannot
// see a stylesheet that hides an element or a heading that collides with the
// block above it. It replaces string-matching, not looking at the page.

func newBrowser(t *testing.T) *browser.Browser {
	t.Helper()
	b := browser.New(
		browser.WithHandler(app.NewRouter().Handler()),
		browser.WithScriptEngine(sobekengine.DefaultEngine()),
	)
	t.Cleanup(b.Close)
	return b
}

func TestDemoPageBuildsItsDOM(t *testing.T) {
	win, err := newBrowser(t).Open("/demo")
	if err != nil {
		t.Fatal(err)
	}
	doc := win.Document()

	h1, _ := doc.QuerySelector("h1")
	if h1 == nil {
		t.Fatal("no h1 on the demo page")
	}
	if got := h1.TextContent(); got != "Server-Driven Reactivity" {
		t.Errorf("h1 is %q", got)
	}

	// The id the handler patches has to exist in the page it patches into.
	// This is the seam that string-matching cannot check, because both halves
	// can contain "counter-value" while referring to different elements.
	if doc.GetElementById("counter-value") == nil {
		t.Error("the counter element the handler patches is not in the page")
	}
	if doc.GetElementById("sse-log") == nil {
		t.Error("the log element the handler patches is not in the page")
	}
}

func TestLandingPageBuildsItsDOM(t *testing.T) {
	win, err := newBrowser(t).Open("/")
	if err != nil {
		t.Fatal(err)
	}
	doc := win.Document()

	for sel, want := range map[string]int{
		".platform-card": 4,
		".feature-card":  6,
		".arch-diagram":  3, // the two SVGs plus the comparison panel
		".code-block":    2,
	} {
		nodes, err := doc.QuerySelectorAll(sel)
		if err != nil {
			t.Errorf("%s: %v", sel, err)
			continue
		}
		if got := nodes.Length(); got != want {
			t.Errorf("%s: %d in the DOM, want %d", sel, got, want)
		}
	}
}

// TestCounterRoundTrip is the test this page most wants, and cannot yet have.
//
// Clicking should make Datastar POST and the SSE response should patch the
// counter to 1. Everything up to Datastar's own start-up works: the client
// compiles and runs, and fetch, ReadableStream and TextDecoder — the transport
// it actually uses — are all present. It stops at MutationObserver.observe
// with an attributeFilter, which gost-dom v0.12 does not implement, so the
// plugins never bind and the click reaches nothing.
//
// The seam this would cover is covered instead by
// TestCounterRoundTripAgainstTheDOM, which does what Datastar would do.
//
// The skip tests for the capability rather than the version, so this starts
// running by itself when the engine grows it.
func TestCounterRoundTrip(t *testing.T) {
	win, err := newBrowser(t).Open("/demo")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := win.Eval(
		`(() => { new MutationObserver(() => {}).observe(
			document.body, {attributes: true, attributeFilter: ["x"]}); return 1 })()`,
	); err != nil {
		t.Skipf("gost-dom cannot filter MutationObserver attributes yet, "+
			"so Datastar's plugins never bind: %v", err)
	}

	btn, err := win.Document().QuerySelector(".counter-demo__btn--increment")
	if err != nil || btn == nil {
		t.Fatalf("no increment button: %v", err)
	}
	btn.(interface{ Click() }).Click()
	if err := win.Clock().RunAll(); err != nil {
		t.Fatalf("settling the clock: %v", err)
	}

	got, _ := win.Document().QuerySelector("#counter-value")
	if text := strings.TrimSpace(got.TextContent()); text != "1" {
		t.Errorf("counter is %q after one click, want \"1\"", text)
	}
}
