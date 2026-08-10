package content

import (
	"strings"
	"testing"
)

// hasBody reports whether a block is something a reader gets to read, as
// opposed to a heading announcing that something follows.
func hasBody(b Block) bool {
	return b.Kind != KindH2 && b.Kind != KindH3
}

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
			case KindPara, KindH2, KindH3:
				prose++
				if strings.TrimSpace(b.Text) == "" {
					t.Errorf("%s: an empty %s block", p.Slug, b.Kind)
				}
			case KindCode:
				code++
				if strings.TrimSpace(b.Text) == "" {
					t.Errorf("%s: an empty code block", p.Slug)
				}
			case KindList:
				if len(b.Items) == 0 {
					t.Errorf("%s: a list with no items", p.Slug)
				}
			case KindTable:
				if len(b.Head) == 0 || len(b.Rows) == 0 {
					t.Errorf("%s: a table with no head or no rows", p.Slug)
				}
				for _, row := range b.Rows {
					if len(row) != len(b.Head) {
						t.Errorf("%s: table row has %d cells, head has %d",
							p.Slug, len(row), len(b.Head))
					}
				}
			case KindNote:
				if len(b.Paras) == 0 {
					t.Errorf("%s: a note with nothing in it", p.Slug)
				}
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
			if b.Kind == KindCode {
				code++
			}
		}
		if code == 0 {
			t.Errorf("%s has no code examples — the original has some, so they were lost in the port", p.Slug)
		}
	}
}

// TestNoHeadingIsEmpty guards the loss that did ship.
//
// The first port had only four block kinds, so every list, table and callout
// vanished and left its heading behind. "Key Features" followed immediately by
// "Platform Support" still renders, still returns 200, and is still a promise
// the page does not keep — 23 headings were in that state. A heading with
// nothing under it is content that was dropped, not content that is terse.
func TestNoHeadingIsEmpty(t *testing.T) {
	for _, p := range Pages {
		for i, b := range p.Blocks {
			if !hasBody(b) {
				// An h2 may introduce an h3 rather than prose of its own.
				if b.Kind == KindH2 && i+1 < len(p.Blocks) && p.Blocks[i+1].Kind == KindH3 {
					continue
				}
				if i == len(p.Blocks)-1 || !hasBody(p.Blocks[i+1]) {
					t.Errorf("%s: %q has nothing under it", p.Slug, b.Text)
				}
			}
		}
	}
}

// TestNoPortingArtifacts guards two mistakes the TSX port made silently.
//
// JSX writes a deliberate space as {" "}, and Go struct tags inside a JS
// template literal are written \`json:"..."\`. Carried across verbatim, both
// render as themselves: readers saw `the{" "} Quick Start` and
// \`json:"name"\`. Neither is caught by anything else here, because both
// produce a page that is complete and merely wrong.
func TestNoPortingArtifacts(t *testing.T) {
	for _, p := range Pages {
		for _, b := range p.Blocks {
			for _, text := range append(append([]string{b.Text}, b.Items...), b.Paras...) {
				if strings.Contains(text, `{" "}`) {
					t.Errorf("%s: leftover JSX spacing in %q", p.Slug, text)
				}
				if strings.Contains(text, "\\`") {
					t.Errorf("%s: JS-escaped backtick in %q", p.Slug, firstLine(text))
				}
			}
		}
	}
}

// TestInlineMarkupIsClosed catches markup that silently renders as itself.
//
// Inline() leaves anything unterminated alone rather than guessing, which is
// the right behaviour at render time and a typo that reaches the reader at
// authoring time. An odd number of backticks in a paragraph is always the
// second one.
func TestInlineMarkupIsClosed(t *testing.T) {
	for _, p := range Pages {
		for _, b := range p.Blocks {
			if b.Kind == KindCode {
				continue // code is never inline-parsed
			}
			for _, text := range append(append([]string{b.Text}, b.Items...), b.Paras...) {
				if strings.Count(text, "`")%2 != 0 {
					t.Errorf("%s: unclosed `code` span in %q", p.Slug, text)
				}
				if strings.Count(text, "**")%2 != 0 {
					t.Errorf("%s: unclosed **strong** span in %q", p.Slug, text)
				}
				if strings.Count(text, "](") != strings.Count(text, "[") {
					t.Errorf("%s: malformed link in %q", p.Slug, text)
				}
			}
		}
	}
}

func firstLine(s string) string {
	if i := strings.IndexByte(s, '\n'); i >= 0 {
		return s[:i]
	}
	return s
}
