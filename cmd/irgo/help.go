// Help text for every command.
//
// Separated from dispatch because main.go was 916 lines, of which two thirds
// were strings: the switch that routes a command and the prose describing it
// change for different reasons and at different times.
package main

import (
	"fmt"
	"strings"
)

func printUsage() {
	fmt.Println(`irgo - one Go codebase for web, desktop, iOS and Android

Usage:
  irgo <noun> <verb> [target] [flags]

One grammar throughout: every command is a noun and a verb.

PROJECT   the repository you are in
  project new <name>       Create one (or "." for the current directory)
  project assets           Regenerate templ + Tailwind CSS
  project test             Run the tests
  project clean [--all]    Remove generated output
  project ci [--force]     Scaffold GitHub Actions workflows
  project upgrade          Take framework updates, leaving your code alone
  project upgrade --check  Name what an upgrade would overwrite (CI-friendly)
  project pin [target]     Which irgo this project builds against
  project config [k] [v]   Show or set a setting (signing, stores, version)

APP       what gets built, run, shipped and installed
  app build <` + buildTargetList() + `>  Build it
  app run <ios|android|desktop>                   Build and launch it
  app deploy cloudflare                           Build the Worker, put it live
  app package <ios|android|macos|windows>         Store artifacts
  app install <ios|android|desktop>               Install a build — no rebuild
  app remove <ios|android|desktop>                Uninstall it
  app reviews <ios|mac|android>                   Monitor store reviews

TOOLS     the toolchains on this machine
  tools doctor [android]   What this host can build; --fix repairs it
  tools install [android]  Install what builds need
  tools remove [android]   Undo it — shows what it will delete, and asks

SERVER    the development server
  server dev               Hot reload
  server serve             No file watching

  version                  Print version information
  help [command]           Detail for one command

Examples:
  irgo project new myapp             Create a project
  irgo server dev                    Hot-reload server on :8080
  irgo tools doctor                  What can this machine build?
  irgo app run ios                   iOS Simulator
  irgo app run ios --device          A USB-connected iPhone
  irgo app build desktop all         Every desktop target this host supports
  irgo app package macos --dmg       Signed .app and a DMG
  irgo app install desktop           Put the built app in /Applications
  irgo app deploy cloudflare         Live on Cloudflare Workers, SSE included
  irgo tools remove --yes            Undo everything irgo installed
  irgo project config ios.team       Which team signs, and what is available
  irgo project pin local ../irgo     Build a checkout you are editing

Nothing needs installing first: toolchains provision themselves when a command
needs them, and go.mod pins the CLI so ` + "`go tool irgo`" + ` always matches the project.`)
}

// printCommandHelp explains one command. The key is the grammar itself —
// "app run", not "run" — because two nouns share verbs (app install puts a
// built app on a device, tools install provisions a toolchain) and a
// verb-keyed lookup answered one of them with the other.
func printCommandHelp(noun, verb string) {
	switch strings.TrimSpace(noun + " " + verb) {

	// ---- project -----------------------------------------------------------

	case "project new":
		fmt.Println(`irgo project new - Create a project, or regenerate this one

Usage:
  irgo project new <name>     Create ./<name>
  irgo project new .          Generate into the current directory
  irgo project new --check    Report what regenerating would change, write nothing

What it writes:
  main.go, handlers/, templates/, static/   your app
  ios/, android/                            native shells
  .github/workflows/                        CI for every target
  go.mod                                    pins the CLI via a tool directive

Files that are yours are seeded once and never overwritten: go.mod, README.md,
irgo.package.toml and appicon.png. Everything else is regenerated, so fix the
template rather than the generated copy.

--check exits non-zero if regenerating would change a file. That asserts the
repo IS unmodified CLI output, which is true of example repos and not of a real
app — for those, use irgo project upgrade --check.`)

	case "project upgrade":
		fmt.Println(`irgo project upgrade - Take framework updates, leaving your code alone

Usage:
  irgo project upgrade           Refresh framework-owned scaffolding
  irgo project upgrade --check   Name what an upgrade would overwrite, change nothing
  irgo project upgrade --diff    Also show what the template holds for your files
  irgo project upgrade --force   Overwrite your files too (destructive)

Framework-owned (replaced): ios/, android/, mobile/, .air.toml, .gitignore,
CLAUDE.md, AGENTS.md, and the generated workflows.
Yours (never rewritten):    main.go, handlers/, templates/, static/, README.md,
                            go.mod, irgo.package.toml, appicon.png.

Anything overwritten is copied to <file>.irgo-bak first.

--check is the CI-friendly form: it exits non-zero when a framework-owned file
has been hand-edited, i.e. when an upgrade is about to discard that edit.`)

	case "project pin":
		fmt.Println(`irgo project pin - Choose which irgo this project builds against

Usage:
  irgo project pin                     Show the current pin and where it came from
  irgo project pin local [dir]         Build a checkout you are editing
  irgo project pin release             Track the published module
  irgo project pin <version>           A published version, e.g. v0.4.0
  irgo project pin <owner>/<repo>@<tag>  A fork

go.mod is the only pin: ` + "`go tool irgo`" + ` builds whatever it names, so there is
nothing installed globally to fall out of step. A pin that does not resolve
leaves go.mod untouched rather than half-written.

A fork keeps the upstream module path, so the proxy cannot serve it:
  go env -w GOPRIVATE='github.com/<owner>/*'`)

	case "project ci":
		fmt.Println(`irgo project ci - Scaffold GitHub Actions workflows

Usage:
  irgo project ci            Write .github/workflows, keeping any that exist
  irgo project ci --force    Regenerate them

Writes build.yml (every target, on the OS each one needs) and release.yml
(signed store artifacts). Both are generated from the CLI — the desktop matrix,
the artifact paths and the action versions come from irgo, so they stay correct
as it changes. Edit the template, not the output.

Two jobs in build.yml are opt-in, off unless the repository sets a variable:
  IRGO_TOOLCHAIN_ROUNDTRIP=true   install → doctor → build → uninstall, and
                                  assert the machine comes back clean
  IRGO_GENERATED_REPO=true        assert the repo still matches project new

The round-trip deletes the Android toolchain to prove the uninstall works, so
it refuses to run anywhere but a github-hosted runner.`)

	case "project clean":
		fmt.Println(`irgo project clean - Remove generated output

Usage:
  irgo project clean         Generated code and build output
  irgo project clean --all   Also the scaffolded native shells

Removes _templ.go, static/css/output.css, build/, tmp/ and dist/. With --all,
ios/Example and android/Example go too; they are scaffolded again on the next
build, which takes a gomobile rebuild.

Nothing you wrote is touched.`)

	case "project config":
		fmt.Println(`irgo project config - Show or set a project setting

Usage:
  irgo project config              Every setting, its value, and where it came from
  irgo project config <key>        One setting
  irgo project config <key> <val>  Set it

Settings live in irgo.package.toml. Precedence: environment variable, then
irgo.package.local.toml (gitignored), then irgo.package.toml.

Secrets belong in the local file or the environment — irgo.package.toml is
committed. Values you have to discover rather than type, such as which signing
teams exist, are reported by irgo tools doctor.`)

	case "project assets":
		fmt.Println(`irgo project assets - Regenerate templ and Tailwind CSS

Usage:
  irgo project assets

Writes _templ.go and static/css/output.css. Both are gitignored and compiled
into the binary, so a plain ` + "`go build`" + ` needs this first; every irgo build and
run does it for you.

templ and the Tailwind standalone binary are installed on demand. There is no
Node, npm or package.json.`)

	case "project test":
		fmt.Println(`irgo project test - Run the tests

Usage:
  irgo project test

Regenerates assets, then runs go test ./... — so tests that render templates
see the current ones rather than whatever was last on disk.`)

	// ---- app ---------------------------------------------------------------

	case "app build":
		fmt.Println(`irgo app build - Build for a target

Usage:
  irgo app build ios              Device framework (Irgo.xcframework)
  irgo app build ios --sim        Simulator build
  irgo app build ios --device     Build and sign for a USB device
  irgo app build android          AAR for the native shell
  irgo app build desktop          This host
  irgo app build desktop <goos>   linux, darwin or windows
  irgo app build cloudflare       A Cloudflare Worker (WASM) in build/worker
  irgo app build all              Everything this host can produce

Flags:
  --sim, -s        Simulator rather than device (iOS)
  --device, -D     A real device (iOS)
  --team <id>      Apple Team ID to sign with

Cloudflare compiles the same router to WebAssembly and serves it from a
Worker, SSE included. Shared state cannot live in a Go variable there — every
request gets a fresh runtime — so keep it in a KV, D1 or Durable Object
binding. Per-connection state within one SSE stream is fine.

Toolchains install themselves: the Android SDK, NDK, JDK and gomobile are
provisioned on first use. Cross-building is limited by the host — macOS can
produce macOS and Windows desktop binaries, Linux only Linux. Ask irgo tools
doctor what this machine can do.`)

	case "app run":
		fmt.Println(`irgo app run - Build and launch

Usage:
  irgo app run ios              iOS Simulator
  irgo app run ios --device     A USB-connected iPhone
  irgo app run android          Android emulator
  irgo app run desktop          Native desktop window

Flags:
  --dev, -d      Hot reload: serves the app from the dev server on :8080 and
                 reloads on save, instead of embedding it in the binary
  --device, -D   A real iPhone rather than the Simulator
  --team <id>    Apple Team ID to sign with
  --built, -b    Desktop: launch the existing build rather than rebuilding

Android uses whatever device or emulator is already connected, and only boots
its own when nothing is. To run a particular AVD, start it first — there is no
flag to disagree with what is actually attached.`)

	case "app package":
		fmt.Println(`irgo app package - Build a store artifact

Usage:
  irgo app package ios        Signed .ipa
  irgo app package android    Signed .aab
  irgo app package macos      Signed .app, notarized; --dmg for a disk image
  irgo app package windows    .msix
  irgo app package setup --check   What each store needs, and what is missing

Every signing setting can be given on the command line as well as in
irgo.package.toml, which is how CI supplies secrets without writing them to disk:

  iOS       --team <id>  --export-method <app-store|ad-hoc|development>
  Android   --keystore <path>  --keystore-pass <s>  --key-alias <s>  --key-pass <s>
  macOS     --identity <cert>  --notarize  --apple-id <email>  --password <s>  --dmg
  Windows   --publisher <dn>  --cert <pfx>  --cert-pass <s>

  --version <v>   Override common.version
  --icon <path>   Override the source icon
  --output, -o    Where to write the artifact

Assets are regenerated first, so a package cannot ship a stale stylesheet.

Signing settings come from irgo.package.toml — see irgo project config. Missing
credentials are reported before the build rather than at the end of it.`)

	case "app deploy":
		fmt.Println(`irgo app deploy - Put the app live

Usage:
  irgo app deploy cloudflare    Build the Worker and deploy it

Builds first, so what goes live is what the current source produces.

Cloudflare needs wrangler, which is a Node program — and Cloudflare does not
support the bun runtime. irgo downloads its own Node into ~/.irgo rather than
asking you to install one, and irgo tools remove takes it away again. A working
node already on PATH is used instead.

Credentials:
  CLOUDFLARE_API_TOKEN   required in CI
                         From a terminal, wrangler opens a browser instead.

The Worker's name and any bindings come from wrangler.toml, which is yours
after irgo seeds it.`)

	case "app install":
		fmt.Println(`irgo app install - Install what you already built

Usage:
  irgo app install ios        Onto the running Simulator
  irgo app install android    Onto the connected device or emulator
  irgo app install desktop    Into /Applications (macOS)

Installs the existing artifact and does not rebuild. Build first if there is
nothing there yet.`)

	case "app remove":
		fmt.Println(`irgo app remove - Uninstall it again

Usage:
  irgo app remove ios
  irgo app remove android
  irgo app remove desktop

The exact inverse of irgo app install. Every install irgo performs can be
undone by irgo, so nothing it puts on a machine has to be hunted down by hand.`)

	case "app reviews":
		fmt.Println(`irgo app reviews - Read and answer store reviews

Usage:
  irgo app reviews <ios|mac|android>
  irgo app reviews ios --new
  irgo app reviews ios --reply <id> --text "..."

Flags:
  --limit <n>     How many to fetch
  --new           Only ones you have not seen
  --reply <id>    Reply to one review
  --text "..."    The reply body

Needs the store credentials in irgo.package.toml under [reviews] — see
irgo project config.`)

	// ---- tools -------------------------------------------------------------

	case "tools doctor":
		fmt.Println(`irgo tools doctor - What this machine can build

Usage:
  irgo tools doctor            Every target, and what is missing for the rest
  irgo tools doctor android    The Android toolchain in detail
  irgo tools doctor --fix      Install what is missing
  irgo tools doctor --strict   Exit non-zero if anything is missing (CI)

Reports the Go toolchain, templ and Tailwind, Xcode and its signing teams, and
the Android SDK/NDK/JDK with the emulator and AVDs.`)

	case "tools install":
		fmt.Println(`irgo tools install - Provision what builds need

Usage:
  irgo tools install                        Everything this host can use
  irgo tools install android                SDK, NDK, JDK 17 and gomobile
  irgo tools install android --emulator     Also a system image and an AVD
  irgo tools install android --avd <name>   Name that AVD (default "irgo")

Everything lands under ~/.irgo or the Android SDK home — no system package
manager, on any OS. Running it again installs only what is missing.

You rarely need this: a build provisions what it needs. It exists so a large
download can be done deliberately rather than in the middle of a build.`)

	case "tools remove":
		fmt.Println(`irgo tools remove - Undo it

Usage:
  irgo tools remove              What irgo installed for this host
  irgo tools remove android      The Android toolchain
  irgo tools remove --all        Everything, including the SDK and AVDs
  irgo tools remove --keep-jdk   Leave the managed JDK in place
  irgo tools remove --yes, -y    Do not ask

Shows what it will delete, with sizes, and asks first. Outside a terminal it
refuses rather than assuming yes.

Only what irgo installed is removed: each install leaves a marker, and anything
without one is left alone, so a toolchain you set up yourself survives.`)

	// ---- server ------------------------------------------------------------

	case "server dev":
		fmt.Println(`irgo server dev - Development server with hot reload

Usage:
  irgo server dev

Serves on http://localhost:8080 and rebuilds on save: templ, Tailwind and the
Go binary. From an Android emulator the same server is http://10.0.2.2:8080,
which is what irgo app run android --dev connects to.

air is installed on demand.`)

	case "server serve":
		fmt.Println(`irgo server serve - Run the app without watching files

Usage:
  irgo server serve

Serves on http://localhost:8080. Regenerates assets once at startup and then
leaves them alone — for checking a build, or running the web target.`)

	// ---- nouns and fallbacks ------------------------------------------------

	case "project", "app", "tools", "server":
		fmt.Printf("irgo %s - %s\n\n", noun, nounSummary[noun])
		fmt.Println("Verbs:")
		for _, v := range nounVerbs[noun] {
			fmt.Printf("  irgo %s %s\n", noun, v)
		}
		fmt.Printf("\nDetail for one:  irgo help %s <verb>\n", noun)

	case "version":
		fmt.Println(`irgo version - Print the version

Usage:
  irgo version

Prints the CLI version and, in a project, which irgo go.mod resolves to — a
release, a fork tag or a local checkout.`)

	default:
		fmt.Printf("No help for %q.\n\n", strings.TrimSpace(noun+" "+verb))
		printUsage()
	}
}

// nounSummary is the one-line description of each noun, used when help is
// asked for a noun rather than a command.
var nounSummary = map[string]string{
	"project": "the repository you are in",
	"app":     "what gets built, run, shipped and installed",
	"tools":   "the toolchains on this machine",
	"server":  "the development server",
}

// verbSummary is the one line each command gets in generated documentation.
// It sits next to the help text so a new command is described once, in the
// same file, rather than in a README that then has to be remembered.
var verbSummary = map[string]string{
	"project new":     "Create a project, or regenerate this one",
	"project clean":   "Remove generated output",
	"project upgrade": "Take framework updates, leaving your code alone",
	"project pin":     "Choose which irgo this project builds against",
	"project ci":      "Scaffold the GitHub Actions workflows",
	"project assets":  "Regenerate templ + Tailwind (builds do this already)",
	"project test":    "Run the tests",
	"project config":  "Show or set a setting (signing, stores, version)",

	"app build":   "Build for ios, android, desktop, cloudflare, or all",
	"app run":     "Build and launch; --dev for hot reload",
	"app package": "Store artifacts (.ipa, .aab, .app, .msix)",
	"app deploy":  "Put the app live on Cloudflare",
	"app install": "Install what you already built — no rebuild",
	"app remove":  "Uninstall it again",
	"app reviews": "Read store reviews",

	"tools install": "Provision what builds need",
	"tools remove":  "Undo it — shows what it will delete, and asks",
	"tools doctor":  "What this host can build; --fix repairs it",

	"server dev":   "Web server with hot reload",
	"server serve": "Web server without file watching",
}

// renderCommandTable writes the command reference that ships in the generated
// README. It is built from the same table the CLI dispatches on, so the
// documented commands are the commands — the previous README was written by
// hand and still described `irgo doctor`, `irgo dev` and `irgo ios team` long
// after all three were renamed.
func renderCommandTable() string {
	var b strings.Builder
	b.WriteString("| Command | What it does |\n|---|---|\n")
	for _, noun := range []string{"project", "app", "tools", "server"} {
		for _, verb := range nounVerbs[noun] {
			key := noun + " " + verb
			fmt.Fprintf(&b, "| `%s` | %s |\n", key, verbSummary[key])
		}
	}
	return strings.TrimRight(b.String(), "\n")
}
