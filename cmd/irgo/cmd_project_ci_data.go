// What the generated CI workflows are built from.
//
// The workflows used to hardcode everything the CLI already knows: which
// runner each target needs, where each build leaves its artifacts, and the
// version of every action, repeated once per job. That is the same knowledge
// twice, in two languages, and the copy in YAML has no way to be wrong out
// loud — it just builds the wrong thing, or silently stops being bumped.
//
// So the templates carry placeholders and this file carries the values. One
// edit here changes every workflow the CLI writes, and `irgo project ci
// --force` (or `upgrade`) rolls it out.
package main

import (
	"fmt"
	"strings"
)

// ciAction pins the actions the workflows use. Bumping one is a single edit
// rather than a search across two files and eight jobs — which is how they
// drifted apart before.
var ciActions = map[string]string{
	"CHECKOUT":        "actions/checkout@v7",
	"SETUP_GO":        "actions/setup-go@v7",
	"UPLOAD_ARTIFACT": "actions/upload-artifact@v7",
}

// desktopTarget is one desktop build: the GOOS the CLI is asked for, the
// runner that can produce it, and where the build leaves it.
type desktopTarget struct {
	name     string // shown in the job name
	target   string // what `app build desktop <target>` takes
	runner   string // GitHub runner label
	artifact string // upload-artifact name
	outDir   string // where the CLI writes it
}

// desktopTargets is the one place that says which desktops CI builds.
//
// Linux pins ubuntu-22.04 deliberately: webview links webkit2gtk-4.0, which
// ubuntu-24.04 no longer ships.
var desktopTargets = []desktopTarget{
	{"Linux", "linux", "ubuntu-22.04", "desktop-linux", "build/desktop/linux"},
	{"macOS", "darwin", "macos-latest", "desktop-macos", "build/desktop/macos"},
	{"Windows", "windows", "windows-latest", "desktop-windows", "build/desktop/windows"},
}

// Artifact locations the CLI writes and CI collects. Named here so a change to
// where a build lands does not require editing YAML by hand.
const (
	iosSimArtifactDir  = "build/ios/DerivedData/Build/Products/Debug-iphonesimulator"
	androidArtifactDir = "build/android"
)

// renderDesktopMatrix writes the matrix include list, indented to sit under
// `include:` in the workflow.
func renderDesktopMatrix(indent string) string {
	var b strings.Builder
	for _, t := range desktopTargets {
		fmt.Fprintf(&b, "%s- { name: %s, target: %s, runner: %s, artifact: %s, out: %s }\n",
			indent, t.name, t.target, t.runner, t.artifact, t.outDir)
	}
	return strings.TrimRight(b.String(), "\n")
}

// renderCITemplate substitutes every placeholder in a workflow template.
func renderCITemplate(body, projectName string) string {
	body = strings.ReplaceAll(body, "{{PROJECT_NAME}}", projectName)
	for key, val := range ciActions {
		body = strings.ReplaceAll(body, "{{"+key+"}}", val)
	}
	body = strings.ReplaceAll(body, "{{IOS_SIM_ARTIFACT_DIR}}", iosSimArtifactDir)
	body = strings.ReplaceAll(body, "{{ANDROID_ARTIFACT_DIR}}", androidArtifactDir)
	// Indent-sensitive, so it is substituted last and matched with its own
	// leading whitespace stripped from the template line.
	body = strings.ReplaceAll(body, "          {{DESKTOP_MATRIX}}", renderDesktopMatrix("          "))
	return body
}
