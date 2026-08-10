# Morpheus's showcase, served by irgo

Morpheus's own documentation site — every page demonstrating every component —
running from an irgo router. The pages are imported from the Morpheus module,
not copied, so this follows whatever version `go.mod` resolves.

## Run it

```sh
cd examples/morpheus-showcase
go generate ./...          # first time, and after any dependency change
go tool irgo server dev
```

`go generate` is required, not optional: `static/` is gitignored because its
215 files and 3.4 MB are copied out of the Go module and regenerate
byte-for-byte. Committing them would mean a second copy of someone else's
release, going stale the moment the dependency moves — which is exactly what
this example exists to avoid.

Then <http://localhost:8080>. Note the **trailing slashes** on routes —
`/button/`, not `/button` — which is the showcase's own convention.

Worth a look: `/components/` for the index, `/theming/` for their theme
switcher, `/bundle-builder/` for the interactive sizer (esbuild and brotli
compiled to wasm).

## Upgrade it

The fork is a published module, so this is the usual two commands:

```sh
go get -u github.com/joeblew999/morpheus && go generate ./...
```

`go get -u` on the replacement, not on `github.com/romshark/morpheus` — the
replace directive names the fork, and that is what has versions to move.

Taking a new upstream release into the fork is a rebase, done once in the fork
rather than by everyone who builds this:

```sh
git fetch upstream && git rebase upstream/main    # on feat/export-site
```
Either way `go generate` is what matters here: it refreshes the assets and
rewrites the embed list. A page added upstream appears with no code written in
this example, and a new image is copied and embedded — both verified by adding
each to the module and regenerating.

Then run it again:

```sh
go tool irgo server dev
```

The fork rebases cleanly over upstream releases, including ones that edit files
the fork moved — git follows the renames. That was tested rather than assumed.

## Why upgrading is that small

**The showcase enumerates itself.** `showcase.Pages()` returns every page with
its route, so mounting them is a loop:

```go
for _, p := range site.Pages(morpheusVersion) {
	r.GET(p.Path, func(ctx *router.Context) (string, error) {
		return renderer.Render(p.Component)
	})
}
```

An earlier version parsed their static site generator to recover the same list.
It worked, and it was wrong: a page added upstream would have been missed in
silence.

**`go generate` owns the assets.** `sync.go` copies two trees out of the module
— `min/` for the stylesheets and bundle the pages link, and `showcase/static`
for the images, icons, fonts and favicon they reference — then rewrites
`static/embed.go` from what is actually there.

Doing that by hand is how thirteen images went missing: `min/` was copied, the
showcase's own tree was not, and the pages rendered well enough to look
finished. `//go:embed all:.` is not valid syntax, so the list is generated
rather than wildcarded.

## Why it needs the fork

```
replace github.com/romshark/morpheus => github.com/joeblew999/morpheus v0.0.0-...
```

Nobody clones anything. Go allows a replacement in a different repository as
long as its `go.mod` still declares the path it is replacing — and the fork
does, because renaming the module would mean rewriting 564 imports across 445
files and conflicting with every upstream release forever. Leaving that line
alone is what keeps the fork a two-file diff.

Three changes, all package-level:

| | |
|---|---|
| `internal/site` → `showcase` | the pages were unimportable |
| `internal/href` → `href` | a page without its route is half a showcase |
| `Pages()` exported | so their generator and any other host read one list |

The third is the one that would help them most: their generator now calls it,
so it cannot drift from the showcase it renders.

**This is not staged for upstream.** Exporting packages someone deliberately
put in `internal/` is their call to make, not a favour to ask — `internal/` is
how a maintainer reserves the right to change something, and taking that away
is a real cost to them for a benefit that is entirely ours. So the fork is
permanent, and it is built to be cheap rather than temporary: upstream's module
line untouched, two files actually edited, the rest renames that `git rebase`
follows on its own. Verified against a release that edited the moved files.

## Where it will and will not run

| target | result |
|---|---|
| web, desktop | fine |
| Cloudflare Workers | **6.53 MB compressed — over the 3 MB free plan**, under the 10 MB paid one |

Not a defect and not worth fixing here: this is a 65-page documentation site
that embeds 209 of its own assets, and embedding them is what makes it work
offline in a WebView. `irgo app build cloudflare` reports the number and the
limit, so the failure is at build time rather than at deploy.

An app that wants Morpheus *and* the free plan should look at
`examples/morpheus`, which is the kit rather than the showcase: 2.47 MB, well
inside it.

## What it costs

```
examples/todo                43 modules
examples/morpheus-showcase   46 modules
```

Three, for the whole kit: Morpheus itself, brotli, and some golang.org/x that
most projects already have. irgo's own graph is untouched at 48 — examples are
separate modules, so nothing here reaches a project that imports irgo.

One thing to watch: Morpheus wants templ v0.3.1020, above irgo's pinned
v0.3.977, so depending on it moves that pin. Harmless here, and worth knowing
if the pin ever matters to you.

## Assets live where the showcase expects them

Its pages link `/static/min/…` for the bundle and `/static/…` for everything
else — Morpheus's layout, not the `/static/css/morpheus/` that
`irgo project assets` produces for a project *using* the kit.

That distinction is worth holding: irgo's sync serves projects using Morpheus;
this serves Morpheus's own site, written against its own paths.

## What this demonstrates

- an irgo router serving a third-party templ application unmodified
- Morpheus components upgrading and behaving inside an irgo app
- no CDN, so it works offline and inside a WebView

Verified in real Chrome — 104 KB of rendered DOM on `/button/` with components
upgraded — and by checking every referenced asset rather than a sample: 34 of
34 assets, and every sampled route.
