# Working on irgo

## If work ends up on integration anyway

`mise run where` shows it — git-stack marks a local commit on a protected
branch:

```
origin/integration  docs: one branch per concern…
⌽ integration       a commit made directly on integration
```

Move it to a branch and it reads clean again:

```sh
git branch feat/rescued
git reset --hard origin/integration
mise run where          # the commit now sits on feat/rescued (ready)
```

Detection on your own machine, before CI. There is deliberately no git hook:
the real fix is that `mise run merge` steps back off integration when it
finishes, so you are never left standing where a commit should not go.

## What is actually enforced

Most of this workflow is convention — the tasks make the right thing shortest
to type, and nothing stops you typing something else. These are the parts that
genuinely stop a mistake:

| | how |
|---|---|
| this fork's main diverging from upstream | CI fails the push or the pull request |
| rebasing main or integration | `git-stack` refuses, after `mise run setup` |
| pushing commits to the upstream repository | `mise run setup` points its push URL at nothing |
| committing generated files | `TestNoGeneratedFilesAreTracked` |
| losing a command's registration | `TestTheCommandSetIsWhatWeThinkItIs` |
| a tool on `@latest` | `TestEveryInstalledToolIsPinned` |
| a build target with no dispatch | `TestEveryDeclaredBuildTargetIsDispatched` |

Everything else is a habit. `mise run setup` is what turns three of those from
habit into refusal, so run it once per clone.

What is deliberately **not** prevented: opening a pull request from this fork.
That is a decision, and it should stay one.

## Where do I work?

On a branch, one per **concern** — not one per idea, question or fix that comes
up while you are working on it.

```sh
mise run start feat/the-workflow    # a branch off integration
… keep committing to it until the concern is done …
mise run check
mise run ship
```

Everything here is meant to reach upstream eventually, so a branch is how a
change stays offerable. The mistake is not making branches — it is making a new
one every time something else comes up.

### The mistake, with the count

Eight branches were made in one afternoon:

```
feat/datastar-syntax        1 commit, merged minutes later
fix/release-task            1 commit, merged minutes later
feat/complete-tasks         1 commit, merged minutes later
feat/template-mise          1 commit …
fix/pin-release-keeps-fork  1 commit …
feat/check-matches-ci       1 commit …
feat/no-push-upstream       1 commit …
docs/point-at-the-workflow  1 commit …
```

Every one of them was the same concern: making the mise workflow work. Each
new fix arrived as a separate remark, and each got its own branch, its own
merge, and its own tag — eleven tags, nine of which nothing ever used.

It should have been **one branch**, committed to until the workflow worked, and
one tag at the end. A branch you merge minutes later and nobody reviews is not
a unit of work; it is a unit of typing.

So: when something new comes up and it is part of what you are already doing,
stay on the branch.

### Offering it upstream

A branch off `integration` carries the fork with it, so it cannot be offered as
it stands. When the concern is done and you want it upstream, re-cut it from
`main`:

```sh
mise run offer fix/the-thing        # a branch off main, carrying only itself
git checkout feat/the-workflow -- <the files>
```

That is a deliberate step, and it is when the change gets shaped for someone
else to read. Doing it per concern is manageable. Doing it per remark, as
above, is what made eight branches out of one.

## What the other branches are

You will see a lot of them in a clone. Only one is a place to work:

| | |
|---|---|
| `integration` | **this fork's trunk.** Where your work starts from, and what gets tagged for release |
| `main` | a mirror of upstream irgo. No fork features, not even `mise.toml`. Never written to |
| `rb/…` | finished work waiting to be offered upstream. Not yours to touch |

Branching from `main` by mistake is the confusing failure: you land on a tree
with none of the fork's code and no `mise` tasks, because upstream has neither.
That is what `main` is *for* — it is the clean base for offering one change
upstream, which is:

```sh
mise run offer fix/the-thing
```

Use that only when you mean to send something to upstream irgo. Expect the
tasks to be missing on such a branch; switch back to `integration` and they
return.

If you lose track, `mise run where` draws the tree.

## Small branches, not one big one

irgo changes arrive as a set of small branches. That is not a style
preference — it is what makes them reviewable, and it is what lets several
people work at once without meeting in the same files.

The code is arranged so they *can* be independent. A command, a build target, a
build step and a tool each declare themselves next to the code that implements
them, so adding one touches that file and nothing else:

```go
register(command{noun: "app", verb: "build", …})   // a command
registerTarget("app build", "web", …, buildWeb)    // a target, and its dispatch
registerAssetStep(assetOrderKit, syncAssets)       // work every build does
registerPreTestStep(installBrowsers)               // provisioning some projects need
```

Before this, every feature edited one central table, so unrelated changes
collided in a file neither was about. Seven branches with one real dependency
between them could not be merged in any order. If you find yourself editing a
shared list to add something, that is the smell — the list should be derived.

## Branch from main, never from your work in progress

```sh
git switch --create fix/the-thing main
```

`main` is the base every branch is cut from, so anything extra in it travels
into all of them. On a fork this matters more: **a fork's main must stay
identical to upstream's**, or a pull request that should be one change arrives
as a hundred. CI enforces it — see `.github/workflows/fork-main.yml`, which
disables itself on the upstream repository.

Work that a fork needs but upstream has not taken lives on an `integration`
branch, and what projects consume is a **tag**. A tag needs no branch, so main
stays clean.

## Versioning this fork

Plain semver, ahead of upstream, no prerelease suffix:

```
v0.6.0        this fork
v0.3.1        upstream's latest
```

Not `v0.5.0-androidapi21.N`, which is what these tags used to be. Anything after
a hyphen is a **prerelease**, and a prerelease sorts *below* the version it
names — so `v0.5.0-androidapi21.5` is older than `v0.5.0`. A fork that is ahead
of upstream while advertising itself as older is backwards: `go get -u` will
never choose it, and the day upstream tags `v0.5.0` every such tag silently
loses.

The module path already says whose fork it is. The tag only has to say which
version, and be bigger than upstream's.

## Moving a stack when its base changes

```sh
mise run snapshot   # save where every branch points, first
mise run sync       # rebase your branches onto the updated parent
mise run verify     # build and test every branch, not just this one
mise run undo       # put them all back, if it went wrong
```

`mise run verify` is the one worth the habit. It runs the tests on *every*
branch in the stack — the failure it catches is a branch that only compiles
because of a change sitting in a later one.

## Before you push

```sh
go test ./...
```

CI runs the same. Some of those tests exist because of specific mistakes and
will tell you so when they fail:

| test | what it caught |
|---|---|
| `TestTheCommandSetIsWhatWeThinkItIs` | a command whose registration was lost in a rebase — silent, because everything derived from it agreed |
| `TestNoGeneratedFilesAreTracked` | fifteen build artifacts committed by `git add -A` on a branch whose `.gitignore` did not cover them |
| `TestEveryDeclaredBuildTargetIsDispatched` | a target advertised in help that failed at run time |
| `TestEveryInstalledToolIsPinned` | a tool on `@latest`, so a build could change with no commit |

Generated files are never committed: `_templ.go`, `static/css/output.css`, and
anything copied out of a dependency. Every build regenerates them.

## Rate

The hardest limit is not tooling. Seven pull requests once sat unread for two
days while nine more branches were built on top of them. Structure lets people
work in parallel; it does not make the work get reviewed. Land one thing before
starting the next.
