package main

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// The scaffolded workflows and the CLI have to agree on what a secret is
// called, and for a long time they did not: the workflow read IOS_TEAM_ID
// while irgo resolved IRGO_IOS_TEAM. Both halves looked right on their own.
// `irgo secrets push github` dutifully set the repository secrets, every job
// saw an empty string, and every job skipped itself — which is indistinguishable
// from "not configured yet", so nothing ever complained.

var secretRef = regexp.MustCompile(`secrets\.([A-Z0-9_]+)`)

// TestWorkflowSecretsExist checks every secret the scaffolded workflows read
// against the registry.
func TestWorkflowSecretsExist(t *testing.T) {
	for _, path := range workflowTemplates(t) {
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		name := filepath.Base(path)
		for _, m := range secretRef.FindAllStringSubmatch(string(data), -1) {
			ref := m[1]
			env := strings.TrimSuffix(ref, base64Suffix)

			cv := registryForEnv(env)
			if cv == nil {
				t.Errorf("%s reads secrets.%s, which irgo does not resolve — "+
					"`irgo secrets push github` will never set it", name, ref)
				continue
			}
			// The base64 form is only meaningful for a value that names a
			// file; asking for it on a plain string means the job reads a
			// variable nothing ever sets.
			if strings.HasSuffix(ref, base64Suffix) && !cv.path {
				t.Errorf("%s reads secrets.%s, but %s carries a value, not a file",
					name, ref, env)
			}
			if !strings.HasSuffix(ref, base64Suffix) && cv.path {
				t.Errorf("%s reads secrets.%s, but %s names a file and has to "+
					"travel as %s%s", name, ref, env, env, base64Suffix)
			}
		}
	}
}

// TestWorkflowsPassNoSigningFlags keeps the workflows a mapping of names.
//
// irgo reads every setting from its own environment variable, so a flag in CI
// is a second way to say the same thing — and the one that silently stops
// matching when a flag is renamed.
func TestWorkflowsPassNoSigningFlags(t *testing.T) {
	flags := map[string]bool{}
	for _, cv := range configRegistry {
		if cv.flag != "" {
			flags[cv.flag] = true
		}
	}
	for _, path := range workflowTemplates(t) {
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		for _, line := range strings.Split(string(data), "\n") {
			if !strings.Contains(line, "irgo app package") {
				continue
			}
			for flag := range flags {
				if strings.Contains(line, flag+" ") {
					t.Errorf("%s passes %s, which the env var already carries:\n  %s",
						filepath.Base(path), flag, strings.TrimSpace(line))
				}
			}
		}
	}
}

func workflowTemplates(t *testing.T) []string {
	t.Helper()
	dir := filepath.Join("templates", "github", "workflows")
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	var out []string
	for _, e := range entries {
		out = append(out, filepath.Join(dir, e.Name()))
	}
	if len(out) == 0 {
		t.Fatal("no workflow templates found")
	}
	return out
}
