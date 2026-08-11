// Package browsertest drives a real browser against an irgo app.
//
// pkg/testing answers what a handler returned. That is enough for most things
// and not enough for anything a browser does after the response arrives:
// whether a Datastar signal updated, whether a web component upgraded, whether
// a script that never loaded left the markup inert. Those failures all render
// perfectly correct HTML.
//
// It is a separate module on purpose. playwright-go pulls in thirty modules,
// and a framework should not put that in the graph of every project that
// imports it merely so a few of them can write browser tests. Opt in with:
//
//	go get github.com/stukennedy/irgo/pkg/browsertest
//
// The browsers themselves are a one-time install, and a test skips rather than
// fails without them — CI that has not installed them stays green instead of
// going permanently red for a reason nobody can act on:
//
//	go run github.com/mxschmitt/playwright-go/cmd/playwright@v0.6100.0 install chromium
package browsertest

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/mxschmitt/playwright-go"
)

// Page is a browser page pointed at a running instance of an app.
type Page struct {
	t    *testing.T
	page playwright.Page

	// URL is where the app is served, for a test that wants to visit another
	// route itself.
	URL string
}

// Open serves the handler and opens a browser on it.
//
// Everything is torn down when the test ends: the server, the browser and the
// driver. A leaked browser outlives the run and the next one starts slower for
// reasons nobody connects to a test.
func Open(t *testing.T, handler http.Handler) *Page {
	t.Helper()
	return open(t, handler, "")
}

// OpenAs is Open with the browser set to a locale.
//
// It sets what a real visitor from that locale sends: navigator.language and
// navigator.languages inside the page, and the Accept-Language header on every
// request the browser makes. Both, because an irgo app can read either — a
// server-rendered page negotiates from the header, and the browser/wasm build
// runs in a service worker where there is no header to read and navigator is
// the only source.
//
// A translation bug is invisible to any test that does not do this. The
// language a test browser reports is the language of the machine running it,
// so a German catalog is never exercised on an English laptop and every
// assertion passes on the source language.
//
//	p := browsertest.OpenAs(t, app.NewRouter(), "de-DE")
//	p.MustHaveText(".tagline", "Servergesteuerte Hypermedia für Go")
func OpenAs(t *testing.T, handler http.Handler, locale string) *Page {
	t.Helper()
	return open(t, handler, locale)
}

func open(t *testing.T, handler http.Handler, locale string) *Page {
	t.Helper()

	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)

	pw, err := playwright.Run()
	if err != nil {
		t.Skipf("browser tests need playwright:\n"+
			"  go run github.com/mxschmitt/playwright-go/cmd/playwright@v0.6100.0 install chromium\n"+
			"  (%v)", err)
	}
	t.Cleanup(func() { pw.Stop() })

	browser, err := pw.Chromium.Launch()
	if err != nil {
		t.Fatalf("launching chromium: %v", err)
	}
	t.Cleanup(func() { browser.Close() })

	// A context rather than browser.NewPage(), because locale is a context
	// property: it has to be set before the page exists, and cannot be changed
	// afterwards without discarding it.
	var opts playwright.BrowserNewContextOptions
	if locale != "" {
		opts.Locale = playwright.String(locale)
	}
	ctx, err := browser.NewContext(opts)
	if err != nil {
		t.Fatalf("opening a browser context: %v", err)
	}
	t.Cleanup(func() { ctx.Close() })

	page, err := ctx.NewPage()
	if err != nil {
		t.Fatalf("opening a page: %v", err)
	}

	// A page error is a script that threw. Without this the symptom is a later
	// assertion timing out on an element that never changed, which reads as
	// the wrong bug entirely.
	page.On("pageerror", func(err error) { t.Errorf("page error: %v", err) })

	// Console errors too, because the failure that matters most here is not a
	// thrown error. A module script served with the wrong MIME type is
	// refused by the browser and only ever mentioned in the console — the
	// page then behaves as though the script did not exist, and every
	// assertion about it times out saying nothing about why.
	page.On("console", func(m playwright.ConsoleMessage) {
		if m.Type() == "error" {
			t.Errorf("console error: %s", m.Text())
		}
	})

	if _, err := page.Goto(srv.URL); err != nil {
		t.Fatalf("loading %s: %v", srv.URL, err)
	}
	return &Page{t: t, page: page, URL: srv.URL}
}

// Click clicks the first element matching a selector.
//
// Force skips the actionability checks. A demo page with a full-screen overlay
// intercepts clicks aimed at what is underneath, and a test about behaviour
// should not fail on hit testing — but a real click is the honest default, so
// this is not it.
func (p *Page) Click(selector string) {
	p.t.Helper()
	if err := p.page.Locator(selector).Click(); err != nil {
		p.t.Fatalf("clicking %s: %v", selector, err)
	}
}

// Eval runs JavaScript in the page and returns the result.
//
// The escape hatch. Some things have no locator — whether a custom element
// upgraded, what a signal holds — and a test that cannot ask them is a test
// that asserts markup instead of behaviour.
func (p *Page) Eval(script string) any {
	p.t.Helper()
	v, err := p.page.Evaluate(script)
	if err != nil {
		p.t.Fatalf("evaluating %s: %v", script, err)
	}
	return v
}

// Text returns an element's text once it exists.
func (p *Page) Text(selector string) string {
	p.t.Helper()
	loc := p.page.Locator(selector)
	if err := loc.WaitFor(); err != nil {
		p.t.Fatalf("waiting for %s: %v", selector, err)
	}
	s, err := loc.TextContent()
	if err != nil {
		p.t.Fatalf("reading %s: %v", selector, err)
	}
	return strings.TrimSpace(s)
}

// Attr returns an attribute once the element exists.
func (p *Page) Attr(selector, name string) string {
	p.t.Helper()
	loc := p.page.Locator(selector)
	if err := loc.WaitFor(); err != nil {
		p.t.Fatalf("waiting for %s: %v", selector, err)
	}
	v, err := loc.GetAttribute(name)
	if err != nil {
		p.t.Fatalf("reading %s of %s: %v", name, selector, err)
	}
	return v
}

// MustHaveText waits until an element contains the given text, and fails with
// what it found instead.
//
// Waiting rather than reading: anything driven by a signal or a script settles
// a moment after the event that caused it, so an immediate read races the
// thing under test and fails intermittently — which is worse than failing
// always, because it gets rerun until it passes.
func (p *Page) MustHaveText(selector, want string) {
	p.t.Helper()
	err := p.page.Locator(selector).
		Filter(playwright.LocatorFilterOptions{HasText: want}).
		WaitFor()
	if err == nil {
		return
	}
	got, _ := p.page.Locator(selector).TextContent()
	p.t.Fatalf("%s never contained %q (it holds %q)", selector, want, strings.TrimSpace(got))
}

// MustExist fails unless a selector matches something.
func (p *Page) MustExist(selector string) {
	p.t.Helper()
	if err := p.page.Locator(selector).WaitFor(); err != nil {
		p.t.Fatalf("%s never appeared: %v", selector, err)
	}
}

// MustHaveAttr waits until an attribute holds the given value.
//
// The attribute equivalent of MustHaveText, and needed for the same reason:
// anything a signal drives lands after the event, so reading immediately
// races it.
func (p *Page) MustHaveAttr(selector, name, want string) {
	p.t.Helper()
	_, err := p.page.WaitForFunction(
		`([sel, attr, want]) => document.querySelector(sel)?.getAttribute(attr) === want`,
		[]any{selector, name, want},
	)
	if err == nil {
		return
	}
	p.t.Fatalf("%s never had %s=%q (it holds %q)",
		selector, name, want, p.Attr(selector, name))
}

// MustEventually waits until a JavaScript expression is true.
//
// The form every assertion about a browser has to take. A module script
// executes after load, a custom element upgrades when its definition arrives,
// a signal updates after the event that changed it — so reading immediately
// tests the moment before the thing under test happened.
//
// Eval is for questions with an answer now. This is for everything else, and
// it is almost always this.
func (p *Page) MustEventually(expr, whyItMatters string) {
	p.t.Helper()
	if _, err := p.page.WaitForFunction(expr, nil); err != nil {
		p.t.Fatalf("never became true: %s\n  %s", expr, whyItMatters)
	}
}
