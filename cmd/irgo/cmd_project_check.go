// Reporting drift instead of fixing it.
//
// Two questions a project needs answered before an upgrade surprises it, both
// of which used to live as shell in a workflow file and so only ever ran in
// CI:
//
//   - `upgrade --check`: has anything the FRAMEWORK owns been hand-edited?
//     Those files are replaced by `irgo project upgrade`, so an edit there is
//     work that is going to be silently thrown away. Every project wants this.
//
//   - `new --check`: does regenerating change a tracked file at all? Only true
//     of a repo whose whole premise is being unmodified CLI output — the demo,
//     or any example repo. It is the check that caught appicon.png being
//     overwritten.
//
// Both are commands rather than YAML so they answer the question locally,
// before a push, which is the only time the answer is cheap.
package main

import (
	"fmt"
	"io/fs"
	"os"
	"strings"
)

// templateDrift lists template files whose content differs from what is on
// disk. onlyFrameworkOwned narrows it to files the CLI maintains.
func templateDrift(modulePath string, onlyFrameworkOwned bool) ([]string, error) {
	var drifted []string

	err := fs.WalkDir(templateFS, "templates", func(path string, d fs.DirEntry, werr error) error {
		if werr != nil || d.IsDir() {
			return werr
		}
		rel := strings.TrimPrefix(path, "templates/")
		if rel == "github" || strings.HasPrefix(rel, "github/") {
			return nil // written by irgo project ci, checked below
		}
		rel = strings.TrimSuffix(rel, ".tmpl")

		if onlyFrameworkOwned && !isFrameworkOwned(rel) {
			return nil
		}
		// Files the project owns are seeded once and then its own, so they are
		// expected to differ and are not drift in either mode.
		if !onlyFrameworkOwned && isProjectOwned(rel) {
			return nil
		}

		want, rerr := templateFS.ReadFile(path)
		if rerr != nil {
			return rerr
		}
		body := renderTemplateBody(string(want), modulePath)

		have, herr := os.ReadFile(rel)
		if herr != nil {
			// Absent is not drift. The native shells under ios/ and android/
			// are gitignored and scaffolded on demand, so a fresh clone has
			// none of them — and nothing that does not exist can be silently
			// overwritten, which is the only thing this reports.
			return nil
		}
		if string(have) != body {
			drifted = append(drifted, rel)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	drifted = append(drifted, ciDrift(modulePath)...)
	return drifted, nil
}

// ciDrift reports workflows that no longer match what `irgo project ci` writes.
func ciDrift(modulePath string) []string {
	var out []string
	_ = fs.WalkDir(ciTemplates, "templates/github", func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		rel := ".github/" + strings.TrimPrefix(path, "templates/github/")
		data, rerr := ciTemplates.ReadFile(path)
		if rerr != nil {
			return nil
		}
		want := renderCITemplate(string(data), baseName(modulePath))
		have, herr := os.ReadFile(rel)
		if herr != nil {
			// A project with no .github/ has not opted into CI; irgo project
			// ci writes it on request.
			return nil
		}
		if string(have) != want {
			out = append(out, rel)
		}
		return nil
	})
	return out
}

// isProjectOwned reports whether a template file is seeded once and then
// belongs to the project — the same set `new` refuses to overwrite.
func isProjectOwned(rel string) bool {
	switch rel {
	case "go.mod", "README.md", packageConfigFile, "appicon.png":
		return true
	}
	return false
}

// runUpgradeCheck reports framework-owned files that have been hand-edited.
func runUpgradeCheck() error {
	modulePath, err := checkPreamble()
	if err != nil {
		return err
	}
	drifted, err := templateDrift(modulePath, true)
	if err != nil {
		return err
	}
	if len(drifted) == 0 {
		fmt.Println("Framework-owned files match irgo — nothing an upgrade would overwrite.")
		return nil
	}

	fmt.Println("These files are maintained by irgo and differ from it:")
	fmt.Println()
	for _, f := range drifted {
		fmt.Printf("  %s\n", f)
	}
	fmt.Println()
	fmt.Println("`irgo project upgrade` replaces them, so any edit here is lost on the")
	fmt.Println("next upgrade. If the change was deliberate, keep it somewhere the CLI")
	fmt.Println("does not own — a plugin file, or a workflow of your own alongside the")
	fmt.Println("generated ones.")
	fmt.Println()
	fmt.Println("  Take the current versions:  irgo project upgrade")
	fmt.Println("  See what changed:           irgo project upgrade --diff")
	return fmt.Errorf("%d framework-owned file(s) differ from irgo", len(drifted))
}

// runNewCheck reports any tracked file that regenerating would change. This is
// the check for a repo that IS generated output; a normal project is supposed
// to differ, so it fails there by design.
func runNewCheck() error {
	modulePath, err := checkPreamble()
	if err != nil {
		return err
	}
	drifted, err := templateDrift(modulePath, false)
	if err != nil {
		return err
	}
	if len(drifted) == 0 {
		fmt.Println("This project matches `irgo project new` output.")
		return nil
	}

	fmt.Println("Regenerating would change these files:")
	fmt.Println()
	for _, f := range drifted {
		fmt.Printf("  %s\n", f)
	}
	fmt.Println()
	fmt.Println("This check asserts the repo IS unmodified CLI output, which is true of")
	fmt.Println("example and demo repos and not of a real project. If this is your app,")
	fmt.Println("you want `irgo project upgrade --check` instead — it only reports the")
	fmt.Println("files the framework owns.")
	fmt.Println()
	fmt.Println("Otherwise fix the template rather than this repo.")
	return fmt.Errorf("%d file(s) differ from `irgo project new` output", len(drifted))
}

func checkPreamble() (string, error) {
	if _, err := os.Stat("go.mod"); err != nil {
		return "", fmt.Errorf("no go.mod here — run this from your project root")
	}
	modulePath, err := getModulePath()
	if err != nil {
		return "", fmt.Errorf("could not determine module path: %w", err)
	}
	return modulePath, nil
}
