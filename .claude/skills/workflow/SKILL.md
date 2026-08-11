---
name: workflow
description: How to make a change in an irgo project — the mise tasks, and why raw git is the wrong tool here.
---

# Making a change

**Use the tasks. Not raw git.** `mise tasks` lists them with a line each, and
the file they live in explains why each exists. They encode decisions this
project has already made and paid for; typing the git underneath skips those.

```sh
mise run wf:branch fix/some-thing   # start, off the trunk, up to date
# ...work...
mise run check                   # before EVERY commit — seconds
git commit
mise run wf:pr                      # push and open the pull request
mise run wf:status                  # what CI is doing
mise run wf:land                    # merge when green, delete the branch, back to trunk
```

CI runs on the pull request. It merges when green, and the branch is deleted
automatically.

## The rules that actually matter

**Never commit to the trunk.** In irgo itself that is `integration`, and it is
protected — the push is rejected, for everyone including the repository owner.
Three commits reached it unreviewed in one afternoon before that was enforced,
every one because a command earlier in a shell chain failed and the rest ran
anyway.

**Run `mise run check` before every commit.** It is `go vet`, the tests and the
wasm build. Seconds.

**Run `mise run verify` when you touch anything platform-shaped** — iOS,
Android, desktop, the Worker, the browser build. `check` never compiles a
native target, which is exactly how a broken one ships.

## If you are an assistant

Three habits, each from a real failure in this repository:

**Check that a command succeeded before relying on it.** Shell chains keep
going after a failure. `git switch` aborting and the next command committing to
whatever branch you were standing on is how junk reached the trunk, three times.

**Never `2>/dev/null` a command whose failure changes what you do next.** That
is what hid the aborted switch.

**Generate before you tidy.** `go mod tidy` cannot see imports that only exist
in ungenerated `*_templ.go`, so tidying first silently removes real
dependencies. Run `irgo project assets` first.

**Run it before you claim it.** Reading the source and reporting what it should
do is not the same as doing it. Nearly every real bug found here was found by
running something, not by reading it.

## Offering a change upstream

Only when the repository owner has agreed. Then:

```sh
mise run wf:offer fix/some-thing <commit>...
irgo project offer-check fix/some-thing --run
```

The check refuses the four ways a pull request wastes a maintainer's time: it
conflicts, it carries files that only exist in this fork, it carries commits
nobody meant to publish, or it fails *their* CI — which it verifies by building
the merged tree with upstream's own steps, not this project's.
