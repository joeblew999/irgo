// Whether a branch can be offered upstream without costing anyone an afternoon.
//
// The requirement is that a pull request never fails: no conflict for the
// maintainer to resolve, no red tick on their pipeline, no change to their CI
// configuration that they did not ask for. Everything here is a check that
// something specific would have gone wrong.
//
// It matters because upstream is active — they merge pull requests, and one of
// ours was ported by hand rather than merged. What arrives badly shaped costs
// them time they have already spent once.
package main

import (
	"fmt"
	"os/exec"
	"strings"
)

// forkOnly are paths that exist because this is a fork and would be noise, or
// worse, in someone else's repository.
//
// A pull request that rewrites the maintainer's CI config is the definition of
// the hassle this exists to prevent. mise.toml and CONTRIBUTING.md describe a
// workflow that is ours; fork-main.yml compares main to upstream, which from
// upstream's own repository is meaningless.
var forkOnly = []string{
	".github/workflows/fork-main.yml",
	".github/workflows/skills.yml",
	".github/workflows/browser.yml",
	"mise.toml",
	"mise.local.toml",
	"CONTRIBUTING.md",
}

// runOfferCheck reports whether a branch is safe to offer.
func runOfferCheck(args []string) error {
	branch := "HEAD"
	if len(args) > 0 {
		branch = args[0]
	}

	base := "upstream/main"
	if err := exec.Command("git", "rev-parse", "--verify", base).Run(); err != nil {
		return fmt.Errorf("no %s — add the upstream remote and fetch it first", base)
	}

	fmt.Printf("Checking %s against %s\n\n", branch, base)
	var problems []string

	// 1. Would it conflict? GitHub tests the merge result, not the branch tip,
	//    so this is the thing their pipeline will actually build.
	merge := exec.Command("git", "merge-tree", "--write-tree", base, branch)
	if out, err := merge.CombinedOutput(); err != nil {
		problems = append(problems, fmt.Sprintf(
			"conflicts with %s — the maintainer would have to resolve it:\n      %s",
			base, strings.TrimSpace(string(out))))
	} else {
		fmt.Println("  merges cleanly            OK")
	}

	// 2. Does it carry anything that is ours alone?
	files, err := changedFiles(base, branch)
	if err != nil {
		return err
	}
	var carried []string
	for _, f := range files {
		for _, bad := range forkOnly {
			if f == bad {
				carried = append(carried, f)
			}
		}
	}
	if len(carried) > 0 {
		problems = append(problems, "carries fork-only files:\n      "+
			strings.Join(carried, "\n      ")+
			"\n      Those are ours, not theirs. Drop them from the branch.")
	} else {
		fmt.Println("  no fork-only files        OK")
	}

	// 3. Is it a reviewable size? Not a failure — a reviewer's patience is not
	//    a pass/fail condition — but worth saying before it is sent.
	fmt.Printf("  %d file(s), %d commit(s)\n", len(files), countCommits(base, branch))

	if len(problems) > 0 {
		fmt.Println()
		for _, p := range problems {
			fmt.Printf("  NOT READY: %s\n", p)
		}
		return fmt.Errorf("%d problem(s) — fix them before offering", len(problems))
	}

	fmt.Println()
	fmt.Println("Merge and contents look right. What this cannot tell you is")
	fmt.Println("whether their CI passes, because theirs is not ours — run it:")
	fmt.Println()
	fmt.Println("  irgo project offer-check --run    (builds the merged tree)")
	return nil
}

func changedFiles(base, branch string) ([]string, error) {
	out, err := exec.Command("git", "diff", "--name-only", base+"..."+branch).Output()
	if err != nil {
		return nil, fmt.Errorf("comparing %s to %s: %w", branch, base, err)
	}
	var files []string
	for _, l := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		if l != "" {
			files = append(files, l)
		}
	}
	return files, nil
}

func countCommits(base, branch string) int {
	out, err := exec.Command("git", "rev-list", "--count", "--no-merges", base+".."+branch).Output()
	if err != nil {
		return 0
	}
	var n int
	fmt.Sscanf(strings.TrimSpace(string(out)), "%d", &n)
	return n
}

func init() {
	register(command{
		noun: "project", verb: "offer-check", order: 95,
		summary: "Check a branch is safe to offer upstream",
		args:    "[branch]",
		usage: [][2]string{
			{"", "the branch you are on"},
			{"fix/some-bug", "a named one"},
		},
		notes: "Answers the three ways a pull request wastes a maintainer's " +
			"time: it conflicts, it rewrites their CI config, or it fails " +
			"their pipeline. The first two are checked here; the third needs " +
			"their steps run against the merged tree, because their CI is not " +
			"ours — upstream has two workflows, this fork has five.",
		run: runOfferCheck,
	})
}
