// CLI tool for creating and managing irgo projects
package main

import (
	"fmt"
	"os"
	"runtime/debug"
	"strings"
)

// version is what the CLI reports and what artifact stamps are compared
// against. It is resolved from the build rather than hardcoded: `go tool irgo`
// builds whatever go.mod pins, so a constant here would describe a release
// nobody is necessarily running — as it did, claiming 0.4.0 for every fork
// build. See cliVersion.
var version = cliVersion()

// fallbackVersion is used only when a binary carries no build information,
// which happens if it was assembled outside the module system.
const fallbackVersion = "0.4.0"

// cliVersion reports the module version this binary was built from, falling
// back to the VCS revision for a local checkout — which is exactly the case
// `irgo pin --local` creates, and worth naming so a surprising result is
// traceable to a working tree rather than a release.
func cliVersion() string {
	bi, ok := debug.ReadBuildInfo()
	if !ok {
		return fallbackVersion
	}
	if v := bi.Main.Version; v != "" && v != "(devel)" {
		return v
	}
	var rev string
	dirty := ""
	for _, s := range bi.Settings {
		switch s.Key {
		case "vcs.revision":
			rev = s.Value
		case "vcs.modified":
			if s.Value == "true" {
				dirty = "-modified"
			}
		}
	}
	if rev != "" {
		if len(rev) > 12 {
			rev = rev[:12]
		}
		return "devel-" + rev + dirty
	}
	return fallbackVersion + "-devel"
}

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	var err error
	switch os.Args[1] {
	case "new":
		if len(os.Args) < 3 {
			fmt.Println("Usage: irgo new <project-name>")
			os.Exit(1)
		}
		err = newProject(os.Args[2])

	case "dev":
		err = runDev()

	case "serve":
		err = runServe()

	case "build":
		if len(os.Args) < 3 {
			fmt.Println("Usage: irgo build <ios|android|desktop|all>")
			os.Exit(1)
		}
		target := os.Args[2]
		if target == "desktop" {
			platform := ""
			if len(os.Args) > 3 {
				platform = os.Args[3]
			}
			err = buildDesktop(platform)
		} else {
			team := ""
			for i := 3; i < len(os.Args)-1; i++ {
				if os.Args[i] == "--team" {
					team = os.Args[i+1]
				}
			}
			err = runBuild(target,
				hasFlag(os.Args[3:], "--sim", "-s"),
				hasFlag(os.Args[3:], "--device", "-D"),
				team)
		}

	case "run":
		if len(os.Args) < 3 {
			fmt.Println("Usage: irgo run <ios|android|desktop> [--dev] [--avd NAME] [--no-window]")
			os.Exit(1)
		}
		platform := os.Args[2]
		devMode := hasFlag(os.Args[3:], "--dev", "-d")
		headless := hasFlag(os.Args[3:], "--no-window", "-nw")
		avd := "irgo"
		for i := 3; i < len(os.Args); i++ {
			if os.Args[i] == "--avd" && i+1 < len(os.Args) {
				avd = os.Args[i+1]
			}
		}

		team := ""
		for i := 3; i < len(os.Args)-1; i++ {
			if os.Args[i] == "--team" {
				team = os.Args[i+1]
			}
		}
		if platform == "desktop" {
			err = runDesktop(devMode, hasFlag(os.Args[3:], "--built", "-b"))
		} else if platform == "ios" && hasFlag(os.Args[3:], "--device", "-D") {
			err = runIOSDevice(team)
		} else {
			err = runMobile(platform, devMode, avd, headless)
		}

	case "package":
		if len(os.Args) < 3 {
			fmt.Println("Usage: irgo package <ios|android|macos|windows> [flags]")
			os.Exit(1)
		}
		target := os.Args[2]
		team, exportMethod, out := "", "app-store", ""
		keystore, keystorePass, keyAlias, keyPass := "", "", "", ""
		version, publisher, icon, cert, certPass := "", "", "", "", ""
		identity, appleID, password := "", "", ""
		notarize, dmg := false, false
		// setup has its own args (store / --check) — don't parse package flags.
		if target != "setup" {
			for i := 3; i < len(os.Args); i++ {
				next := func() string {
					if i+1 < len(os.Args) {
						i++
						return os.Args[i]
					}
					return ""
				}
				switch os.Args[i] {
				case "--team":
					team = next()
				case "--export-method":
					exportMethod = next()
				case "--keystore":
					keystore = next()
				case "--keystore-pass":
					keystorePass = next()
				case "--key-alias":
					keyAlias = next()
				case "--key-pass":
					keyPass = next()
				case "--version":
					version = next()
				case "--publisher":
					publisher = next()
				case "--icon":
					icon = next()
				case "--cert":
					cert = next()
				case "--cert-pass":
					certPass = next()
				case "--identity":
					identity = next()
				case "--apple-id":
					appleID = next()
				case "--password":
					password = next()
				case "--notarize":
					notarize = true
				case "--dmg":
					dmg = true
				case "-o", "--output":
					out = next()
				default:
					fmt.Printf("Unknown flag: %s\n", os.Args[i])
					os.Exit(1)
				}
			}
		}
		switch target {
		case "setup":
			// irgo package setup              → static guide (where to get every value)
			// irgo package setup <store>      → interactive wizard for that store
			// irgo package setup --check      → status report for every store
			// irgo package setup --check <s>  → status report for one store
			check := false
			store := ""
			for _, a := range os.Args[3:] {
				switch a {
				case "--check", "-c":
					check = true
				default:
					store = a
				}
			}
			if check {
				stores := []string{"ios", "android", "windows", "macos", "reviews-apple-ios", "reviews-apple-mac", "reviews-android"}
				if store != "" {
					stores = []string{store}
				}
				for _, s := range stores {
					checkStoreConfig(s)
				}
			} else if store != "" {
				if storeConfigValues(store) == nil {
					fmt.Printf("unknown setup target: %s (use ios, android, windows, macos, or reviews-*)\n", store)
					os.Exit(1)
				}
				missing := missingStoreConfig(store)
				if len(missing) == 0 {
					fmt.Printf("Nothing missing for %s — defaults cover it. Run `irgo package setup` for the full guide.\n", store)
				} else if wErr := runSetupWizard(store, missing); wErr != nil {
					err = wErr
				}
			} else {
				packageSetupGuide()
			}
		case "ios":
			err = packageIOS(team, exportMethod, out)
		case "android":
			err = packageAndroid(keystore, keystorePass, keyAlias, keyPass, version, icon, out)
		case "macos":
			err = packageMacOS(identity, notarize, appleID, team, password, dmg, icon, out)
		case "windows":
			err = packageWindows(publisher, version, icon, cert, certPass, out)
		default:
			err = fmt.Errorf("unknown package target: %s (use ios, android, macos, windows, or setup)", target)
		}

	case "ios":
		if len(os.Args) > 2 && os.Args[2] == "team" {
			err = runIOSTeamCmd(os.Args[3:])
		} else {
			err = fmt.Errorf("usage: irgo ios team [TEAM_ID]")
		}

	case "clean":
		err = runClean(hasFlag(os.Args[2:], "--all", "-a"))

	case "uninstall":
		// The inverse of `irgo run`: remove the app from the simulator,
		// emulator or this machine. Distinct from uninstall-tools, which
		// removes the toolchain rather than the app.
		target := "all"
		if len(os.Args) > 2 {
			target = os.Args[2]
		}
		err = runAppUninstall(target)

	case "templ":
		err = runTempl()

	case "assets":
		err = ensureAssets()

	case "ci":
		err = runCI(hasFlag(os.Args[2:], "--force", "-f"))

	case "pin":
		err = runPin(os.Args[2:])

	case "upgrade":
		err = runUpgrade(hasFlag(os.Args[2:], "--force"), hasFlag(os.Args[2:], "--diff"))

	case "test":
		err = runTest()

	case "install-tools":
		if len(os.Args) > 2 && os.Args[2] == "android" {
			avd := "irgo"
			for i := 3; i < len(os.Args); i++ {
				if os.Args[i] == "--avd" && i+1 < len(os.Args) {
					avd = os.Args[i+1]
				}
			}
			err = installAndroidTools(hasFlag(os.Args[3:], "--emulator", "-e"), avd)
		} else {
			err = installTools()
		}

	case "uninstall-tools":
		// Every install path has an inverse: bare form undoes install-tools
		// (templ/air/gomobile/gobind + mingw-w64), `android` undoes the
		// Android toolchain.
		if len(os.Args) > 2 && os.Args[2] == "android" {
			err = uninstallAndroidTools(hasFlag(os.Args[3:], "--remove-jdk"))
		} else {
			err = uninstallTools(hasFlag(os.Args[2:], "--all"))
		}

	case "doctor":
		// Bare `irgo doctor` answers the first question a dev has on a new
		// machine: what can I actually build here?
		if len(os.Args) > 2 && os.Args[2] == "android" {
			err = doctorAndroid()
		} else if len(os.Args) > 2 && !strings.HasPrefix(os.Args[2], "-") {
			err = fmt.Errorf("unknown doctor target: %s (use `irgo doctor` or `irgo doctor android`)", os.Args[2])
		} else if hasFlag(os.Args[2:], "--fix") {
			err = runDoctorFix()
		} else {
			err = doctorHost(hasFlag(os.Args[2:], "--strict"))
		}

	case "reviews":
		err = reviewsCommand(os.Args[2:])

	case "version", "-v", "--version":
		fmt.Printf("irgo %s\n", version)
		// Build info reports the *required* module version, so a project
		// tracking a fork or a checkout would be told v0.4.0 while running
		// something else entirely. go.mod has the answer.
		if r := projectReplacement(); r != "" {
			fmt.Printf("  running: %s\n", r)
		}

	case "help", "-h", "--help":
		if len(os.Args) > 2 {
			printCommandHelp(os.Args[2])
		} else {
			printUsage()
		}

	default:
		fmt.Printf("Unknown command: %s\n", os.Args[1])
		printUsage()
		os.Exit(1)
	}

	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Println(`irgo - Hypermedia framework for mobile and desktop apps

Usage:
  irgo <command> [arguments]

Commands:
  new <name>       Create a new irgo project
  dev              Run development server with hot reload
  serve            Run server without file watching
  build <target>   Build for mobile/desktop (ios, android, desktop, or all)
  run <platform>   Build and run on simulator or desktop
  package <target> Package for stores (ios .ipa, android .aab, macos .app/.dmg, windows .msix)
  package setup    Guide: how to get every store config value
  reviews <ios|android>   Monitor app store reviews (reply on android)
  ios team [ID]    List development teams, or select one for device builds
  clean [--all]    Remove generated output (--all: also node_modules/caches)
  uninstall <p>    Remove the installed app (ios, android, desktop, or all)
  templ            Generate templ files
  assets           Regenerate embedded assets (templ + Tailwind CSS)
  ci [--force]     Scaffold GitHub Actions workflows for every target
  pin [target]     Show or change which irgo this project builds against
  upgrade          Refresh framework scaffolding, leaving your code alone
  test             Run tests
  install-tools    Install required dev tools (gomobile, templ, air)
  install-tools android   Install Android SDK + NDK (+ emulator with --emulator)
  uninstall-tools  Remove the Go tools irgo installed (--all: also yours)
  uninstall-tools android Remove everything install-tools android installed
  doctor [--strict] Report what this host can and cannot build
  doctor --fix      Repair what can be repaired automatically
  doctor android   Check the Android toolchain is correctly installed
  version          Print version information
  help [command]   Show help for a command

Examples:
  irgo new myapp         Create a new project
  irgo dev               Start dev server with hot reload
  irgo run ios           Build and run on iOS Simulator
  irgo run ios --dev     Hot-reload mode (connects to dev server)
  irgo run ios --device  Build, install and launch on a USB-connected iPhone
  irgo run android       Build and run on Android Emulator
  irgo run android --dev Hot-reload mode (connects to dev server)
  irgo run desktop       Run as desktop app
  irgo run desktop --dev Desktop app with devtools enabled
  irgo run desktop --built
                         Launch the app from irgo build desktop
  irgo build ios         Build iOS framework only
  irgo build ios --sim   Build the runnable iOS Simulator app
  irgo build ios --device --team ID
                         Build the Release app for a device / App Store
  irgo build desktop     Build desktop app for current platform
  irgo build desktop all Build every desktop app this host supports
                         (macOS -> macOS + Windows; installs mingw-w64 if needed)`)
}

func printCommandHelp(cmd string) {
	switch cmd {
	case "new":
		fmt.Println(`irgo new - Create a new irgo project

Usage:
  irgo new <project-name>
  irgo new .              Initialize in current directory

Creates a new project with:
  - main.go           App entry point
  - handlers/         Route handlers
  - templates/        Templ templates
  - static/           CSS and JS assets
  - dev.sh            Development script
  - Makefile          Build targets

Environment:
  IRGO_REPLACE  Pin irgo to a published fork in the generated go.mod, e.g.
                "github.com/joeblew999/irgo v0.4.0-androidapi21.24" ("@" between
                module and version also works). Set this in the consuming repo so
                its app is regenerable from the CLI alone — no hand-edited go.mod.
  IRGO_PATH     Pin to a local irgo checkout instead (for hacking on irgo itself).
                IRGO_REPLACE wins when both are set.`)

	case "dev":
		fmt.Println(`irgo dev - Run development server with hot reload

Usage:
  irgo dev

Starts:
  - Air for Go hot reloading
  - Templ file watcher
  - Tailwind CSS watcher (if configured)

Server runs at http://localhost:8080`)

	case "install-tools":
		fmt.Println(`irgo install-tools - Install required development tools

Usage:
  irgo install-tools             Install Go tools (gomobile, templ, air)
  irgo install-tools android     Install Android SDK + NDK + cmdline-tools
  irgo install-tools android --emulator
                                 Also install the emulator + system image + AVD
  irgo install-tools android --emulator --avd <name>
                                 AVD name (default "irgo")

Android toolchain (pinned, known-good, fully cross-platform):
  - JDK 17 — auto-downloaded (Temurin via Adoptium) into ~/.irgo/jdks, no
    brew/apt/winget needed; respects an existing JAVA_HOME when it is JDK 17
  - cmdline-tools, platform-tools, platforms android-34/35, build-tools
  - NDK r26 (required: gomobile's bind defaults to API 16, NDK r27+ rejects it)

Respects ANDROID_HOME (defaults: ~/Library/Android/sdk on macOS,
~/Android/Sdk on Linux, %LOCALAPPDATA%\Android\Sdk on Windows).`)

	case "uninstall-tools":
		fmt.Println(`irgo uninstall-tools android - Remove the Android toolchain

Usage:
  irgo uninstall-tools android [--remove-jdk]

Removes everything install-tools android put in place: gomobile/gobind, the
SDK directory (only if irgo provisioned it), ~/.android, ~/.gradle, temp
clones, and (macOS) emulator prefs. --remove-jdk also removes the managed
JDK at ~/.irgo/jdks.`)

	case "doctor":
		fmt.Println(`irgo doctor - What this host can build

Usage:
  irgo doctor          Capability report for this machine
  irgo doctor --strict Same, but exit non-zero on CLI pin drift (for CI)
  irgo doctor --fix    Repair what irgo can, then list what needs you

--fix repoints xcode-select when it is aimed at the Command Line Tools instead
of Xcode, accepts the Xcode licence and installs its components, and downloads
an iOS simulator runtime if none exists. Each needs sudo or a large download,
which is why it is opt-in rather than part of every build.

Two things cannot be automated: signing in an Apple ID needs credentials and
2FA, and Developer Mode is a toggle Apple gates on the device itself. --fix
gets you to the last step of each and opens Xcode for the first.
  irgo doctor android  Verify the Android toolchain in detail

The plain form lists every target with one of three verdicts:
  ready           buildable now
  auto-installs   buildable; irgo provisions what is missing on first build
  NOT ON THIS OS  impossible here, whatever you install

iOS needs macOS, Linux desktop needs Linux, and Windows desktop builds from
Windows or from macOS via mingw-w64. 'irgo build all' skips what the host
cannot do rather than failing, so the same command works in CI everywhere.`)

	case "build":
		fmt.Println(`irgo build - Build for mobile and desktop platforms

Usage:
  irgo build ios             Build iOS framework (.xcframework)
  irgo build android         Build Android library (.aar)
  irgo build desktop         Build desktop app for current platform
  irgo build desktop macos   Build desktop app for macOS
  irgo build desktop windows Build desktop app for Windows
  irgo build desktop linux   Build desktop app for Linux
  irgo build all             Build all mobile platforms

Requirements:
  - iOS: Xcode and gomobile
  - Android: Android SDK and gomobile
  - Desktop: CGO enabled (C compiler required)
    - macOS: Xcode Command Line Tools
    - Windows: MinGW-w64 or similar
    - Linux: GCC and WebKit2GTK dev packages

Output:
  - iOS: build/ios/Irgo.xcframework
  - Android: build/android/irgo.aar
  - Desktop macOS: build/desktop/macos/<app>.app
  - Desktop Windows: build/desktop/windows/<app>.exe
  - Desktop Linux: build/desktop/linux/<app>`)

	case "templ":
		fmt.Println(`irgo templ - Generate templ files

Usage:
  irgo templ

Runs 'templ generate' to compile .templ files to Go code.`)

	case "run":
		fmt.Println(`irgo run - Build and run on simulator or desktop

Usage:
  irgo run ios              Build and run on iOS Simulator
  irgo run ios --dev        Run iOS with hot-reload (connects to dev server)
  irgo run android          Build and run on Android Emulator
  irgo run android --dev    Run Android with hot-reload (connects to dev server)
  irgo run desktop          Run as desktop app
  irgo run desktop --dev    Run desktop app with devtools enabled

Flags:
  --dev, -d    Development mode.
               - Mobile: Connects to the dev server for hot-reload
                 (iOS Simulator: localhost:8080, Android Emulator: 10.0.2.2:8080)
               - Desktop: Enables browser devtools in webview

Requirements:
  - iOS: Xcode with iOS Simulator
  - Android: Android Studio with emulator
  - Desktop: CGO enabled (see 'irgo help build' for details)

Mobile standard mode (without --dev):
  1. Builds the Go framework with gomobile
  2. Builds the native app project
  3. Installs and launches on simulator/emulator

Mobile dev mode (with --dev):
  1. Starts the dev server with hot reload (air on localhost:8080)
  2. Builds the gomobile framework/AAR only if it doesn't exist yet
     (delete build/ios/Irgo.xcframework or android/Example/app/libs/irgo.aar
     to force a rebuild - the native app still links against it)
  3. Builds, installs and launches the native app on the simulator/emulator
  4. The app loads its UI from the dev server, so Go code changes
     are reflected instantly without rebuilding the native app

Desktop mode:
  1. Starts local HTTP server on auto-selected port
  2. Opens native webview window pointing to localhost
  3. Closes server when window is closed`)

	case "serve":
		fmt.Println(`irgo serve - Run the web server without file watching

Usage: irgo serve

Serves the app over plain HTTP with no rebuild-on-change. Use 'irgo dev' for
hot reload, which air provides natively — neither needs anything installed.`)

	case "uninstall":
		fmt.Println(`irgo uninstall - Remove the installed app

Usage:
  irgo uninstall ios       From the booted simulator
  irgo uninstall android   From the attached device or emulator
  irgo uninstall desktop   From /Applications (macOS)
  irgo uninstall           All of the above

The inverse of irgo run. Not to be confused with irgo uninstall-tools, which
removes the toolchain rather than the app.

An app that is not installed is not an error — the goal is that it is gone.
Reach for this when a stale install is the suspect: the app keeps launching
with old assets or an old bridge API and nothing explains why.`)

	case "pin":
		fmt.Println(`irgo pin - Which irgo does this project build against?

Usage:
  irgo pin                        Show the current pin and what it means
  irgo pin --release              Track the published module
  irgo pin <version>              Track a specific published version
  irgo pin <owner>/<repo>@<tag>   Track a fork
  irgo pin --local [dir]          Build a checkout you are editing

Using irgo and working on irgo are the same activity pointed at different
versions. 'go tool irgo' builds whatever go.mod names, so this is the only
difference between the two — there is no separate toolchain, install step or
set of commands for contributors.

  irgo pin --local ../irgo    then edit the CLI; the next 'go tool irgo'
                              already runs your change, with nothing to
                              reinstall and nothing to tag
  irgo pin --release          go back to the published build

A fork keeps the upstream module path, so Go fetches it from GitHub rather
than the proxy:  go env -w GOPRIVATE='github.com/<owner>/*'`)

	case "upgrade":
		fmt.Println(`irgo upgrade - Move an existing project to this CLI version

Usage:
  irgo upgrade          Refresh framework scaffolding
  irgo upgrade --diff   Also show what the template holds for your files
  irgo upgrade --force  Overwrite your files too (destructive)

Files are split by who owns them.

  Framework — rewritten, because they must match the CLI in use:
    ios/, android/      native shells (builds regenerate these anyway)
    .air.toml           hot-reload config
    .gitignore          tracks which generated paths exist
    .github/workflows   CI
    CLAUDE.md           framework documentation

  Yours — seeded once, never rewritten:
    main.go, app/, handlers/, templates/, static/, mobile/
    irgo.package.toml   your signing team and store settings
    go.mod              your dependencies

When one of your files differs from the current template, upgrade names it
rather than touching it. That is expected for anything you have edited; look
only when an upgrade note says the framework changed how it works.

--force overwrites your code too. That is what a repo which IS generated
output wants, and what an application never does.`)

	case "ci":
		fmt.Println(`irgo ci - Scaffold GitHub Actions workflows

Usage:
  irgo ci           Write .github/workflows (existing files are kept)
  irgo ci --force   Overwrite them

Writes two workflows:

  build.yml    Tests, web binary, desktop for Linux/macOS/Windows, the iOS
               framework and simulator app, and the Android AAR — each on a
               runner of the OS it requires. No secrets needed.
  release.yml  Store packages on tag push. Every job is skipped unless its
               signing secrets exist, so this is useful before you have any.

Nothing is installed first. The workflows call 'go tool irgo', which builds the
CLI version this module requires straight from go.mod — including through a
replace directive when you track a fork. So the pin is go.mod and there is
nothing to keep in step by hand.

There are no SDK/NDK/JDK setup steps either: the CLI provisions its own
toolchains, so 'go tool irgo build android' works on a bare runner.`)

	case "clean":
		fmt.Println(`irgo clean - Remove generated output

Usage:
  irgo clean         Build output, native shells, generated code
  irgo clean --all   Also node_modules and Gradle caches

Removes only what irgo produces — compiled output, store packages, the
scaffolded ios/Example and android/Example shells, _templ.go, the generated
stylesheet, and the gomobile go.work. Nothing you wrote is touched, and
everything removed is rebuilt by the next build.

Reach for this when a stale artifact is the suspect: a framework built by an
older irgo exposes an older bridge API, and the native shells then fail to
compile against it.

--all additionally drops dependency trees and caches, which are slow to
refetch — so the plain form stays quick.`)

	case "assets":
		fmt.Println(`irgo assets - Regenerate the embedded assets

Usage: irgo assets

Regenerates _templ.go and static/css/output.css. Both are gitignored yet
embedded into every build, so a fresh clone has neither.

Every 'irgo build' / 'irgo run' does this for you. Run it directly only when
invoking the Go toolchain yourself (plain 'go test' / 'go build'), which
otherwise compiles against missing templates and ships an unstyled app.

Installs the templ generator (pinned to your go.mod templ version) and the
frontend dependencies if they are absent. Skips CSS when the project has no
"css" script; prefers bun, falls back to npm.`)

	case "test":
		fmt.Println(`irgo test - Run the Go test suite

Usage: irgo test

Equivalent to: go test -v ./...`)

	case "help":
		fmt.Println(`irgo help - Show help

Usage:
  irgo help            List every command
  irgo help <command>  Detail for one command`)

	case "version":
		fmt.Println(`irgo version - Print the CLI version

Usage: irgo version | irgo -v | irgo --version`)

	case "package":
		fmt.Println(`irgo package - Build store-ready artifacts

Usage:
  irgo package ios       [--team ID] [--export-method M] [-o OUT]   .ipa    (macOS only)
  irgo package android   [--keystore F --keystore-pass P --key-alias A --key-pass P] [-o OUT]   .aab
  irgo package macos     [--identity ID] [--notarize --apple-id ID --team ID --password P] [--dmg]   (macOS only)
  irgo package windows   [--publisher CN] [--version V] [--cert F --cert-pass P]   .msix
  irgo package setup [store]     Interactive wizard for the values a store needs
  irgo package setup --check     Report what is set, where it came from, and
                                 exactly how to supply what is missing

Configuration precedence, highest first:
  1. CLI flag                    irgo package android --keystore ...
  2. Environment                 IRGO_ANDROID_KEYSTORE=...  (what CI secrets set)
  3. irgo.package.local.toml     your machine, gitignored — SECRETS go here
  4. irgo.package.toml           shared with your team, committed
  5. auto-derived                e.g. the Team ID read from Xcode

Config is read from irgo.package.toml; flags override it. App icons come from
a single appicon.png. Signing material is never written to the repo.`)

	case "reviews":
		fmt.Println(`irgo reviews - Monitor app store reviews

Usage:
  irgo reviews ios       Recent App Store reviews (iOS)
  irgo reviews mac       Recent App Store reviews (macOS)
  irgo reviews android   Recent Play Store reviews, and reply to them

Credentials come from irgo.package.toml - 'irgo package setup --check' reports
what is missing.`)

	default:
		fmt.Printf("Unknown command: %s\n", cmd)
		printUsage()
	}
}
