---
name: toki
description: Writing translatable text in an irgo project — TIK syntax, the ordering rule, and the locale match that fails silently.
---

# Translations in irgo (toki)

irgo uses [toki](https://github.com/romshark/toki), which extracts **TIKs**
(Textual Internationalization Keys) from Go source and keeps one catalog per
locale. Text stays in the source, readable, in the language it was written in.

> The normative TIK grammar is
> [romshark/tik/SPECIFICATION.md](https://github.com/romshark/tik/blob/main/SPECIFICATION.md).
> This document is the irgo-specific part: where TIKs go, what order things run
> in, and the three mistakes that produce no error.

```sh
irgo i18n init      # once, naming the language the source is written in
irgo i18n add de    # a language to translate into
irgo i18n edit      # browser UI over the catalogs
irgo i18n check     # fail if a translation is unfinished (CI)
```

## Writing a TIK

The string literal *is* the key. Write it as normal text with placeholders in
braces:

```go
t.String(`Welcome back!`)
t.String(`You have {# new messages}`, unread)
t.String(`Hello {name}, it is {date-full}`, name, when)
```

| form | means |
|---|---|
| `{# things}` | a cardinal plural — the `#` is the number |
| `{name}` | a string placeholder |
| `{date-full}`, `{time-short}` | a formatted date/time |
| `[context] body` | a context prefix to disambiguate two identical texts |

Use a context when the same English text needs different translations:

```go
t.String(`[navigation] Home`)   // the page
t.String(`[building] Home`)     // the place
```

Both produce the ICU message; the `[context]` is not rendered.

**Plurals are not `if n == 1`.** English has two forms, Polish four, Japanese
one. Writing `{# new messages}` hands that to CLDR. Writing your own branch
hard-codes English into every language the app will ever have.

## In an irgo project, text lives in `.templ` files

Pass a reader into the component:

```templ
templ Inbox(t tokibundle.Reader, unread int) {
	<h2>{ t.String(`Welcome back!`) }</h2>
	<p>{ t.String(`You have {# new messages}`, unread) }</p>
}
```

### templ runs before toki. Always.

toki reads **Go**, and a templ component is not Go until templ has generated
it. Run toki first and it reports:

```
scan.files: 0
tiks.total: 0
```

— a clean, quiet, successful run that extracted nothing from a project whose
every string lives in `.templ` files. Nothing in that output says the tool
never saw the app.

irgo's asset pipeline is `templ` → registered steps → CSS, and the toki step is
registered in the middle, so every build gets this right. Do not call `toki
generate` by hand before `templ generate`.

## The three failures that produce no error

### 1. Discarding the match confidence

**This is the one that matters.** toki's own quick start writes:

```go
reader, _ := tokibundle.Match(i18n.Preferred(r)...)   // WRONG
```

x/text's matcher **never fails**. Asked for a language with no catalog, it
returns the first supported one and reports `language.No` beside it — the value
that underscore threw away. An app with `de` and `en` catalogs serves **German**
to a French visitor. Nothing errors, nothing logs, the page renders perfectly.

It cannot happen in development, because the languages you test with are
exactly the ones you have catalogs for. It happens only to the users the
feature exists to reach.

```go
reader := i18n.Reader(tokibundle.Match, tokibundle.Default, i18n.Preferred(r)...)
```

`i18n.Reader` checks the confidence and falls back to the source language.

### 2. An SSE fragment that forgot the reader

Handlers that patch fragments after first render need it too:

```go
sse.PatchTempl(templates.ConnectionStatus(lang.For(ctx.Request), true))
```

Miss it and that one corner of the page reverts to the source language while
everything around it stays translated — on a live update, so it is invisible
in a static page test.

`irgo i18n init` writes `lang/lang.go` for exactly this reason — use it rather
than calling `tokibundle.Match` anywhere:

```go
templates.Page(lang.For(ctx.Request))
```

If a project predates it, re-running `irgo i18n init` writes the file without
touching the bundle.

### 3. Mobile without `SetLocales`

A gomobile process inherits no shell environment, so `LANG` is unset, and the
WebView requests reaching the bridge do not carry `Accept-Language`. The native
shells irgo generates already call it at startup:

```kotlin
IrgoBridge.setLocales(resources.configuration.locales.toLanguageTags())
```
```swift
MobileSetLocales(Locale.preferredLanguages.joined(separator: ","))
```

A hand-written shell that omits it serves the source language on every phone.
Both were verified on a device: Android with `persist.sys.locale=de-DE`, iOS
with `simctl launch ... -AppleLanguages "(de)"`.

## Picking the locale

`i18n.Preferred(r)` answers "what does this user want?", which every target
asks differently — `Accept-Language` on web and Workers, `LC_ALL`/`LANG` on
desktop, `navigator.languages` in the browser, `SetLocales` on mobile. Pass the
whole ordered list, not just the first: a user asking for `de-CH, de, en` who
gets English when a `de` catalog exists has been ignored.

## Testing a translation

A test browser reports the locale of the machine running it, so a German
catalog is never exercised on an English laptop and every assertion passes on
the source language. Set it explicitly:

```go
p := browsertest.OpenAs(t, app.NewRouter(), "de-DE")
p.MustHaveText(".tagline", "Servergesteuerte Hypermedia für Go")
```

`OpenAs` sets both halves of what a real visitor sends — `navigator.language(s)`
and `Accept-Language` — which matters because different targets read different
ones.

Always include a locale you have **no** catalog for and assert the *source*
language. That is the case that fails silently, and the only one that would
still pass if `Match` were called directly.

## Catalogs

`tokibundle/catalog_<locale>.arb` is what a translator edits — ICU messages:

```json
"msgb19...": "Du hast {var0, plural, =0{keine neuen Nachrichten} one{# neue Nachricht} other{# neue Nachrichten}}"
```

- After editing a `.arb`, **regenerate** (`irgo project assets`, or any build).
  The Go catalogs are generated from it; editing the `.arb` alone changes
  nothing at runtime.
- `''` is ICU's escape for a single quote, not a typo.
- **Your source locale starts incomplete.** `{# new messages}` yields only the
  `other` form, so `en` sits at 50% until you add `one`. That is expected, and
  `irgo i18n check` is what tells you.

## Commit the whole of `tokibundle/`

Including the generated `*_gen.go`, which is the opposite of how `*_templ.go`
is treated. It cannot be rebuilt from what would be left: the default locale
lives only in `bundle_gen.go`, so toki will not start without it, and it
analyses Go source that stops compiling once the imported package is gone.

templ has neither problem, which is why the instinct to ignore generated code
is wrong here specifically.
