# irgo + Morpheus

[Morpheus](https://github.com/romshark/morpheus) is a web component UI kit that
targets server-driven architectures. This example uses it in an irgo project.

```sh
go run github.com/stukennedy/irgo/cmd/irgo server dev
```

## What you need

**Nothing to install.** Morpheus is opt-in, so a project asks for it:

```sh
go get github.com/romshark/morpheus
```

Then `irgo project assets` — which every build runs — copies its stylesheets
and bundle out of the Go module into `static/`. There is nothing to download
and no version to keep in step, because the assets *are* the module the templ
wrappers were compiled against.

It is not scaffolded into new projects. `irgo project new` produces a Datastar
demo that most projects delete on the first day, so a UI kit wired into it
would be wired into something throwaway.

## Why copied out of the module

Morpheus documents a CDN install, which is right for an ordinary web app and
wrong for an irgo one: the same code runs in an iOS and Android WebView, a
desktop shell and a Cloudflare Worker, where a second origin is an offline
failure rather than a convenience.

## Two directories

| | |
|---|---|
| `static/css/morpheus/` | irgo's. Refreshed every build, never edit |
| `static/css/` | yours. Never touched |

A theme is a file named `theme-<name>.css`, so a project writing its own would
eventually pick a name the kit also ships. Keeping the copies in their own
directory means one is regenerated and the other is not.

## Themes switch without a request

A theme is a set of custom properties scoped to a class on the root element:

```css
:root.theme-ocean { --accent: #0aa; --page-bg: #f7fdff; }
```

So switching one is a class change, which Datastar does from a signal:

```html
<main data-signals:theme="'default'">
  <div data-attr:class="'theme-' + $theme">
    <button data-on:click="$theme = 'ocean'">ocean</button>
```

**A colon, not a hyphen.** `data-attr:class` and `data-signals:theme`. The
hyphen forms parse, render, and bind nothing — see the note at the end.

No stylesheet swap, no reload, no round trip. Writing your own theme is the
same shape — put it in `static/css` and link it after the kit's.

## Not every component has JavaScript

`neo-badge` is styled by `morpheus.css` and registers nothing. `neo-button`,
`neo-popover` and the other interactive ones are custom elements the bundle
defines.

That distinction matters when testing: asserting that a badge "upgraded" is
asserting something that was never meant to be true. `ui_test.go` checks
`neo-button` for exactly this reason.

## Testing it

The interesting failures all render correct HTML — a component that never
upgraded, a stylesheet that 404s, a signal that never updated. `pkg/testing`
cannot see any of them, because they happen after the response.

```go
p := browsertest.Open(t, app.NewRouter())
p.MustEventually(`customElements.get('neo-button') !== undefined`, "...")
p.MustHaveAttr(`[data-attr\:class]`, "class", "theme-ocean")
```

[`pkg/browsertest`](../../pkg/browsertest) is a small wrapper over
playwright-go, in its own module so irgo's dependency graph does not grow by
thirty modules for everyone. Install the browsers once:

```sh
go run github.com/mxschmitt/playwright-go/cmd/playwright@v0.6100.0 install chromium
```

Without them the tests skip rather than fail, so CI that has not installed them
stays green.

**Assertions must wait.** A module script executes after load, a custom element
upgrades when its definition arrives, a signal updates after the event. Reading
immediately tests the moment before the thing under test happened — which is
why the harness offers `MustEventually` and `MustHaveAttr` rather than plain
reads.

---

# Picking this up in a new session

Everything below is state, not documentation. It is here so a session can
resume without re-deriving what was already measured.

## Where things are

| | |
|---|---|
| branch | `feat/ui-morpheus`, off `upstream/main`, **uncommitted work** |
| irgo `pkg/browsertest` | new nested module — playwright-go wrapper |
| this example | works, all six tests pass |
| shadcn-templ | abandoned; fork branch deleted, fork back on upstream main |
| clones | `../../../morpheus`, `../../../shadcn-templ`, `../../../datapages` |

## The bug that was here, and what fixed it

`TestThemeSwitchIsAClassChange` failed for a long session. The markup was
`data-signals-theme` and `data-attr-class` — hyphens where Datastar wants
colons. The page rendered, the element was there, Datastar was loaded, the
signal attribute was in the DOM, and nothing bound.

It was found by reading Datastar's own reference, not by any test, and the
correct syntax was already in irgo's README the whole time. That is why the
reference is now a skill an assistant loads on its own, synced into
`.claude/skills/` by every build. See `irgo project skills`.

The diagnostic that isolated it: switch to the object form, `data-attr="{class:
'theme-' + $theme}"`. It produced `class="theme-"` — the binding working and
the signal empty — which separated "syntax wrong" from "signal missing" in one
step. Worth remembering; the two look identical from the outside.

## Measurements, so they are not redone

```
irgo baseline Worker              2.40 MB compressed (3.00 MB limit)
+ Morpheus                        2.46 MB   +0.06
+ shadcn-templ (4 components)     2.86 MB   +0.46   ← why it lost
playwright-go                     30 modules → its own nested module
irgo module graph                 48, unchanged
```

## Decisions already made, with the reason

- **Morpheus over shadcn-templ** — 8× cheaper in the Worker, first-class
  Datastar package, no fork needed, no Tailwind scanning problem.
- **Opt-in, not scaffolded** — `project new` emits a Datastar demo most
  projects delete, so anything wired into it is wired into something
  throwaway. An earlier version scaffolded it and needed a generated blank
  import to survive `go mod tidy`; both were reverted.
- **Assets copied from the module cache, not downloaded** — they ship in the
  module, so the version cannot drift from the wrappers. An earlier version
  fetched GitHub releases and pinned separately; deleted.
- **`pkg/browsertest` is a separate module** — playwright-go is 30 modules and
  a framework should not put that in every consumer's graph.

## Next, in order

1. ~~Diagnose the failing test.~~ Done — hyphen for colon, above. All six
   tests pass, three of them in a real browser.
2. ~~Import Morpheus's own demo as a second example.~~ Done —
   `examples/morpheus-showcase`, all 65 pages served and tested.

   Its pages are imported rather than copied, so it cannot fall behind. That
   needs them exported and upstream keeps them in `internal/`, so it depends on
   a fork — **which is not going upstream**: `internal/` is how a maintainer
   reserves the right to change something, and asking them to give that up is a
   real cost to them for a benefit that is ours alone.

   So the fork is built to be cheap rather than temporary. It leaves upstream's
   `module` line alone, which is what lets `go.mod` replace across repositories
   without anyone cloning anything, and keeps it a two-file diff that `git
   rebase` carries over releases on its own.

3. **toki** (`github.com/romshark/toki`, i18n) — on its own branch, not this
   one.
4. **iOS WebView** — Morpheus needs `@starting-style` and `allow-discrete`
   (Safari 17.5). Likely still *functions* on 17.0–17.4 without transition
   animation, since CSS degrades rather than throwing. **Unverified** — needs
   a simulator.

## Things that cost time, so they are not repeated

- gost-dom loads the Datastar module and silently never evaluates it. Use
  playwright.
- `--dump-dom` renders but cannot interact.
- Assertions about a browser must **wait**. A module script runs after load; a
  custom element upgrades when its definition arrives; a signal updates after
  its event. Three tests failed on this before the harness grew
  `MustEventually`.
- A module rejected for MIME type appears **only** in the console, never as a
  page error. The harness now watches both.
- `neo-badge` is not a custom element. Only interactive components register.
- **Datastar attributes take a colon.** `data-attr:class`, `data-signals:theme`,
  `data-on:click`. A hyphen renders and binds nothing, and no handler test can
  see it. `.claude/skills/datastar/SKILL.md` is the reference; it is synced in
  by every build.
