// CLI tool for creating and managing irgo projects
package main

import (
	"fmt"
	"os"
)

var version = "0.4.0"

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

		if platform == "desktop" {
			err = runDesktop(devMode)
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

	case "templ":
		err = runTempl()

	case "assets":
		err = ensureAssets()

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
		} else if len(os.Args) > 2 {
			err = fmt.Errorf("unknown doctor target: %s (use `irgo doctor` or `irgo doctor android`)", os.Args[2])
		} else {
			err = doctorHost()
		}

	case "reviews":
		err = reviewsCommand(os.Args[2:])

	case "version", "-v", "--version":
		fmt.Printf("irgo %s\n", version)

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
  templ            Generate templ files
  assets           Regenerate embedded assets (templ + Tailwind CSS)
  test             Run tests
  install-tools    Install required dev tools (gomobile, templ, air)
  install-tools android   Install Android SDK + NDK (+ emulator with --emulator)
  uninstall-tools  Remove the Go tools irgo installed (--all: also yours)
  uninstall-tools android Remove everything install-tools android installed
  doctor           Report what this host can and cannot build
  doctor android   Check the Android toolchain is correctly installed
  version          Print version information
  help [command]   Show help for a command

Examples:
  irgo new myapp         Create a new project
  irgo dev               Start dev server with hot reload
  irgo run ios           Build and run on iOS Simulator
  irgo run ios --dev     Hot-reload mode (connects to dev server)
  irgo run android       Build and run on Android Emulator
  irgo run android --dev Hot-reload mode (connects to dev server)
  irgo run desktop       Run as desktop app
  irgo run desktop --dev Desktop app with devtools enabled
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
hot reload (that path needs entr; this one has no extra dependency).`)

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
  irgo package setup --check     Report what is configured and what is missing

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
