// Setting up translations while a project is being created.
//
// The scaffolded templates are written as TIKs, so a new project is
// translatable from the first commit rather than after a conversion nobody
// gets round to. That means the bundle has to exist before the project
// compiles, and creating it is a sequence rather than a command.
//
// The order is forced and each step explains the one before it:
//
//	toki generate -l en   writes the bundle, then fails analysing it — the
//	                      project does not depend on x/text yet
//	go mod tidy           adds what the bundle just imported
//	templ generate        makes the components Go, which is the only form
//	                      toki can read TIKs out of
//	toki generate         extracts them, now that all three are true
//
// Run out of order it does not fail loudly. It reports scanning zero files and
// writes an empty catalog, and the project looks translated until someone adds
// a language and finds nothing in it.
package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

// demoLocale is the second language a new project ships with.
//
// A project with one catalog cannot show that any of this works: every string
// renders in the source language whether the machinery runs or not. With a
// second one, switching the browser's language changes the page, which is the
// whole feature made visible in five seconds.
//
// German because it exercises what English hides — a plural rule that differs
// from English in the zero case, and words long enough to break a layout that
// assumed English. Delete the catalog if the project does not want it; nothing
// else refers to it.
const demoLocale = "de"

// demoTranslations are keyed by the English source text rather than by message
// id, because ids are content hashes toki computes and neither of us knows them
// until it has run.
var demoTranslations = map[string]string{
	"Server-driven hypermedia for Go": "Servergesteuerte Hypermedia für Go",
	"Connection":                      "Verbindung",
	"Pulses":                          "Impulse",
	"Latency":                         "Latenz",
	"Live":                            "Live",
	"Connecting...":                   "Verbinde...",
	"Move your cursor to see server-driven reactivity. Every visual change flows through Go handlers via SSE.": "Bewege den Mauszeiger, um servergesteuerte Reaktivität zu sehen. Jede visuelle Änderung läuft über Go-Handler via SSE.",
}

// bootstrapI18n creates the bundle for a project being scaffolded.
//
// Best-effort throughout. A network failure fetching toki should not leave a
// half-created project — everything else about it is already valid, and `irgo
// i18n init` finishes the job later.
func bootstrapI18n(dir, locale string) error {
	if err := ensureGoTool("toki"); err != nil {
		return fmt.Errorf("toki: %w", err)
	}

	// Before toki, not after. The scaffolded handlers import this package, so
	// without it the project does not compile — and toki reads Go by analysing
	// it, so every later step reports finding nothing.
	if _, err := writeLangHelperIn(dir); err != nil {
		return fmt.Errorf("writing lang/lang.go: %w", err)
	}

	run := func(name string, args ...string) error {
		cmd := exec.Command(name, args...)
		cmd.Dir = dir
		return cmd.Run()
	}

	// Fails analysing what it just wrote, which is expected: the project does
	// not depend on x/text or go-playground/locales yet. The bundle is on disk
	// either way, which is what the next steps need.
	_ = run("toki", "generate", "-l", locale, "-q")

	if err := run(goBin(), "mod", "tidy"); err != nil {
		return fmt.Errorf("adding the bundle's dependencies: %w", err)
	}
	if err := run("templ", "generate"); err != nil {
		return fmt.Errorf("generating templates: %w", err)
	}
	if err := run("toki", "generate", "-q"); err != nil {
		return fmt.Errorf("extracting texts: %w", err)
	}

	if err := addDemoLocale(dir, run); err != nil {
		// Not fatal. The project has working translations in one language;
		// the second one only exists to demonstrate them.
		fmt.Printf("Note: could not add the %s catalog: %v\n", demoLocale, err)
	}
	return run(goBin(), "mod", "tidy")
}

// addDemoLocale creates the second catalog and fills it in.
func addDemoLocale(dir string, run func(string, ...string) error) error {
	if err := run("toki", "generate", "-t", demoLocale, "-q"); err != nil {
		return err
	}

	// The source catalog maps id -> English text. Inverting it turns the
	// translations above, which are keyed by text, into the ids toki wants.
	src, err := readARB(filepath.Join(dir, tokiBundleDir, "catalog_"+sourceLocaleOf(dir)+".arb"))
	if err != nil {
		return err
	}
	dstPath := filepath.Join(dir, tokiBundleDir, "catalog_"+demoLocale+".arb")
	dst, err := readARB(dstPath)
	if err != nil {
		return err
	}

	for id, v := range src {
		english, ok := v.(string)
		if !ok || len(id) == 0 || id[0] == '@' {
			continue
		}
		if german, ok := demoTranslations[english]; ok {
			dst[id] = german
		}
	}

	body, err := json.MarshalIndent(dst, "", "  ")
	if err != nil {
		return err
	}
	if err := os.WriteFile(dstPath, append(body, '\n'), 0o644); err != nil {
		return err
	}
	// Regenerate so the Go catalog matches the .arb just written. Without this
	// the file is translated and the app still renders English, which is the
	// most confusing possible state.
	return run("toki", "generate", "-q")
}

// sourceLocaleOf finds which catalog is the source, by elimination.
func sourceLocaleOf(dir string) string {
	entries, err := os.ReadDir(filepath.Join(dir, tokiBundleDir))
	if err != nil {
		return "en"
	}
	for _, e := range entries {
		n := e.Name()
		if len(n) > 12 && n[:8] == "catalog_" && filepath.Ext(n) == ".arb" {
			loc := n[8 : len(n)-4]
			if loc != demoLocale {
				return loc
			}
		}
	}
	return "en"
}

func readARB(path string) (map[string]any, error) {
	body, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var m map[string]any
	if err := json.Unmarshal(body, &m); err != nil {
		return nil, err
	}
	return m, nil
}
