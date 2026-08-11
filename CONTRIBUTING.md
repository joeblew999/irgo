# Working on irgo

**`mise.toml` is the source of truth.** It holds every command and why it
exists, next to the command itself, where it cannot drift from what actually
runs.

```sh
mise install     # every tool this workflow needs, including gh
gh auth login    # once per machine — mise installs gh, it cannot log you in
mise run setup   # once per clone — protects the trunk, blocks pushes to upstream
mise tasks       # every command, with a line each
```

Read `mise.toml` when you want the reasoning. This file has only the things
that are not commands.

## The loop

Three or four people share this trunk, so nothing lands on it directly.

```sh
mise run branch fix/some-thing   # off the trunk, up to date
# ...work...
mise run check                   # before every commit
git commit
mise run pr                      # pushes and opens the pull request
```

CI runs on the pull request. It merges when it is green and somebody has looked
at it, and GitHub deletes the branch on merge.

**These branches are short-lived** — hours, or a day. One per change. That is
deliberately not what this repository had before: twenty-four long-lived
branches that were *workspaces*, where you had to know which one owned a file
before you could edit it, kept alive for pull requests nobody opened. A branch
that exists for an afternoon and is deleted on merge is a different thing
entirely.

**Nobody commits to `integration`, including whoever owns the repository.** It
is protected and the rule applies to everyone — `enforce_admins` is on.

That is not caution for its own sake. While the owner was exempt, three commits
reached the trunk unreviewed in a single afternoon, each because a command
earlier in a shell chain failed and the rest ran anyway. GitHub reported
"Bypassed rule violations" and let them through. A rule that binds three people
out of four is a rule you cannot reason about.

If CI ever wedges and something must land, turn protection off, push, turn it
back on. Two minutes, deliberate, and visible in the settings log — which is
the point.

`mise run check` before every commit. It is `go vet`, the tests and the wasm
build — seconds. `mise run verify` when you touch anything platform-shaped,
because `check` never compiles a native target and that is how a broken one
ships.

## Offering something upstream

When a change should go to upstream, and the repository owner has agreed:

```sh
mise run offer fix/some-bug <sha>...
```

It cuts from `main`, replays only the commits you name, and pushes. The branch
carries that change and nothing else.

If a cherry-pick conflicts, the change depends on something only the fork has.
That is information, not a failure — it was never offerable, and branching
earlier would not have made it so.

Then check it, before anyone else has to:

```sh
irgo project offer-check fix/some-bug --run
```

A pull request wastes a maintainer's time in three ways, and this refuses all
three. It **conflicts** — checked with `merge-tree` against `upstream/main`,
since GitHub builds the merge result and not your branch tip. It **rewrites
their CI** — upstream has two workflows and this fork has five, so a branch
carrying `skills.yml`, `browser.yml`, `mise.toml` or this file
is refused by name. Or it **fails their pipeline** — so `--run` merges into a
throwaway worktree and runs *upstream's* steps, not ours: `go vet`, the wasm
build, `go test`.

Proven both ways: all six current candidates pass, and a branch with a
deliberate compile error is refused naming the symbol.

**Do not open a pull request without asking the repository's owner.** Seven were
opened once and withdrawn. Worth knowing, though: upstream does merge pull
requests — `#6`, `#13` and `#15` are theirs — and one of ours was **ported by
hand** rather than merged. They read what arrives, so what arrives should be
worth reading.

## Finish a branch when it is done

```sh
mise run done <branch>
```

There used to be a command to start a branch and none to end one, which is the
entire reason twenty-four accumulated.

The safety is git's own: `branch -d`, lowercase, refuses anything not merged
into the branch you are standing on, and the task stands on `integration`
first. It is deliberately not `-D` — a refusal means the work exists nowhere
else, which is exactly when you want to be stopped.

Retiring a merged branch loses nothing. `mise run merge` makes a real merge
commit, so the tip stays addressable forever as `<merge-commit>^2` and reachable
from `integration`; a branch can be recreated from it byte-for-byte, and then
offered.

## The branches

| | |
|---|---|
| `integration` | the trunk. Assembled by merging, tagged for release |
| `fix/…`, `feat/…` | a pending offer upstream — not work in progress |

There should be very few of the third kind, and each should be one commit that
applies cleanly to `main`. Six survive from the twenty-four that once existed;
the rest were fork-internal and were never going anywhere, so they were retired
once merged.

## What is waiting to be offered

Eleven branches, each gated green by `irgo project offer-check --run` against
upstream's own CI on the merged tree. They are offers, not workspaces — work
happens on `integration`.

`refactor/self-registering` goes first. Four of the others need `hooks.go` from
it and are branched on top; that was discovered rather than assumed, when
`fix/datastar` failed on `main` with `undefined: registerAssetStep`.

| | |
|---|---|
| `refactor/self-registering` | commands, targets and build steps declare themselves. **Land first** |
| `fix/scaffolding`, `fix/toolchain`, `fix/datastar`, `feat/android-toolchain`, `feat/agent-skills` | stacked on it |
| `fix/scaffold-formatting`, `fix/upgrade-preserves-gitignore`, `fix/mobile-clone-durable`, `feat/tailwind-sees-dependency-components`, `pr1/datastar-sourcemap` | independent, any order |

**One needs framing rather than just sending.** `feat/android-toolchain`
overlaps upstream's `c147158`, which pinned `-androidapi 21` to make the Android
pipeline work on the toolchains of the day — the same pin those old
`v0.4.0-androidapi21.N` tags were named after. This does not duplicate that fix;
it moves past it, to NDK r29 and API 36, because Play's floor has risen since.
Say so in the pull request. Arriving as an unexplained revert of somebody's
recent work is how a good change gets refused.

Every other overlap is incidental — `c147158` touched 26 files under
`cmd/irgo`, so almost anything does. Overlap is not conflict, and the gate
proves the difference by building the merged tree.

## What is enforced, and what is habit

Most of this is habit. The tasks make the right thing shortest to type, and
nothing stops you typing something else. These genuinely stop a mistake:

| | how |
|---|---|
| a pull request that would conflict, rewrite their CI, or fail it | `irgo project offer-check --run` |
| a target that stopped building | `mise run verify` — `check` never compiles a native one |
| the trunk breaking | CI, which now runs on `integration` — before this, it only ever ran on `main`, which receives no commits |
| deleting a branch whose work is nowhere else | `git branch -d`, via `mise run done` |
| a red commit reaching the trunk | branch protection — `integration` requires CI and a pull request |
| landing without anyone looking | branch protection — a pull request is required to merge |
| pushing to the upstream repository | `mise run setup` points its push URL at nothing |
| rebasing `main` or `integration` | git-stack refuses, after `setup` |
| committing generated files | `TestNoGeneratedFilesAreTracked` |
| losing a command's registration | `TestTheCommandSetIsWhatWeThinkItIs` |
| a tool on `@latest` | `TestEveryInstalledToolIsPinned` |
| a target with no dispatch | `TestEveryDeclaredBuildTargetIsDispatched` |
| unformatted code | `TestEverythingIsFormatted` |

Nothing runs CI on `integration` — every workflow is `branches: [main]` — so
`mise run check` before a commit is not a courtesy, it is the only gate. This
table used to claim CI caught commits authored on `integration`; it never did,
and a table that advertises a guard which does not exist is worse than one that
admits the gap.

`integration` is the only branch here that is not an offer. There is no local
`main`: it mirrored upstream and never received a commit, so it was one more
thing to explain and to keep in sync. Git already tracks upstream's code as
`upstream/main`, which is what `mise run offer` cuts from.

That also retired `fork-main.yml`, which guarded the mirror. Worth knowing why
it is no loss: **GitHub registers workflows from the default branch**, and while
`main` held that, only three of five workflows were known to Actions —
`fork-main.yml` among the missing. It had never been able to run at all, while
this document claimed it was enforcing something.

GitHub also deletes a head branch when its pull request merges, which is
`mise run done` performed by the host.

There is deliberately no git hook. A workflow that stops leaving you in the
wrong place beats one that refuses you afterwards.

## Versioning

Plain semver, ahead of upstream, no prerelease suffix:

```
v0.7.0        this fork
v0.3.1        upstream's latest
```

Not `v0.5.0-androidapi21.N`, which these tags used to be. Anything after a
hyphen is a **prerelease** and sorts *below* the version it names — so a fork
that is ahead was advertising itself as older, `go get -u` would never choose
it, and the day upstream tags `v0.5.0` every such tag silently loses.

Tag when something needs the change, not once per merge.

## Rate

The hardest limit is not tooling. Seven pull requests once sat unread for two
days while nine more branches were built on top of them. Structure lets people
work in parallel; it does not make the work get reviewed.

Land one thing before starting the next.
