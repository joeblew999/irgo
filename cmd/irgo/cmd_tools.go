// Toolchain: installing, verifying and removing the tools a build needs
// (templ, air, gomobile/gobind, mingw-w64) plus the generated assets they
// produce. Every install here has a matching uninstall.
package main

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

// Where irgo keeps what it installs.
//
// These paths were spelled out with filepath.Join(homeDir(), ".irgo", ...) in
// seven places across four files, so nothing owned the layout and moving it
// meant finding them all. Every caller asks here now.
func irgoHome() string { return filepath.Join(homeDir(), ".irgo") }

// irgoBinDir holds single-binary tools irgo downloads.
func irgoBinDir() string { return filepath.Join(irgoHome(), "bin") }

// goTools are the tools irgo installs with `go install`.
//
// This list was written out in five places — install, remove, doctor and two
// tests — and had already drifted: install was missing gobind, so `tools
// install` left behind a tool that `tools remove` then offered to delete.
func goTools() []string { return []string{"templ", "air", "gomobile", "gobind"} }

// runTempl generates templ files
func runTempl() error {
	if err := ensureGoTool("templ"); err != nil {
		return err
	}

	fmt.Println("Generating templ files...")
	return runCommand("templ", "generate")
}

// goToolPkg maps a tool name to the module to `go install`. templ is pinned to
// the project's go.mod version: the generator and the library must match, and
// @latest drift breaks generated code.
// pinTempl is the fallback templ version, used when a project's own cannot be
// read from its go.mod.
const pinTempl = "v0.3.977"

// pinAir is the hot-reload watcher's version. Pinned for the same reason as
// everything else irgo installs: a build that changes without a commit is a
// build nobody can reproduce.
const pinAir = "v1.63.0"

// sops decrypts the secrets a deploy needs, when a project keeps them
// encrypted in the repository rather than in a keychain. Installed when a
// fnox.toml actually declares a sops provider, not before.
const pinSops = "v3.12.2"

func goToolPkg(name string) string {
	switch name {
	case "templ":
		// The project's own templ version first: the generator and the runtime
		// package have to agree, or generated code fails to compile against
		// the library. Falling back to @latest made that disagreement possible
		// whenever the version could not be read.
		if v := templVersionFromGoMod(); v != "" {
			return "github.com/a-h/templ/cmd/templ@v" + v
		}
		return "github.com/a-h/templ/cmd/templ@" + pinTempl
	case "air":
		return "github.com/air-verse/air@" + pinAir
	case "sops":
		return "github.com/getsops/sops/v3/cmd/sops@" + pinSops
	case "gomobile", "gobind":
		// Same commit as the x/mobile checkout gomobile builds against.
		// Installing @latest while pinning the source is how a tool ends up
		// disagreeing with the code it drives: gomobile started requiring
		// golang.org/x/mobile in the project's dependency graph, and every
		// mobile build broke with no commit in this repository or the
		// project's.
		return "golang.org/x/mobile/cmd/" + name + "@" + pinXMobile
	}
	return ""
}

// irgoToolsDir holds one marker file per tool irgo installed itself, so
// uninstall can remove exactly those and leave a developer's own copies alone.
// Without it an uninstall would either delete tools irgo never owned or, if it
// played safe and skipped them, leave a half-provisioned machine that silently
// fails to re-provision.
func irgoToolsDir() string {
	return filepath.Join(irgoHome(), "tools")
}

func markToolInstalled(name string) {
	dir := irgoToolsDir()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return
	}
	// Record which version was installed, not just that one was. Asking the
	// tool afterwards does not work: air and gobind have no version flag at
	// all, and parsing their usage text finds the Go version instead, which is
	// how doctor first reported air as drifted when it was not.
	body := "irgo " + version + "\n"
	if pkg := goToolPkg(name); pkg != "" {
		if _, v, ok := strings.Cut(pkg, "@"); ok {
			body += "pin " + v + "\n"
		}
	}
	_ = os.WriteFile(filepath.Join(dir, name), []byte(body), 0o644)
}

// clearToolMarker forgets that irgo installed something. Markers left behind
// keep ~/.irgo populated, which makes an otherwise-clean machine look
// provisioned — and a machine that cannot reach a known-clean state hides
// provisioning bugs.
func clearToolMarker(name string) {
	_ = os.Remove(filepath.Join(irgoToolsDir(), name))
}

func toolInstalledByIrgo(name string) bool {
	_, err := os.Stat(filepath.Join(irgoToolsDir(), name))
	return err == nil
}

// ensureGoTool installs a Go-based tool when missing rather than printing an
// install command for someone to copy. `go install` behaves the same on macOS,
// Linux and Windows, so this needs no per-OS branching.
func ensureGoTool(name string) error {
	if _, err := exec.LookPath(name); err == nil {
		return nil
	}
	// mise first when it can provide the tool: it installs into a directory
	// the developer's own version manager can see and clean up, rather than a
	// GOBIN that irgo then has to prepend to PATH. Only some are in its
	// registry — templ, gomobile and gobind are not — so `go install` remains
	// the answer for the rest, and the fallback for all of them.
	if spec, ok := miseSpec(name); ok {
		if bin := miseTool(spec, name); bin != "" {
			markToolInstalled(name)
			return prependToPATH(filepath.Dir(bin))
		}
	}
	pkg := goToolPkg(name)
	if pkg == "" {
		return fmt.Errorf("%s not found, and no install source is known for it", name)
	}
	fmt.Printf("%s not found — installing %s...\n", name, pkg)
	if err := runCommand(goBin(), "install", pkg); err != nil {
		return fmt.Errorf("installing %s: %w", name, err)
	}
	markToolInstalled(name)
	// `go install` lands in GOBIN (default $GOPATH/bin), which is often absent
	// from PATH — prepend it so the tool resolves for the rest of this process.
	if dir := gobinDir(); dir != "" {
		_ = os.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
	}
	if _, err := exec.LookPath(name); err != nil {
		return fmt.Errorf("%s still not found after installing it — is %s on your PATH?", name, gobinDir())
	}
	return nil
}

// miseSpec maps a tool to what mise is asked for, at irgo's own pin.
//
// Every one of these was verified to resolve — `mise ls-remote <spec>` lists
// the exact version. An earlier version of this table claimed templ, gomobile
// and tailwindcss were not available, on the strength of grepping
// `mise registry`, which is capped at a thousand entries and lists none of
// them. The backends have them all.
//
// The pin travels in the spec, so no mise.toml is involved and the developer's
// own config is never consulted: `mise install templ@0.3.977` means that
// version whatever their config says.
func miseSpec(name string) (spec string, ok bool) {
	v := func(pin string) string { return strings.TrimPrefix(pin, "v") }
	switch name {
	case "templ":
		if p := templVersionFromGoMod(); p != "" {
			return "go:github.com/a-h/templ/cmd/templ@" + p, true
		}
		return "go:github.com/a-h/templ/cmd/templ@" + v(pinTempl), true
	// The short registry name, not the backend-qualified one. mise installs
	// sops@3.12.2 and aqua:getsops/sops@3.12.2 into different directories, so
	// asking for the qualified form would ignore a sops the developer already
	// had and install a second copy beside it. tailwindcss is the exception:
	// it is not in the registry, so only the backend spec resolves.
	case "air":
		return "air@" + v(pinAir), true
	case "sops":
		return "sops@" + v(pinSops), true
	case "gomobile":
		return "go:golang.org/x/mobile/cmd/gomobile@" + pinXMobile, true
	case "gobind":
		return "go:golang.org/x/mobile/cmd/gobind@" + pinXMobile, true
	}
	return "", false
}

// miseSpecFor covers every tool with a mise spec, including the two that are
// not go-installed.
func miseSpecFor(name string) (string, bool) {
	switch name {
	case "tailwindcss":
		return "aqua:tailwindlabs/tailwindcss@" + strings.TrimPrefix(pinTailwind, "v"), true
	case "node":
		return "node@" + pinNode, true
	case "jdk":
		// doctor calls the row jdk; mise calls it java. Missing this pruned
		// the marker for a JDK that was installed and in use, which would have
		// left tools remove unable to give it back.
		return "java@" + pinJDK, true
	}
	return miseSpec(name)
}

// prependToPATH makes a freshly installed tool resolvable for the rest of
// this process, wherever it landed.
func prependToPATH(dir string) error {
	if dir == "" {
		return nil
	}
	return os.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
}

// runCSS rebuilds the Tailwind stylesheet. static/css/output.css is generated
// and gitignored, but it is embedded into every build — so skipping it ships an
// unstyled app. No-op for projects without a Tailwind entry point.
func runCSS() error {
	const in, out = "static/css/input.css", "static/css/output.css"
	if _, err := os.Stat(in); err != nil {
		return nil // not a Tailwind project
	}
	bin, err := ensureTailwind()
	if err != nil {
		return err
	}
	fmt.Println("Building CSS...")
	return runCommand(bin, "-i", in, "-o", out, "--minify")
}

// ensureAssets regenerates everything that is gitignored yet embedded into a
// build: _templ.go, the Tailwind stylesheet, and whatever the project
// generates for itself. Every build path runs this, so a fresh clone builds
// correctly without the caller sequencing it by hand.
func ensureAssets() error {
	if err := runTempl(); err != nil {
		return err
	}
	return runCSS()
}

// ensureAssetsAndGenerate is ensureAssets plus the project's own generators.
//
// Kept separate because ensureAssets is the hot path: .air.toml runs
// `irgo project assets` on every save, and generators are not save-cheap. A
// project whose generators shell out to `go test` turns one keystroke into a
// test compile, and a watch loop into a fork bomb — 95 of them, the first time
// this ran. Generation belongs where correctness matters and latency does not:
// builds, deploys and tests.
func ensureAssetsAndGenerate() error {
	if err := ensureAssets(); err != nil {
		return err
	}
	return runGoGenerate()
}

// runGoGenerate runs the project's own generators.
//
// Some projects derive content from source rather than typing it — a docs site
// generating its API reference from go/doc, say. That belongs to the project,
// not to irgo, so this runs the standard Go mechanism and stays ignorant of
// what any particular project generates. A project with no //go:generate
// directives spends a few milliseconds here and produces nothing.
//
// It runs after templ and Tailwind, so a generator can rely on a package that
// compiles.
func runGoGenerate() error {
	if !hasGoGenerate() {
		return nil
	}
	fmt.Println("Running go generate...")
	return runCommand("go", "generate", "./...")
}

// hasGoGenerate reports whether the project declares any generator, so the
// common case prints nothing and pays nothing.
func hasGoGenerate() bool {
	found := false
	_ = filepath.Walk(".", func(path string, info os.FileInfo, err error) error {
		if err != nil || found {
			return nil
		}
		if info.IsDir() {
			switch info.Name() {
			case "node_modules", "vendor", "build", "tmp", ".git":
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(path, ".go") {
			return nil
		}
		data, err := os.ReadFile(path)
		if err == nil && bytes.Contains(data, []byte("\n//go:generate")) {
			found = true
		}
		return nil
	})
	return found
}

// installTools installs required development tools
func installTools() error {
	fmt.Println("Installing irgo development tools...")
	fmt.Println()

	// The same table doctor reports from, so `tools install` cannot provide a
	// different set of tools than `tools doctor` describes. Each row carries
	// how to get itself; this is the eager form of what every build does
	// lazily when a call needs something.
	//
	// Skipped here: the heavy toolchains. The Android SDK is hundreds of
	// megabytes and only an Android build needs it, so it stays lazy — the
	// closing message says so.
	eager := map[string]bool{}
	for _, name := range goTools() {
		eager[name] = true
	}
	eager["tailwindcss"] = true

	for _, row := range toolLocators() {
		if !eager[row.name] {
			continue
		}
		if row.path != "" {
			fmt.Printf("  %s: already installed\n", row.name)
			continue
		}
		if row.ensure == nil {
			fmt.Printf("  %s: not something irgo can install\n", row.name)
			continue
		}
		if err := row.ensure(); err != nil {
			fmt.Printf("  Warning: %v\n", err)
			continue
		}
		fmt.Printf("  %s: installed\n", row.name)
	}

	if err := installOSPackages(); err != nil {
		fmt.Printf("Warning: %v\n", err)
	}

	fmt.Println()
	fmt.Println("Initializing gomobile...")
	if err := runCommand("gomobile", "init"); err != nil {
		fmt.Printf("Warning: gomobile init failed: %v\n", err)
	}

	fmt.Println()
	fmt.Println("Done. Nothing else is required up front:")
	fmt.Println("  Android — irgo installs the JDK, SDK and NDK on the first")
	fmt.Println("            android build or run. Android Studio is NOT needed.")
	fmt.Println("            Check it with: irgo tools doctor android")
	if runtime.GOOS == "darwin" {
		fmt.Println("  iOS     — needs Xcode from the App Store (the one thing irgo")
		fmt.Println("            cannot install for you).")
		fmt.Println("  Windows — irgo installs mingw-w64 when cross-compiling.")
	} else {
		fmt.Printf("  iOS     — deploys to a physical device from %s via xtool +\n", runtime.GOOS)
		fmt.Println("            a Swift toolchain (https://xtool.sh); run 'xtool setup'")
		fmt.Println("            once. The Simulator itself still needs macOS.")
	}
	return nil
}

// uninstallTools is the exact inverse of `irgo tools install`: it removes the
// Go tools irgo installed, and nothing else. Every install path in the CLI has
// a matching uninstall — without one you cannot return a machine to a known
// state, and a provisioning bug hides behind whatever was left lying around
// instead of surfacing on the next run.
//
// Marker-guarded: a tool irgo did not install is reported and kept, so a
// developer's own templ/air survives. Pass all to override that.
// removalTally counts outcomes so the summary reflects everything considered,
// not just the Go tools.
type removalTally struct{ removed, kept, missing int }

// report prints one aligned line and records the outcome. Aligned columns
// because the previous output interleaved three kinds of thing in one flat
// list, and you could not tell a Go tool from a Homebrew package.
func (t *removalTally) report(name, state, detail string) {
	switch state {
	case "removed":
		t.removed++
	case "kept":
		t.kept++
	default:
		t.missing++
	}
	if detail != "" {
		fmt.Printf("  %-14s %-9s %s\n", name, state, detail)
		return
	}
	fmt.Printf("  %-14s %s\n", name, state)
}

// uninstallTools is the exact inverse of `irgo tools install`: it removes what
// irgo installed, and nothing else. Every install path in the CLI has a
// matching removal — without one you cannot return a machine to a known state,
// and a provisioning bug hides behind whatever was left lying around instead of
// surfacing on the next run.
//
// Marker-guarded: a tool irgo did not install is reported and kept, so a
// developer's own templ or gomobile survives. Pass all to override that.
// uninstallTools removes what irgo installed — all of it, including Android.
//
// scope narrows it to one area ("android"), yes skips the prompt, all also
// removes copies irgo did not install, and keepJDK preserves the managed JDK,
// which is the slowest thing here to re-download.
func uninstallTools(scope string, all, yes, keepJDK bool) error {
	var p removalPlan
	switch scope {
	case "android":
		planAndroid(&p, keepJDK)
	default:
		planTools(&p, all)
		planHostPackages(&p, all)
		planAndroid(&p, keepJDK)
	}

	if p.empty() {
		fmt.Println("Nothing to remove — irgo has not installed anything here.")
		if len(p.kept) > 0 {
			fmt.Printf("Present but not installed by irgo: %s\n", strings.Join(p.kept, ", "))
			fmt.Println("  --all removes them too.")
		}
		return nil
	}

	ok, err := confirmRemoval(&p, yes)
	if err != nil {
		return err
	}
	if !ok {
		return nil
	}

	fmt.Println("Removing:")
	var t removalTally
	for _, act := range p.acts {
		act(&t)
	}

	// Build residue from mobile builds: the temp x/mobile clone and the local
	// go.work that ensureMobileBuildSetup generates.
	_ = os.RemoveAll(filepath.Join(os.TempDir(), "golang-mobile"))
	_ = os.Remove("go.work")
	_ = os.Remove("go.work.sum")
	pruneIrgoStateDir()

	fmt.Printf("\n%d removed, %d kept, %d absent.\n", t.removed, t.kept, t.missing)
	return nil
}

func checkTool(name, installCmd string) error {
	_, err := exec.LookPath(name)
	if err != nil {
		return fmt.Errorf("%s not found. Install with: %s", name, installCmd)
	}
	return nil
}

// pruneIrgoStateDir removes ~/.irgo once nothing is left in it. Marker files
// are irgo's own bookkeeping, so leaving an empty tree behind makes an
// otherwise-clean machine look provisioned.
func pruneIrgoStateDir() {
	// os.Remove only succeeds on an empty directory, which is exactly the
	// condition we want — never delete state that is still in use.
	_ = os.Remove(irgoToolsDir())
	_ = os.Remove(irgoHome())
}
