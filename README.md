# Irgo

A hypermedia-driven application framework that uses Go as a runtime kernel with Datastar. Build native iOS, Android, and **desktop** apps using Go, HTML, and Datastar - no JavaScript frameworks required.

## Working on irgo

[CONTRIBUTING.md](CONTRIBUTING.md) has the workflow: where to branch from, what
is enforced versus habit, and how a release reaches a project.

```sh
mise install       # the branch tools
mise run wf:setup     # once per clone — protects main, blocks pushes to upstream
mise tasks         # everything else, with a line each
```

## Shipping to China

[CHINA.md](CHINA.md) — the filing sequence that gates every mainland
deployment, the one blocked import a generated project still ships with, and
why 10% of Chinese phones cannot run the Android build at all.

[PUSH.md](PUSH.md) — there is no push in irgo on any platform yet. What to
build first so that the Chinese vendors are the last mile rather than the
design.

## Key Features

- **Go-Powered Apps**: Write your backend logic in Go, compile to native mobile frameworks or desktop apps
- **Datastar for Interactivity**: Use Datastar's hypermedia approach with SSE instead of complex JavaScript
- **Cross-Platform**: Single codebase for iOS, Android, desktop (macOS, Windows, Linux), and web
- **Virtual HTTP (Mobile)**: No network sockets - requests are intercepted and handled directly by Go, with **true streaming SSE**
- **Native Capabilities**: One API for haptics, share sheets, clipboard, secure storage, notifications and more — `native.Call(...)` from Go, `irgo.native(...)` from the WebView, with pluggable Swift/Kotlin plugins
- **Sessions That Just Work**: Persistent cookie jar on mobile — standard `http.SetCookie` auth flows survive app restarts
- **Native Webview (Desktop)**: Real HTTP server with native webview window
- **Type-Safe Templates**: Use [templ](https://templ.guide) for compile-time checked HTML templates
- **Hot Reload Development**: Edit Go/templ code and see changes instantly

## Native Capabilities

Call platform features from anywhere in your app — no per-platform code:

```html
<!-- From the WebView (Datastar expressions) -->
<button data-on:click="irgo.native('haptics.impact', {style: 'light'})">Tap</button>
<button data-on:click="irgo.native('share.text', {text: 'Check this out!'})">Share</button>
```

```go
// From a Go handler
import "github.com/stukennedy/irgo/pkg/native"

native.Call(ctx.Context(), "notifications.show", native.Params{
    "title": "Order placed", "body": "We'll notify you when it ships",
})
```

Built-in plugins: `device.info`, `haptics.*`, `clipboard.*`, `share.text`,
`browser.open`, `storage.*` (Keychain / SharedPreferences),
`notifications.*`, `toast.show`. Add your own by implementing the
`IrgoPlugin` protocol/interface in Swift or Kotlin and registering it with
`IrgoNative` — it's instantly callable from Go and JS. Register Go
fallbacks with `native.Register` so the same code runs on web and desktop.

## Architecture

### Mobile Architecture (iOS/Android)

```
┌─────────────────────────────────────────────────────────────┐
│                      Mobile App                              │
│  ┌───────────────────────────────────────────────────────┐  │
│  │                 WebView (Datastar)                     │  │
│  │  • HTML rendered by Go templates                       │  │
│  │  • Datastar handles interactions via irgo:// scheme    │  │
│  └──────────────────────┬────────────────────────────────┘  │
│                         │                                    │
│  ┌──────────────────────▼────────────────────────────────┐  │
│  │           Native Bridge (Swift / Kotlin)               │  │
│  │  • Intercepts irgo:// requests                          │  │
│  │  • Routes to Go via gomobile                           │  │
│  └──────────────────────┬────────────────────────────────┘  │
│                         │                                    │
│  ┌──────────────────────▼────────────────────────────────┐  │
│  │              Go Runtime (gomobile bind)                │  │
│  │  • HTTP router (chi-based)                             │  │
│  │  • Template rendering (templ)                          │  │
│  │  • Business logic                                      │  │
│  └───────────────────────────────────────────────────────┘  │
└─────────────────────────────────────────────────────────────┘
```

### Desktop Architecture (macOS/Windows/Linux)

```
┌─────────────────────────────────────────────────────────────┐
│                     Desktop App                              │
│  ┌───────────────────────────────────────────────────────┐  │
│  │            Native Webview Window                       │  │
│  │  (System webview engine - Chromium/WebKit)            │  │
│  │  Navigates to: http://localhost:PORT                   │  │
│  └──────────────────────┬────────────────────────────────┘  │
│                         │                                    │
│  ┌──────────────────────▼────────────────────────────────┐  │
│  │         Go HTTP Server (localhost:PORT)                │  │
│  │  • Page Routes (Templ → HTML)                          │  │
│  │  • API Routes (SSE responses)                          │  │
│  │  • Static Asset Server (/static/*)                     │  │
│  └───────────────────────────────────────────────────────┘  │
└─────────────────────────────────────────────────────────────┘
```

## Quick Start

### Prerequisites

Go, and nothing else you have to install yourself.

The CLI provisions what a build needs, when that build needs it: templ, the
Tailwind standalone binary, air, gomobile, and the Android JDK/SDK/NDK — all
into `~/.irgo` or the Android SDK home, on macOS, Linux and Windows alike. No
system package manager is involved, and `irgo tools remove` undoes it.

Ask what this machine can do, and what it is missing:

```bash
go tool irgo tools doctor
```

The one thing irgo cannot install is Xcode, because only Apple can; `doctor`
says so plainly. Android Studio is *not* required.

**For desktop development:**
- CGO enabled (C compiler required)
- macOS: Xcode Command Line Tools (included with Xcode)
- Windows: MinGW-w64 or similar C compiler
- Linux: GCC and WebKit2GTK dev packages (`apt install libwebkit2gtk-4.0-dev`)

### Install Irgo CLI

A prebuilt binary, on macOS, Linux or Windows, without a Go toolchain:

```bash
mise use -g ubi:joeblew999/irgo
```

Or take it straight from the [releases
page](https://github.com/joeblew999/irgo/releases) — one archive per platform,
containing a single self-contained `irgo`. The templates it scaffolds from are
compiled into it, so there is nothing to install beside it.

Or from source:

```bash
git clone https://github.com/joeblew999/irgo.git
cd irgo/cmd/irgo
go install .
```

Note that `go install` **cannot** fetch this CLI by path. The module keeps
upstream's name — `module github.com/stukennedy/irgo` — which is what lets a
project depend on this fork through a `replace`, and the price is that

```bash
go install github.com/joeblew999/irgo/cmd/irgo@latest   # does not work
```

fails with `module declares its path as: github.com/stukennedy/irgo`, while the
`stukennedy` path installs upstream's CLI, which is a different and much older
program. The releases above are the only one-step way to get *this* one.

Whichever route, `irgo version` reports the same thing, because none of them
stamps a number in: it is read back out of the build, so a binary cannot claim
to be a release it was not built from.

### Create a New Project

```bash
irgo project new myapp
cd myapp
go mod tidy
```

### Run as Desktop App

```bash
irgo app run desktop         # Run as desktop app
irgo app run desktop --dev   # With devtools enabled
```

### Development with Hot Reload (Web)

```bash
irgo server dev                 # Start dev server at http://localhost:8080
```

### iOS Development

```bash
irgo app run ios --dev       # Hot-reload with iOS Simulator
irgo app run ios             # Production build
```

#### iOS on Linux (xtool)

On Linux, `irgo app run ios` deploys to a USB/network-connected physical
device using [xtool](https://xtool.sh) instead of Xcode + Simulator, and
`irgo app build ios` cross-compiles the device slice (ios-arm64) of the
framework with the xtool Darwin SDK. Install xtool and a Swift 6.1+
toolchain, run `xtool setup` once (requires Xcode.xip), and connect a
device. The app itself is a SwiftPM project scaffolded at `ios/App`.

In `--dev` mode the device connects to the dev server over your LAN
(override the URL with `IRGO_DEV_SERVER=http://<host>:8080`). If your
Darwin SDK lives outside `~/.swiftpm/swift-sdks`, point `IRGO_DARWIN_SDK`
at it. `irgo tools doctor` reports what is missing.

### Build for Production

```bash
# Desktop
irgo app build desktop           # Build for current platform
irgo app build desktop macos     # Build macOS .app bundle
irgo app build desktop windows   # Build Windows .exe
irgo app build desktop linux     # Build Linux binary

# Mobile
irgo app build ios               # Build iOS framework
irgo app build android           # Build Android AAR
```

## Project Structure

```
myapp/
├── main.go              # Mobile/web entry point (build tag: !desktop)
├── main_desktop.go      # Desktop entry point (build tag: desktop)
├── go.mod               # Go module definition
├── .air.toml            # Air hot reload configuration
│
├── app/
│   └── app.go           # Router setup and app configuration
│
├── handlers/
│   └── handlers.go      # HTTP handlers (business logic)
│
├── templates/
│   ├── layout.templ     # Base HTML layout
│   ├── pages.templ      # Page templates
│   └── components.templ # Reusable components
│
├── static/
│   ├── css/
│   │   ├── input.css    # Tailwind source
│   │   └── output.css   # Generated CSS
│   └── js/
│       └── datastar.js  # Datastar library
│
├── mobile/
│   └── mobile.go        # Mobile bridge setup
│
├── ios/                 # iOS Xcode project
├── android/             # Android project
│
└── build/
    ├── ios/             # Built iOS framework
    ├── android/         # Built Android AAR
    └── desktop/         # Built desktop apps
        ├── macos/       # macOS .app bundle
        ├── windows/     # Windows .exe
        └── linux/       # Linux binary
```

## Desktop Development

### How Desktop Mode Works

Desktop mode uses a different architecture than mobile:

1. **Real HTTP Server**: A Go HTTP server starts on an auto-selected localhost port
2. **Native Webview**: A native window with an embedded browser engine opens
3. **Standard HTTP**: The webview navigates to the localhost URL - standard HTTP requests

This means your app works identically to the web dev server, but packaged as a native desktop app.

### Desktop Entry Point

Projects include a `main_desktop.go` with build tag `//go:build desktop`:

```go
//go:build desktop

package main

import (
    "flag"
    "fmt"
    "net/http"

    "myapp/app"
    "github.com/stukennedy/irgo/desktop"
)

func main() {
    devMode := flag.Bool("dev", false, "Enable devtools")
    flag.Parse()

    r := app.NewRouter()

    mux := http.NewServeMux()
    staticDir := desktop.FindStaticDir()
    mux.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir(staticDir))))
    mux.Handle("/", r.Handler())

    config := desktop.DefaultConfig()
    config.Title = "My App"
    config.Debug = *devMode

    desktopApp := desktop.New(mux, config)

    fmt.Println("Starting desktop app...")
    if err := desktopApp.Run(); err != nil {
        fmt.Printf("Error: %v\n", err)
    }
}
```

### Desktop Configuration

```go
config := desktop.Config{
    Title:     "My App",      // Window title
    Width:     1024,          // Window width
    Height:    768,           // Window height
    Resizable: true,          // Allow window resize
    Debug:     false,         // Enable browser devtools
    Port:      0,             // 0 = auto-select available port
}
```

### Running Desktop Apps

```bash
# Run directly (compiles and runs)
irgo app run desktop

# With devtools (for debugging)
irgo app run desktop --dev
```

### Building Desktop Apps

```bash
# Build for current platform
irgo app build desktop

# Build for specific platform
irgo app build desktop macos     # Creates build/desktop/macos/MyApp.app
irgo app build desktop windows   # Creates build/desktop/windows/MyApp.exe
irgo app build desktop linux     # Creates build/desktop/linux/MyApp
```

### Desktop vs Mobile: Key Differences

| Aspect | Mobile | Desktop |
|--------|--------|---------|
| HTTP | Virtual (no sockets) | Real localhost server |
| Bridge | gomobile + native code | None (direct HTTP) |
| Entry point | `main.go` | `main_desktop.go` |
| Build tag | `!desktop` | `desktop` |
| CGO | Not required | Required (webview) |

## Writing Handlers

Irgo supports two types of handlers:

### Standard Handlers (Full Page Loads)

Return `(string, error)` with HTML:

```go
r.GET("/about", func(ctx *router.Context) (string, error) {
    return renderer.Render(templates.AboutPage())
})
```

### Datastar SSE Handlers

Return `error` and use `ctx.SSE()` for responses:

```go
r.DSPost("/todos", func(ctx *router.Context) error {
    var signals struct {
        Title string `json:"title"`
    }
    ctx.ReadSignals(&signals)

    todo := createTodo(signals.Title)

    sse := ctx.SSE()
    sse.PatchTempl(templates.TodoItem(todo))
    sse.PatchSignals(map[string]any{"title": ""}) // Clear input
    return nil
})
```

## Writing Templates

Templates use [templ](https://templ.guide) with Datastar attributes:

```go
// templates/pages.templ
package templates

templ HomePage() {
    @Layout("Home") {
        <main class="container mx-auto p-4">
            <h1 class="text-2xl font-bold">Welcome to Irgo</h1>

            <div data-signals="{name: ''}">
                <input
                    type="text"
                    data-bind:name
                    placeholder="Your name"
                    class="border p-2 rounded"
                />
                <button
                    data-on:click="@post('/greeting')"
                    class="bg-blue-500 text-white px-4 py-2 rounded"
                >
                    Greet
                </button>
                <div id="greeting"></div>
            </div>
        </main>
    }
}

templ TodoItem(todo Todo) {
    <div id={ "todo-" + todo.ID } class="flex items-center gap-2 p-2">
        <input
            type="checkbox"
            checked?={ todo.Done }
            data-on:click={ fmt.Sprintf("@patch('/todos/%s')", todo.ID) }
        />
        <span>{ todo.Title }</span>
        <button
            data-on:click={ fmt.Sprintf("@delete('/todos/%s')", todo.ID) }
            class="text-red-500"
        >Delete</button>
    </div>
}
```

## CLI Commands

```bash
# Create new project
irgo project new myapp
irgo project new .              # Initialize in current directory

# Development
irgo server dev                # Start dev server with hot reload (web)
irgo app run desktop        # Run as desktop app
irgo app run desktop --dev  # Desktop with devtools
irgo app run ios --dev      # Hot reload with iOS Simulator
irgo app run android --dev  # Hot reload with Android Emulator

# Production builds
irgo app build desktop      # Build desktop app for current platform
irgo app build desktop macos/windows/linux  # Cross-platform builds
irgo app build ios          # Build iOS framework
irgo app build android      # Build Android AAR
irgo app build all          # Build all mobile platforms

irgo app run ios            # Build and run on iOS Simulator
irgo app run android        # Build and run on Android Emulator

# Utilities
irgo project assets              # Generate templ files
irgo project skills              # Reference material for agents (builds sync it)
irgo tools install      # Install required dev tools
irgo version            # Print version
irgo help [command]     # Show help
```

## Agent Skills

`.claude/skills/` holds the reference an AI assistant loads when it writes
templates. It is committed, and it ships **inside the irgo module**, so every
project gets it from the version its `go.mod` names — no download, nothing to
pin twice, and a project cannot end up on a different version than the
framework it builds against.

| | |
|---|---|
| `datastar` | Every Datastar attribute and action. Vendored from [datapages](https://github.com/romshark/datapages) |
| a kit's own | Arrive from the kit's module — Morpheus ships three |

```sh
irgo project skills --sources   # where each one came from
irgo project skills --check     # compare the vendored ones against upstream
irgo project skills --update    # take upstream's version (from this checkout)
```

Skills that ship inside a module cannot go stale — their version is `go.mod`'s.
Only vendored ones can, which is what `--check` is for; CI runs it weekly.
Provenance for those lives in [`cmd/irgo/skills.go`](cmd/irgo/skills.go), which
is both the record a human reads and the one `--check` fetches against, so the
two cannot disagree.

This exists because of a real failure: `data-attr-class` was written instead of
`data-attr:class`. It parses, it renders, the markup looks right, and the
binding silently does nothing. No handler test can see it. The correct syntax
was already in the project README — which the assistant had not read, because a
600-line README is not in context and a skill is.
## Translations

Opt-in. A project without it has no bundle and no extra dependency.

```sh
irgo i18n init          # set up, using the language your source is written in
irgo i18n add de fr     # languages to translate into
irgo i18n edit          # a browser UI over the catalogs
irgo i18n check         # fail if a translation is unfinished — for CI
```

Text stays in your source, readable, in the language you wrote it in.
[toki](https://github.com/romshark/toki) extracts it into a catalog per locale:

```go
reader := i18n.Reader(tokibundle.Match, tokibundle.Default, i18n.Preferred(r)...)
reader.String(`You have {# new messages}`, unread)
```

`{# new messages}` carries the plural rules with it. English has two forms,
Polish has four, Japanese has one — so `if n == 1` in a framework has already
decided the app is wrong in most of the world.

**Use `i18n.Reader`, not `tokibundle.Match` directly.** The matcher never
fails: asked for a language you have no catalog for, it returns the first one
you *do* have and reports its lack of confidence in a second return value.
Discard that value — as `reader, _ := tokibundle.Match(...)` does — and a
French visitor to a German-and-English app gets German. Nothing errors, nothing
logs, and the page renders perfectly in a language nobody chose. It cannot
happen in development, because the languages you test with are the ones you
have catalogs for.

`i18n.Preferred` answers the other half — what *this* user wants — which every
target asks differently: `Accept-Language` on web and Workers, `LC_ALL`/`LANG`
on desktop, `navigator.languages` in the browser, the OS on mobile.

Regenerating happens on every build, after templ and before the CSS. That
order is not decoration: toki reads Go, and a templ component is not Go until
templ has generated it — run it first and it cheerfully reports scanning zero
files.

## Datastar Overview

[Datastar](https://data-star.dev) is a lightweight (~11KB) hypermedia framework that powers Irgo's interactivity:

- **SSE (Server-Sent Events)**: Server pushes HTML fragments to update the DOM
- **Reactive Signals**: Client-side state with `data-signals` and `$variable` syntax
- **Declarative Actions**: `data-on:click="@get('/api')"` triggers server requests

### Key Datastar Attributes

| Attribute | Description | Example |
|-----------|-------------|---------|
| `data-signals` | Initialize state | `data-signals="{count: 0}"` |
| `data-bind:X` | Two-way binding | `data-bind:name` |
| `data-on:event` | Event handler | `data-on:click="@post('/api')"` |
| `data-text` | Dynamic text | `data-text="$count"` |
| `data-show` | Conditional display | `data-show="$visible"` |

## Troubleshooting

### Desktop: "CGO_ENABLED=0" error

Desktop builds require CGO. Ensure you have a C compiler:
- macOS: `xcode-select --install`
- Windows: Install MinGW-w64
- Linux: `apt install build-essential`

### Desktop: Webview not showing

Check that WebKit2GTK is installed (Linux):
```bash
apt install libwebkit2gtk-4.0-dev
```

### "Module not found" errors

```bash
go mod tidy
```

### Hot reload not working

1. Check if air is running
2. Verify `.air.toml` configuration
3. Make sure `_templ.go` is NOT in `exclude_regex`

### Port 8080 already in use

```bash
lsof -i :8080
kill <PID>
```

## License

Apache License 2.0 — see [LICENSE](LICENSE) for details.

## Acknowledgments

- [Datastar](https://data-star.dev) - The hypermedia framework that powers Irgo
- [templ](https://templ.guide) - Type-safe HTML templating for Go
- [chi](https://github.com/go-chi/chi) - Lightweight Go router
- [webview](https://github.com/webview/webview) - Native webview for desktop
- [gomobile](https://pkg.go.dev/golang.org/x/mobile) - Go on mobile platforms
- [air](https://github.com/air-verse/air) - Live reload for Go
