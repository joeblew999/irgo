// Package app provides the shared application setup.
// This is imported by both main.go (desktop) and mobile/mobile.go (mobile).
package app

import (
	"io/fs"
	"net/http"

	"docs-templ/clidoc"
	"docs-templ/content"
	"docs-templ/handlers"
	"docs-templ/static"
	"docs-templ/templates"
	"github.com/stukennedy/irgo/pkg/render"
	"github.com/stukennedy/irgo/pkg/router"
)

var Renderer = render.NewTemplRenderer()

// NewRouter creates a new router with all app routes configured.
func NewRouter() *router.Router {
	r := router.New()

	// Serve embedded static files (works for both web and mobile)
	staticFS, _ := fs.Sub(static.Files, ".")
	r.Static("/static", http.FS(staticFS))

	// Home page
	r.GET("/", func(ctx *router.Context) (string, error) {
		return Renderer.Render(templates.DocsHome())
	})

	// Documentation pages. The CLI reference is rendered from what the binary
	// declares rather than written by hand, which is the whole reason this
	// site is a Go app.
	r.GET("/docs/cli", func(ctx *router.Context) (string, error) {
		byNoun := map[string][]clidoc.Command{}
		for _, n := range clidoc.Nouns() {
			byNoun[n] = clidoc.For(n)
		}
		return Renderer.Render(templates.CLIReference(clidoc.Nouns(), byNoun))
	})
	// Every prose page, from one declaration each.
	for _, p := range content.Pages {
		page := p
		r.GET("/docs/"+page.Slug, func(ctx *router.Context) (string, error) {
			return Renderer.Render(templates.Prose(page))
		})
	}

	// Mount handlers
	handlers.Mount(r)

	return r
}
