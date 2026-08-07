// What irgo installed, and whether it is what irgo pins.
//
// `doctor` used to answer "is gomobile present?", which stopped being the
// useful question the day gomobile changed behaviour underneath a pinned
// checkout and every mobile build failed. Present is not the same as correct:
// a tool from before the pin existed, or one a person installed themselves,
// reports exactly the same way and behaves differently.
//
// So this reports the version each tool actually is, next to the version irgo
// asks for, and says plainly when they differ and what to run.
package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// toolStatus is one row.
type toolStatus struct {
	name    string
	path    string // where it is, "" when absent
	have    string // version reported by the tool
	want    string // version irgo pins
	managed bool   // irgo installed it
}

// installedVersion is the version irgo recorded when it installed the tool.
//
// Read from the marker rather than from the tool: air and gobind have no
// version flag, and scanning their usage output for something version-shaped
// finds the Go version in a path — which is exactly how the first draft of
// this reported air as drifted when it was not. A wrong version is worse than
// no version, because it sends someone to reinstall a tool that was fine.
func installedVersion(name string) string {
	data, err := os.ReadFile(filepath.Join(irgoToolsDir(), name))
	if err != nil {
		return ""
	}
	for _, line := range strings.Split(string(data), "\n") {
		if v, ok := strings.CutPrefix(line, "pin "); ok {
			// Trim the "v" here and in wantedVersion, so a pin written
			// "v1.63.0" and one written "1.63.0" compare equal rather than
			// reporting drift that does not exist.
			return strings.TrimPrefix(strings.TrimSpace(v), "v")
		}
	}
	return ""
}

// wantedVersion is the version irgo installs, taken from the same table the
// installer uses so the two cannot disagree.
func wantedVersion(tool string) string {
	pkg := goToolPkg(tool)
	if pkg == "" {
		return ""
	}
	_, v, ok := strings.Cut(pkg, "@")
	if !ok {
		return ""
	}
	return strings.TrimPrefix(v, "v")
}

// goToolStatuses reports every Go tool irgo installs.
func goToolStatuses() []toolStatus {
	var out []toolStatus
	for _, name := range []string{"templ", "air", "gomobile", "gobind"} {
		st := toolStatus{name: name, want: wantedVersion(name), managed: toolInstalledByIrgo(name)}
		if p, err := exec.LookPath(name); err == nil {
			st.path = p
			st.have = installedVersion(name)
		}
		out = append(out, st)
	}
	return out
}

// nodeStatus reports the Node irgo downloads for wrangler. It is 176 MB and
// nothing mentioned it until `tools remove` listed it, which is a poor way to
// discover a large download.
func nodeStatus() toolStatus {
	st := toolStatus{name: "node", want: pinNode, managed: toolInstalledByIrgo("node")}
	bin := filepath.Join(managedNodeHome(), "bin", "node")
	if !pathExists(bin) {
		if p, err := exec.LookPath("node"); err == nil {
			bin = p
		} else {
			return st
		}
	}
	if nodeWorks(bin) {
		st.path = bin
		// node --version is reliable, unlike the Go tools'.
		if out, err := exec.Command(bin, "--version").Output(); err == nil {
			st.have = strings.TrimSpace(strings.TrimPrefix(string(out), "v"))
		}
	}
	return st
}

// printToolVersions writes the section.
func printToolVersions() {
	rows := append(goToolStatuses(), nodeStatus())

	w := 0
	for _, r := range rows {
		if len(r.name) > w {
			w = len(r.name)
		}
	}

	fmt.Println()
	fmt.Println("Tools irgo installs:")
	var drifted []string
	for _, r := range rows {
		switch {
		case r.path == "":
			fmt.Printf("  %-*s  %-12s  installed when a build needs it\n", w, r.name, "absent")
		case r.have == "":
			// Present but not installed by irgo, so there is no record of what
			// it is and no honest claim to make about it.
			fmt.Printf("  %-*s  %-12s  %s\n", w, r.name, "yours", r.path)
		case r.want != "" && !versionMatches(r.have, r.want):
			fmt.Printf("  %-*s  %-12s  irgo pins %s\n", w, r.name, r.have, r.want)
			drifted = append(drifted, r.name)
		default:
			fmt.Printf("  %-*s  %-12s  %s\n", w, r.name, r.have, whose(r))
		}
	}

	if len(drifted) > 0 {
		fmt.Println()
		fmt.Printf("Not the version irgo pins: %s\n", strings.Join(drifted, ", "))
		fmt.Println("A tool that disagrees with the source it drives is how mobile builds")
		fmt.Println("broke before: gomobile changed under a pinned checkout. Replace them:")
		fmt.Println("  irgo tools remove   # only what irgo installed")
		fmt.Println("  irgo tools install")
	}
}

// versionMatches compares loosely: a pseudo-version pins a commit, and the
// tool reports something else entirely, so an exact match is not always
// possible or meaningful.
func versionMatches(have, want string) bool {
	if have == want || strings.HasPrefix(want, have) || strings.HasPrefix(have, want) {
		return true
	}
	// x/mobile pins a commit; gomobile reports "unknown" or a date. Nothing
	// useful to compare, so do not claim drift.
	if len(want) == 40 || strings.Contains(want, "-") && len(want) > 20 {
		return true
	}
	return false
}

// whose says where a tool came from, because "irgo installed this" and "you
// installed this" have different consequences for `tools remove`.
func whose(r toolStatus) string {
	if r.managed {
		return "installed by irgo"
	}
	return r.path
}
