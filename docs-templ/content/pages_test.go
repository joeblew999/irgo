package content

import (
	"strings"
	"testing"
)

// TestEveryPageIsUsable catches the failure a rendered site hides: a page that
// exists, returns 200 and says nothing. The port carried the prose across
// mechanically, so a page that lost its content would otherwise look fine.
func TestEveryPageIsUsable(t *testing.T) {
	if len(Pages) == 0 {
		t.Fatal("no pages")
	}
	seen := map[string]bool{}
	for _, p := range Pages {
		if p.Slug == "" || p.Title == "" {
			t.Errorf("page %q has no slug or title", p.Title)
		}
		if seen[p.Slug] {
			t.Errorf("duplicate slug %q — one page would shadow the other", p.Slug)
		}
		seen[p.Slug] = true

		if strings.ContainsAny(p.Slug, " _.") || strings.ToLower(p.Slug) != p.Slug {
			t.Errorf("slug %q is not url-shaped", p.Slug)
		}

		var prose, code int
		for _, b := range p.Blocks {
			switch b.Kind {
			case Para, H2, H3:
				prose++
			case Code:
				code++
			}
			if strings.TrimSpace(b.Text) == "" {
				t.Errorf("%s: an empty %s block", p.Slug, b.Kind)
			}
		}
		if prose < 2 {
			t.Errorf("%s has %d prose blocks — it would render nearly blank", p.Slug, prose)
		}
		t.Logf("%-18s %2d prose, %2d code", p.Slug, prose, code)
	}
}

// TestPagesCarryTheirExamples guards the loss that nearly shipped.
//
// The port was mechanical, and the first pass matched only single-line code
// blocks — so the prose came across and 158 examples did not. Every page
// rendered, returned 200, and read like documentation with the useful half
// removed. A page of instructions with no code is the failure worth catching.
func TestPagesCarryTheirExamples(t *testing.T) {
	for _, p := range Pages {
		var code int
		for _, b := range p.Blocks {
			if b.Kind == Code {
				code++
			}
		}
		if code == 0 {
			t.Errorf("%s has no code examples — the original has some, so they were lost in the port", p.Slug)
		}
	}
}
