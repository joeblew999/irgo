// Package secrets resolves the secrets a command needs, without a wrapper.
//
// Deploying and signing need credentials, and the usual answer is to put
// another language's binary in front of every command: `fnox exec -- irgo app
// deploy cloudflare`. That works until someone forgets it, and forgetting it
// fails as "no API token" rather than "you did not run the wrapper".
//
// So irgo reads the contract itself. It is the same fnox.toml other repos
// already declare and the same keychain entries fnox already writes — this is
// interoperability, not a replacement. fnox is a maintained tool with leases,
// sync, a TUI and half a dozen providers; reimplementing it would be a second
// thing to own. A program only ever needs the read path, and the read path is
// this file.
//
// Two providers cover what a build needs. keychain is for a laptop: nothing
// touches the repository. sops is for CI: the secrets are encrypted in the
// repository and the runner holds one key. Both are read here; writing them
// stays with fnox and sops, which are better at it.
//
//	fnox set -p keychain CLOUDFLARE_API_TOKEN <token>
//	sops encrypt --in-place secrets.enc.yaml
//
// Values are never logged. The only thing this package will say about a secret
// is whether it resolved.
package secrets

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

// ConfigName is the file that declares which secrets a repository uses.
const ConfigName = "fnox.toml"

// Secret is one declared secret: where to get it, and what to ask for.
type Secret struct {
	Name     string
	Provider string
	Value    string
}

// Config is a parsed fnox.toml.
type Config struct {
	// Path is the file this came from, for error messages.
	Path string
	// Providers maps a provider name to its settings, e.g. keychain -> service.
	Providers map[string]map[string]string
	// Secrets are declared in file order, so failures report predictably.
	Secrets []Secret

	// Envs holds per-environment overlays, keyed by name, declared as
	// [env.staging.secrets]. Same names, different sources.
	Envs map[string][]Secret

	// EnvNames is the declared environments in file order, for listing.
	EnvNames []string
}

// For returns the config as one environment sees it: the base secrets, with
// any the environment redeclares replaced.
//
// Real deployments are the same application against different backing
// services, and the difference is almost never which secrets exist — it is
// which values they take. So an environment overlays rather than replaces, and
// a name declared once at the top keeps working everywhere.
func (c *Config) For(env string) *Config {
	if env == "" || len(c.Envs[env]) == 0 {
		return c
	}
	override := map[string]Secret{}
	for _, s := range c.Envs[env] {
		override[s.Name] = s
	}
	out := *c
	out.Secrets = nil
	seen := map[string]bool{}
	for _, s := range c.Secrets {
		if o, ok := override[s.Name]; ok {
			s = o
			seen[s.Name] = true
		}
		out.Secrets = append(out.Secrets, s)
	}
	// Secrets only one environment has — a staging-only feature flag.
	for _, s := range c.Envs[env] {
		if !seen[s.Name] {
			out.Secrets = append(out.Secrets, s)
		}
	}
	return &out
}

// HasEnv reports whether an environment is declared, so a typo in --env is an
// error rather than silently falling back to production values.
func (c *Config) HasEnv(env string) bool {
	_, ok := c.Envs[env]
	return ok
}

// Find locates the nearest fnox.toml at or above dir.
//
// Walking up matters because commands run from a subdirectory: the docs site
// is built from docs-templ, and the credentials belong to the repository.
func Find(dir string) (string, bool) {
	abs, err := filepath.Abs(dir)
	if err != nil {
		return "", false
	}
	for {
		candidate := filepath.Join(abs, ConfigName)
		if _, err := os.Stat(candidate); err == nil {
			return candidate, true
		}
		parent := filepath.Dir(abs)
		if parent == abs {
			return "", false
		}
		abs = parent
	}
}

// Load parses the nearest fnox.toml. A missing file is not an error: most
// projects have no secrets, and every command has to work without one.
func Load(dir string) (*Config, bool, error) {
	path, ok := Find(dir)
	if !ok {
		return nil, false, nil
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, false, err
	}
	cfg, err := parse(string(data))
	if err != nil {
		return nil, false, fmt.Errorf("%s: %w", path, err)
	}
	cfg.Path = path
	return cfg, true, nil
}

// Apply resolves every declared secret and puts it in the environment.
//
// An existing environment variable always wins, so CI — which supplies the
// same names from repository secrets — is unaffected, and so is anyone who
// exported one by hand for a single command.
//
// Returns the names it set. A secret that cannot be resolved is reported and
// skipped rather than fatal: a command that does not need it should still run,
// and a command that does will fail with its own, better message.
func Apply(dir, env string) (applied []string, skipped map[string]error, err error) {
	cfg, ok, err := Load(dir)
	if err != nil || !ok {
		return nil, nil, err
	}
	if env != "" && !cfg.HasEnv(env) {
		return nil, nil, fmt.Errorf("%s declares no environment %q", cfg.Path, env)
	}
	cfg = cfg.For(env)
	skipped = map[string]error{}
	for _, s := range cfg.Secrets {
		if _, present := os.LookupEnv(s.Name); present {
			continue
		}
		value, err := cfg.resolve(s)
		if err != nil {
			skipped[s.Name] = err
			continue
		}
		if value == "" {
			skipped[s.Name] = fmt.Errorf("resolved to an empty value")
			continue
		}
		if err := os.Setenv(s.Name, value); err != nil {
			skipped[s.Name] = err
			continue
		}
		applied = append(applied, s.Name)
	}
	return applied, skipped, nil
}

// Uses reports whether any declared secret needs the given provider type.
//
// Lets a caller install a tool only when the config asks for it: a project
// keeping its secrets in a keychain should never be told about sops.
func (c *Config) Uses(kind string) bool {
	for _, s := range c.Secrets {
		t := s.Provider
		if declared := c.Providers[s.Provider]["type"]; declared != "" {
			t = declared
		}
		if t == kind {
			return true
		}
	}
	return false
}

// resolve fetches one secret from its provider.
func (c *Config) resolve(s Secret) (string, error) {
	kind := s.Provider
	settings := c.Providers[s.Provider]
	if t := settings["type"]; t != "" {
		kind = t
	}

	switch kind {
	case "keychain":
		service := settings["service"]
		if service == "" {
			service = "fnox"
		}
		return keychain(service, s.Value)
	case "sops":
		file := settings["file"]
		if file == "" {
			return "", fmt.Errorf("the %q provider needs a file = \"...\"", s.Provider)
		}
		return c.sops(settings, file, s.Value)
	case "env":
		v, ok := os.LookupEnv(s.Value)
		if !ok {
			return "", fmt.Errorf("$%s is not set", s.Value)
		}
		return v, nil
	case "", "plain", "literal":
		return s.Value, nil
	default:
		return "", fmt.Errorf("provider %q is not one this reads — "+
			"use fnox for it, or declare a keychain entry", kind)
	}
}

// sops decrypts a key out of a SOPS-encrypted file.
//
// The binary rather than the library, deliberately. SOPS is Go and importing
// github.com/getsops/sops/v3/decrypt would be the obvious move, but it brings
// the AWS, GCP, Azure and Vault SDKs with it: irgo goes from 48 modules to
// 341, and every project that runs `go tool irgo` carries them. Shelling out
// costs one exec and nothing else, and `go install github.com/getsops/sops/v3`
// is how you get it.
//
// This is the provider to use when secrets must reach CI: a SOPS file is
// encrypted and committed, and CI needs one key rather than a keychain.
func (c *Config) sops(settings map[string]string, file, key string) (string, error) {
	path := file
	if !filepath.IsAbs(path) {
		path = filepath.Join(filepath.Dir(c.Path), file)
	}
	if _, err := os.Stat(path); err != nil {
		return "", fmt.Errorf("%s: no such file", file)
	}
	cmd := exec.Command("sops", "decrypt", "--extract", extractPath(key), path)
	cmd.Env = os.Environ()

	// The age key can itself be a declared secret, which is the arrangement
	// worth having: one key in the keychain, everything else encrypted in the
	// repository. Passing it explicitly rather than relying on it happening to
	// be resolved first — declaration order is not something a config file
	// should have to get right.
	if name := settings["age_key"]; name != "" {
		value, err := c.resolveNamed(name)
		if err != nil {
			return "", fmt.Errorf("the age key for %s: %w", file, err)
		}
		cmd.Env = append(cmd.Env, "SOPS_AGE_KEY="+value)
	}

	out, err := cmd.Output()
	if err != nil {
		if _, lookErr := exec.LookPath("sops"); lookErr != nil {
			return "", fmt.Errorf("sops is not installed — " +
				"go install github.com/getsops/sops/v3/cmd/sops@latest")
		}
		return "", fmt.Errorf("sops could not read %s from %s "+
			"(is the decryption key available?)", key, file)
	}
	return strings.TrimRight(string(out), "\r\n"), nil
}

// resolveNamed resolves another declared secret by name, for a provider that
// needs one — the age key a sops file is encrypted to.
//
// A sops-backed key would be a cycle: the key needed to decrypt the file
// cannot live in the file. Refused rather than recursed.
func (c *Config) resolveNamed(name string) (string, error) {
	if v, ok := os.LookupEnv(name); ok && v != "" {
		return v, nil
	}
	for _, s := range c.Secrets {
		if s.Name != name {
			continue
		}
		kind := s.Provider
		if t := c.Providers[s.Provider]["type"]; t != "" {
			kind = t
		}
		if kind == "sops" {
			return "", fmt.Errorf("%s is itself a sops secret — the key to a file cannot live in it", name)
		}
		return c.resolve(s)
	}
	return "", fmt.Errorf("%s is not declared in [secrets]", name)
}

// extractPath turns a dotted key into the index expression sops wants:
// "cloudflare.api_token" becomes ["cloudflare"]["api_token"].
func extractPath(key string) string {
	var b strings.Builder
	for _, part := range strings.Split(key, ".") {
		b.WriteString(`["` + part + `"]`)
	}
	return b.String()
}

// keychain reads a generic password from the platform's own store.
func keychain(service, account string) (string, error) {
	if runtime.GOOS != "darwin" {
		return "", fmt.Errorf("keychain lookups need macOS; set $%s in the environment instead", account)
	}
	out, err := exec.Command("security", "find-generic-password",
		"-s", service, "-a", account, "-w").Output()
	if err != nil {
		// Both forms, because irgo does not need fnox: reading is this
		// function, and fnox is only how a human writes the entry. Someone
		// without it should not be blocked on installing a Rust toolchain to
		// set one password.
		return "", fmt.Errorf("not in the %q keychain. Set it with either:\n"+
			"    fnox set -p keychain %s        (prompts, nothing in shell history)\n"+
			"    security add-generic-password -U -s %s -a %s -w",
			service, account, service, account)
	}
	return strings.TrimRight(string(out), "\r\n"), nil
}

// parse reads the subset of TOML fnox.toml uses: [providers.<name>] tables and
// a [secrets] table of inline tables.
//
// Hand-rolled for the same reason the package config parser is: this is the
// only TOML the CLI reads, the shape is fixed, and a dependency for it would
// be carried by every project that builds irgo.
func parse(src string) (*Config, error) {
	cfg := &Config{Providers: map[string]map[string]string{}, Envs: map[string][]Secret{}}
	section := ""

	for i, raw := range strings.Split(src, "\n") {
		line := strings.TrimSpace(raw)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]") {
			section = strings.TrimSpace(line[1 : len(line)-1])
			if name, ok := strings.CutPrefix(section, "providers."); ok {
				if _, exists := cfg.Providers[name]; !exists {
					cfg.Providers[name] = map[string]string{}
				}
			}
			if name, ok := strings.CutPrefix(section, "env."); ok {
				name = strings.TrimSuffix(name, ".secrets")
				if _, exists := cfg.Envs[name]; !exists {
					cfg.Envs[name] = nil
					cfg.EnvNames = append(cfg.EnvNames, name)
				}
			}
			continue
		}

		key, value, ok := strings.Cut(line, "=")
		if !ok {
			return nil, fmt.Errorf("line %d: expected key = value", i+1)
		}
		key = strings.TrimSpace(key)
		value = strings.TrimSpace(value)

		switch {
		case strings.HasPrefix(section, "providers."):
			name := strings.TrimPrefix(section, "providers.")
			cfg.Providers[name][key] = unquote(value)

		case section == "secrets":
			s, err := parseSecret(key, value)
			if err != nil {
				return nil, fmt.Errorf("line %d: %w", i+1, err)
			}
			cfg.Secrets = append(cfg.Secrets, s)

		// [env.staging.secrets]
		case strings.HasPrefix(section, "env.") && strings.HasSuffix(section, ".secrets"):
			name := strings.TrimSuffix(strings.TrimPrefix(section, "env."), ".secrets")
			s, err := parseSecret(key, value)
			if err != nil {
				return nil, fmt.Errorf("line %d: %w", i+1, err)
			}
			if _, seen := cfg.Envs[name]; !seen {
				cfg.EnvNames = append(cfg.EnvNames, name)
			}
			cfg.Envs[name] = append(cfg.Envs[name], s)
		}
	}
	return cfg, nil
}

// parseSecret reads either an inline table or a bare string.
//
//	NAME = { provider = "keychain", value = "NAME" }
//	NAME = "literal"
func parseSecret(name, value string) (Secret, error) {
	s := Secret{Name: name}
	if !strings.HasPrefix(value, "{") {
		s.Value = unquote(value)
		return s, nil
	}
	inner, ok := strings.CutSuffix(strings.TrimPrefix(value, "{"), "}")
	if !ok {
		return s, fmt.Errorf("unterminated inline table for %s", name)
	}
	for _, field := range splitFields(inner) {
		k, v, ok := strings.Cut(field, "=")
		if !ok {
			continue
		}
		switch strings.TrimSpace(k) {
		case "provider":
			s.Provider = unquote(strings.TrimSpace(v))
		case "value":
			s.Value = unquote(strings.TrimSpace(v))
		}
	}
	if s.Value == "" {
		s.Value = name // fnox defaults the lookup key to the secret's name
	}
	return s, nil
}

// splitFields splits on commas that are not inside quotes.
func splitFields(s string) []string {
	var out []string
	var cur strings.Builder
	var quote byte
	for i := 0; i < len(s); i++ {
		c := s[i]
		switch {
		case quote != 0:
			if c == quote {
				quote = 0
			}
			cur.WriteByte(c)
		case c == '"' || c == '\'':
			quote = c
			cur.WriteByte(c)
		case c == ',':
			out = append(out, cur.String())
			cur.Reset()
		default:
			cur.WriteByte(c)
		}
	}
	if strings.TrimSpace(cur.String()) != "" {
		out = append(out, cur.String())
	}
	return out
}

func unquote(s string) string {
	s = strings.TrimSpace(s)
	if len(s) >= 2 {
		if (s[0] == '"' && s[len(s)-1] == '"') || (s[0] == '\'' && s[len(s)-1] == '\'') {
			return s[1 : len(s)-1]
		}
	}
	return s
}
