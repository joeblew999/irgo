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

## One branch per concern

Not one per idea, question or fix that comes up while you are working. When
something new arrives and it belongs to what you are already doing, stay on the
branch.

Eight branches were made in one afternoon here, all for one concern — making
the workflow work. Each held a single commit, each was merged minutes later,
each was reviewed by nobody, and eleven tags came with them of which nine were
never used by anything.

A branch you merge minutes later and nobody reviews is not a unit of work.

## Never commit on integration

It is assembled by merging, not authored. A commit made there exists nowhere
else — and because a branch cut from `integration` carries the whole fork with
it, that work cannot be offered upstream without being re-cut by hand. It fails
silently: everything builds, the tests pass, and nobody finds out until someone
tries to send the change somewhere. Sixteen commits landed there in one
afternoon and four pieces of work had to be rescued afterwards.

`mise run merge` steps back off `integration` so you are not left standing
there. `mise run where` shows it if one lands anyway.

## The branches

| | |
|---|---|
| `integration` | the trunk. Assembled by merging, tagged for release |
| `main` | mirrors upstream. No fork features, not even `mise.toml`. Never written to |
| `rb/…`, `feat/…` | work, and what eventually becomes a pull request |

Everything here is meant to reach upstream eventually. **Do not open a pull
request without asking the repository's owner** — seven were opened once and
sat unread for two days while the work moved underneath them; all seven were
withdrawn.

## What is enforced, and what is habit

Most of this is habit. The tasks make the right thing shortest to type, and
nothing stops you typing something else. These genuinely stop a mistake:

| | how |
|---|---|
| `main` diverging from upstream | CI |
| commits authored on `integration` | CI |
| pushing to the upstream repository | `mise run setup` points its push URL at nothing |
| rebasing `main` or `integration` | git-stack refuses, after `setup` |
| committing generated files | `TestNoGeneratedFilesAreTracked` |
| losing a command's registration | `TestTheCommandSetIsWhatWeThinkItIs` |
| a tool on `@latest` | `TestEveryInstalledToolIsPinned` |
| a target with no dispatch | `TestEveryDeclaredBuildTargetIsDispatched` |
| unformatted code | `TestEverythingIsFormatted` |

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
