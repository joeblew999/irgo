# Desktop on glaze: replace webview_go (cgo) with crgimenes/glaze (cgo-free)

Status: planned · 2026-09-30 · upstream issue [stukennedy/irgo#16](https://github.com/stukennedy/irgo/issues/16)

## Why

irgo desktop builds need cgo today (`github.com/webview/webview_go`: WebKitGTK / WebView2 /
WKWebView through C), so a Windows or Linux desktop binary needs that platform's C toolchain.
glaze binds the same OS webviews through purego: `CGO_ENABLED=0` builds for every desktop target,
cross-compiled from one Mac. Mobile is unaffected (gomobile), and glaze stays desktop-only by design
(its maintainer, glaze#30).

## Facts (checked 2026-09-30)

- The whole desktop webview use is `desktop/webview_desktop.go` (`//go:build desktop`), 9 calls:
  `New`, `Destroy`, `SetTitle`, `SetSize(w, h, HintNone|HintFixed)`, `Init`, `Navigate`, `Run`,
  `Bind`, `Eval`. `webview_stub.go` covers `!desktop`.
- glaze v0.0.61 has all nine with the same shape (it descends from the same webview API). Two
  differences: `glaze.New(debug)` returns `(WebView, error)`; `Bind` returns `error` (irgo already
  returns an error from `App.Bind`).
- irgo never calls `runtime.LockOSThread()`. glaze on macOS needs the window created on the main
  OS thread — irgo-windows-vm's `glaze-all` example crashed with SIGTRAP in `windowInit` for this
  exact reason when run on the Mac.
- `cmd/irgo/app_desktop_build.go` builds with `go build -tags desktop` (cgo on).
- glaze on Windows is proven by irgo-windows-vm (Windows 11 ARM64): full glaze test suite and all
  probes pass on v0.0.61. Known limits there: absolute `app://` URLs do not load on Windows
  (virtual host); `SetAppIcon` unsupported on Windows; WebView2 stale registration → glaze#34.
- Branch: irgo's working branch is `feat/fork-portable-workflow`; `upstream` = stukennedy/irgo
  (write access; PRs merged there). This change branches from `upstream/main`.

## Change (phase 1: swap only, same behaviour)

1. `go.mod`: add `github.com/crgimenes/glaze v0.0.61`; remove `github.com/webview/webview_go`.
2. `desktop/webview_desktop.go`: import glaze; `type webviewHandle = glaze.WebView`;
   `a.wv, err = glaze.New(a.config.Debug)` and return the error (today a missing runtime would
   crash); `glaze.HintNone` / `glaze.HintFixed`; `Bind` returns glaze's error. Everything else
   unchanged (Init secret injection, Navigate to the loopback URL, Run, Destroy).
3. Main thread: in the `desktop` build, `func init() { runtime.LockOSThread() }` in package
   `desktop`, with a comment (macOS AppKit requires the main thread; Windows/Linux harmless), and
   make sure `runWebview` is reached from `main`'s goroutine (check `desktop/app.go` flow).
4. Builds: `cmd/irgo/app_desktop_build.go` sets `CGO_ENABLED=0` for desktop builds and allows
   `GOOS=windows|linux` targets from macOS (no platform C toolchain needed any more). Linux still
   needs WebKitGTK *at runtime* on the target machine — document it, it is not a build dependency.
5. Docs: README desktop section — "cgo-free; cross-compile desktop for every OS from one machine";
   drop the "CGO must be enabled / CGO_ENABLED=0 error" troubleshooting (README "Desktop:
   CGO_ENABLED=0 error", docs-templ pages) since it no longer applies.

Out of scope (phase 2, its own plan): the stable app origin from #16 (glaze scheme handlers
instead of the loopback server + secret). Blocked in part by glaze's Windows `app://` limitation.

## Dev loop

- **Inner loop, macOS, seconds:** `go run -tags desktop .` in irgo-demo against the local irgo
  (`go.work` or `replace`), window opens, JS↔Go `Bind` works.
- **Windows gate:** `CGO_ENABLED=0 GOOS=windows GOARCH=arm64 go build -tags desktop` the irgo-demo
  desktop binary → `irgo-winvm app-create -gui <exe>` in irgo-windows-vm (logged-in VM; if not,
  `irgo-winvm vm-repair -reboot`).
- **Linux:** cross-build must succeed (`CGO_ENABLED=0 GOOS=linux`); running it needs a Linux desktop
  (not available yet — note, don't block on it).

## Verify

1. `CGO_ENABLED=0 go build -tags desktop ./...` for darwin/arm64, darwin/amd64, windows/arm64,
   windows/amd64, linux/amd64, linux/arm64 — all green.
2. irgo's tests: `go test ./...` and `go vet -tags desktop ./...`.
3. irgo-demo desktop on macOS: window, title, size, page loads from the loopback server, secret
   injected (`window.__IRGO_SECRET__`), a bound function round-trips.
4. Same irgo-demo desktop `.exe` on Windows 11 ARM via irgo-windows-vm: same checks.
5. `irgo build desktop` produces a working app without a C compiler (run with `CC=false` to prove it).

## Upstream

PR to stukennedy/irgo from `feat/desktop-glaze`, referencing #16; after merge, comment on #16 with
the results and close it. irgo-demo regenerated with `irgo project upgrade` if templates change.
