# Shipping an irgo app into mainland China

Two things are in the way, and only one of them is code.

The first is regulatory: nothing serves on port 80 or 443 from a mainland
address without an ICP filing, and a filing needs a Chinese corporate sponsor.
That sequence is fixed, it is not negotiable, and it takes weeks. It is
Phases 1 and 2 below, from our local partner.

The second is that the app has to survive the network it lands on. That part is
ours, and most of it is already done — irgo embeds its own client, so there is
no CDN to be blocked. What is left is short and specific, and it is Phase 3.

> **Status: unverified in country.** Phases 1, 2 and 4 are our partner's
> checklist, written down so it does not live in a chat thread. The irgo notes
> attached to each are checked against this repository and cite the file. The
> section on HarmonyOS NEXT is the open question, and it is first because it
> decides how much of Phase 4 is even reachable. Sources are at the bottom, and
> one of them is a vendor selling the service — orientation, not authority.

## A demo does not have to wait for a filing

Hosting outside the mainland — Hong Kong or Singapore — needs **no ICP filing
at all**. That is the fast path to something a Chinese audience can open, and
it is available this week rather than in a month. What it costs:

- **Slower, and unpredictably so.** Cross-border traffic is the bottleneck the
  filing exists to remove. Our SSE streams are long-lived connections across
  exactly that boundary, so this is the target most likely to show buffering and
  drops. Measure it (Phase 4) rather than assuming it is fine.
- **It is a website, not an app.** The store listings want a filing number —
  Apple asks for it before a China-region listing, and the local Android stores
  ask for it alongside the SCC. Offshore hosting reaches a browser, and that is
  all. *(Confirm the current app-filing rules with the partner; enforcement has
  tightened since 2023.)*
- **Weaker on Baidu**, if discovery ever matters.

So the two speeds are: **Hong Kong now, for a demo people can open**, and the
filing running in parallel because it takes weeks and gates everything else.
Start Phase 1 the day the demo goes up, not after it.

## The 10% that is not Android

Roughly one device in ten sold in China now runs **HarmonyOS NEXT** (HarmonyOS
5). NEXT dropped the AOSP compatibility layer: it does not install or run APKs.
It is a different operating system that happens to be made by Huawei, and it
takes `.hap` packages built with ArkTS/ArkUI, listed separately in AppGallery.

For irgo this is not a packaging problem, it is a runtime problem. The mobile
targets work by compiling the Go kernel into the app and serving requests
through the native bridge — no socket, no server (see "Mobile Architecture" in
[README.md](README.md)). That needs a Go toolchain targeting the platform, and
there is no OpenHarmony target in upstream Go. `gomobile` cannot produce a
`.hap`.

So the three honest options for that 10%, in order of how much they cost:

1. **The web build.** Same Go code, served from the mainland host, opened in the
   device browser. Works on every device regardless of OS, and it is the target
   we already deploy. It loses on-device Go — the kernel runs on the server —
   which matters only for the offline paths.
2. **A `.hap` webview shell** wrapping that same URL. Gets us an AppGallery
   listing and an icon on the home screen; still a server-side kernel. Written
   in ArkTS by someone who has done it before, which is not us.
3. **A WeChat Mini Program.** Reaches 100% of devices, ignores the OS question
   entirely, and is a separate app to build and maintain.

**Ask the partner:** whether the NEXT share matters for the demo audience, or
whether the browser is enough for the first cut. Options 2 and 3 are real
projects; option 1 costs nothing because we already build it.

## What we have to fix in irgo

Everything below is detailed in the phase it belongs to. This is the list on one
screen, because the engineering work is small and it is easy to lose inside the
paperwork.

| | Where | Why it matters | State |
|---|---|---|---|
| Google Fonts `@import` | [`cmd/irgo/templates/static/css/input.css:1`](cmd/irgo/templates/static/css/input.css), and so every generated project | A blocked stylesheet import does not degrade the page, it hangs it | **Open** — one line, and the same line in [`docs-templ/static/css/site.css:7`](docs-templ/static/css/site.css) |
| SSE buffering | streaming responses set no `X-Accel-Buffering: no` | Behind a buffering CDN or proxy the app loads and then never updates | **Open** — a header, plus CDN config on those paths |
| Android artifact is `.aab` | [`cmd/irgo/app_android_package.go:123`](cmd/irgo/app_android_package.go) | Play's format; the Chinese stores want a universal signed APK | **Open** — no `assembleRelease` path in the CLI |
| No Go target for HarmonyOS NEXT | the mobile bridge compiles the kernel in | ~10% of Chinese phones cannot run the mobile build at all | **Decision, not a fix** — see above |
| Datastar from a CDN | [`pkg/datastarjs`](pkg/datastarjs), served at `/_irgo/datastar.js` | `cdn.jsdelivr.net` is unreliable-to-blocked inside the firewall | **Done** — and `TestNoUnpinnedDatastarCDN` keeps it done |
| Google push (FCM) | nothing in the tree | Nothing to strip — and also no push on any platform today | **N/A** — if push is ever needed in China it is per-vendor (Huawei, Xiaomi, OPPO, vivo), never FCM |

## Phase 1 — Legal sponsorship and the domain

- [ ] **Secure a local corporate sponsor.** Foreign individuals cannot hold an
      ICP filing. A licensed Chinese company or agency is the legal owner and
      sponsor of every filing below.

      There are two shapes this takes: **own the entity** — a WFOE, joint
      venture or representative office, which is a company formation and months
      of it — or **borrow one**, partnering with a local entity or distributor
      that already holds the licence and files on our behalf. The partner we
      have is the second, and it is why this is weeks rather than a quarter.
      Whose name the filing sits under is worth being deliberate about: it is
      the legal owner of the service in China.

- [ ] **Register the domain with an MIIT-approved Chinese registrar.** A domain
      held elsewhere cannot be filed.
- [ ] **Pass real-name verification** — the sponsor's business licence and the
      authorised contact's ID, submitted to the registrar. The domain does not
      activate until this clears.

**What gets asked for**, so it can be gathered once rather than four times: the
Chinese business licence and registration certificates; proof of domain
ownership; the legal representative's and the contact person's ID details,
**with a mainland China phone number**; and the site's own particulars — domain,
IP, hosting provider, a description of the content, and the security measures in
place. That last one is a description we write, and it should match what the app
actually does.

## Phase 2 — Infrastructure and licensing

- [ ] **Provision mainland servers** — Tencent Cloud, Alibaba Cloud or Huawei
      Cloud, physically in mainland China.
- [ ] **Obtain the ICP filing (备案)** through the cloud provider's portal,
      which submits it to the provincial Communications Administration. One
      filing covers the web app and the mobile app. The provider keeps 80 and
      443 blocked until the government approves it: **typically 2–4 weeks.**

      **There are two ICPs, and the difference is months.** The *filing* (备案)
      above is free and covers a non-commercial site — informational, a
      brochure, a demo. The moment the service earns revenue — e-commerce, SaaS
      subscriptions, paid memberships — it needs the **commercial ICP licence**
      (经营性ICP许可证): stricter criteria, real cost, and **2–3 months** rather
      than 2–4 weeks. Decide which one this app is *before* filing, because
      shipping a demo under the free filing and then adding payment is not a
      change of plan, it is a restart. If the China plan ends in a paid product,
      start the licence early and run the demo on the filing meanwhile.
- [ ] **Complete the PSB filing** with the local Public Security Bureau, within
      30 days of the ICP number landing.
- [ ] **Register a Software Copyright Certificate (SCC)** with the Copyright
      Protection Center of China. A hard prerequisite for the local Android
      stores, so it gates Phase 4 — start it early.

**What this means for the build.** `irgo app cloudflare deploy` is not the China
deployment. The Worker target is a global edge, unfilable, and outside the
data-localisation rule below; it stays the target everywhere else. China is a
plain Linux host inside a mainland cloud, running the same binary — this is the
same server build the desktop and web targets already use, so there is no new
code path, only a new host.

## Phase 3 — The Great Firewall scrub

A single blocked script does not degrade the page. It hangs it, for the length
of the connection timeout, on every load.

- [ ] **Strip blocked dependencies** — Google Analytics, Google Fonts,
      reCAPTCHA, Meta SDKs, YouTube embeds.

      **We have one.** [`cmd/irgo/templates/static/css/input.css:1`](cmd/irgo/templates/static/css/input.css)
      opens with `@import url('https://fonts.googleapis.com/css2?...')`, so
      every project generated by `irgo project new` inherits a blocking request
      to `fonts.googleapis.com` in its first stylesheet. The docs site has the
      same import at [`docs-templ/static/css/site.css:7`](docs-templ/static/css/site.css).
      Self-host the two families, or drop to system fonts — CJK text is not
      being set in Space Grotesk anyway.

      **We do not have the other one.** Datastar is embedded in
      [`pkg/datastarjs`](pkg/datastarjs) and served from `/_irgo/datastar.js`,
      never from `cdn.jsdelivr.net`, and `TestNoUnpinnedDatastarCDN` in
      [`cmd/irgo/help_test.go:328`](cmd/irgo/help_test.go) fails the build if a
      template reintroduces the CDN. That test was written to stop client and
      Go library drifting apart; it happens to be the reason the app renders at
      all behind the firewall. There is no Firebase, FCM or `google-services`
      anywhere in the tree — nothing to strip, and also no push. See below.

- [ ] **Integrate local services** — Baidu or Amap for maps, WeChat or Alipay
      for auth and payment. We currently use none of these categories, which is
      the cheapest possible position to be in. Keep it that way for the demo.
- [ ] **Configure a mainland CDN**, activated with the ICP number, to cache
      assets across provinces.

      **Exempt the SSE routes.** irgo is SSE end to end — every Datastar patch
      is a stream. The Datastar SDK sets `Cache-Control: no-cache` and
      `Connection: keep-alive` (`datastar-go@v1.1.0/datastar/sse.go:47`) but
      **not** `X-Accel-Buffering: no`, and a buffering reverse proxy in front of
      a stream produces an app that looks alive and never updates. Set
      `X-Accel-Buffering: no` on streaming responses and turn caching and
      buffering off for those paths at the CDN. This is the failure most likely
      to be blamed on "the firewall" when it is our configuration.

- [ ] **Implement data localisation** — user data generated in China stays on
      the mainland servers. Practically: no shared database with the rest of the
      deployment, and no third-party service outside China in the request path.

      The filing is permission to serve, not a compliance certificate. Three
      laws apply regardless: **PIPL** (personal information — consent, and
      cross-border transfer rules), the **Data Security Law**, and the
      **Cybersecurity Law**. Between them they cover where data lives, what
      leaves the country and under what approval, and what security and logging
      we are expected to have. Non-compliance is not only a fine: the service
      gets blocked at the infrastructure, and the hosting provider can suspend
      or refuse it. Our architecture makes the hard part easy — the Go kernel is
      one binary with its own data, so a mainland deployment is a separate host
      with a separate database, not a special case in the code.

## Phase 4 — QA and distribution

### Testing on Chinese devices

Testing from here measures the wrong network and, for NEXT, the wrong operating
system.

- [ ] **Network, from inside.** [17ce.com](https://www.17ce.com) or Baidu's
      performance testing — load time and reachability from dozens of mainland
      cities and carriers. Overseas tools cannot see what we need to see.
      Test the SSE routes specifically, not just the first page load: buffering
      shows up as a page that loads fast and then does nothing.
- [ ] **Real devices, remotely.** Huawei DevEco cloud testing (the only lab
      with HarmonyOS NEXT hardware), Tencent WeTest, Alibaba MQC. These are
      device farms inside China, so they cover the device *and* the network.
      Ask the partner which they already have an account with.
- [ ] **Real devices, locally.** The partner side-loads the APK — that path
      needs no store listing and no SCC, so it is the first end-to-end test
      available and it should come long before any submission. iOS is harder:
      TestFlight external testing needs a review pass, internal testing needs
      them on the team.

### Submission

- [ ] **Local Android stores** — Tencent MyApp, Huawei AppGallery, Xiaomi,
      OPPO, vivo. Each takes a manual upload of the APK, the SCC and the app's
      ICP number. There is no single submission.

      **Gap:** `irgo app android package` builds a signed **.aab** via gradle
      `bundleRelease` ([`cmd/irgo/app_android_package.go:123`](cmd/irgo/app_android_package.go)),
      which is a Play format. The Chinese stores want a universal signed APK.
      Today that means `bundletool build-apks` off the .aab, or an
      `assembleRelease` path the CLI does not have yet. Worth adding before the
      first submission rather than during it.

- [ ] **China App Store (iOS)** — App Store Connect needs the ICP filing number
      and the local compliance details before the app can be listed in the
      China region.
- [ ] **HarmonyOS NEXT**, if we decide it is in scope — a separate AppGallery
      listing, a separate package. See the top of this file.

## What is still open

Questions for the partner, in the order the answers unblock work:

1. **Filing or licence?** Does this end as a free demo or a paid product? Free
   filing is 2–4 weeks, the commercial licence is 2–3 months, and the answer
   decides which clock starts today.
2. **Which cloud** — Tencent, Alibaba or Huawei? It decides the ICP portal, the
   CDN and the device lab in one answer.
3. **Do we go offshore first?** Hong Kong hosting puts a demo in front of people
   in days with no filing, while the filing runs. Their call, since they know
   the audience and whether "opens in a browser" is enough for it.
4. Is the browser enough for the HarmonyOS NEXT share of that audience?
5. Do they hold, or can they sponsor, the SCC application? It gates every
   Android store and it is the slowest item after the filing itself.
6. Which device farm do they already use, so we test where they can watch.

## Sources

- Our partner's checklist, sent 11 August 2026 — Phases 1, 2 and 4.
- Brocent, ["ICP Licensing in China for Foreign Enterprises"](https://www.brocent.com/blog/posts/5),
  20 May 2025 — the two ICP types and their timelines, the entity options, the
  document list, the offshore-hosting trade-off, and PIPL/DSL/CSL. Brocent sells
  managed IT services into China, so read it as orientation and confirm anything
  load-bearing with the partner.
- The irgo notes are checked against this repository at the commit that added
  this file, and cite `file:line`. If a line has moved, the claim is still
  checkable — grep for it.
