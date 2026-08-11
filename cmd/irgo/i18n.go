// Translating an irgo app, via toki.
//
// toki extracts Textual Internationalization Keys from Go source and keeps a
// catalog per locale. The text stays in the code, readable, in the language it
// was written in:
//
//	reader.String(`You have {# new messages}`, n)
//
// and the catalogs hold what each locale makes of it, including the plural
// rules — which are not a detail. English has two forms, Polish has four, and
// Japanese has one; a framework that ships `if n == 1` has already decided the
// app will be wrong in most of the world.
//
// # Order matters, and it is the whole reason this is an asset step
//
// toki reads Go, and a templ component is not Go until templ has generated it.
// Run toki first and it reports `scan.files: 0` — a clean, quiet, successful
// run that extracted nothing, on a project whose every string lives in .templ
// files. Nothing about that output says the tool never saw the app.
//
// irgo's asset pipeline is templ, then registered steps, then CSS. Registering
// here means the ordering is a property of the build rather than something
// each developer has to know, and `irgo i18n check` cannot report a stale
// catalog just because it ran too early.
//
// # No second list of locales
//
// Which locales a project has is decided by the catalogs on disk. irgo does
// not keep its own copy in project config. Two lists of the same thing is a
// bug this codebase has already shipped twice — the config registry and the
// key list, then the formatting test's directories — and both times the
// symptom was silence: a setting that resolved to nothing, a check that looked
// at four directories out of six.
package main

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// tokiBundleDir is toki's default output package, and the one irgo assumes.
// The flag exists (-b) but a project that moves it gains nothing and loses
// every default in this file.
const tokiBundleDir = "tokibundle"

// hasI18n reports whether this project uses i18n at all.
//
// Its absence is how i18n stays opt-in. A framework that generates a
// translation bundle for every new project adds a dependency, a code
// generator and a directory of ICU messages to a great many apps that will
// only ever speak one language.
func hasI18n() bool {
	st, err := os.Stat(tokiBundleDir)
	return err == nil && st.IsDir()
}

// projectLocales lists the locales this project has, newest information first:
// the catalogs themselves.
func projectLocales() []string {
	entries, err := os.ReadDir(tokiBundleDir)
	if err != nil {
		return nil
	}
	var locales []string
	for _, e := range entries {
		name := e.Name()
		// catalog_en.arb, catalog_pt-BR.arb — the .arb is the translator's
		// file. catalog_en_gen.go is generated from it and would double-count.
		if !strings.HasPrefix(name, "catalog_") || !strings.HasSuffix(name, ".arb") {
			continue
		}
		locales = append(locales, strings.TrimSuffix(strings.TrimPrefix(name, "catalog_"), ".arb"))
	}
	sort.Strings(locales)
	return locales
}

// runI18nInit sets up the bundle, once.
func runI18nInit(args []string) error {
	locale := "en"
	if len(args) > 0 {
		locale = args[0]
	}
	if hasI18n() {
		fmt.Printf("This project already has i18n (%s/).\n", tokiBundleDir)

		// Not simply a no-op. A project set up before irgo scaffolded the
		// helper has the bundle and not the file, and re-running init is
		// exactly what someone in that position would try. Writing it here is
		// the difference between that working and it saying "already done"
		// about the half they have.
		created, err := writeLangHelper()
		if err != nil {
			return fmt.Errorf("writing lang/lang.go: %w", err)
		}
		if created {
			reportLangHelper(true)
		}

		fmt.Println()
		fmt.Println("  Add a language:  irgo i18n add <locale>")
		return nil
	}
	if err := ensureGoTool("toki"); err != nil {
		return err
	}

	// templ first, for the reason in this file's header: toki reads Go, and
	// the components are not Go yet.
	if err := runTempl(); err != nil {
		return err
	}

	fmt.Printf("Setting up i18n with %s as the source language...\n", locale)

	// The first generate writes the bundle and then fails analysing it, and
	// that is expected rather than broken. bundle_gen.go imports x/text and
	// go-playground/locales, which the project does not have yet — so toki
	// emits the file, immediately re-reads it, and reports three packages it
	// cannot import. The error names the generated file, which reads as toki
	// having produced something invalid.
	//
	// The fix is `go mod tidy` and then a second pass, which is what toki's own
	// quick start does. Done here because a developer who has to know that has
	// been handed a broken first command.
	firstPass := runCommand("toki", "generate", "-l", locale)
	if err := runCommand(goBin(), "mod", "tidy"); err != nil {
		return fmt.Errorf("adding the bundle's dependencies: %w", err)
	}
	if err := runCommand("toki", "generate"); err != nil {
		// Only now is a failure real: the dependencies are present, so this is
		// something else. Report the first pass too, since it came first and
		// may be the actual cause.
		if firstPass != nil {
			return fmt.Errorf("generating the bundle: %w", firstPass)
		}
		return fmt.Errorf("generating the bundle: %w", err)
	}

	created, err := writeLangHelper()
	if err != nil {
		return fmt.Errorf("writing lang/lang.go: %w", err)
	}

	// <html lang> must say which language the page is actually in. Left at the
	// scaffolded "en", a German page tells a screen reader to pronounce German
	// with English phonetics, and an Arabic one renders backwards for want of
	// a dir attribute.
	changedLayouts, skippedLayouts := upgradeLayoutForI18n()

	fmt.Println()
	fmt.Println("Done. Write text as a TIK and toki will extract it:")
	fmt.Println()
	fmt.Println("    t.String(`You have {# new messages}`, n)")
	fmt.Println()
	fmt.Println("  The braces carry the plural rules, so this is right in every")
	fmt.Println("  language. `if n == 1` is right in about half of them.")

	reportLangHelper(created)

	if changedLayouts > 0 {
		fmt.Println()
		fmt.Printf("Updated %d layout(s): <html> now reports the rendered language\n", changedLayouts)
		fmt.Println("  and its writing direction, which assistive technology relies on.")
	}
	for _, p := range skippedLayouts {
		fmt.Println()
		fmt.Printf("Note: %s has an <html> tag irgo did not recognise, so it was\n", p)
		fmt.Println("      left alone. Set the language yourself:")
		fmt.Println("        <html lang={ i18n.Lang(ctx) } dir={ i18n.Dir(ctx) }>")
	}

	fmt.Println()
	fmt.Println("  Add a language:      irgo i18n add de")
	fmt.Println("  Edit translations:   irgo i18n edit")
	fmt.Println("  Check completeness:  irgo i18n check")
	return nil
}

// runI18nAdd creates a catalog for another locale.
func runI18nAdd(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("which locale? e.g. irgo i18n add de\n\n" +
			"  A BCP 47 tag: de, fr, pt-BR, en-US")
	}
	if err := requireI18n(); err != nil {
		return err
	}
	if err := ensureGoTool("toki"); err != nil {
		return err
	}
	if err := runTempl(); err != nil {
		return err
	}

	cmd := []string{"generate"}
	for _, l := range args {
		cmd = append(cmd, "-t", l)
	}
	if err := runCommand("toki", cmd...); err != nil {
		return err
	}

	fmt.Println()
	fmt.Printf("Catalogs written. They start empty — %s/catalog_<locale>.arb is\n", tokiBundleDir)
	fmt.Println("what a translator edits, and `irgo i18n edit` opens a UI for it.")
	return nil
}

// runI18nCheck fails when a catalog is incomplete.
//
// Separate from `irgo project check` and deliberately so: an app mid-
// translation is a normal state to be in, and failing every build over a
// half-finished German catalog would teach people to stop running the check.
// This is the one you put in CI before a release.
func runI18nCheck(args []string) error {
	if err := requireI18n(); err != nil {
		return err
	}
	if err := ensureGoTool("toki"); err != nil {
		return err
	}
	if err := runTempl(); err != nil {
		return err
	}
	if err := runCommand("toki", "lint", "-require-complete"); err != nil {
		return fmt.Errorf("%w\n\n"+
			"  Some translations are missing. See which, and fill them in:\n"+
			"    irgo i18n edit", err)
	}
	fmt.Println()
	fmt.Printf("All %d locale(s) complete: %s\n",
		len(projectLocales()), strings.Join(projectLocales(), ", "))
	return nil
}

// runI18nEdit opens toki's translation UI.
func runI18nEdit(args []string) error {
	if err := requireI18n(); err != nil {
		return err
	}
	if err := ensureGoTool("toki"); err != nil {
		return err
	}
	fmt.Println("Opening the translation editor. Ctrl-C when you are done.")
	return runCommand("toki", append([]string{"webedit"}, args...)...)
}

// runI18nList shows what the project has, and how much of it is done.
func runI18nList(args []string) error {
	if !hasI18n() {
		fmt.Println("This project has no translations yet.")
		fmt.Println("  Set them up:  irgo i18n init")
		return nil
	}
	locales := projectLocales()
	fmt.Printf("%d locale(s) in %s/:\n\n", len(locales), tokiBundleDir)
	for _, l := range locales {
		fmt.Printf("  %-8s %s\n", l, filepath.Join(tokiBundleDir, "catalog_"+l+".arb"))
	}
	fmt.Println()
	fmt.Println("  Completeness:  irgo i18n check")
	return nil
}

// requireI18n gives the same answer everywhere rather than each verb inventing
// its own way of saying the bundle is missing.
func requireI18n() error {
	if hasI18n() {
		return nil
	}
	return fmt.Errorf("this project has no %s/ — run: irgo i18n init", tokiBundleDir)
}

// regenerateI18n keeps the bundle current on every build.
//
// Silent when the project has no bundle, which is most of them. When it does,
// a build that did not regenerate would compile the previous extraction: text
// changed in a template, catalogs unchanged, and the app serves the old string
// with no error anywhere.
func regenerateI18n() error {
	if !hasI18n() {
		return nil
	}

	// A bundle directory with no bundle_gen.go in it is the state a project
	// reaches by gitignoring the generated files, and toki cannot recover from
	// it: the default locale is recorded in bundle_gen.go and nowhere else, so
	// `toki generate` refuses to start and asks for -l — an error that names a
	// flag rather than the problem. It also analyses Go source, which no longer
	// compiles once the package the app imports has gone.
	//
	// Worth its own message because the instinct is exactly wrong here. The
	// bundle looks generated, and irgo does gitignore generated code elsewhere
	// — but templ rebuilds *_templ.go from .templ files, needing nothing to
	// compile first, and this cannot.
	if !pathExists(filepath.Join(tokiBundleDir, "bundle_gen.go")) {
		return fmt.Errorf("%s/ has catalogs but no bundle_gen.go\n\n"+
			"  It holds the default locale, so toki cannot regenerate without\n"+
			"  it — commit the whole of %s/, including the generated files.\n"+
			"  Unlike *_templ.go, this is not rebuildable from what is left.\n\n"+
			"  To recreate it now, naming the language your source is written in:\n"+
			"    irgo i18n init <locale>",
			tokiBundleDir, tokiBundleDir)
	}

	if err := ensureGoTool("toki"); err != nil {
		return err
	}
	// -q: on a normal build this has nothing to report, and six lines of
	// scan statistics per build trains people to stop reading build output.
	return runCommand("toki", "generate", "-q")
}

func init() {
	// After templ, because toki reads Go and a component is not Go until
	// templ has run. Before the CSS, which does not care either way.
	registerAssetStep(assetOrderDocs, regenerateI18n)

	register(command{
		noun: "i18n", verb: "init", order: 10,
		summary: "Set up translations for this project",
		args:    "[locale]",
		usage: [][2]string{
			{"", "English as the source language"},
			{"de", "when the app is written in German"},
		},
		notes: "The locale here is the language the text in your source code " +
			"is already written in, not one you want to translate into. " +
			"Everything else is measured against it.\n\n" +
			"Opt-in: a project without this has no translation bundle and no " +
			"toki dependency.",
		run: runI18nInit,
	})
	register(command{
		noun: "i18n", verb: "add", order: 20,
		summary: "Add a language to translate into",
		args:    "<locale>...",
		usage: [][2]string{
			{"de", "German"},
			{"de fr pt-BR", "several at once"},
		},
		notes: "A BCP 47 tag. Regional variants work and inherit: pt-BR falls " +
			"back to pt, and en-GB to en, so a catalog only carries what " +
			"actually differs.",
		run: runI18nAdd,
	})
	register(command{
		noun: "i18n", verb: "list", order: 30,
		summary: "Show the locales this project has",
		usage: [][2]string{
			{"", "every locale, and the file a translator edits"},
		},
		notes: "Read from the catalogs on disk rather than from project " +
			"config, so there is only ever one answer to what languages " +
			"this app speaks.",
		run: runI18nList,
	})
	register(command{
		noun: "i18n", verb: "edit", order: 40,
		summary: "Open the translation editor in a browser",
		usage: [][2]string{
			{"", "a UI over every catalog"},
		},
		notes: "toki's web UI. The alternative is hand-editing ICU messages " +
			"in .arb files, where a missing plural form is a quiet runtime " +
			"fallback rather than an error.",
		run: runI18nEdit,
	})
	register(command{
		noun: "i18n", verb: "check", order: 50,
		summary: "Fail if any translation is incomplete",
		usage: [][2]string{
			{"", "exits non-zero if a catalog is unfinished"},
		},
		notes: "For CI, before a release. Deliberately not part of `irgo " +
			"project check`: a half-translated app is a normal state to be " +
			"in, and failing every build over it would teach people to stop " +
			"running the check that catches everything else.",
		run: runI18nCheck,
	})
}
