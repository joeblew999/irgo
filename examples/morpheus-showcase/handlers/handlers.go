package handlers

import (
	site "github.com/romshark/morpheus/showcase"

	"github.com/stukennedy/irgo/pkg/render"
	"github.com/stukennedy/irgo/pkg/router"
)

// morpheusVersion is stamped into each page frame by the showcase itself.
const morpheusVersion = "v0.1.0"

var renderer = render.NewTemplRenderer()

// Mount serves Morpheus's showcase from an irgo router.
//
// The showcase enumerates itself, so this is a loop rather than a route table.
// A page added upstream appears here on the next `go get -u` with nothing to
// regenerate — which is the whole point: an example that has to be
// re-derived when its source moves is an example that is quietly wrong.
//
// An earlier version parsed their static site generator to recover the same
// list. It worked and it was wrong: a page added upstream would have been
// missed in silence.
func Mount(r *router.Router) {
	for _, p := range site.Pages(morpheusVersion) {
		component := p.Component
		r.GET(p.Path, func(ctx *router.Context) (string, error) {
			return renderer.Render(component)
		})
	}
}
