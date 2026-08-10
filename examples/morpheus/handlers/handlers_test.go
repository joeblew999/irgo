package handlers_test

import (
	"testing"

	irgotest "github.com/stukennedy/irgo/pkg/testing"

	"morpheus/app"
)

// What a handler returns. The browser-level facts — whether a component
// upgraded, whether a signal applied — are in ui_test.go, because no amount of
// reading the response can answer them.

func TestHomePageRenders(t *testing.T) {
	resp := irgotest.NewClient(app.NewRouter()).Get("/")
	resp.AssertOK(t)
	resp.AssertContains(t, "irgo + Morpheus")
}

// TestComponentsAreServerRendered — the markup arrives from Go, not from a
// script that builds it later. That is the difference between this and a
// client-side component library, and it is worth asserting rather than
// assuming.
func TestComponentsAreServerRendered(t *testing.T) {
	resp := irgotest.NewClient(app.NewRouter()).Get("/")
	resp.AssertOK(t)
	for _, tag := range []string{"<neo-badge", "<neo-alert", "<neo-button"} {
		resp.AssertContains(t, tag)
	}
}

// TestAssetsAreLinked — the three things a Morpheus page needs. Missing any of
// them renders correct HTML with no styling or behaviour, so the response is
// where it is cheapest to catch.
func TestAssetsAreLinked(t *testing.T) {
	resp := irgotest.NewClient(app.NewRouter()).Get("/")
	resp.AssertOK(t)
	resp.AssertContains(t, "/static/css/morpheus/morpheus.css")
	resp.AssertContains(t, "/static/css/morpheus/theme-default.css")
	resp.AssertContains(t, "/static/js/morpheus.js")
}
