// The reference material an AI agent needs to work on an irgo project.
//
// Every actor here — the maintainer, the developer, and the assistant either
// of them is driving — writes Datastar attributes by hand, and getting one
// wrong fails in the one way nothing catches: the page renders, the markup
// looks right, and the binding silently does nothing. `data-signals-theme`
// instead of `data-signals:theme` cost a full debugging session, and it was
// found by reading this skill rather than by any test.
//
// So the skills are not documentation to go and find. They are synced into the
// project like any other generated asset, and for the same reason: something
// the build guarantees is present is something nobody has to remember.
//
// They arrive the way Morpheus's stylesheets do — copied out of the Go module
// cache. A module zip carries its dotfiles, so `.claude/skills` ships inside
// the module, which means the skill a project gets is the one that shipped
// with the version its go.mod names. There is no separate download, nothing to
// pin twice, and no way for the two to drift.
//
// That also settles where irgo's own copy lives. irgo is a module, so its
// repository root .claude/skills is at once the copy its own maintainers read
// and the copy every project is served from — one file, not a canonical
// original plus a template duplicate that quietly falls behind it.
package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

const irgoModule = "github.com/stukennedy/irgo"

// skillsDir is where agents look. Claude Code reads it, and the convention is
// common enough that others do too.
var skillsDir = filepath.Join(".claude", "skills")

// skillModules are the modules that ship skills inside their module zip.
//
// Discovered from go.mod rather than listed here. A UI kit, an i18n library,
// anything a project depends on may carry the reference for using it well, and
// a hardcoded list would mean irgo had to know each one by name and be
// released before a project could benefit.
//
// Direct requirements only: an indirect dependency is not something the
// project chose, and its opinions are not what someone working here needs.
func skillModules() []string {
	mods := []string{irgoModule}

	out, err := exec.Command(goBin(), "mod", "edit", "-json").Output()
	if err != nil {
		return mods
	}
	var f struct {
		Require []struct {
			Path     string
			Indirect bool
		}
	}
	if err := json.Unmarshal(out, &f); err != nil {
		return mods
	}
	for _, r := range f.Require {
		if !r.Indirect && r.Path != irgoModule {
			mods = append(mods, r.Path)
		}
	}
	return mods
}

// syncSkills copies every available skill into the project.
//
// Called from ensureAssets, so it happens on every build rather than being a
// command someone has to know about — which is the whole point, since the
// actor who most needs the skill is the one who does not know it exists.
//
// Silent when there is nothing to do. A module that is not a dependency is
// skipped, and a file that already matches is not rewritten.
func syncSkills() error {
	for _, mod := range skillModules() {
		dir := moduleDir(mod)
		if dir == "" {
			// Not a dependency. Nothing to say: a project that does not use
			// Morpheus should not hear about Morpheus's skills.
			continue
		}
		src := filepath.Join(dir, ".claude", "skills")
		if _, err := os.Stat(src); err != nil {
			// A module that ships none, or a release that moved them. Neither
			// is this build's problem.
			continue
		}
		if err := copySkillTree(src, skillsDir); err != nil {
			return fmt.Errorf("syncing skills from %s: %w", mod, err)
		}
	}
	return nil
}

// copySkillTree copies the skill directories under src into dst.
//
// Directories only, and only one level down: a skill is a directory holding a
// SKILL.md, and the files that sit beside them at the top level are the source
// module's own bookkeeping rather than anything a project should receive.
func copySkillTree(src, dst string) error {
	entries, err := os.ReadDir(src)
	if err != nil {
		return err
	}
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		from := filepath.Join(src, e.Name())
		to := filepath.Join(dst, e.Name())

		// Copying a directory onto itself, which is what irgo's own repository
		// does, has to be a no-op rather than an error.
		if sameDir(from, to) {
			continue
		}
		if err := copyDirFiles(from, to); err != nil {
			return err
		}
	}
	return nil
}

// copyDirFiles copies a skill's files, including any nested under it.
func copyDirFiles(from, to string) error {
	return filepath.WalkDir(from, func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		rel, err := filepath.Rel(from, path)
		if err != nil {
			return err
		}
		return copyGenerated(path, filepath.Join(to, rel))
	})
}

// A skill irgo keeps its own copy of, and where that copy came from.
//
// Only skills that do not ship in a Go module need one of these. Morpheus's
// travel inside its module, so their version is whatever go.mod resolves and
// there is nothing here to record or to go stale. Datastar's do not: irgo does
// not depend on datapages and should not take a whole module to carry a
// document, so irgo vendors it and this is the record of that.
//
// commit is the commit that last touched the file, not the repository head
// when it was taken. That is the thing a check has to compare against — a head
// moves for reasons that have nothing to do with the skill.
type skillSource struct {
	name    string
	repo    string // https://github.com/<owner>/<repo>
	ref     string // branch the path is read from
	path    string // path within the repository
	commit  string // last commit touching path, when vendored
	taken   string // when it was vendored
	license string
	why     string
}

// vendoredSkills is the single record of where irgo's own skills came from.
//
// In Go rather than a data file on purpose: irgo has no TOML dependency and a
// skill record is not worth acquiring one, and a record the updater reads
// directly cannot disagree with a record a human reads.
var vendoredSkills = []skillSource{{
	name:    "datastar",
	repo:    "https://github.com/romshark/datapages",
	ref:     "main",
	path:    ".skills/datastar/SKILL.md",
	commit:  "b895b239ee79cb6d56761d32e258f4cbeddcb9c3",
	taken:   "2026-08-10",
	license: "MIT",
	why: "irgo ships Datastar, so the reference for writing it belongs with " +
		"irgo. Taking a module dependency on datapages to carry one document " +
		"would put its whole graph in every project that builds an irgo app.",
}}

// rawURL is where the file is fetched from to check it.
func (s skillSource) rawURL() string {
	repo := strings.TrimPrefix(s.repo, "https://github.com/")
	return "https://raw.githubusercontent.com/" + repo + "/" + s.ref + "/" + s.path
}

// localPath is irgo's copy.
func (s skillSource) localPath() string {
	return filepath.Join(skillsDir, s.name, "SKILL.md")
}

func runSkills(args []string) error {
	switch {
	case hasFlag(args, "--check"):
		return skillsCheck()
	case hasFlag(args, "--update"):
		return skillsUpdate()
	case hasFlag(args, "--sources"):
		return skillsSources()
	}
	if err := syncSkills(); err != nil {
		return err
	}
	return skillsReport()
}

// skillsReport lists what the project ended up with, and from where.
func skillsReport() error {
	entries, err := os.ReadDir(skillsDir)
	if err != nil {
		fmt.Println("No skills. This is not an irgo project, or irgo is not a dependency of it.")
		return nil
	}
	var names []string
	for _, e := range entries {
		if e.IsDir() {
			names = append(names, e.Name())
		}
	}
	if len(names) == 0 {
		fmt.Println("No skills.")
		return nil
	}
	fmt.Printf("%s\n\n", skillsDir)
	for _, n := range names {
		fmt.Printf("  %s\n", n)
	}
	fmt.Printf("\n%d skill(s). Refreshed on every build; edit upstream, not here.\n", len(names))
	return nil
}

// skillsSources prints where each vendored skill came from.
func skillsSources() error {
	fmt.Println("Vendored — irgo keeps its own copy, so these can go stale:")
	fmt.Println()
	for _, s := range vendoredSkills {
		fmt.Printf("  %s\n", s.name)
		fmt.Printf("    from     %s/blob/%s/%s\n", s.repo, s.ref, s.path)
		fmt.Printf("    commit   %s\n", s.commit)
		fmt.Printf("    taken    %s (%s)\n", s.taken, s.license)
		fmt.Printf("    %s\n\n", wrapAt(s.why, 68, "    "))
	}
	// Only the dependencies that actually carry skills. Listing every direct
	// requirement would name a dozen modules that ship none, which reads as a
	// dozen things to go and look at.
	var carrying []string
	for _, mod := range skillModules() {
		if mod == irgoModule {
			continue
		}
		if dir := moduleDir(mod); dir != "" {
			if _, err := os.Stat(filepath.Join(dir, ".claude", "skills")); err == nil {
				carrying = append(carrying, mod)
			}
		}
	}
	if len(carrying) > 0 {
		fmt.Println("From the module cache — version is go.mod's, cannot drift:")
		fmt.Println()
		for _, mod := range carrying {
			fmt.Printf("  %s\n", mod)
		}
		fmt.Println()
	}
	fmt.Println("Check the vendored ones against upstream:  irgo project skills --check")
	return nil
}

// skillsCheck compares each vendored skill with upstream.
//
// The reason this exists: a vendored copy is a copy, and the failure mode of a
// copy is that it stops being one without anything happening. Nothing else in
// irgo can notice — the build does not read it and no test covers upstream.
func skillsCheck() error {
	client := &http.Client{Timeout: 30 * time.Second}
	stale := 0

	for _, s := range vendoredSkills {
		have, err := os.ReadFile(s.localPath())
		if err != nil {
			fmt.Printf("  %-12s missing: %v\n", s.name, err)
			stale++
			continue
		}
		resp, err := client.Get(s.rawURL())
		if err != nil {
			fmt.Printf("  %-12s could not reach upstream: %v\n", s.name, err)
			continue
		}
		body, readErr := io.ReadAll(resp.Body)
		resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			// A moved path looks exactly like a stale copy from here, and the
			// fix is not the same, so do not call it either one.
			fmt.Printf("  %-12s upstream returned %s — the path may have moved:\n               %s\n",
				s.name, resp.Status, s.rawURL())
			continue
		}
		if readErr != nil {
			fmt.Printf("  %-12s could not read upstream: %v\n", s.name, readErr)
			continue
		}
		if string(body) == string(have) {
			fmt.Printf("  %-12s up to date\n", s.name)
			continue
		}
		stale++
		fmt.Printf("  %-12s CHANGED upstream (%d bytes here, %d there)\n",
			s.name, len(have), len(body))
		fmt.Printf("               %s\n", s.rawURL())
	}

	if stale > 0 {
		fmt.Printf("\n%d skill(s) differ from upstream. Take the change with:\n", stale)
		fmt.Println("  irgo project skills --update")
		fmt.Println()
		fmt.Println("Then update the commit in cmd/irgo/skills.go, which is what")
		fmt.Println("records the version this copy was taken at.")
	}
	return nil
}

// skillsUpdate takes upstream's version of each vendored skill.
//
// Only runnable from irgo's own checkout: the copy it writes is the one every
// project is served from, so updating it anywhere else would write a file the
// next build overwrites and change nothing for anyone.
func skillsUpdate() error {
	if _, err := os.Stat(filepath.Join("cmd", "irgo", "skills.go")); err != nil {
		return fmt.Errorf("run this from a checkout of %s\n"+
			"  A project's skills are copies. irgo's is the original, and\n"+
			"  updating it is what changes them for everyone", irgoModule)
	}

	client := &http.Client{Timeout: 30 * time.Second}
	changed := 0

	for _, s := range vendoredSkills {
		resp, err := client.Get(s.rawURL())
		if err != nil {
			return fmt.Errorf("fetching %s: %w", s.name, err)
		}
		body, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			return fmt.Errorf("reading %s: %w", s.name, err)
		}
		if resp.StatusCode != http.StatusOK {
			return fmt.Errorf("%s: upstream returned %s for\n  %s\n"+
				"  If the file moved, update its path in cmd/irgo/skills.go",
				s.name, resp.Status, s.rawURL())
		}

		have, _ := os.ReadFile(s.localPath())
		if string(have) == string(body) {
			fmt.Printf("  %-12s unchanged\n", s.name)
			continue
		}
		if err := os.MkdirAll(filepath.Dir(s.localPath()), 0o755); err != nil {
			return err
		}
		if err := os.WriteFile(s.localPath(), body, 0o644); err != nil {
			return err
		}
		changed++
		fmt.Printf("  %-12s updated (%d -> %d bytes)\n", s.name, len(have), len(body))
	}

	if changed > 0 {
		fmt.Println()
		fmt.Println("Now update `commit` and `taken` in cmd/irgo/skills.go for what")
		fmt.Println("changed — they are the record of which version this copy is,")
		fmt.Println("and a copy whose record says otherwise is worse than none.")
		fmt.Println()
		fmt.Println("The commit that last touched it:")
		for _, s := range vendoredSkills {
			repo := strings.TrimPrefix(s.repo, "https://github.com/")
			fmt.Printf("  gh api 'repos/%s/commits?path=%s&per_page=1' --jq '.[0].sha'\n",
				repo, s.path)
		}
	}
	return nil
}

// wrapAt wraps text to width, indenting continuation lines.
func wrapAt(text string, width int, indent string) string {
	var out strings.Builder
	col := 0
	for _, w := range strings.Fields(text) {
		if col > 0 && col+1+len(w) > width {
			out.WriteString("\n" + indent)
			col = 0
		} else if col > 0 {
			out.WriteString(" ")
			col++
		}
		out.WriteString(w)
		col += len(w)
	}
	return out.String()
}

func init() {
	registerAssetStep(assetOrderDocs, syncSkills)

	register(command{
		noun: "project", verb: "skills", order: 55,
		summary: "Reference material for whoever writes the code, human or not",
		usage: [][2]string{
			{"", "Sync into .claude/skills and list what is there"},
			{"--sources", "Where each one came from"},
			{"--check", "Compare the vendored ones against upstream"},
			{"--update", "Take upstream's version (from an irgo checkout)"},
		},
		notes: `Every build does the sync already, so this is for looking rather than doing.

Skills ship inside Go modules, so the version you get is the one your go.mod
names. Nothing is downloaded and there is no version to pin twice.

They are committed rather than gitignored: an assistant reads them the moment a
repository is cloned, which is before any build has run.`,
	})
}
