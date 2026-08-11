# Working on irgo

**`mise.toml` is the source of truth.** It holds every command and why it
exists, next to the command itself, where it cannot drift from what actually
runs.

```sh
mise install     # the branch tools
mise run setup   # protects main and integration, blocks pushes to upstream
mise tasks       # every command, with a line each
```

Read `mise.toml` when you want the reasoning. This file has only the things
that are not commands.

## Work on integration

Commit there. It is the trunk, it is what gets tagged, and it is what projects
consume — and it is now also where the work happens.

That reverses what this document used to say, so here is the reasoning rather
than just the instruction. The old rule was that every change be made on a
branch cut from `main`, so it stayed offerable upstream. It cost more than it
returned. Twenty-four branches accumulated, every one merged, fifty merge
commits into `integration` in a single day — and the property it was protecting
was already gone: `cmd_project_upgrade.go` appeared in 21 of those 24 branches
and `help_test.go` in 19, not because nineteen changes touched them but because
each branch inherited everything beneath it. The last one built carried
nineteen commits. That is a stack, not a set of reviewable pull requests.

Meanwhile the thing it bought is exercised approximately never: this fork does
not open pull requests upstream without asking, after seven were opened once and
withdrawn.

A commit on `integration` is not stranded. `mise run offer` replays it onto a
branch cut from `main` whenever somebody wants that, and what comes out is
cleaner than the stacked branch would have been.

`mise run check` before you commit. No workflow triggers on `integration` — all
of them are `branches: [main]` — so that check is the only gate there is.

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

**Do not open a pull request without asking the repository's owner.** Seven were
opened once and sat unread for two days while the work moved underneath them;
all seven were withdrawn.

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
| `main` | mirrors upstream. No fork features, not even `mise.toml`. Never written to |
| `fix/…`, `feat/…` | a pending offer upstream — not work in progress |

There should be very few of the third kind, and each should be one commit that
applies cleanly to `main`. Six survive from the twenty-four that once existed;
the rest were fork-internal and were never going anywhere, so they were retired
once merged.

## What is enforced, and what is habit

Most of this is habit. The tasks make the right thing shortest to type, and
nothing stops you typing something else. These genuinely stop a mistake:

| | how |
|---|---|
| `main` diverging from upstream | CI (`fork-main.yml`) |
| deleting a branch whose work is nowhere else | `git branch -d`, via `mise run done` |
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
