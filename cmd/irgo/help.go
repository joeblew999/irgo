// Help text for every command.
//
// Separated from dispatch because main.go was 916 lines, of which two thirds
// were strings: the switch that routes a command and the prose describing it
// change for different reasons and at different times.
package main

import "fmt"

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

APP       what gets built, shipped and installed
  app build <ios|android|desktop|all>      Build it
  app run <ios|android|desktop>            Build and launch it
  app package <ios|android|macos|windows>  Store artifacts
  app install <platform>   Install what you already built — no rebuild
  app remove <platform>    Uninstall it
  app reviews <ios|mac|android>            Monitor store reviews

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
  irgo tools remove --yes            Undo everything irgo installed
  irgo project config ios.team       Which team signs, and what is available
  irgo project pin --local ../irgo   Build a checkout you are editing

Earlier spellings (irgo build, irgo doctor, irgo dev, ...) still work and say
where they moved.

Nothing needs installing first: toolchains provision themselves when a command
needs them, and go.mod pins the CLI so ` + "`go tool irgo`" + ` always matches the project.`)
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

	case "install-tools", "tools":
		fmt.Println(`irgo tools - The toolchains on this machine

Usage:
  irgo tools doctor [android] [--fix|--strict]   What this host can build
  irgo tools install [android] [--emulator]      Install what builds need
  irgo tools remove  [android] [--yes] [--all] [--keep-jdk]   Undo it

You rarely need install: every build provisions what it needs, when it needs
it. It exists to set a machine up in one go, and remove exists so that can be
undone — a machine that cannot return to a known state hides provisioning bugs
behind whatever was left lying around.

WHAT IRGO MANAGES

Go tools, via go install into GOBIN:
  templ      pinned to the templ version in your go.mod, so the generator and
             the library cannot disagree
  air        hot reload for irgo dev
  gomobile   and gobind, for the iOS/Android bindings

Downloaded and pinned, under ~/.irgo:
  Tailwind   v4.3.3 standalone binary — no Node, npm or bun anywhere
  JDK 17     Temurin via Adoptium, into ~/.irgo/jdks; an existing JAVA_HOME is
             respected when it is actually 17 (Gradle 8.2 fails on 21+)

Android SDK, into ANDROID_HOME:
  cmdline-tools, platform-tools, build-tools 35, platforms 34 and 35
  NDK r26    required: gomobile binds at API 16 by default and r27+ rejects it
  emulator   with --emulator, plus a system image and an AVD (default "irgo")

Host packages, via brew/apt/pacman, only when a build actually needs them:
  mingw-w64  cross-compiling the Windows desktop app from macOS
  webkit2gtk GTK3 + WebKit2GTK for the Linux desktop webview

Android Studio is not involved, and nothing here is required up front.

REMOVING

  irgo tools remove              everything irgo installed, Android included
  irgo tools remove android      only the Android toolchain
  irgo tools remove --all        also copies irgo did not install
  irgo tools remove --keep-jdk   spare the managed JDK, the slowest to refetch
  irgo tools remove --yes        skip the confirmation

It shows what it will delete, with sizes, and asks first — the Android SDK
alone is several gigabytes. Outside a terminal it refuses rather than assuming
yes, so a script cannot quietly wipe an SDK; pass --yes there.

Removal is marker-guarded: anything irgo did not install is reported and kept,
so your own templ or JDK survives. --all overrides that.

ANDROID_HOME defaults to ~/Library/Android/sdk on macOS, ~/Android/Sdk on
Linux, %LOCALAPPDATA%\Android\Sdk on Windows.`)

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

	case "app", "uninstall":
		fmt.Println(`irgo app - The app installed on a device, simulator or this machine

Usage:
  irgo app install <platform>   Install an already-built app
  irgo app remove <platform>    Remove it

  irgo app remove ios       From the booted simulator
  irgo app remove android   From the attached device or emulator
  irgo app remove desktop   From /Applications (macOS)
  irgo app remove           All of the above

The inverse of irgo run. 'irgo tools remove' is the separate thing that removes
the toolchain — these used to be one hyphen apart, which was a trap.

install does not build. It takes what is already there — the packaged artifact
if one exists, since that is what ships, otherwise the development build — so
you can check the thing itself rather than the thing plus a rebuild:

  ios       the Simulator app from 'irgo build ios --sim'
  android   the debug APK, via adb
  desktop   the .app into /Applications (macOS), packaged build preferred

An app that is not installed is not an error — the goal is that it is gone.
Reach for this when a stale install is the suspect: the app keeps launching
with old assets or an old bridge API and nothing explains why.`)

	case "ios":
		fmt.Println(`irgo ios - iOS-specific settings

Usage:
  irgo ios team          List the development teams Xcode knows about
  irgo ios team <ID>     Use that team for device builds and packaging

The selection is written to irgo.package.toml under [ios] team, so it applies
to every later build without being passed again. A flag or IRGO_IOS_TEAM still
wins for a one-off.

With several teams and none chosen, a device build stops and lists them rather
than picking one — the wrong team fails later as an Apple-side error that never
mentions the team.`)

	case "config":
		fmt.Println(`irgo project config - Settings for this project

Usage:
  irgo project config                  Every setting, its value and its source
  irgo project config <key>            One setting
  irgo project config <key> <value>    Set it

Settings live in irgo.package.toml, which is committed. Precedence, highest
first: a command flag, an environment variable (what a CI secret is called),
irgo.package.local.toml (gitignored, for secrets), then this file.

Secrets — passwords, certificates, keys — are marked when you set them and
belong in irgo.package.local.toml or the environment, not the committed file.

This replaced 'irgo ios team', which was the only command named after a
platform. Platforms are targets (irgo app build ios), never nouns, and one
accessor covers every setting rather than growing a command per value. Where
valid values are discoverable rather than looked up — signing teams — they are
listed alongside.`)

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
  irgo upgrade --check  Report what an upgrade would overwrite, change nothing
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
  irgo clean --all   Also Gradle caches and other slow-to-refetch state

Removes only what irgo produces — compiled output, store packages, the
scaffolded ios/Example and android/Example shells, _templ.go, the generated
stylesheet, and the gomobile go.work. Nothing you wrote is touched, and
everything removed is rebuilt by the next build.

Reach for this when a stale artifact is the suspect: a framework built by an
older irgo exposes an older bridge API, and the native shells then fail to
compile against it.

--all additionally drops caches that are slow to refetch, so the plain form
stays quick.`)

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

// deprecated warns that a command has moved, then runs it anyway. Renaming
// without an alias turns a naming cleanup into a broken script for everyone
// who already had one.
