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
		if len(os.Args) >= 3 && os.Args[2] == "mobile" {
			// Scaffold the ios/Example + android/Example apps into the
			// current project (idempotent: missing-only).
			err = scaffoldExamples()
			break
		}
		if len(os.Args) < 3 {
			fmt.Println("Usage: irgo new <project-name> | irgo new mobile")
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
			err = runBuild(target)
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

	case "templ":
		err = runTempl()

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
		if len(os.Args) < 3 || os.Args[2] != "android" {
			fmt.Println("Usage: irgo uninstall-tools android [--remove-jdk]")
			os.Exit(1)
		}
		err = uninstallAndroidTools(hasFlag(os.Args[3:], "--remove-jdk"))

	case "doctor":
		if len(os.Args) < 3 || os.Args[2] != "android" {
			fmt.Println("Usage: irgo doctor android")
			os.Exit(1)
		}
		err = doctorAndroid()

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
  new mobile       Scaffold ios/Example + android/Example into current project
  dev              Run development server with hot reload
  serve            Run server without file watching
  build <target>   Build for mobile/desktop (ios, android, desktop, or all)
  run <platform>   Build and run on simulator or desktop
  templ            Generate templ files
  test             Run tests
  install-tools    Install required dev tools (gomobile, templ, air)
  install-tools android   Install Android SDK + NDK (+ emulator with --emulator)
  uninstall-tools android Remove everything install-tools android installed
  doctor android   Check the Android toolchain is correctly installed
  version          Print version information
  help [command]   Show help for a command

Examples:
  irgo new myapp         Create a new project
  irgo new mobile        Scaffold the mobile example apps in the current project
  irgo dev               Start dev server with hot reload
  irgo run ios           Build and run on iOS Simulator
  irgo run ios --dev     Hot-reload mode (connects to dev server)
  irgo run android       Build and run on Android Emulator
  irgo run android --dev Hot-reload mode (connects to dev server)
  irgo run desktop       Run as desktop app
  irgo run desktop --dev Desktop app with devtools enabled
  irgo build ios         Build iOS framework only
  irgo build desktop     Build desktop app for current platform`)
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
  - Makefile          Build targets`)

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
		fmt.Println(`irgo doctor android - Verify the Android toolchain

Usage:
  irgo doctor android

Checks ANDROID_HOME, JDK 17, sdkmanager, NDK, emulator and AVDs, printing
what is OK and what is MISSING. Exits non-zero if anything critical is
missing — safe to run in CI to assert an install worked.`)

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

	default:
		fmt.Printf("Unknown command: %s\n", cmd)
		printUsage()
	}
}
