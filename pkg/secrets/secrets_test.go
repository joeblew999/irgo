package secrets

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

const sample = `# A comment.
[providers.keychain]
type = "keychain"
service = "fnox"

[providers.shell]
type = "env"

[secrets]
CLOUDFLARE_API_TOKEN  = { provider = "keychain", value = "CLOUDFLARE_API_TOKEN" }
IRGO_IOS_TEAM         = { provider = "keychain", value = "IRGO_IOS_TEAM" }
FROM_ENV              = { provider = "shell", value = "SOME_ENV_VAR" }
DEFAULTED             = { provider = "keychain" }
LITERAL               = "just-a-value"
`

func TestParse(t *testing.T) {
	cfg, err := parse(sample)
	if err != nil {
		t.Fatal(err)
	}
	if got := cfg.Providers["keychain"]["service"]; got != "fnox" {
		t.Errorf("keychain service is %q", got)
	}
	if len(cfg.Secrets) != 5 {
		t.Fatalf("parsed %d secrets, want 5", len(cfg.Secrets))
	}

	// Declaration order is preserved, so failures report predictably.
	if cfg.Secrets[0].Name != "CLOUDFLARE_API_TOKEN" {
		t.Errorf("first secret is %q", cfg.Secrets[0].Name)
	}
	// fnox lets you omit the lookup key; it defaults to the secret's name.
	for _, s := range cfg.Secrets {
		if s.Name == "DEFAULTED" && s.Value != "DEFAULTED" {
			t.Errorf("DEFAULTED resolves against %q, want its own name", s.Value)
		}
	}
}

func TestResolveEnvAndLiteral(t *testing.T) {
	cfg, err := parse(sample)
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("SOME_ENV_VAR", "from-the-environment")

	for _, s := range cfg.Secrets {
		switch s.Name {
		case "FROM_ENV":
			got, err := cfg.resolve(s)
			if err != nil || got != "from-the-environment" {
				t.Errorf("FROM_ENV = %q, %v", got, err)
			}
		case "LITERAL":
			got, err := cfg.resolve(s)
			if err != nil || got != "just-a-value" {
				t.Errorf("LITERAL = %q, %v", got, err)
			}
		}
	}
}

// TestUnknownProviderSaysWhatToDo — this reads a subset of what fnox writes,
// so meeting a provider it does not implement has to point somewhere.
func TestUnknownProviderSaysWhatToDo(t *testing.T) {
	cfg, err := parse(`
[providers.op]
type = "onepassword"

[secrets]
TOKEN = { provider = "op", value = "op://vault/item/field" }
`)
	if err != nil {
		t.Fatal(err)
	}
	_, err = cfg.resolve(cfg.Secrets[0])
	if err == nil {
		t.Fatal("expected an error for an unimplemented provider")
	}
	if !contains(err.Error(), "fnox") {
		t.Errorf("the error should point at fnox; it says %q", err)
	}
}

// TestEnvironmentWins — CI supplies these names from repository secrets, and
// anyone can export one for a single command. Neither should be overridden.
func TestEnvironmentWins(t *testing.T) {
	dir := t.TempDir()
	write(t, filepath.Join(dir, ConfigName), `
[secrets]
ALREADY_SET = "from-the-file"
`)
	t.Setenv("ALREADY_SET", "from-the-environment")

	applied, _, err := Apply(dir, "")
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range applied {
		if name == "ALREADY_SET" {
			t.Error("Apply overwrote a variable that was already set")
		}
	}
	if os.Getenv("ALREADY_SET") != "from-the-environment" {
		t.Error("the environment lost")
	}
}

// TestNoConfigIsNotAnError — most projects have no secrets, and every command
// has to work without a fnox.toml.
func TestNoConfigIsNotAnError(t *testing.T) {
	applied, skipped, err := Apply(t.TempDir(), "")
	if err != nil {
		t.Fatalf("a project with no fnox.toml should not error: %v", err)
	}
	if len(applied) != 0 || len(skipped) != 0 {
		t.Errorf("applied %v, skipped %v", applied, skipped)
	}
}

// TestFindWalksUp — commands run from a subdirectory (the docs site builds
// from docs-templ) but credentials belong to the repository.
func TestFindWalksUp(t *testing.T) {
	root := t.TempDir()
	write(t, filepath.Join(root, ConfigName), "[secrets]\n")
	nested := filepath.Join(root, "docs", "deep")
	if err := os.MkdirAll(nested, 0o755); err != nil {
		t.Fatal(err)
	}
	got, ok := Find(nested)
	if !ok {
		t.Fatal("did not find the config from a subdirectory")
	}
	if filepath.Base(got) != ConfigName {
		t.Errorf("found %q", got)
	}
}

// TestApplyNeverReportsValues — the point of this package is that a secret
// reaches the process without reaching a log. Apply returns names only.
func TestApplyNeverReportsValues(t *testing.T) {
	dir := t.TempDir()
	write(t, filepath.Join(dir, ConfigName), `
[secrets]
SOME_TOKEN = "super-secret-value"
`)
	t.Setenv("SOME_TOKEN", "")
	os.Unsetenv("SOME_TOKEN")

	applied, skipped, err := Apply(dir, "")
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range applied {
		if contains(name, "super-secret") {
			t.Error("Apply returned a value, not a name")
		}
	}
	for name, err := range skipped {
		if contains(name+err.Error(), "super-secret") {
			t.Error("a skip reason leaked the value")
		}
	}
}

func write(t *testing.T, path, body string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
}

func contains(haystack, needle string) bool {
	return len(haystack) >= len(needle) &&
		(haystack == needle || len(needle) == 0 ||
			indexOf(haystack, needle) >= 0)
}

func indexOf(h, n string) int {
	for i := 0; i+len(n) <= len(h); i++ {
		if h[i:i+len(n)] == n {
			return i
		}
	}
	return -1
}

// TestSopsProvider decrypts a real SOPS file, if sops and a key are available.
//
// Skipped rather than mocked: the value of this provider is that it drives the
// actual binary, and a fake exec would test the fake.
func TestSopsProvider(t *testing.T) {
	if _, err := exec.LookPath("sops"); err != nil {
		t.Skip("sops is not on PATH")
	}
	keygen, err := exec.LookPath("age-keygen")
	if err != nil {
		t.Skip("age-keygen is not on PATH")
	}

	dir := t.TempDir()
	keyFile := filepath.Join(dir, "key.txt")
	if out, err := exec.Command(keygen, "-o", keyFile).CombinedOutput(); err != nil {
		t.Skipf("age-keygen failed: %s", out)
	}
	pub := agePublicKey(t, keyFile)

	enc := filepath.Join(dir, "secrets.enc.yaml")
	write(t, enc, "cloudflare:\n    api_token: the-token\n")
	cmd := exec.Command("sops", "encrypt", "--age", pub, "--in-place", enc)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("sops encrypt: %s", out)
	}

	write(t, filepath.Join(dir, ConfigName), `
[providers.sops]
type = "sops"
file = "secrets.enc.yaml"

[secrets]
CLOUDFLARE_API_TOKEN = { provider = "sops", value = "cloudflare.api_token" }
`)

	t.Setenv("SOPS_AGE_KEY_FILE", keyFile)
	os.Unsetenv("CLOUDFLARE_API_TOKEN")

	applied, skipped, err := Apply(dir, "")
	if err != nil {
		t.Fatal(err)
	}
	if len(applied) != 1 {
		t.Fatalf("applied %v, skipped %v", applied, skipped)
	}
	if got := os.Getenv("CLOUDFLARE_API_TOKEN"); got != "the-token" {
		t.Errorf("decrypted to %q", got)
	}
}

func agePublicKey(t *testing.T, keyFile string) string {
	t.Helper()
	data, err := os.ReadFile(keyFile)
	if err != nil {
		t.Fatal(err)
	}
	for _, line := range strings.Split(string(data), "\n") {
		if _, pub, ok := strings.Cut(line, "public key: "); ok {
			return strings.TrimSpace(pub)
		}
	}
	t.Fatal("no public key in the generated file")
	return ""
}

// TestExtractPath — sops indexes with ["a"]["b"], and a dotted key is how a
// fnox.toml would name it.
func TestExtractPath(t *testing.T) {
	if got := extractPath("cloudflare.api_token"); got != `["cloudflare"]["api_token"]` {
		t.Errorf("extractPath = %s", got)
	}
	if got := extractPath("TOKEN"); got != `["TOKEN"]` {
		t.Errorf("extractPath = %s", got)
	}
}

// TestSopsKeyFromKeychainProvider is the arrangement worth recommending: one
// age key held wherever secrets already live, everything else encrypted in the
// repository. Here the key comes from another declared secret rather than from
// a file on disk.
func TestSopsKeyFromAnotherSecret(t *testing.T) {
	if _, err := exec.LookPath("sops"); err != nil {
		t.Skip("sops is not on PATH")
	}
	keygen, err := exec.LookPath("age-keygen")
	if err != nil {
		t.Skip("age-keygen is not on PATH")
	}

	dir := t.TempDir()
	keyFile := filepath.Join(dir, "key.txt")
	if out, err := exec.Command(keygen, "-o", keyFile).CombinedOutput(); err != nil {
		t.Skipf("age-keygen failed: %s", out)
	}
	pub := agePublicKey(t, keyFile)
	priv := agePrivateKey(t, keyFile)

	enc := filepath.Join(dir, "secrets.enc.yaml")
	write(t, enc, "cloudflare:\n    api_token: composed\n")
	if out, err := exec.Command("sops", "encrypt", "--age", pub,
		"--in-place", enc).CombinedOutput(); err != nil {
		t.Fatalf("sops encrypt: %s", out)
	}

	// The age key is declared like any other secret. In a real repository it
	// would be `provider = "keychain"`; here it comes from the environment so
	// the test touches no keychain.
	write(t, filepath.Join(dir, ConfigName), `
[providers.shell]
type = "env"

[providers.sops]
type = "sops"
file = "secrets.enc.yaml"
age_key = "SOPS_AGE_KEY"

[secrets]
CLOUDFLARE_API_TOKEN = { provider = "sops", value = "cloudflare.api_token" }
SOPS_AGE_KEY = { provider = "shell", value = "TEST_AGE_KEY" }
`)
	t.Setenv("TEST_AGE_KEY", priv)
	os.Unsetenv("SOPS_AGE_KEY")
	os.Unsetenv("CLOUDFLARE_API_TOKEN")
	os.Unsetenv("SOPS_AGE_KEY_FILE")

	// Declared first, so this also proves the order does not matter: the sops
	// secret is resolved before the key it needs.
	applied, skipped, err := Apply(dir, "")
	if err != nil {
		t.Fatal(err)
	}
	if os.Getenv("CLOUDFLARE_API_TOKEN") != "composed" {
		t.Errorf("applied %v, skipped %v", applied, skipped)
	}
}

// TestSopsKeyCannotLiveInTheFileItUnlocks — a cycle that would otherwise
// recurse until the stack ran out.
func TestSopsKeyCannotLiveInTheFileItUnlocks(t *testing.T) {
	cfg, err := parse(`
[providers.sops]
type = "sops"
file = "secrets.enc.yaml"
age_key = "SOPS_AGE_KEY"

[secrets]
SOPS_AGE_KEY = { provider = "sops", value = "age.key" }
`)
	if err != nil {
		t.Fatal(err)
	}
	os.Unsetenv("SOPS_AGE_KEY")
	if _, err := cfg.resolveNamed("SOPS_AGE_KEY"); err == nil {
		t.Fatal("expected a refusal, not a cycle")
	}
}

func agePrivateKey(t *testing.T, keyFile string) string {
	t.Helper()
	data, err := os.ReadFile(keyFile)
	if err != nil {
		t.Fatal(err)
	}
	for _, line := range strings.Split(string(data), "\n") {
		if strings.HasPrefix(line, "AGE-SECRET-KEY-") {
			return strings.TrimSpace(line)
		}
	}
	t.Fatal("no private key in the generated file")
	return ""
}

// TestUsesOnlyReportsDeclaredProviders — the CLI installs a tool when the
// config asks for it, so a keychain-only project must not report sops.
func TestUsesOnlyReportsDeclaredProviders(t *testing.T) {
	cfg, err := parse(`
[providers.keychain]
type = "keychain"

[secrets]
TOKEN = { provider = "keychain", value = "TOKEN" }
`)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Uses("sops") {
		t.Error("a keychain-only project should not pull in sops")
	}
	if !cfg.Uses("keychain") {
		t.Error("it does use keychain")
	}
}

// Environments are the same names against different backing services. The
// overlay has to be a genuine overlay: a name declared once at the top has to
// keep working in every environment, or every environment ends up restating
// the whole file and they drift.
func TestEnvironmentOverlay(t *testing.T) {
	cfg, err := parse(`
[providers.demo]
type = "literal"

[secrets]
SHARED   = { provider = "demo", value = "same-everywhere" }
DATABASE = { provider = "demo", value = "prod-db" }

[env.staging.secrets]
DATABASE     = { provider = "demo", value = "staging-db" }
STAGING_ONLY = { provider = "demo", value = "extra" }
`)
	if err != nil {
		t.Fatal(err)
	}

	if got := cfg.EnvNames; len(got) != 1 || got[0] != "staging" {
		t.Fatalf("EnvNames = %v, want [staging]", got)
	}
	if !cfg.HasEnv("staging") || cfg.HasEnv("production") {
		t.Error("HasEnv does not match what is declared")
	}

	base := valuesOf(cfg)
	if base["DATABASE"] != "prod-db" {
		t.Errorf("base DATABASE = %q, want prod-db", base["DATABASE"])
	}
	if _, ok := base["STAGING_ONLY"]; ok {
		t.Error("base config carries a staging-only secret")
	}

	staging := valuesOf(cfg.For("staging"))
	if staging["DATABASE"] != "staging-db" {
		t.Errorf("staging DATABASE = %q, want staging-db — the overlay did not win",
			staging["DATABASE"])
	}
	if staging["SHARED"] != "same-everywhere" {
		t.Errorf("staging SHARED = %q — a name declared once must survive the overlay",
			staging["SHARED"])
	}
	if staging["STAGING_ONLY"] != "extra" {
		t.Error("a secret only one environment declares was dropped")
	}

	// The base config must not be mutated by asking for an environment: the
	// same *Config is reused across commands in one process.
	if again := valuesOf(cfg); again["DATABASE"] != "prod-db" {
		t.Errorf("For() mutated the base config: DATABASE = %q", again["DATABASE"])
	}
}

// An unknown environment has to be an error. Silently falling back to the
// defaults means a typo in --env pushes production values to staging.
func TestUnknownEnvironmentIsAnError(t *testing.T) {
	dir := t.TempDir()
	write(t, filepath.Join(dir, ConfigName), `
[providers.demo]
type = "literal"

[secrets]
A = { provider = "demo", value = "1" }
`)
	if _, _, err := Apply(dir, "nope"); err == nil {
		t.Fatal("Apply accepted an environment the file does not declare")
	}
}

func valuesOf(cfg *Config) map[string]string {
	out := map[string]string{}
	for _, s := range cfg.Secrets {
		out[s.Name] = s.Value
	}
	return out
}
