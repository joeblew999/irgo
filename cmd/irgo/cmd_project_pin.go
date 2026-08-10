// Choosing which irgo a project builds against.
//
// There is no separate class of person here. Using irgo and working on irgo are
// the same activity pointed at different versions: a published release, a fork
// tag, or a checkout on your disk. `go tool irgo` reads whichever go.mod names,
// so one command covers all three and nothing else in the project changes.
package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

const upstreamModule = "github.com/stukennedy/irgo"

// runPin reports or changes the pinned CLI.
func runPin(args []string) error {
	if _, err := os.Stat("go.mod"); err != nil {
		return fmt.Errorf("no go.mod here — run irgo project pin from your project root")
	}
	if len(args) == 0 {
		return showPin()
	}

	// Bare words, like every other command: nothing in the CLI takes a flag
	// where a word will do.
	target := args[0]
	switch {
	case target == "local":
		dir := "."
		if len(args) > 1 {
			dir = args[1]
		}
		return pinLocal(dir)
	case target == "release":
		return pinRelease()
	default:
		return pinVersion(target)
	}
}

// looksLikeVersion reports whether target is a version rather than a word
// someone meant as a keyword. Without this check a typo is silently turned
// into a tag ("local" -> "vlocal") that no repository has, and the pin only
// fails later in `go mod tidy` — after go.mod has already been rewritten.
func looksLikeVersion(target string) bool {
	v := strings.TrimPrefix(target, "v")
	return v != "" && v[0] >= '0' && v[0] <= '9'
}

// showPin prints what the project builds against and where that came from.
func showPin() error {
	data, err := os.ReadFile("go.mod")
	if err != nil {
		return err
	}
	var require, replace string
	for _, raw := range strings.Split(string(data), "\n") {
		line := strings.TrimSpace(raw)
		if strings.HasPrefix(line, "replace "+upstreamModule) {
			if i := strings.Index(line, "=>"); i >= 0 {
				replace = strings.TrimSpace(line[i+2:])
			}
		}
		if strings.HasPrefix(line, upstreamModule+" v") || strings.HasPrefix(line, "require "+upstreamModule) {
			require = strings.TrimSpace(strings.TrimPrefix(line, "require "))
		}
	}

	fmt.Println("irgo project pin (from go.mod)")
	fmt.Println()
	if replace == "" {
		fmt.Printf("  published release: %s\n", strings.TrimSpace(require))
		fmt.Println("  `go tool irgo` builds that version from the module proxy.")
	} else if strings.HasPrefix(replace, ".") || strings.HasPrefix(replace, "/") {
		fmt.Printf("  local checkout: %s\n", replace)
		fmt.Println("  `go tool irgo` builds your working tree — edit the CLI and")
		fmt.Println("  the next command already reflects it. No install step.")
	} else {
		fmt.Printf("  fork: %s\n", replace)
		fmt.Println("  `go tool irgo` clones and builds that tag on demand.")
	}

	fmt.Println()
	fmt.Println("Change it:")
	fmt.Println("  irgo project pin release              track the published module")
	fmt.Println("  irgo project pin <version>            a published version")
	fmt.Println("  irgo project pin <owner>/<repo>@<tag> track a fork")
	fmt.Println("  irgo project pin local [dir]          build a checkout you are editing")
	return nil
}

// pinLocal points the project at a working tree, which is how you work ON the
// CLI: edit, run, no install, no tagging, no reinstall.
func pinLocal(dir string) error {
	abs, err := filepath.Abs(dir)
	if err != nil {
		return err
	}
	if _, err := os.Stat(filepath.Join(abs, "go.mod")); err != nil {
		return fmt.Errorf("%s is not a Go module — point it at an irgo checkout", abs)
	}
	// The strict check, not a substring one: "module github.com/stukennedy/
	// irgo-tools" contains the irgo path and is not irgo.
	if !isIrgoCheckout(abs) {
		return fmt.Errorf("%s is a Go module but not irgo (its go.mod declares a different module)", abs)
	}

	// go.work, not a replace in go.mod.
	//
	// go.mod is committed, so a replace pointing at /Users/someone/checkout
	// travels with the repository: it builds on one machine and nowhere else,
	// and CI fails on a path that does not exist. That is not a discipline
	// problem to be solved by remembering to undo it — it is the wrong file.
	//
	// go.work is gitignored (it already was, for gomobile), so a local pin
	// cannot be committed even deliberately, and CI — which has no go.work —
	// builds the published module without being told to. Go added workspaces
	// for exactly this.
	if err := dropCommittedLocalReplace(); err != nil {
		return err
	}
	if _, err := os.Stat("go.work"); err != nil {
		if _, err := runCommandQuiet(goBin(), "work", "init", "."); err != nil {
			return fmt.Errorf("creating go.work: %w", err)
		}
	}
	if _, err := runCommandQuiet(goBin(), "work", "use", abs); err != nil {
		return fmt.Errorf("adding %s to go.work: %w", abs, err)
	}

	fmt.Printf("Pinned to your checkout: %s\n", abs)
	fmt.Println("`go tool irgo` now builds that tree — edits take effect immediately.")
	fmt.Println()
	fmt.Println("Written to go.work, which is gitignored. go.mod still names the")
	fmt.Println("published version, so a commit, a teammate and CI are unaffected.")
	fmt.Println("Undo with: irgo project pin release")
	return nil
}

// dropCommittedLocalReplace removes a local-path replace an older irgo wrote
// into go.mod, so upgrading fixes the repository rather than leaving a pin that
// only works on one machine.
func dropCommittedLocalReplace() error {
	data, err := os.ReadFile("go.mod")
	if err != nil {
		return nil
	}
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, "replace "+upstreamModule) {
			continue
		}
		i := strings.Index(line, "=>")
		if i < 0 {
			continue
		}
		target := strings.TrimSpace(line[i+2:])
		if !strings.HasPrefix(target, ".") && !strings.HasPrefix(target, "/") {
			continue // a fork or a version, which is a real pin and stays
		}
		fmt.Println("  removed a local replace from go.mod — that only ever built here")
		return goModEdit("-dropreplace", upstreamModule)
	}
	return nil
}

// pinRelease drops any replace, returning to the published module.
func pinRelease() error {
	// The workspace first: it wins over go.mod, so dropping the replace while
	// leaving a `use` in place would report a release pin and keep building
	// the checkout.
	if err := dropWorkspaceUse(); err != nil {
		return err
	}

	// A replace naming a module and version is a real pin — a fork, usually —
	// and "release" means stop using the local checkout, not abandon it. A
	// project pinned to a fork that gained this command would otherwise be
	// dropped back to upstream, which has none of what it depends on, and the
	// only symptom would be code that stopped compiling.
	if pin := forkPin(); pin != "" {
		fmt.Printf("Using the pinned release again: %s\n", pin)
		fmt.Println("The local checkout is no longer in the workspace.")
		return tidy()
	}

	restore, err := snapshotGoMod()
	if err != nil {
		return err
	}
	if err := goModEdit("-dropreplace", upstreamModule); err != nil {
		return err
	}
	if err := tidy(); err != nil {
		restore()
		return err
	}
	fmt.Println("Pinned to the published module (replace removed).")
	fmt.Println("Change the version with: irgo project pin <version>")
	return nil
}

// pinVersion accepts a bare version for the published module, or
// owner/repo@version for a fork.
func pinVersion(target string) error {
	if strings.Contains(target, "/") {
		repo, version, ok := strings.Cut(target, "@")
		if !ok {
			return fmt.Errorf("a fork needs a version: irgo project pin %s@<tag>", target)
		}
		mod := repo
		if !strings.HasPrefix(mod, "github.com/") {
			mod = "github.com/" + mod
		}
		restore, err := snapshotGoMod()
		if err != nil {
			return err
		}
		if err := goModEdit("-replace", upstreamModule+"="+mod+"@"+version); err != nil {
			return err
		}
		if err := tidy(); err != nil {
			restore()
			return err
		}
		fmt.Printf("Pinned to fork %s %s\n", mod, version)
		// A fork keeps the upstream module path, so the proxy cannot serve it.
		fmt.Println()
		fmt.Println("A fork is fetched straight from GitHub. If this is the first one,")
		fmt.Printf("tell Go not to use the proxy for it:\n  go env -w GOPRIVATE='%s/*'\n",
			strings.TrimSuffix(mod, "/"+filepath.Base(mod)))
		return nil
	}

	if !looksLikeVersion(target) {
		return fmt.Errorf("%q is not a version. Did you mean one of:\n"+
			"  irgo project pin local [dir]     a checkout on this machine\n"+
			"  irgo project pin release         the latest published release\n"+
			"  irgo project pin v0.4.0          a published version\n"+
			"  irgo project pin owner/irgo@tag  a fork", target)
	}
	if !strings.HasPrefix(target, "v") {
		target = "v" + target
	}
	// Pinning rewrites go.mod before Go gets a chance to reject the version, so
	// a bad tag would otherwise leave the project unbuildable and the person
	// holding a broken go.mod they never edited.
	restore, err := snapshotGoMod()
	if err != nil {
		return err
	}
	if err := goModEdit("-dropreplace", upstreamModule); err != nil {
		return err
	}
	if err := goModEdit("-require", upstreamModule+"@"+target); err != nil {
		restore()
		return err
	}
	if err := tidy(); err != nil {
		restore()
		return fmt.Errorf("%w\n\ngo.mod left unchanged — %s was not pinned", err, target)
	}
	fmt.Printf("Pinned to published %s\n", target)
	return nil
}

// snapshotGoMod returns a function that puts go.mod (and go.sum) back as they
// are right now.
func snapshotGoMod() (func(), error) {
	mod, err := os.ReadFile("go.mod")
	if err != nil {
		return nil, err
	}
	sum, sumErr := os.ReadFile("go.sum")
	return func() {
		os.WriteFile("go.mod", mod, 0644)
		if sumErr == nil {
			os.WriteFile("go.sum", sum, 0644)
		}
	}, nil
}

// goCommand runs the Go toolchain irgo itself was built with.
func goCommand(args ...string) *exec.Cmd { return exec.Command(goBin(), args...) }

func goModEdit(args ...string) error {
	full := append([]string{"mod", "edit"}, args...)
	cmd := goCommand(full...)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("go mod edit failed: %s", strings.TrimSpace(string(out)))
	}
	return nil
}

// tidy resolves the new pin. -mod=mod because a tool directive plus a changed
// replace needs go.sum updated, which the default readonly mode refuses.
func tidy() error {
	cmd := goCommand("mod", "tidy")
	cmd.Env = append(os.Environ(), "GOFLAGS=-mod=mod")
	if out, err := cmd.CombinedOutput(); err != nil {
		// A failing tidy means the pin does not resolve, which used to be
		// reported as a warning while go.mod kept the unbuildable edit.
		// Callers restore the previous go.mod, so this has to be an error.
		return fmt.Errorf("go mod tidy: %s\n"+
			"If this is a fork, Go may need: go env -w GOPRIVATE='github.com/<owner>/*'",
			strings.TrimSpace(string(out)))
	}
	fmt.Println("go.mod updated — `go tool irgo` now uses it.")
	return nil
}

// projectReplacement returns the replacement this project pins irgo to, or ""
// when it tracks the published module (or there is no go.mod here).
func projectReplacement() string {
	data, err := os.ReadFile("go.mod")
	if err != nil {
		return ""
	}
	for _, raw := range strings.Split(string(data), "\n") {
		line := strings.TrimSpace(raw)
		if !strings.HasPrefix(line, "replace "+upstreamModule) {
			continue
		}
		if i := strings.Index(line, "=>"); i >= 0 {
			return strings.TrimSpace(line[i+2:])
		}
	}
	return ""
}

// dropWorkspaceUse removes the irgo checkout from go.work, and the file too if
// nothing else is left using it.
//
// Leaving an empty go.work behind is not harmless: it changes module
// resolution for every command run in this directory, and a workspace with one
// bare `use .` is a thing to debug later with no reason to exist.
func dropWorkspaceUse() error {
	data, err := os.ReadFile("go.work")
	if err != nil {
		return nil
	}
	var uses []string
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		line = strings.TrimPrefix(line, "use ")
		line = strings.Trim(line, "()\t ")
		if line == "" || strings.HasPrefix(line, "go ") || line == "use" {
			continue
		}
		uses = append(uses, line)
	}

	for _, u := range uses {
		if u == "." {
			continue
		}
		abs := u
		if !filepath.IsAbs(abs) {
			abs, _ = filepath.Abs(u)
		}
		if isIrgoCheckout(abs) {
			if _, err := runCommandQuiet(goBin(), "work", "edit", "-dropuse", u); err != nil {
				return err
			}
			fmt.Printf("  removed %s from go.work\n", u)
		}
	}

	// Re-read: if only "." remains, the workspace exists for no reason.
	data, err = os.ReadFile("go.work")
	if err != nil {
		return nil
	}
	remaining := 0
	for _, line := range strings.Split(string(data), "\n") {
		t := strings.TrimSpace(line)
		if strings.HasPrefix(t, "use ") && strings.TrimSpace(strings.TrimPrefix(t, "use ")) != "." {
			remaining++
		}
	}
	if remaining == 0 {
		_ = os.Remove("go.work")
		_ = os.Remove("go.work.sum")
	}
	return nil
}
func init() {
	register(command{
		noun: "project", verb: "pin", order: 3,
		summary: "Choose which irgo this project builds against",
		args:    "[local|release|<version>]",
		usage: [][2]string{
			{"", "Show the current pin and where it came from"},
			{"local [dir]", "Build a checkout you are editing"},
			{"release", "Track the published module"},
			{"<version>", "A published version, e.g. v0.4.0"},
			{"<owner>/<repo>@<tag>", "A fork"},
		},
		notes: "`local` writes go.work, which is gitignored, so working on the CLI cannot\n" +
			"leak into a commit and CI keeps building the published module.\n\n" +
			"go.mod is the pin everything else uses: `go tool irgo` builds whatever it names, so there is\n" +
			`nothing installed globally to fall out of step. A pin that does not resolve
leaves go.mod untouched rather than half-written.

A fork keeps the upstream module path, so the proxy cannot serve it:
  go env -w GOPRIVATE='github.com/<owner>/*'`,
	})
}

// forkPin is the replace naming a module and version, or "" if there is none.
//
// Distinguished from a local-path replace, which is a developer's checkout and
// is meant to be dropped, and from no replace at all.
func forkPin() string {
	data, err := os.ReadFile("go.mod")
	if err != nil {
		return ""
	}
	for _, raw := range strings.Split(string(data), "\n") {
		line := strings.TrimSpace(raw)
		if !strings.HasPrefix(line, "replace "+upstreamModule) {
			continue
		}
		i := strings.Index(line, "=>")
		if i < 0 {
			continue
		}
		target := strings.TrimSpace(line[i+2:])
		if strings.HasPrefix(target, ".") || strings.HasPrefix(target, "/") {
			return "" // a local path, which release is meant to undo
		}
		return target
	}
	return ""
}
