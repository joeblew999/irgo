package content

import (
	"os"
	"testing"
)

// TestLLMsTxtMatchesTheFramework catches the copy going stale.
//
// The Introduction page points AI assistants at /llms.txt, and the site serves
// it from static/. It has to be a copy: docs-templ is its own Go module, and
// go:embed cannot reach up to the framework's own llms.txt. A copy nobody
// checks is a copy that drifts, and the failure is invisible — the file still
// serves, it just describes an older framework.
func TestLLMsTxtMatchesTheFramework(t *testing.T) {
	const canonical = "../../llms.txt"

	want, err := os.ReadFile(canonical)
	if err != nil {
		t.Skipf("no framework llms.txt to compare against: %v", err)
	}
	got, err := os.ReadFile("../static/llms.txt")
	if err != nil {
		t.Fatalf("the site has no llms.txt, but the Introduction page links to it: %v", err)
	}
	if string(got) != string(want) {
		t.Errorf("static/llms.txt has drifted from %s — copy it across:\n\tcp %s docs-templ/static/llms.txt",
			canonical, canonical)
	}
}
