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
	"runtime"
	"strings"
)

// Where a tool comes from. Constants because these were nine string literals
// across two functions, and a wording change meant finding them all.
const (
	howGoInstall = "go install"
	howDownload  = "downloaded by irgo"
	howAndroid   = "Android SDK, by irgo"
	howMise      = "mise, else downloaded by irgo"
	howPlatform  = "provided by the platform"
	howHostPkg   = "host package manager"
	howOptional  = "optional — irgo uses it if present"
	howPrereq    = "prerequisite — compiles irgo"
)

// sourceOf says where the copy on this machine actually came from.
//
// Two questions kept getting conflated. How irgo *would* install a tool is a
// fact about irgo — miseSpec and goToolPkg answer it. How the copy that is
// here *did* arrive is a fact about the machine, and irgo does not always know:
// templ may be irgo's `go install` or the developer's.
//
// This answers the second, and asks the authority rather than reading the path.
// mise is asked whether it owns the binary; a substring check called templ a
// mise package because it sits in the bin directory of a Go that mise manages,
// and `tools remove` then printed a `mise uninstall templ` that matched
// nothing.
func sourceOf(name, path string) string {
	if path == "" {
		return ""
	}
	// mise, asked directly: is this path inside an install it owns?
	//
	// Containment rather than equality: a row may hold the binary (node) or
	// the directory (the JDK, which is a JAVA_HOME), and both live under the
	// same install.
	if spec, ok := miseSpecFor(name); ok {
		if dir := miseInstallDir(spec); dir != "" && strings.HasPrefix(path, dir) {
			return "mise"
		}
	}
	switch {
	case strings.HasPrefix(path, androidHome()):
		return howAndroid
	case strings.HasPrefix(path, irgoHome()):
		return howDownload
	case gobinDir() != "" && strings.HasPrefix(path, gobinDir()):
		return howGoInstall
	case goToolPkg(name) != "":
		// irgo installs this with `go install`, and it is not a mise package,
		// so wherever it landed that is how it got there.
		return howGoInstall
	}
	return "yours"
}

// miseInstallDir is where mise put a spec, or "" if it does not have it.
func miseInstallDir(spec string) string {
	mise, ok := miseCmd()
	if !ok {
		return ""
	}
	out, err := exec.Command(mise, "where", spec).Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

// toolStatus is one row.
type toolStatus struct {
	name    string
	path    string // where it is, "" when absent
	have    string // version reported by the tool
	want    string // version irgo pins
	managed bool   // irgo installed it, per the marker in ~/.irgo/tools
	// ensure gets the tool when it is missing. nil means irgo cannot provide
	// it — Xcode, a C compiler — and doctor says so rather than pretending.
	ensure func() error
	// how it is obtained, in a few words. Printed, because otherwise the only
	// way to learn whether a tool comes from `go install`, a download or mise
	// is to read the source — and reading the source is what doctor is for.
	how string
	// size on disk, for directories irgo owns. "" for everything else.
	size string
	// remove undoes the install, when irgo was the one that did it. nil means
	// there is nothing of irgo's to take back.
	remove func() error
}

// irgoDiskUse reports how much space a tool irgo installed is using.
//
// Only under irgoHome: measuring the platform's own directories would be slow
// and none of irgo's business. Returns "" when there is nothing to say.
func irgoDiskUse(path string) string {
	if path == "" || !strings.HasPrefix(path, irgoHome()) {
		return ""
	}
	// The tool's own directory, not the binary — except in ~/.irgo/bin, which
	// holds one file per tool, so walking up there would report the whole
	// directory as the size of each one. tailwindcss read 76 MB that way.
	dir := path
	if filepath.Dir(path) == irgoBinDir() {
		if fi, err := os.Stat(path); err == nil {
			return humanBytes(fi.Size())
		}
		return ""
	}
	for filepath.Dir(dir) != irgoHome() && filepath.Dir(dir) != "/" && dir != "" {
		dir = filepath.Dir(dir)
	}
	var total int64
	_ = filepath.Walk(dir, func(_ string, fi os.FileInfo, err error) error {
		if err == nil && !fi.IsDir() {
			total += fi.Size()
		}
		return nil
	})
	return humanBytes(total)
}

func humanBytes(n int64) string {
	switch {
	case n == 0:
		return ""
	case n > 1<<30:
		return fmt.Sprintf("%.1f GB", float64(n)/(1<<30))
	case n > 1<<20:
		return fmt.Sprintf("%d MB", n/(1<<20))
	}
	return fmt.Sprintf("%d KB", n/(1<<10))
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
	for _, name := range goTools() {
		tool := name
		st := toolStatus{
			name: name, want: wantedVersion(name), managed: toolInstalledByIrgo(name),
			ensure: func() error { return ensureGoTool(tool) },
			how:    howGoInstall,
		}
		if p, err := exec.LookPath(name); err == nil {
			st.path = p
			st.have = installedVersion(name)
		}
		out = append(out, st)
	}
	return out
}

// nodeStatus reports the Node used for wrangler. It is 176 MB when irgo has to
// download one, and nothing mentioned it until `tools remove` listed it, which
// is a poor way to discover a large download.
//
// Asks nodeBin rather than working the path out again. The copy that used to
// live here searched in the opposite order — managed before PATH — and did not
// know about node.exe, so doctor could name a Node that no build would use.
// A tool's own file decides where it is; doctor reports what it says.
func nodeStatus() toolStatus {
	st := toolStatus{
		name: "node", want: pinNode, managed: toolInstalledByIrgo("node"),
		ensure: func() error { _, err := nodeBin(true); return err },
		how:    howMise,
	}
	bin, err := nodeBin(false)
	if err != nil {
		return st
	}
	st.path = bin
	// node --version is reliable, unlike the Go tools'.
	if out, err := exec.Command(bin, "--version").Output(); err == nil {
		st.have = strings.TrimSpace(strings.TrimPrefix(string(out), "v"))
	}
	return st
}

// toolLocators is every external program irgo runs, and the code that already
// knows where it is.
//
// Each entry calls the owning file's resolver rather than rebuilding a path.
// doctor used to work out node's location itself and got it wrong two ways, so
// it could name a binary no build would use. Nothing here derives anything: a
// tool's own file decides, and doctor reports what it says.
//
// Every developer's machine answers "where is everything" with one command,
// and adding a tool means adding one line.
func toolLocators() []toolStatus {
	sdk := androidHome()

	// found reports a path when something is actually there, and how to get
	// it when it is not.
	found := func(name, path, how string, ensure func() error) toolStatus {
		st := toolStatus{name: name, ensure: ensure, how: how}
		if path == "" {
			return st
		}
		if _, err := os.Stat(path); err != nil {
			return st
		}
		st.path = path
		return st
	}
	// onPath is for tools found by name. ensure may be nil: irgo cannot
	// install Xcode or a system C compiler, and says so instead.
	onPath := func(name, how string, ensure func() error) toolStatus {
		st := toolStatus{name: name, ensure: ensure, how: how}
		if p, err := exec.LookPath(name); err == nil && runs(p) {
			st.path = p
		}
		return st
	}

	rows := append(goToolStatuses(), nodeStatus())

	// Tools mise provides, resolved through mise rather than by name.
	//
	// sops was in the platform list, so doctor looked it up on PATH, found a
	// shim that fails with "no version is set", and reported absent — while
	// mise had the binary all along. A tool mise provides has to be asked of
	// mise, like every other row here.
	rows = append(rows, miseRow("sops"))

	// Downloaded by irgo.
	rows = append(rows,
		found("tailwindcss", tailwindPath(), howDownload, func() error {
			_, err := ensureTailwind()
			return err
		}),
	)

	// The Android toolchain, each from the function that owns it.
	jdk, _ := detectJDK17(false)
	androidTools := func() error { return ensureAndroidToolchain(false, "") }
	rows = append(rows,
		found("jdk", jdk, howDownload, func() error {
			_, ok := detectJDK17(true)
			if !ok {
				return fmt.Errorf("could not install a JDK 17")
			}
			return nil
		}),
		found("sdkmanager", locateSdkmanager(sdk), howAndroid, androidTools),
		found("avdmanager", locateAvdmanager(sdk), howAndroid, androidTools),
		found("adb", adbBin(), howAndroid, androidTools),
		found("emulator", emulatorBin(sdk), howAndroid, func() error { return ensureAndroidToolchain(true, "") }),
		found("ndk", ndkDir(sdk), howAndroid, androidTools),
	)

	// Provided by the platform or the developer. irgo cannot install these,
	// but where they are is exactly what someone debugging a build needs.
	for _, name := range platformTools() {
		rows = append(rows, onPath(name, platformHow(name), platformEnsure(name)))
	}

	pruneStaleMarkers(rows)

	// Everything below needs the finished set, so it runs here rather than in
	// whichever command happens to call this. An earlier version attached
	// removal inside doctor's printer, so `tools remove` — which reads this
	// function directly — found nothing removable at all.
	for i := range rows {
		// Where it actually came from beats where it would come from.
		if src := sourceOf(rows[i].name, rows[i].path); src != "" {
			rows[i].how = src
		}
		// Disk, for the things irgo put there: 485 MB of Node and JDK sat in
		// ~/.irgo duplicating what mise had, and only du could tell you.
		rows[i].size = irgoDiskUse(rows[i].path)
		// The inverse of how it was installed.
		attachRemove(&rows[i])
		// Ask the tool itself when irgo has no marker for it.
		if rows[i].have == "" {
			rows[i].have = toolVersion(rows[i].path)
		}
	}
	return rows
}

// pruneStaleMarkers drops records of tools that are no longer there.
//
// ~/.irgo/tools says what irgo installed, which is the only way to know that a
// mise or GOBIN copy is irgo's rather than the developer's. Nothing swept it,
// so it accumulated: a marker for node long after that directory was deleted,
// and one for entr, a tool irgo no longer uses at all.
//
// A marker for something absent is not harmless. It is the thing `tools
// remove` consults to decide what it may delete.
func pruneStaleMarkers(rows []toolStatus) {
	dir := irgoToolsDir()
	entries, err := os.ReadDir(dir)
	if err != nil {
		return
	}
	known := map[string]bool{}
	for _, r := range rows {
		if r.path != "" {
			known[r.name] = true
		}
	}
	for _, e := range entries {
		if e.IsDir() {
			continue // the mise markers, pruned by spec below
		}
		name := e.Name()
		if known[name] || isHostPackage(name) {
			continue
		}
		_ = os.Remove(filepath.Join(dir, name))
	}

}

// isHostPackage keeps markers for brew and apt packages, which are not tools
// in the table and have no path to check.
func isHostPackage(name string) bool {
	for _, p := range hostPackageKeys() {
		if p == name {
			return true
		}
	}
	return false
}

// runs reports whether a binary actually executes.
//
// Existing is not the same as working: a version manager's shim is a real file
// that fails with "no version is set" when nothing selects one. doctor named
// the sops shim as though a build could use it, which is the same mistake
// nodeWorks was written to stop.
func runs(path string) bool {
	if path == "" {
		return false
	}
	// Only shims are distrusted. Testing every binary was worse than the bug:
	// go, codesign and security do not take --version, exited non-zero, and
	// doctor called three working tools ABSENT.
	if !strings.Contains(path, string(filepath.Separator)+"shims"+string(filepath.Separator)) {
		return true
	}
	out, err := exec.Command(path, "--version").CombinedOutput()
	if err == nil {
		return true
	}
	// A shim with nothing selected says so, and cannot be used by a build.
	return !strings.Contains(string(out), "No version is set")
}

// toolVersion asks a binary what it is.
//
// The marker irgo writes only exists for tools irgo installed, which after
// mise took over most of them left fifteen of twenty-two rows blank. Running
// the tool covers the rest — carefully, because air and gobind have no version
// flag and their usage text contains a Go version that is not theirs.
func toolVersion(path string) string {
	if path == "" {
		return ""
	}
	out, err := exec.Command(path, "--version").Output()
	if err != nil {
		return ""
	}
	// The first token shaped like a version, and nothing else. "tailwindcss
	// v4.3.3" and "git version 2.51.0" both land here; a usage banner does not.
	for _, field := range strings.Fields(string(out)) {
		v := strings.TrimPrefix(strings.TrimSuffix(field, ","), "v")
		if versionShaped(v) {
			return v
		}
	}
	return ""
}

// versionShaped is a digit, a dot, a digit — deliberately strict, so a path or
// a date does not read as a version.
func versionShaped(s string) bool {
	dots, digits := 0, 0
	for _, r := range s {
		switch {
		case r == '.':
			dots++
		case r >= '0' && r <= '9':
			digits++
		case r == '-' || r == '+' || (r >= 'a' && r <= 'z'):
			// build metadata, allowed after the numbers
		default:
			return false
		}
	}
	if dots < 1 || digits < 2 || s == "" || s[0] < '0' || s[0] > '9' {
		return false
	}
	// A trailing dot is not a version: xcrun prints "xcrun version 72." and
	// that read as 72.
	return s[len(s)-1] != '.'
}

// attachRemove gives a row the inverse of how it was installed.
//
// Only what irgo installed: the markers decide. A mise tool is uninstalled by
// the exact spec irgo asked for, never by name, so a developer's own version
// is never a candidate.
func attachRemove(r *toolStatus) {
	name := r.name
	if spec, ok := miseSpecFor(name); ok && installedViaMise(spec) {
		r.remove = func() error {
			mise, ok := miseCmd()
			if !ok {
				return fmt.Errorf("mise is gone, so %s cannot be uninstalled through it", name)
			}
			if err := exec.Command(mise, "uninstall", spec).Run(); err != nil {
				return err
			}
			return nil
		}
		return
	}
	// irgo's own directories.
	if r.path != "" && strings.HasPrefix(r.path, irgoHome()) {
		dir := irgoOwnedDir(r.path)
		r.remove = func() error {
			if err := os.RemoveAll(dir); err != nil {
				return err
			}
			clearToolMarker(name)
			return nil
		}
		return
	}
	if toolInstalledByIrgo(name) && r.how == howGoInstall {
		r.remove = func() error {
			removeToolPath(name)
			clearToolMarker(name)
			return nil
		}
	}
}

// irgoOwnedDir is what removing a tool should delete: its own directory under
// ~/.irgo, or the single file when it lives in ~/.irgo/bin.
func irgoOwnedDir(path string) string {
	if filepath.Dir(path) == irgoBinDir() {
		return path
	}
	dir := path
	for filepath.Dir(dir) != irgoHome() && filepath.Dir(dir) != "/" && dir != "" {
		dir = filepath.Dir(dir)
	}
	return dir
}

// miseRow resolves a tool through mise, at irgo's pin.
func miseRow(name string) toolStatus {
	st := toolStatus{
		name:   name,
		how:    howGoInstall,
		ensure: func() error { return ensureGoTool(name) },
	}
	spec, ok := miseSpec(name)
	if !ok {
		return st
	}
	if mise, have := miseCmd(); have {
		st.path = miseWhere(mise, spec, name)
	}
	if st.path == "" {
		// Not through mise: whatever is on PATH, if it runs.
		if p, err := exec.LookPath(name); err == nil && runs(p) {
			st.path = p
		}
	}
	return st
}

// platformHow says where a host tool comes from.
func platformHow(name string) string {
	switch name {
	case "sops":
		return howGoInstall
	case "gcc", "pkg-config", "x86_64-w64-mingw32-gcc":
		return howHostPkg
	case "mise":
		return howDownload
	case "go":
		return howPrereq
	}
	return howPlatform
}

// platformEnsure returns how to obtain a host tool, or nil when irgo cannot.
//
// sops is a Go program, so `go install` reaches it. The rest — Xcode, a C
// compiler, git — come from the platform, and claiming otherwise would send
// someone to run a command that cannot work.
func platformEnsure(name string) func() error {
	switch name {
	case "mise":
		return func() error { _, err := ensureMise(); return err }
	case "sops":
		return func() error { return ensureGoTool("sops") }
	case "gcc", "pkg-config":
		return func() error { return ensureOSPackage(name) }
	case "x86_64-w64-mingw32-gcc":
		return func() error { return ensureOSPackage("mingw-w64") }
	}
	return nil
}

// platformTools are the host programs irgo runs but does not provide. Listed
// per OS, because naming xcrun on Linux would report a permanent absence.
func platformTools() []string {
	// mise first: on a machine that uses it, most rows above resolve to a
	// path inside its installs directory, so it is the tool that explains
	// where the others came from.
	common := []string{"mise", "go", "git"}
	switch runtime.GOOS {
	case "darwin":
		// mingw is the Windows cross-compiler. `tools remove` offered it back
		// while doctor never mentioned it, so the only way to learn irgo had
		// installed it was to try deleting things.
		return append(common, "xcrun", "xcodebuild", "codesign", "security",
			"clang", "x86_64-w64-mingw32-gcc")
	case "linux":
		return append(common, "gcc", "pkg-config", "x86_64-w64-mingw32-gcc")
	case "windows":
		return append(common, "powershell")
	}
	return common
}

// printToolVersions writes the section.
func printToolVersions() {
	rows := toolLocators()

	w := 0
	for _, r := range rows {
		if len(r.name) > w {
			w = len(r.name)
		}
	}

	// Width for the "how" column too, so the three read as columns.
	hw := 0
	for _, r := range rows {
		if len(r.how) > hw {
			hw = len(r.how)
		}
	}

	fmt.Println()
	fmt.Println("Tools — where they are, and where they come from:")
	var drifted []string
	for _, r := range rows {
		// One line, one format. Five near-identical Printf calls differed only
		// in the last two columns, which is how the path came to be missing
		// from one of them.
		// One meaning per column: what it is, where it came from, which
		// version, where it lives. The version column used to say "yours",
		// which after the source column became derived was the same word
		// twice and a version nowhere.
		state, note := r.have, r.path
		if state == "" {
			state = "-"
		}
		switch {
		case r.path == "" && r.ensure == nil:
			state, note = "-", "ABSENT — install it yourself"
		case r.path == "":
			state, note = "-", "absent — arrives when a build needs it"
		case r.want != "" && r.have != "" && !versionMatches(r.have, r.want):
			note = r.path + " — irgo pins " + r.want
			drifted = append(drifted, r.name)
		}
		if r.size != "" {
			note = r.size + "  " + note
		}
		fmt.Printf("  %-*s  %-*s  %-12s  %s\n", w, r.name, hw, r.how, state, note)
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
