// Removing what irgo installed.
//
// This deletes gigabytes — the Android SDK and NDK, a JDK, Homebrew formulae —
// so it plans first, shows exactly what it will touch, and asks. A command that
// destroys that much on a bare invocation is a trap, however accurate its
// output afterwards.
//
// Android is not a separate command. "Remove what irgo installed" that leaves
// behind the largest thing irgo installed is a promise the CLI does not keep,
// and it is the part someone will forget.
package main

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// removalPlan is what a removal would do, computed before anything is deleted.
type removalPlan struct {
	// group -> lines to show under it.
	groups []planGroup
	// kept lists things irgo did not install, which are reported and skipped.
	kept []string
	// notes are printed after the plan: things irgo will not touch, and what
	// to run instead.
	notes []string
	acts  []func(*removalTally)
}

type planGroup struct {
	name  string
	lines []string
}

func (p *removalPlan) add(group, line string, act func(*removalTally)) {
	for i := range p.groups {
		if p.groups[i].name == group {
			p.groups[i].lines = append(p.groups[i].lines, line)
			p.acts = append(p.acts, act)
			return
		}
	}
	p.groups = append(p.groups, planGroup{name: group, lines: []string{line}})
	p.acts = append(p.acts, act)
}

func (p *removalPlan) empty() bool { return len(p.acts) == 0 }

// confirmRemoval shows the plan and asks. Outside a terminal it refuses rather
// than assuming yes: a script that deletes an SDK because nobody was watching
// is worse than one that stops and says what it needs.
func confirmRemoval(p *removalPlan, yes bool) (bool, error) {
	fmt.Println("This will remove what irgo installed on this machine:")
	for _, g := range p.groups {
		fmt.Println()
		fmt.Printf("%s:\n", g.name)
		for _, l := range g.lines {
			fmt.Printf("  %s\n", l)
		}
	}
	for _, n := range p.notes {
		fmt.Println()
		fmt.Println(n)
	}
	if len(p.kept) > 0 {
		fmt.Println()
		fmt.Printf("Kept (irgo did not install these): %s\n", strings.Join(p.kept, ", "))
		fmt.Println("  --all removes them too.")
	}
	fmt.Println()

	if yes {
		return true, nil
	}
	if !interactive() {
		return false, fmt.Errorf("refusing to remove without confirmation.\n" +
			"  Nothing has been deleted. Re-run with --yes to proceed:\n" +
			"    irgo tools remove --yes")
	}
	if !confirm("Proceed?", false) {
		fmt.Println("Nothing removed.")
		return false, nil
	}
	return true, nil
}

// confirm asks, unless told not to. Outside a terminal it refuses rather than
// assuming yes: a script that meant to pass --yes should say so.
func confirm(question string, yes bool) bool {
	if yes {
		return true
	}
	if !interactive() {
		return false
	}
	fmt.Printf("%s [y/N] ", question)
	line, err := bufio.NewReader(os.Stdin).ReadString('\n')
	if err != nil {
		return false
	}
	switch strings.ToLower(strings.TrimSpace(line)) {
	case "y", "yes":
		return true
	}
	return false
}

// planGoTools adds the go-installed tools irgo owns.
// planMiseTools offers back the tools irgo asked mise to install.
//
// Only those irgo installed: the marker decides. A mise tool that was already
// on the machine belongs to the developer and probably to other projects too,
// and uninstalling their node because irgo happened to use it would be the
// worst thing this command could do.
//
// mise uninstall rather than deleting the directory, so mise's own view stays
// correct.
// planTools adds every tool irgo installed, from the table doctor reports.
//
// This replaced four planners — Go tools, mise tools, downloads, node — that
// each rediscovered paths doctor had already found, in their own way. One of
// them searched a different order than the build did, which is how remove
// could offer a copy nothing was using.
//
// The table knows where each tool is and how to take it back; this decides
// what to show and groups it.
func planTools(p *removalPlan, all bool) {
	var mise []toolStatus
	claimed := map[string]bool{}
	for _, row := range toolLocators() {
		// Tools that live in mise are reported, not removed. See planMiseNote.
		if row.how == "mise" {
			mise = append(mise, row)
			continue
		}
		// The Android section reports these with the rest of that toolchain,
		// where they make sense together. Listing them here too offered the
		// same JDK twice.
		switch row.name {
		case "jdk", "sdkmanager", "avdmanager", "adb", "emulator", "ndk":
			if row.path != "" {
				claimed[irgoOwnedDir(row.path)] = true
			}
			continue
		}
		if row.remove != nil && strings.HasPrefix(row.path, irgoHome()) {
			claimed[irgoOwnedDir(row.path)] = true
		}
		if row.remove == nil {
			// Nothing of irgo's to remove. Worth naming when it is present and
			// irgo could have installed it, so the report is not silently
			// short.
			if row.path != "" && row.ensure != nil && !all {
				p.kept = append(p.kept, row.name)
			}
			continue
		}
		name, undo := row.name, row.remove
		where := row.path
		if row.size != "" {
			where += "  (" + row.size + ")"
		}
		p.add(removalGroup(row), fmt.Sprintf("%-14s %s", name, where),
			func(tally *removalTally) {
				if err := undo(); err != nil {
					tally.report(name, "failed", err.Error())
					return
				}
				tally.report(name, "removed", row.path)
			})
	}
	planIrgoLeftovers(p, claimed)
	planMiseNote(p, mise)
}

// planMiseNote lists the tools irgo uses from mise, with the command to
// remove them — rather than removing them itself.
//
// irgo cannot tell which of these it installed. This machine's sops 3.12.2 was
// installed in April and its tailwindcss today, both by the same mechanism
// into the same place, and mise records no owner. A marker file could have
// tracked it, at the cost of a folder to keep swept.
//
// But ownership is the wrong question. These live where the developer's own
// version manager can see them and other projects may be using them, so the
// decision is theirs — and printing the command costs nothing while getting it
// wrong costs someone else's build.
func planMiseNote(p *removalPlan, rows []toolStatus) {
	if len(rows) == 0 {
		return
	}
	var lines []string
	for _, r := range rows {
		if spec, ok := miseSpecFor(r.name); ok {
			lines = append(lines, "  mise uninstall "+spec)
		}
	}
	if len(lines) == 0 {
		return
	}
	p.notes = append(p.notes,
		"From mise (shared with anything else on this machine that uses them):",
		strings.Join(lines, "\n"),
		"  mise prune   # anything no config asks for")
}

// planIrgoLeftovers adds anything still sitting in ~/.irgo that no tool row
// points at any more.
//
// The table only names the copy a build would use. Once node came from mise,
// the 176 MB irgo had downloaded stopped being referenced by any row and
// would have been left on disk forever — invisible to the one command whose
// job is finding it.
//
// irgo owns ~/.irgo entirely, so anything in it is irgo's to offer back.
func planIrgoLeftovers(p *removalPlan, claimed map[string]bool) {
	entries, err := os.ReadDir(irgoHome())
	if err != nil {
		return
	}
	for _, e := range entries {
		// tools holds the markers, bin is handled per-file, jdks belongs to
		// the Android section which reports it with the rest of that toolchain.
		switch e.Name() {
		case "tools", "jdks":
			continue
		}
		dir := filepath.Join(irgoHome(), e.Name())
		if claimed[dir] || !isDir(dir) {
			continue
		}
		if empty, _ := os.ReadDir(dir); len(empty) == 0 {
			continue
		}
		name := e.Name()
		p.add("Downloads", fmt.Sprintf("%-14s %s%s", name, dir, dirSizeNote(dir)),
			func(tally *removalTally) {
				if err := os.RemoveAll(dir); err != nil {
					tally.report(name, "failed", err.Error())
					return
				}
				clearToolMarker(name)
				tally.report(name, "removed", dir)
			})
	}
}

// removalGroup is the heading a tool appears under, taken from where it came
// from rather than from a second list.
func removalGroup(row toolStatus) string {
	switch row.how {
	case "mise":
		return "mise tools"
	case howGoInstall:
		return "Go tools"
	}
	return "Downloads"
}

// planHostPackages adds packages installed through the host package manager.
func planHostPackages(p *removalPlan, all bool) {
	mgr := pkgManager()
	if mgr == "" {
		return
	}
	for _, pkg := range osPackages() {
		pk := pkg
		name := pk.pkgNameFor(mgr)
		if name == "" || !pk.probe() {
			continue
		}
		if !all && !toolInstalledByIrgo(pk.key) {
			p.kept = append(p.kept, pk.key)
			continue
		}
		p.add(fmt.Sprintf("Host packages (%s)", mgr), fmt.Sprintf("%-14s %s", pk.key, name),
			func(tally *removalTally) {
				cmd := pkgRemoveCmd(mgr, name)
				if out, err := runCommandQuiet(cmd[0], cmd[1:]...); err != nil {
					tally.report(pk.key, "failed", firstLine(strings.TrimSpace(out)))
				} else {
					tally.report(pk.key, "removed", "via "+mgr)
				}
				clearToolMarker(pk.key)
			})
	}
}

// planAndroid adds the Android toolchain, which is by far the largest thing
// irgo installs and therefore the one most worth showing before deleting.
func planAndroid(p *removalPlan, keepJDK bool) {
	home := homeDir()
	sdk := androidHome()

	// Only an SDK irgo provisioned carries the marker; never delete a
	// developer's own.
	if pathExists(filepath.Join(sdk, toolchainMarker)) {
		p.add("Android", fmt.Sprintf("%-14s %s%s", "SDK/NDK", sdk, dirSizeNote(sdk)), func(tally *removalTally) {
			if os.RemoveAll(sdk) == nil {
				tally.report("android-sdk", "removed", sdk)
			} else {
				tally.report("android-sdk", "failed", sdk)
			}
		})
	} else if pathExists(sdk) {
		p.kept = append(p.kept, "android SDK (not provisioned by irgo)")
	}

	// The JDK, when irgo downloaded it. A JDK from mise belongs to mise, and
	// `mise uninstall java@...` is how it goes.
	if jdk := managedJDKRoot(); !keepJDK && pathExists(jdk) {
		p.add("Android", fmt.Sprintf("%-14s %s%s", "JDK 17", jdk, dirSizeNote(jdk)), func(tally *removalTally) {
			if os.RemoveAll(jdk) == nil {
				tally.report("managed-jdk", "removed", jdk)
			} else {
				tally.report("managed-jdk", "failed", jdk)
			}
		})
	}

	for _, c := range []struct{ label, path string }{
		{"AVDs/adb", filepath.Join(home, ".android")},
		{"Gradle cache", filepath.Join(home, ".gradle")},
	} {
		cc := c
		if !pathExists(cc.path) {
			continue
		}
		p.add("Android", fmt.Sprintf("%-14s %s%s", cc.label, cc.path, dirSizeNote(cc.path)), func(tally *removalTally) {
			if os.RemoveAll(cc.path) == nil {
				tally.report(cc.label, "removed", cc.path)
			} else {
				tally.report(cc.label, "failed", cc.path)
			}
		})
	}
}

// dirSizeNote reports an approximate size, because "this deletes 4 GB" is the
// fact that changes whether someone says yes.
func dirSizeNote(path string) string {
	out, err := runCommandQuiet("du", "-sh", path)
	if err != nil {
		return ""
	}
	if f := strings.Fields(out); len(f) > 0 {
		return "  (" + f[0] + ")"
	}
	return ""
}
