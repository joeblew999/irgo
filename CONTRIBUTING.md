# Working on irgo

```sh
mise install     # the tools, pinned and prebuilt
mise run setup   # protect main, once per clone
mise tasks       # what you can run, and what each does
```

Everything below is plain git. The tools make the repetitive parts cheap;
nothing here requires them, and CI uses none of them.

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
