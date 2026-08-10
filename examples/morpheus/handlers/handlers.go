package handlers

import (
	"morpheus/templates"

	"github.com/stukennedy/irgo/pkg/render"
	"github.com/stukennedy/irgo/pkg/router"
)

var renderer = render.NewTemplRenderer()

// Mount registers the example's routes.
//
// One page. What this example demonstrates happens in the browser, so the
// server has nothing interesting to do.
func Mount(r *router.Router) {
	r.GET("/", func(ctx *router.Context) (string, error) {
		return renderer.Render(templates.HomePage())
	})
}
