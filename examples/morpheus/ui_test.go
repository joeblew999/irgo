package main

import (
	"testing"

	"github.com/stukennedy/irgo/pkg/browsertest"

	"morpheus/app"
)

// These assert what a browser does, which is the only place the interesting
// failures live. Every one of them renders correct HTML:
//
//   - a web component that never upgraded, because the bundle did not load
//   - a stylesheet that 404s, because nothing refreshed it into static/
//   - a Datastar signal that never updated, because an expression was wrong
//
// Serving the page and reading the markup would pass in all three cases.

// TestComponentsUpgrade — Morpheus components are custom elements, and a
// custom element that never upgraded is an unknown tag: it renders its
// children unstyled and does nothing.
func TestComponentsUpgrade(t *testing.T) {
	p := browsertest.Open(t, app.NewRouter())

	p.MustExist("neo-badge")
	p.MustExist("neo-alert")

	// neo-button, not neo-badge. Only components with behaviour register: a
	// badge is styled by morpheus.css and defines nothing, so asserting it
	// upgraded is asserting something that was never meant to be true.
	//
	// Waited for, not read: the bundle is a module script, so it executes
	// after load. Reading immediately tests the moment before it ran.
	p.MustEventually(`customElements.get('neo-button') !== undefined`,
		"the bundle did not load, or did not register the element")
}

// TestAssetsAreServed — the three assets irgo refreshes out of the Go module.
// A 404 here is silent in the markup and total in the browser.
func TestAssetsAreServed(t *testing.T) {
	p := browsertest.Open(t, app.NewRouter())

	for _, path := range []string{
		"/static/css/morpheus/morpheus.css",
		"/static/css/morpheus/theme-default.css",
		"/static/js/morpheus.js",
	} {
		p.MustEventually(
			`fetch('`+path+`').then(r => r.status === 200)`,
			"served non-200, so the page is missing its styling or behaviour")
	}
}

// TestThemeSwitchIsAClassChange — a theme is custom properties scoped to a
// class, so switching one is a class change rather than a stylesheet swap.
// That is what makes it instant and offline: no request, no reload.
func TestThemeSwitchIsAClassChange(t *testing.T) {
	p := browsertest.Open(t, app.NewRouter())

	const themed = "[data-attr\\:class]"

	// Datastar applies the class once it has processed the page, which is
	// after the module executes.
	p.MustHaveAttr(themed, "class", "theme-default")

	p.Click(`[data-on\:click*="ocean"]`)

	// Waiting, not reading: the class lands a moment after the click, so an
	// immediate read races it and fails intermittently.
	p.MustHaveAttr(themed, "class", "theme-ocean")
}
