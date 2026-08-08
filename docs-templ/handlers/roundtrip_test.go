package handlers_test

import (
	"regexp"
	"strings"
	"testing"

	"github.com/gost-dom/browser"
	"github.com/gost-dom/browser/dom"
	"github.com/gost-dom/browser/scripting/sobekengine"
	irgotest "github.com/stukennedy/irgo/pkg/testing"

	"docs-templ/app"
)

// The counter's whole round trip, against a real DOM.
//
// gost-dom has no EventSource yet, so Datastar's own client cannot open the
// stream (see browser_test.go). This does what Datastar would do: it reads the
// endpoint off the button's own attribute, sends the request, and applies the
// patches the server sends back to the document that was loaded.
//
// That covers the seam nothing else does. The handler test proves the SSE is
// right and the page test proves the ids exist; only this notices when the id
// in the template stops being the id the handler patches, because here the
// patch has to find something to replace.

var (
	// @post('/api/demo/increment') — the endpoint as the page declares it.
	actionRE = regexp.MustCompile(`@(get|post|put|patch|delete)\('([^']+)'\)`)
	// data: elements <html…>
	elementsRE = regexp.MustCompile(`(?m)^data: elements (.*)$`)
	idRE       = regexp.MustCompile(`\bid="([^"]+)"`)
)

func TestCounterRoundTripAgainstTheDOM(t *testing.T) {
	b := browser.New(
		browser.WithHandler(app.NewRouter().Handler()),
		browser.WithScriptEngine(sobekengine.DefaultEngine()),
	)
	t.Cleanup(b.Close)

	win, err := b.Open("/demo")
	if err != nil {
		t.Fatal(err)
	}
	doc := win.Document()

	// Follow the page rather than a hardcoded URL: if the button stops
	// pointing at the handler, this test stops finding it.
	btn, err := doc.QuerySelector(".counter-demo__btn--increment")
	if err != nil || btn == nil {
		t.Fatalf("no increment button: %v", err)
	}
	expr, ok := btn.GetAttribute("data-on:click")
	if !ok {
		t.Fatal("the increment button has no data-on:click")
	}
	m := actionRE.FindStringSubmatch(expr)
	if m == nil {
		t.Fatalf("cannot read an endpoint out of %q", expr)
	}
	method, path := m[1], m[2]
	if method != "post" {
		t.Fatalf("expected the counter to POST, got %s", method)
	}

	before := textOf(t, doc, "counter-value")
	if before != "0" {
		t.Fatalf("counter starts at %q, want \"0\"", before)
	}

	// Send what the page's signals hold at this point.
	resp := irgotest.NewClient(app.NewRouter().Handler()).
		Datastar().
		PostJSON(path, `{"count": 0, "log": []}`)
	resp.AssertOK(t)

	patches := elementsRE.FindAllStringSubmatch(string(resp.Body), -1)
	if len(patches) == 0 {
		t.Fatal("the response patched no elements")
	}

	applied := 0
	for _, p := range patches {
		html := p[1]
		id := idRE.FindStringSubmatch(html)
		if id == nil {
			t.Errorf("patched element carries no id: %.60s", html)
			continue
		}
		target := doc.GetElementById(id[1])
		if target == nil {
			t.Errorf("the handler patches #%s, which is not in the page it patches into", id[1])
			continue
		}
		if err := target.SetOuterHTML(html); err != nil {
			t.Errorf("applying the patch for #%s: %v", id[1], err)
			continue
		}
		applied++
	}
	if applied == 0 {
		t.Fatal("no patch could be applied")
	}

	// The DOM the reader would be looking at.
	if got := textOf(t, doc, "counter-value"); got != "1" {
		t.Errorf("counter reads %q after one increment, want \"1\"", got)
	}
	if log := textOf(t, doc, "sse-log"); !strings.Contains(log, "POST "+path) {
		t.Errorf("the event log does not mention the request; it reads %q", log)
	}
}

func textOf(t *testing.T, doc dom.Document, id string) string {
	t.Helper()
	el := doc.GetElementById(id)
	if el == nil {
		t.Fatalf("no #%s in the document", id)
	}
	return strings.TrimSpace(el.TextContent())
}
