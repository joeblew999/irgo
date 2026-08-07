package clidoc

import (
	"encoding/json"
	"os/exec"
	"testing"
)

// TestEmbeddedCLIMatchesTheBinary is the whole reason this page is generated.
//
// The old site's CLI reference was written by hand and drifted every time a
// command moved: it documented `irgo doctor`, `irgo dev` and `irgo ios team`
// long after all three were renamed, and never mentioned nine flags that
// existed. Generating it only helps if the generated copy is refreshed, so a
// stale one fails here rather than shipping.
//
// Refresh with:  go tool irgo help --json > clidoc/cli.json
func TestEmbeddedCLIMatchesTheBinary(t *testing.T) {
	out, err := exec.Command("go", "tool", "irgo", "help", "--json").Output()
	if err != nil {
		t.Skipf("cannot run the CLI here: %v", err)
	}

	var live, embedded []Command
	if err := json.Unmarshal(out, &live); err != nil {
		t.Fatalf("the CLI's --json output does not parse: %v", err)
	}
	if err := json.Unmarshal(cliJSON, &embedded); err != nil {
		t.Fatalf("the embedded cli.json does not parse: %v", err)
	}

	if len(live) != len(embedded) {
		t.Fatalf("the CLI has %d commands and the docs have %d — refresh with:\n"+
			"  go tool irgo help --json > clidoc/cli.json", len(live), len(embedded))
	}

	for i := range live {
		l, e := live[i], embedded[i]
		if l.Name() != e.Name() {
			t.Errorf("command %d: the CLI has %q, the docs have %q", i, l.Name(), e.Name())
			continue
		}
		if l.Summary != e.Summary {
			t.Errorf("%s: summary drifted\n  CLI:  %s\n  docs: %s", l.Name(), l.Summary, e.Summary)
		}
		if len(l.Flags) != len(e.Flags) {
			t.Errorf("%s: the CLI has %d flags, the docs show %d",
				l.Name(), len(l.Flags), len(e.Flags))
		}
	}
}
