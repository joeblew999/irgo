// The Morpheus web component kit, when a project depends on it.
//
// Morpheus is two halves that have to agree: a Go module carrying the templ
// wrappers, and static assets carrying the styling and behaviour. Its
// documented install points the assets at a CDN, which is right for an
// ordinary web app and wrong for an irgo one — the same code runs in an iOS
// and Android WebView, a desktop shell and a Cloudflare Worker, where a second
// origin is an offline failure rather than a convenience.
//
// The assets ship inside the Go module, so there is nothing to download. `go
// get` has already put them on disk at exactly the version the templ wrappers
// were compiled against, and copying them out is the same kind of derived
// output as _templ.go or the Tailwind stylesheet: regenerated on every build,
// gitignored, never edited.
//
// The first version of this fetched them from GitHub releases and pinned a
// version alongside go.mod's. That is two records of one fact, and the one
// irgo wrote down could drift from the one the compiler used.
package main

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

const morpheusModule = "github.com/romshark/morpheus"

// morpheusAssetDir is irgo's to regenerate. A project's own stylesheets —
// including its own themes — live in static/css beside it and are never
// touched.
var morpheusAssetDir = filepath.Join("static", "css", "morpheus")

// morpheusTheme is the stylesheet a project uses unless it says otherwise.
//
// Themes are small — under 5 KB each — and switching one is switching a
// stylesheet, so irgo copies the lot rather than making the choice a setting
// it has to store and keep in step with what a layout actually links.
const morpheusTheme = "default"

func runUI(args []string) error {
	switch uiVerb(args) {
	case "themes", "":
		return uiThemes()
	}
	return fmt.Errorf("unknown: irgo ui %s", uiVerb(args))
}

// syncMorpheusAssets copies the kit's stylesheets and bundle out of the module
// into the project, when the project depends on it.
//
// Called from ensureAssets, so it happens on every build rather than being a
// command someone has to know about. A project that does not use Morpheus does
// nothing here.
func syncMorpheusAssets() error {
	dir := morpheusDir()
	if dir == "" {
		// Not a dependency. Silent unless the project still expects the
		// assets — see morpheusOrphanedLinks for why that happens without
		// anyone doing anything wrong.
		return morpheusOrphanedLinks()
	}
	min := filepath.Join(dir, "min")
	if _, err := os.Stat(min); err != nil {
		// A future release could rearrange this. Say so rather than leaving a
		// page that renders unknown elements with no styling and no clue.
		return fmt.Errorf("%s has no min/ directory — irgo does not know where "+
			"its assets moved to", morpheusModule)
	}

	// Into a directory of their own. A theme is a file named theme-<name>.css,
	// and a project writing its own would sooner or later pick a name the kit
	// also ships — at which point a build would silently overwrite it. Keeping
	// the copies under morpheus/ means the whole directory is irgo's to
	// regenerate and everything beside it is the project's.
	for _, a := range []struct{ src, dst string }{
		{"morpheus.css", filepath.Join(morpheusAssetDir, "morpheus.css")},
		{"bundle.js", filepath.Join("static", "js", "morpheus.js")},
	} {
		if err := copyMorpheusFile(filepath.Join(min, a.src), a.dst); err != nil {
			return err
		}
	}

	// Every theme, not the configured one: they are a few KB each, and a
	// layout that links a theme irgo did not copy is a page with no colours
	// and nothing to explain why.
	themes, _ := filepath.Glob(filepath.Join(min, "theme-*.css"))
	for _, t := range themes {
		if err := copyMorpheusFile(t, filepath.Join(morpheusAssetDir, filepath.Base(t))); err != nil {
			return err
		}
	}

	// Copied is not the same as used, and the difference is invisible.
	return morpheusUnlinkedAssets()
}

// morpheusOrphanedLinks reports a layout that links assets nothing refreshes.
//
// The module is only kept by `go mod tidy` while something imports it. Remove
// the last component from a page and the next tidy prunes it — but the layout
// still links three files, so they stop being refreshed and eventually 404.
//
// Normally writeMorpheusKeep prevents this. It remains for the case that file
// is deleted by hand, which is the documented way to opt out — and which is
// only correct if the layout's links go too.
//
// It is worse than it sounds because it takes two steps: the first tidy keeps
// the module, since the generated _templ.go still imports it. Only after a
// build regenerates that file does the next tidy drop it. By then the edit
// that caused it is in a different session, and nothing connects the two.
//
// So this looks at what the project asks for rather than what it has.
// morpheusUnlinkedAssets reports assets that nothing on the page asks for.
//
// The mirror of morpheusOrphanedLinks, and the one a new project actually
// hits. `go get` the kit, use a component, build: the assets are copied, they
// are served, the custom element is in the markup — and the page has no
// stylesheet and no bundle, because adding those is the one step that is the
// project's rather than irgo's.
//
// Nothing about that looks wrong. The element renders its children as an
// unknown tag, so there is text on the screen; it is simply unstyled, and the
// interactive components never upgrade. The obvious readings are all wrong —
// the kit is broken, irgo did not copy anything, the version is off.
//
// Checked by what the templates ask for rather than by what is on disk,
// because the assets are always on disk by the time this runs.
func morpheusUnlinkedAssets() error {
	missing, err := headsWithoutMorpheus()
	if err != nil || len(missing) == 0 {
		return nil
	}

	fmt.Println("Note: Morpheus's assets are in static/, but these do not link them:")
	fmt.Println()
	for _, m := range missing {
		fmt.Printf("        %s\n", m)
	}
	fmt.Println()
	fmt.Println("      Components rendered through them will be unstyled, and the")
	fmt.Println("      interactive ones will not upgrade. Add to each <head>:")
	fmt.Println()
	fmt.Println(`        <link rel="stylesheet" href="/static/css/morpheus/morpheus.css"/>`)
	fmt.Printf("        <link rel=\"stylesheet\" href=\"/static/css/morpheus/theme-%s.css\"/>\n", morpheusTheme)
	fmt.Println(`        <script type="module" src="/static/js/morpheus.js"></script>`)
	fmt.Println()
	fmt.Println("      The theme is a class on the root element, so swapping that")
	fmt.Println("      line is how you change it. See: irgo ui themes")
	return nil
}

// headsWithoutMorpheus finds the templ blocks that render a <head> and do not
// link the kit, as "file.templ:Name".
//
// Per block rather than per project, because a scaffolded project has two —
// Layout and FullscreenPage, each with their own complete <head> — and the
// home page uses the second. Adding the links to "the layout" therefore has a
// fifty-fifty chance of changing nothing at all, with no error to say so: the
// page still renders, the component is still in the markup, and it is still
// unstyled. A check that asked whether *any* template mentioned the kit went
// quiet at exactly that moment, which made it worse than no check.
func headsWithoutMorpheus() ([]string, error) {
	var missing []string
	err := filepath.WalkDir("templates", func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() || !strings.HasSuffix(path, ".templ") {
			return nil
		}
		body, err := os.ReadFile(path)
		if err != nil {
			return nil
		}
		// A templ block runs to the next one. Good enough to attribute a
		// </head> to the block it is in, which is all this needs.
		name := ""
		var block strings.Builder
		flush := func() {
			if name == "" {
				return
			}
			b := block.String()
			if !strings.Contains(b, "</head>") {
				return
			}
			if strings.Contains(b, "/static/css/morpheus/") ||
				strings.Contains(b, "/static/js/morpheus.js") {
				return
			}
			missing = append(missing, fmt.Sprintf("%s:%s", path, name))
		}
		for _, line := range strings.Split(string(body), "\n") {
			if rest, ok := strings.CutPrefix(strings.TrimSpace(line), "templ "); ok {
				flush()
				block.Reset()
				name, _, _ = strings.Cut(rest, "(")
				continue
			}
			block.WriteString(line)
			block.WriteString("\n")
		}
		flush()
		return nil
	})
	return missing, err
}

func morpheusOrphanedLinks() error {
	var linking []string
	err := filepath.WalkDir("templates", func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() || !strings.HasSuffix(path, ".templ") {
			return nil
		}
		body, err := os.ReadFile(path)
		if err != nil {
			return nil
		}
		if strings.Contains(string(body), "/static/css/morpheus/") ||
			strings.Contains(string(body), "/static/js/morpheus.js") {
			linking = append(linking, path)
		}
		return nil
	})
	if err != nil || len(linking) == 0 {
		return nil
	}

	fmt.Printf("Note: %s links Morpheus assets, but the module is not a dependency.\n",
		strings.Join(linking, ", "))
	fmt.Println("      Nothing refreshes them, so they will 404 once removed.")
	fmt.Println()
	fmt.Println("      Either use a component again, which is what keeps the module:")
	fmt.Printf("        go get %s\n", morpheusModule)
	fmt.Println("      or drop the three <link>/<script> lines from the layout.")
	return nil
}

// copyMorpheusFile writes one asset, skipping the work when it is already
// current.
//
// Compared by content, not size. An upgrade that happens to leave a file the
// same length — a colour changed, a selector renamed — would be skipped, and
// the project would keep serving the old asset against new wrappers with
// nothing to show for it. Modification time is no better: the module cache
// stamps its files when it extracts them, not when they were written.
//
// Reading two files that are usually identical is cheap next to the build that
// follows.
func copyMorpheusFile(src, dst string) error {
	return copyGenerated(src, dst)
}

// morpheusDir is where the module is on this machine, or "" when the project
// does not depend on it.
func morpheusDir() string {
	return moduleDir(morpheusModule)
}

// uiThemes names the themes the kit ships, read from the module rather than
// listed here so a new one appears without an irgo release.
func uiThemes() error {
	dir := morpheusDir()
	if dir == "" {
		fmt.Printf("This project does not depend on %s.\n\n", morpheusModule)
		fmt.Printf("  go get %s\n", morpheusModule)
		fmt.Println()
		fmt.Println("Its assets are copied into static/ on the next build.")
		return nil
	}

	themes, err := filepath.Glob(filepath.Join(dir, "min", "theme-*.css"))
	if err != nil || len(themes) == 0 {
		return fmt.Errorf("no themes found in %s/min", dir)
	}

	fmt.Println("Themes, refreshed into static/css/morpheus on every build:")
	for _, t := range themes {
		name := strings.TrimSuffix(strings.TrimPrefix(filepath.Base(t), "theme-"), ".css")
		fmt.Printf("  %-10s /static/css/morpheus/%s\n", name, filepath.Base(t))
	}
	fmt.Println()
	fmt.Println("In your layout's <head>:")
	fmt.Println(`  <link rel="stylesheet" href="/static/css/morpheus/morpheus.css"/>`)
	fmt.Printf("  <link rel=\"stylesheet\" href=\"/static/css/morpheus/theme-%s.css\"/>\n", morpheusTheme)
	fmt.Println(`  <script type="module" src="/static/js/morpheus.js"></script>`)
	fmt.Println()
	fmt.Println("A theme is a set of custom properties scoped to a class on the")
	fmt.Println("root element, so switching one at runtime is switching that class:")
	fmt.Println()
	fmt.Printf("  <html class=\"theme-%s\">\n", morpheusTheme)
	fmt.Println(`  <html data-class-theme-ocean="$dark">   (with Datastar)`)
	fmt.Println()
	fmt.Println("Writing your own is the same shape. Put it in static/css, which")
	fmt.Println("irgo never touches, and link it after the kit's:")
	fmt.Println()
	fmt.Println("  /* static/css/theme-mine.css */")
	fmt.Println("  :root.theme-mine { --accent: #f50; --page-bg: #fff; }")
	return nil
}

// uiVerb is the first non-flag argument.
func uiVerb(args []string) string {
	for _, a := range args {
		if !strings.HasPrefix(a, "-") {
			return a
		}
	}
	return ""
}

func init() {
	// Before the CSS, so Tailwind sees anything the kit's stylesheets bring.
	registerAssetStep(assetOrderKit, syncMorpheusAssets)

	register(command{
		noun: "ui", verb: "themes", order: 0,
		summary: "The themes a kit ships, and how to add your own",
		usage:   [][2]string{{"", "List what static/css/morpheus holds"}},
	})
}
