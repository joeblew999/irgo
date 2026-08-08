# docs-templ

Built with **irgo** — one Go codebase for web, desktop, iOS and Android, using
Go + [templ](https://templ.guide) + [Datastar](https://data-star.dev).

> **This file is the single source of truth, for people and for AI assistants.**
> There is no separate CLAUDE.md or AGENTS.md holding different instructions —
> those exist only as pointers here, so anything reading the repo lands in the
> same place. Keep it that way: two docs drift, and the one an agent happens to
> read wins.

## Quickstart

```bash
go tool irgo tools doctor      # what this machine can build, and what needs setup
go tool irgo server dev         # hot-reload server on http://localhost:8080
```

Starting a *new* project is the one case `go tool` cannot cover — it needs a
go.mod to read, and there is not one yet:

```bash
go run github.com/stukennedy/irgo/cmd/irgo@latest new myapp
cd myapp && go tool irgo tools doctor
```

After that the project pins its own CLI and `go tool irgo` is all you need.

`go tool irgo` builds the CLI version this module requires, straight from
go.mod — nothing to install, nothing to keep on PATH, nothing to keep in step.
The `tool` directive in go.mod is the only pin.

## Architecture

```
User Interaction → Datastar Request → Go Handler → Templ Template → SSE Response → DOM Update
```

**Key principle:** the server returns HTML fragments over SSE, not JSON.
Datastar applies them to the DOM. There is no client-side state to keep in
sync, and no API layer to write twice.

## Project Structure

```
├── main.go              # Web/mobile entry (//go:build !desktop)
├── main_desktop.go      # Desktop entry (//go:build desktop)
├── app/app.go           # Router setup and route definitions
├── handlers/            # HTTP handlers returning HTML or SSE
├── templates/           # .templ layouts, pages and components
├── static/
│   ├── css/input.css    # Tailwind source (yours)
│   └── css/output.css   # generated — do not edit
├── mobile/mobile.go     # gomobile bridge (framework-owned)
├── ios/Example/         # native iOS shell — generated, not committed
├── android/Example/     # native Android shell — generated, not committed
├── irgo.package.toml    # store/signing settings (yours; secrets go in
│                        #   irgo.package.local.toml, which is gitignored)
├── .air.toml            # hot-reload config (framework-owned)
└── .github/workflows/   # CI for every target (framework-owned)
```

Anything marked generated is rebuilt on demand and gitignored. `_templ.go` and
`static/css/output.css` are generated **but embedded into builds**, so every
build regenerates them — you never have to remember.

## Commands

Everything runs through `go tool irgo`. `go tool irgo help <command>` has the
detail; `doctor` reports what your machine can actually do.

| Command | What it does |
|---|---|
| `project new` | Create a project, or regenerate this one |
| `project clean` | Remove generated output |
| `project upgrade` | Take framework updates, leaving your code alone |
| `project pin` | Choose which irgo this project builds against |
| `project ci` | Scaffold the GitHub Actions workflows |
| `project assets` | Regenerate templ + Tailwind (builds do this already) |
| `project generate` | Run the project's own generators (builds do this already) |
| `project test` | Run the tests |
| `project config` | Show or set a setting (signing, stores, version) |
| `app build <ios|android|desktop|cloudflare|all>` | Build it |
| `app run <ios|android|desktop>` | Build and launch it |
| `app package <ios|android|macos|windows>` | Store artifacts |
| `app deploy <cloudflare>` | Build the Worker and put it live |
| `app install <ios|android|desktop>` | Install a build — no rebuild |
| `app remove <ios|android|desktop>` | Uninstall it again |
| `app reviews <ios|mac|android>` | Monitor store reviews |
| `tools install` | Provision what builds need |
| `tools remove` | Undo it — shows what it will delete, and asks |
| `tools doctor` | What this host can build; --fix repairs it |
| `server dev` | Web server with hot reload |
| `server serve` | Web server without file watching |

Every command explains itself, including its flags:

```bash
go tool irgo help              # the whole grammar
go tool irgo help app          # what app accepts
go tool irgo help app run      # detail and flags for one command
```

**Nothing needs installing first.** Toolchains provision themselves when a
command needs them: `build android` fetches JDK 17, the SDK and the NDK into
`~/.irgo` — Android Studio is not involved. Undo it all with
`tools remove android` — it shows what it will delete, with sizes, and asks
first. The one exception is Xcode, which only
Apple can install; `doctor` says so plainly.

## Generated content

This site is an irgo project like any other — `server dev`, `app build`,
`app deploy` all work the same way, and a plain irgo project needs nothing
below. What it adds is content generated from source rather than typed, and
that stays out of the irgo CLI on purpose: highlighting themes and API
references are this site's concern, not every project's.

```bash
go tool irgo project generate   # refresh everything below
go tool irgo project test       # fails if any of it is stale
```

Builds, deploys and tests run the generators for you, so in practice you never
type the first line. It is deliberately not part of `project assets`, which
runs on every save under hot reload.

| Generated | From | Why |
|---|---|---|
| `clidoc/cli.json` | `irgo help --json` | The CLI reference cannot describe a command the binary does not have |
| `apidoc/api.json` | `go/doc` over the framework packages | The hand-written version documented nine methods that never existed |
| `apidoc/env.json` | `os.Getenv` calls in the source | The old page listed `IRGO_PORT`, which nothing reads |
| `content/highlighted_gen.go`, `static/css/chroma.css` | chroma | Highlighting at render time put 1 MB of lexers in the Worker |

Each has a test that fails when the file no longer matches its source, so
forgetting to regenerate is a red build rather than a wrong page. CI runs
`project test`, which does both.

The code samples on the realtime page are not strings: they live in
`content/examples/`, are compiled by `go build ./...`, and the pages read the
marked regions back out. That page previously documented a hub that did not
exist.

## Working on irgo itself

There is no separate setup for contributors, and no second toolchain. Using
irgo and working on irgo are the same activity pointed at different versions:

```bash
go tool irgo project pin                      # what am I building against?
go tool irgo project pin local ../irgo        # build a checkout you are editing
# edit the CLI — the next `go tool irgo` already runs your change
go tool irgo project pin release              # back to the published build
```

`pin local` writes a `replace` into go.mod, so nothing is installed, nothing is
tagged, and nothing has to be reinstalled between edits. The same command
tracks a fork:

```bash
go tool irgo project pin joeblew999/irgo@v0.4.0-androidapi21.65
go env -w GOPRIVATE='github.com/joeblew999/*'   # forks bypass the proxy, once
```

A fork keeps the upstream module path — that is what makes its `replace`
transparent, and also why Go fetches it from GitHub rather than the proxy.

## Updating the framework

```bash
go tool irgo project upgrade
```

Refreshes framework-owned files — the native shells, `.air.toml`, `.gitignore`,
`.github/workflows`, `mobile/`, this README. It **never** rewrites your code:
`main.go`, `app/`, `handlers/`, `templates/`, `static/`, `go.mod` and
`irgo.package.toml` are yours. Files of yours that differ from the current
template are listed, not touched — `--diff` shows what changed upstream.

To move the CLI version, edit the requirement in go.mod; everything follows.

## Router & Handlers

### Standard Handlers (Full Page Loads)

Standard handlers return `(string, error)`. The string is HTML.

```go
import (
    "github.com/stukennedy/irgo/pkg/router"
    "github.com/stukennedy/irgo/pkg/render"
)

// Full page load
r.GET("/", func(ctx *router.Context) (string, error) {
    return renderer.Render(templates.HomePage())
})
```

### Datastar SSE Handlers

Datastar handlers return `error` only and use `ctx.SSE()` for responses.

```go
// Datastar SSE endpoint
r.DSGet("/greeting", func(ctx *router.Context) error {
    var signals struct {
        Name string `json:"name"`
    }
    ctx.ReadSignals(&signals)

    sse := ctx.SSE()
    return sse.PatchTempl(templates.Greeting(signals.Name))
})

r.DSPost("/todos", createTodo)
r.DSPut("/todos/{id}", updateTodo)
r.DSPatch("/todos/{id}", toggleTodo)
r.DSDelete("/todos/{id}", deleteTodo)
```

### Context Methods

**Input:**
- `ctx.Param("id")` - URL path parameter
- `ctx.Query("q")` - Query string parameter
- `ctx.FormValue("name")` - Form field value
- `ctx.Header("X-Custom")` - Request header
- `ctx.ReadSignals(&signals)` - Parse Datastar signals from request

**Datastar Detection:**
- `ctx.IsDatastar()` - true if Accept: text/event-stream

**SSE Output (for Datastar handlers):**
```go
sse := ctx.SSE()
sse.PatchTempl(templates.Component())      // Patch templ component
sse.PatchHTML(`<div id="x">HTML</div>`)    // Patch raw HTML
sse.PatchSignals(map[string]any{...})      // Update client signals
sse.Remove("#element-id")                   // Remove element
sse.Redirect("/new-url")                    // Navigate browser
```

**Standard Output (for full page handlers):**
- Return HTML string from handler
- `ctx.Redirect("/path")` - HTTP redirect
- `ctx.NotFound("message")` - 404 response
- `ctx.BadRequest("message")` - 400 response
- `ctx.NoContent()` - 204 response

## Templ Templates

Templ is a type-safe HTML templating language that compiles to Go.

### Basic Syntax

```go
// templates/components.templ
package templates

// Component with parameters
templ UserCard(name string, email string) {
    <div class="card">
        <h2>{ name }</h2>
        <p>{ email }</p>
    </div>
}

// Component with children
templ Card(title string) {
    <div class="card">
        <h3>{ title }</h3>
        { children... }
    </div>
}

// Usage
templ ProfilePage() {
    @Card("Profile") {
        <p>Content goes here</p>
    }
}

// Conditionals
templ Status(active bool) {
    if active {
        <span class="text-green-500">Active</span>
    } else {
        <span class="text-red-500">Inactive</span>
    }
}

// Loops
templ UserList(users []User) {
    <ul>
        for _, user := range users {
            <li>{ user.Name }</li>
        }
    </ul>
}

// Conditional attributes
templ Checkbox(checked bool) {
    <input type="checkbox" checked?={ checked }/>
}

// Dynamic classes
templ Item(done bool) {
    <span class={ "item", templ.KV("line-through", done) }>Item</span>
}

// Safe URLs
templ Link(url string) {
    <a href={ templ.SafeURL(url) }>Link</a>
}

// Raw HTML (use sparingly)
templ RawContent(html string) {
    @templ.Raw(html)
}
```

### Rendering in Handlers

```go
renderer := render.NewTemplRenderer()

// Standard handler
func handler(ctx *router.Context) (string, error) {
    return renderer.Render(templates.MyComponent(data))
}

// Datastar handler
func sseHandler(ctx *router.Context) error {
    sse := ctx.SSE()
    return sse.PatchTempl(templates.MyComponent(data))
}
```

## Datastar Patterns

This project uses **Datastar** from `https://data-star.dev/`. Key concepts:
- **Signals**: Reactive client-side state
- **SSE**: Server responses as event streams
- **`data-*` attributes**: Declarative behavior

### Signals (Client-Side State)

```go
// Initialize signals
templ Counter() {
    <div data-signals="{count: 0}">
        <span data-text="$count">0</span>
        <button data-on:click="$count++">+</button>
    </div>
}

// Two-way binding
templ SearchForm() {
    <div data-signals="{query: ''}">
        <input type="text" data-bind:query placeholder="Search..."/>
        <span data-text="$query.length + ' characters'"></span>
    </div>
}
```

### Server Requests

```go
// GET request
templ LoadButton() {
    <button data-on:click="@get('/data')">Load</button>
    <div id="result"></div>
}

// POST request
templ TodoForm() {
    <div data-signals="{title: ''}">
        <input type="text" data-bind:title placeholder="New todo"/>
        <button data-on:click="@post('/todos')">Add</button>
    </div>
    <ul id="todo-list"></ul>
}

// DELETE request
templ DeleteButton(id string) {
    <button data-on:click={ fmt.Sprintf("@delete('/todos/%s')", id) }>
        Delete
    </button>
}
```

### Event Modifiers

```go
// Debounce input (wait 300ms after typing stops)
templ SearchInput() {
    <input
        type="text"
        data-bind:query
        data-on:input__debounce.300ms="@get('/search')"
        placeholder="Search..."
    />
}

// Prevent default form submission
templ Form() {
    <form data-on:submit__prevent="@post('/submit')">
        <input type="text" data-bind:name/>
        <button type="submit">Submit</button>
    </form>
}

// Trigger once (lazy loading)
templ LazyLoad() {
    <div data-on:intersect__once="@get('/lazy-content')">
        Loading...
    </div>
}
```

### Conditional Display

```go
// Show/hide based on signal
templ Modal() {
    <div data-signals="{showModal: false}">
        <button data-on:click="$showModal = true">Open</button>
        <div data-show="$showModal" class="modal">
            <p>Modal content</p>
            <button data-on:click="$showModal = false">Close</button>
        </div>
    </div>
}

// Dynamic classes
templ TabButton(name string) {
    <button
        data-class:active="$activeTab === 'name'"
        data-on:click="$activeTab = 'name'"
    >
        { name }
    </button>
}
```

### Loading Indicators

```go
templ LoadButton() {
    <div data-signals="{loading: false}">
        <button
            data-on:click="@get('/slow-endpoint')"
            data-indicator:loading
            data-attr:disabled="$loading"
        >
            <span data-show="!$loading">Load Data</span>
            <span data-show="$loading">Loading...</span>
        </button>
    </div>
}
```

## Native Capabilities

Call platform features (haptics, clipboard, share, storage, notifications)
with one API on every platform.

From templates / Datastar expressions (promise-based `irgo.native`):

```go
templ ShareButton(text string) {
    <button data-on:click={ fmt.Sprintf("irgo.native('share.text', {text: %q})", text) }>
        Share
    </button>
}
```

From Go handlers:

```go
import "github.com/stukennedy/irgo/pkg/native"

native.Call(ctx.Context(), "haptics.impact", native.Params{"style": "light"})
```

Built-in methods: `device.info`, `haptics.impact/notification/selection`,
`clipboard.read/write`, `share.text`, `browser.open`,
`storage.get/set/remove`, `notifications.requestPermission/show`,
`toast.show` (Android). Unsupported methods return
`native.ErrNotSupported` — degrade gracefully. Register Go fallbacks with
`native.Register(method, handler)` so web/desktop work too. Custom native
features: implement `IrgoPlugin` in `ios/.../IrgoPlugins.swift` or
`android/.../IrgoPlugins.kt` and register it.

## Sessions & Cookies

Standard `http.SetCookie` session auth works on all platforms — the mobile
bridge keeps a persistent cookie jar (sessions survive app restarts). Use
`mobile.ClearCookies()` for logout.

## Script Order

The layout must load the framework JS bridge **before** Datastar (both are
served automatically — no files to copy):

```html
<script src="/_irgo/bridge.js"></script>
<script src="/static/js/datastar.js"></script>
```

## Streaming / Real-Time

SSE streams progressively on every platform, including mobile. Long-lived
handlers must watch for disconnect:

```go
r.DSGet("/live", func(ctx *router.Context) error {
    sse := ctx.SSE()
    for {
        select {
        case <-sse.Context().Done():
            return nil // client went away
        case update := <-updates:
            sse.PatchTempl(templates.LiveRow(update))
        }
    }
})
```

## Build Tags

The framework uses Go build tags to separate platform code:

```go
//go:build !desktop    // Mobile/web builds (main.go)
//go:build desktop     // Desktop builds only (main_desktop.go)
```

- `go build .` → uses `main.go` (mobile/web)
- `go build -tags desktop .` → uses `main_desktop.go`
- `go tool irgo app run desktop` → automatically adds `-tags desktop`

## Common Handler Patterns

### CRUD Operations

```go
func Mount(r *router.Router) {
    // Full page - list
    r.GET("/", func(ctx *router.Context) (string, error) {
        items := db.GetItems()
        return renderer.Render(templates.ItemsPage(items))
    })

    // SSE - create
    r.DSPost("/items", func(ctx *router.Context) error {
        var signals struct {
            Name string `json:"name"`
        }
        ctx.ReadSignals(&signals)

        if signals.Name == "" {
            return ctx.SSE().PatchTempl(templates.Error("Name required"))
        }

        item := db.CreateItem(signals.Name)
        sse := ctx.SSE()
        sse.PatchTempl(templates.ItemRow(item))
        sse.PatchSignals(map[string]any{"name": ""}) // Clear input
        return nil
    })

    // SSE - update
    r.DSPatch("/items/{id}", func(ctx *router.Context) error {
        id := ctx.Param("id")
        item := db.ToggleItem(id)
        return ctx.SSE().PatchTempl(templates.ItemRow(item))
    })

    // SSE - delete
    r.DSDelete("/items/{id}", func(ctx *router.Context) error {
        id := ctx.Param("id")
        db.DeleteItem(id)
        return ctx.SSE().Remove("#item-" + id)
    })
}
```

### Validation Errors

```go
r.DSPost("/register", func(ctx *router.Context) error {
    var signals struct {
        Email string `json:"email"`
    }
    ctx.ReadSignals(&signals)

    if !isValidEmail(signals.Email) {
        return ctx.SSE().PatchTempl(templates.FieldError("email", "Invalid email"))
    }

    // Success - redirect to dashboard
    return ctx.SSE().Redirect("/dashboard")
})
```

## Datastar Attribute Reference

| Attribute | Description | Example |
|-----------|-------------|---------|
| `data-signals` | Initialize signals | `data-signals="{count: 0}"` |
| `data-bind:X` | Two-way binding | `data-bind:name` |
| `data-text` | Text content | `data-text="$count"` |
| `data-show` | Show/hide | `data-show="$visible"` |
| `data-class:X` | Conditional class | `data-class:active="$isActive"` |
| `data-attr:X` | Dynamic attribute | `data-attr:disabled="$loading"` |
| `data-on:event` | Event handler | `data-on:click="@get('/data')"` |
| `data-indicator:X` | Loading indicator | `data-indicator:loading` |

### HTTP Actions

| Expression | Description |
|------------|-------------|
| `@get('/url')` | GET request |
| `@post('/url')` | POST request |
| `@put('/url')` | PUT request |
| `@patch('/url')` | PATCH request |
| `@delete('/url')` | DELETE request |

### Event Modifiers

| Modifier | Description |
|----------|-------------|
| `__prevent` | Prevent default |
| `__stop` | Stop propagation |
| `__once` | Trigger once |
| `__debounce.Xms` | Debounce (e.g., `__debounce.300ms`) |
| `__throttle.Xms` | Throttle (e.g., `__throttle.100ms`) |

## Tips

1. **Always read files before editing** - understand existing code first
2. **You do not need to run templ by hand** — every build and `go tool irgo server dev` regenerates `_templ.go` and the stylesheet
3. **Use `go tool irgo server dev`** during development for hot reload
4. **Return HTML fragments via SSE**, not JSON - this is hypermedia-driven
5. **Elements need IDs** for Datastar to patch them
6. **Use signals for client state** - avoid unnecessary server roundtrips
7. **Prefer small, focused components** that can be reused and patched independently
8. **Test in desktop mode** with `go tool irgo app run desktop --dev` for browser devtools
