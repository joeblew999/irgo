# Working on irgo

## Once per clone

```sh
mise install     # the branch tools
mise run setup   # protects main and integration, blocks pushes to upstream
mise tasks       # everything, with a line each
```

## A normal day

```sh
mise run start feat/the-thing   # a branch off integration
… work, committing as you go …
mise run check                  # exactly what CI runs
mise run ship                   # push it
```

When it is done:

```sh
mise run merge                  # onto integration, then steps back off
mise run tag v0.7.0             # only when something needs it
mise run publish
```

## One branch per concern

Not one per idea, question or fix that comes up while you are working. When
something new arrives and it belongs to what you are already doing, **stay on
the branch**.

Eight branches were made in one afternoon here:

```
feat/datastar-syntax        1 commit, merged minutes later
fix/release-task            1 commit, merged minutes later
feat/complete-tasks         1 commit, merged minutes later
… five more the same …
```

All one concern — making the workflow work. Each new remark got its own
branch, its own merge and its own tag: eleven tags, nine of which nothing ever
used. It should have been one branch and one tag.

A branch you merge minutes later and nobody reviews is not a unit of work.

## Never commit on integration

It is assembled by merging, not authored. A commit made there exists nowhere
else — and because a branch cut from `integration` carries the whole fork with
it, that work cannot be offered upstream without being re-cut by hand. It fails
silently: everything builds, the tests pass, and nobody finds out until someone
tries to send the change somewhere.

Sixteen commits landed there in one afternoon and four pieces of work had to be
rescued afterwards.

`mise run merge` steps back off `integration` when it finishes, so you are not
left standing where a commit should not go. If one lands anyway, `mise run
where` shows it — git-stack marks a local commit on a protected branch:

```
origin/integration  docs: one branch per concern…
⌽ integration       a commit made directly on integration
```

Move it and it reads clean again:

```sh
git branch feat/rescued
git reset --hard origin/integration
```

## Working on irgo and an app at the same time

```sh
cd your-app
mise run link ../irgo    # build against your checkout
… edit either repository, changes are live …
mise run unlink          # back to the pinned release
```

`link` writes `go.work`, which is gitignored — it cannot reach a commit, and CI
keeps building the released version. No tag, no version number, nothing to
undo by hand. Only tag when you actually want to publish.

## Offering it upstream

Everything here is meant to reach upstream eventually. A branch off
`integration` carries the fork, so it cannot be offered as it stands — re-cut
it from `main`, which mirrors upstream:

```sh
mise run offer fix/the-thing
git checkout feat/the-concern -- <the files>
```

That is a deliberate step, and it is where the change gets shaped for someone
else to read. Manageable per concern; impossible per remark.

**Do not open the pull request without asking the repository's owner.** Seven
were opened once and sat unread for two days while the work moved underneath
them; all seven were withdrawn.

## The branches

| | |
|---|---|
| `integration` | the trunk. Assembled by merging, tagged for release |
| `main` | mirrors upstream. No fork features, not even `mise.toml`. Never written to |
| `rb/…`, `feat/…` | work, and what eventually becomes a pull request |

## What is enforced, and what is habit

Most of this is habit — the tasks make the right thing shortest to type, and
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

Deliberately not prevented: opening a pull request, and committing wherever you
like with `git`. There is no git hook. A workflow that stops leaving you in the
wrong place beats one that refuses you afterwards.

## Versioning

Plain semver, ahead of upstream, no prerelease suffix:

```
v0.6.10       this fork
v0.3.1        upstream's latest
```

Not `v0.5.0-androidapi21.N`, which these tags used to be. Anything after a
hyphen is a **prerelease** and sorts *below* the version it names — so a fork
that is ahead was advertising itself as older, `go get -u` would never choose
it, and the day upstream tags `v0.5.0` every such tag silently loses.

Tag when something needs the change, not once per merge.

## Before you push

```sh
mise run check
```

It runs what CI runs. Some of those tests exist because of a specific mistake
and will say so when they fail — the list is in the table above.

Generated files are never committed: `_templ.go`, `static/css/output.css`, and
anything copied out of a dependency. Every build regenerates them.

## Rate

The hardest limit is not tooling. Seven pull requests once sat unread for two
days while nine more branches were built on top of them. Structure lets people
work in parallel; it does not make the work get reviewed.

Land one thing before starting the next.
