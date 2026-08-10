// Package app provides the shared application setup.
// This is imported by both main.go (desktop) and mobile/mobile.go (mobile).
package app

import (
	"io/fs"
	"net/http"

	"docs-templ/apidoc"
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

	// llms.txt is what the Introduction page points an AI assistant at, so it
	// has to be text/plain rather than a rendered page — hence HandleFunc
	// rather than the usual GET.
	r.HandleFunc("/llms.txt", func(w http.ResponseWriter, req *http.Request) {
		data, err := static.Files.ReadFile("llms.txt")
		if err != nil {
			http.Error(w, "llms.txt is missing from the build", http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.Write(data)
	})

	// The landing page, the interactive demo, and the documentation index.
	r.GET("/", func(ctx *router.Context) (string, error) {
		return Renderer.Render(templates.HomePage())
	})
	r.GET("/demo", func(ctx *router.Context) (string, error) {
		return Renderer.Render(templates.DemoPage())
	})
	r.GET("/constellation", func(ctx *router.Context) (string, error) {
		return Renderer.Render(templates.ConstellationPage())
	})
	r.GET("/docs", func(ctx *router.Context) (string, error) {
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
	// The API reference, generated from the framework's source for the same
	// reason the CLI reference is generated from the binary.
	r.GET("/docs/api", func(ctx *router.Context) (string, error) {
		return Renderer.Render(templates.APIReference(apidoc.Packages()))
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
