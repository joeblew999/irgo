# For assistants working in this repository

Read `.claude/skills/workflow/SKILL.md` first. It is the short version of how a
change is made here, and it is kept next to the commands rather than in a
document that drifts from them.

The one line version: **use `mise run …`, never raw git for branch work, and run
`mise run check` before every commit.** `mise tasks` lists everything.

`.claude/skills/` also holds the references for the libraries this framework
ships — Datastar's attribute syntax and toki's TIK format. Both describe
failures that render perfectly and do nothing, which no test catches.
