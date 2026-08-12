# Push notifications in irgo

There is no push in irgo today — on any platform. `grep -ril
"fcm\|apns\|firebase\|PushKit"` across `*.go`, `*.kt` and `*.swift` returns
nothing, and [CHINA.md](CHINA.md) records the same absence from the other
direction: there is no Firebase to strip because there was never any to add.

That is the starting position, and it decides the order of everything below.
The question that prompted this file was the Chinese one — five vendor SDKs,
five sets of credentials, a manifest full of proprietary background services.
It is the hardest instance of push there is, and building it first means
designing the general case by accident, from its worst case. So: the general
case first, iOS as the first real backend, and the Chinese vendors as the last
mile.

> **Status: design, not implementation.** Nothing here is built. The claims
> about *this repository* cite `file:line` and are checked at the commit that
> added this file. The claims about *vendors* — who requires a mainland
> entity, which console wants a store listing first, what the quotas are — are
> **not verified in country**, carry the same weight as the rest of CHINA.md's
> partner checklist, and are collected into questions at the bottom rather
> than asserted.

## Why the general case first

Push has one shape everywhere: ask permission, get a token, give the token to a
server, have the server hand a message to a delivery service that wakes the
app. What changes per platform is the delivery service and its credentials —
not the shape.

The temptation with China is to start from Huawei, because Huawei is the
constraint. The cost of that is a design carrying Huawei's vocabulary — its
message categories, its two-tier classification — into the parts that every
other platform has to use too. Start from the intersection and each vendor is
an implementation of an interface. Start from Huawei and each *other* platform
is an exception to Huawei.

There is also a scheduling argument. Everything Chinese is gated on things that
take months and are not ours: see "What gates the Android matrix" below. The
platform-neutral half is gated on nothing. It can be built, shipped and used
outside China while the filing and the SCC are still moving.

## The seam already exists

irgo has a native-capability bridge, and push needs no new one.

A platform capability is an `IrgoPlugin` — a namespace and a `handle` method —
registered into a per-process registry:

- Kotlin: the `IrgoPlugin` interface at
  [`IrgoNative.kt:24`](cmd/irgo/templates/android/Example/app/src/main/kotlin/com/irgo/IrgoNative.kt),
  registered by `IrgoNative.register` at
  [`IrgoNative.kt:58`](cmd/irgo/templates/android/Example/app/src/main/kotlin/com/irgo/IrgoNative.kt)
- Swift: the `IrgoPlugin` protocol at
  [`IrgoNative.swift:6`](cmd/irgo/templates/ios/Example/Example/IrgoNative.swift),
  registered at [`IrgoNative.swift:37`](cmd/irgo/templates/ios/Example/Example/IrgoNative.swift)

Go reaches them through [`pkg/native/native.go:91`](pkg/native/native.go)
(`native.Call`), templates through `irgo.native('namespace.method', {...})`, and
a method no platform implements comes back as `ErrNotSupported`
([`native.go:36`](pkg/native/native.go)) rather than a crash. Go-side fallbacks
register at [`native.go:74`](pkg/native/native.go).

**And the rendering half is already written.** `NotificationsPlugin`
([`IrgoPlugins.kt:306`](cmd/irgo/templates/android/Example/app/src/main/kotlin/com/irgo/IrgoPlugins.kt))
and `IrgoNotificationsPlugin`
([`IrgoPlugins.swift:342`](cmd/irgo/templates/ios/Example/Example/IrgoPlugins.swift))
already request the runtime permission and post a local notification. What they
cannot do is fire when the app is not running — that is the whole of what push
adds. Push does not need a new way to *show* a notification. It needs remote
delivery into the one that exists.

One correction worth writing down, because the external analysis that prompted
this file got it backwards: **the WebView never sees the token.** A WebView
cannot receive one. The native plugin captures it and hands it to Go across the
bridge, exactly as `device.info` hands over the model string.

## What to build

Three pieces, none of which mention a vendor.

**1. A `push.*` plugin namespace**, alongside `notifications.*`, with the same
method on every platform:

| Method | Returns |
|---|---|
| `push.requestPermission` | `{granted}` |
| `push.token` | `{token, provider}` — `provider` is how the server knows which sender to use |
| `push.onMessage` | delivered inbound, app-foregrounded; background delivery is the OS's job |

`provider` in the token response is the load-bearing field. It is what lets one
token store hold APNs, FCM and five Chinese vendors without the client
knowing anything about routing.

**2. Token registration as an ordinary Go route.** The token crosses into Go
through the bridge and then goes wherever the app's own router sends it — no
special path, no framework-owned endpoint. On web and desktop, a Go fallback
registered with `native.Register` keeps the same handler compiling and
degrading rather than turning into a build tag.

**3. `pkg/push` — a `Sender` interface and a token store.**

```go
type Message struct {
    Title, Body  string
    CollapseKey  string
    Priority      Priority
    Data         map[string]string
    Extras       map[string]any // per-provider, opaque to everything else
}

type Sender interface {
    Send(ctx context.Context, token string, m Message) error
}
```

`Message` must stay the **intersection**, not the union. Huawei's message
category and Xiaomi's channel id go in `Extras`, keyed by provider. This is the
one design decision worth defending: the moment Huawei's taxonomy is a field on
the shared struct, every other backend grows a comment explaining that it
ignores that field, and the interface stops being an interface.

## APNs first

The first concrete `Sender` should be Apple's, for four reasons that have
nothing to do with preferring iOS:

- **No Chinese entity required.** It is buildable this week, by us, with no
  partner in the loop.
- **APNs is not blocked in mainland China.** So the iOS half of a China launch
  is finished by the same work that serves everywhere else — the Chinese
  problem is Android-only.
- **The credential is already documented here.** APNs token auth uses a `.p8`
  key from App Store Connect; [`irgo.package.toml`](cmd/irgo/templates/irgo.package.toml)
  already describes that key material and where to get it, for review
  monitoring.
- **iOS already builds and signs** in this project, so there is a device to
  test on without a new toolchain.

FCM is the natural second, and it is the one that covers Android everywhere
except China.

## The Android matrix, and what gates it

Five vendors, not four. This is the correction that matters most to CHINA.md's
existing note.

| Channel | Covers | Needs a mainland entity | Needs a store listing first | Go library |
|---|---|---|---|---|
| FCM | Android outside China | no | no | official |
| Huawei HMS Push | Huawei devices (Android) | overseas developer accounts exist; the mainland service is separate | no | none — REST |
| Honor Push | Honor devices | likely | likely | none — REST |
| Xiaomi MiPush | Xiaomi, Redmi | likely | no | `yilee/xiaomi-push`, unmaintained |
| OPPO Push | OPPO, OnePlus, Realme | likely | reportedly yes | none — REST |
| vivo Push | vivo, iQOO | likely | reportedly yes | none — REST |
| HarmonyOS NEXT Push Kit | ~10% of Chinese phones | — | — | **unreachable** |

Three things follow.

**Honor is not Huawei.** It was sold off in 2020 and has its own push service.
A `Build.MANUFACTURER.contains("huawei")` branch — which is how every
integration guide starts — silently misses every Honor device. And branching on
`MANUFACTURER` at all is the weaker check: what matters is whether that
vendor's push service is present and usable on the device in hand.

**HarmonyOS NEXT is not on this list by accident.** Huawei's *Android* HMS Push
SDK is not NEXT's Push Kit. NEXT takes `.hap` packages built with ArkTS, which
`gomobile` cannot produce — the same wall CHINA.md already documents for the
app itself. Push does not make that 10% any more reachable than it already was,
and the same three options apply: the web build, a `.hap` shell someone else
writes, or a WeChat Mini Program.

**The Go side is ours to write.** There is no meaningful Go SDK for four of
these. Each is an OAuth-ish token exchange followed by a JSON POST — on the
order of 150–250 lines each, plus the token-refresh and error-mapping that
never appear in the sample code. That is real work but it is *known* work. It
is not the expensive part.

### The expensive part is the paperwork, and CHINA.md already owns it

The vendor consoles want a mainland business entity, and OPPO and vivo
reportedly want the app already listed in their stores. CHINA.md Phase 4 puts
those listings behind the Software Copyright Certificate, which it flags as the
slowest item after the filing itself.

So the dependency runs: **sponsor entity → SCC → store listing → push
credentials → push.** Push is downstream of the slowest thing in the China
plan. It is a post-launch feature by construction, unless the partner already
holds accounts.

Practically, that means the vendor developer accounts belong in Phase 1's
document-gathering list — collected once with everything else — rather than
discovered later as a separate errand.

### Why not an aggregator — and it is not delivery quality

JPush, Getui and UMeng exist precisely because the above is tedious, and the
usual framing is that direct integration buys better delivery. It does not.
Aggregators sit *on top of* the same vendor channels; the delivery is the same
delivery.

What direct integration actually buys, here, is compliance. CHINA.md Phase 3
requires data localisation and says it plainly: no third-party service outside
China in the request path. An aggregator is a third party holding device tokens
and message contents, which is a PIPL question before it is an engineering one.
That is the argument — cost and data path, not reliability.

## The build problem, and the precedent that answers it

Every integration guide says to add dependencies to `app/build.gradle.kts` and
services to `AndroidManifest.xml`. In irgo those files are **generated**:
`android/Example/` is gitignored and rebuilt from
`cmd/irgo/templates/android/Example/` by `irgo app build android` and refreshed
by `irgo project upgrade`. Hand-edits survive until the next build.

The mechanism that does work already exists, once:
[`i18n_manifests.go:82`](cmd/irgo/i18n_manifests.go) opens the *generated*
manifest and injects `android:localeConfig`, driven by what is in
`tokibundle/`. Diff the template manifest against a generated one in any irgo
project and that attribute is the only difference — which is both the proof
that the shell is generated and the proof that CLI-side injection is the
supported way to change it.

Push follows the same shape, with one addition it cannot do without: **a
per-project opt-in.** No irgo app should ship five vendors' background services
and broadcast receivers by default — it is dead weight for every project
outside China, and it is the kind of manifest that gets an app questioned in
review. An `[android.push]` section in
[`irgo.package.toml`](cmd/irgo/templates/irgo.package.toml) — which is the
project's file, not the framework's — naming the enabled vendors, driving the
manifest entries, the gradle dependencies, and the placement of the vendor
config files (`agconnect-services.json` and friends).

The config files are secrets-adjacent and per-vendor. They belong wherever
`irgo.package.local.toml` already puts credentials: gitignored, or supplied by
environment in CI.

## What is still open

For the partner, in the order the answers unblock work:

1. **Is push in scope for the China demo at all?** Given the dependency chain
   above, a demo can ship without it and probably should. If push is a
   launch requirement, the vendor accounts have to start in Phase 1.
2. **Who holds the vendor developer accounts** under the sponsoring entity —
   the same partner who holds the ICP filing, or someone else? It is the same
   question as the filing ownership, and it deserves the same deliberateness.
3. **Do OPPO and vivo actually require a store listing before issuing push
   credentials?** If so, push cannot precede the SCC, and that is worth knowing
   before anyone plans around it.
4. **Does the demo audience's device mix make the Chinese vendors urgent at
   all** — or does iOS plus FCM cover enough of it that Android-in-China push
   waits for the paid product?

For us, independent of any of that:

5. Build the `push.*` plugin surface and `pkg/push` with APNs behind it. It is
   gated on nothing, it is useful outside China immediately, and it is what
   every vendor implementation plugs into.

## Sources

- This repository, at the commit that added this file; every `file:line` above
  is checked against it. If a line has moved, the claim is still checkable —
  grep for it.
- [CHINA.md](CHINA.md) for the filing sequence, the SCC gate, the
  data-localisation rule and the HarmonyOS NEXT problem. This file does not
  restate them; it depends on them.
- Vendor requirements — entity, store listing, quotas, the Honor split — are
  from an assistant with a mid-2026 knowledge cutoff and are **unverified in
  country**. Every one of them is load-bearing for scheduling, so confirm with
  the partner before planning around it. They are written as questions above
  for that reason.
