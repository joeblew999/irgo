package main

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

// The registry is the single declaration of every setting. These tests are
// what make that true: without them "single declaration" is a comment, and the
// four switch statements this replaced each drifted precisely because nothing
// checked they agreed.

// TestEverySettingRoundTrips writes each setting into a config file and reads
// it back through the real resolver.
//
// This is the bug that kept happening: a setting was added to the parser and
// missed in the reader, or the other way round, and it silently resolved to
// empty. Nothing failed to compile, and packaging just behaved as though the
// value had never been set.
func TestEverySettingRoundTrips(t *testing.T) {
	for _, cv := range configRegistry {
		if cv.tomlSection == "" {
			continue // environment-only, e.g. a deploy token
		}
		t.Run(cv.tomlSection+"."+cv.tomlKey, func(t *testing.T) {
			want := "roundtrip-" + cv.tomlKey
			if cv.boolField != nil {
				want = "true"
			}
			inConfigDir(t, "["+cv.tomlSection+"]\n"+cv.tomlKey+" = \""+want+"\"\n")

			if got := valueFromConfig(cv); got != want {
				t.Errorf("wrote %s.%s = %q, read back %q\n"+
					"the setting is declared but does not survive the file — "+
					"check its field accessor in configRegistry",
					cv.tomlSection, cv.tomlKey, want, got)
			}
		})
	}
}

// TestEverySettingIsReachable checks the parts of an entry other code depends
// on being there.
func TestEverySettingIsReachable(t *testing.T) {
	for _, cv := range configRegistry {
		name := cv.tomlSection + "." + cv.tomlKey
		if cv.tomlSection == "" {
			name = cv.env
		}

		if cv.env == "" {
			t.Errorf("%s: no env var, so it can never be supplied by CI", name)
		}
		if len(cv.targets) == 0 {
			t.Errorf("%s: belongs to no target, so no command will ever ask for it", name)
		}
		if cv.display == "" {
			t.Errorf("%s: no display name, so it renders blank in setup and doctor", name)
		}
		if cv.tomlSection != "" && cv.field == nil && cv.boolField == nil {
			t.Errorf("%s: names a toml key with nowhere to put the value", name)
		}
		if cv.tomlSection == "" && (cv.field != nil || cv.boolField != nil) {
			t.Errorf("%s: has a struct field but no toml key, so nothing can write it", name)
		}
	}
}

// TestEveryTargetIsListed keeps configStores honest. It decides display order
// and what `secrets list` groups by, so a target that gains a setting and is
// missed here is invisible rather than merely unsorted.
func TestEveryTargetIsListed(t *testing.T) {
	for _, cv := range configRegistry {
		for _, target := range cv.targets {
			if !slices.Contains(configStores, target) {
				t.Errorf("%s.%s targets %q, which is not in configStores",
					cv.tomlSection, cv.tomlKey, target)
			}
		}
	}
}

// TestEnvVarsAreUnique guards the lookup classify and push depend on. Two
// settings sharing an env var means one of them silently wins.
func TestEnvVarsAreUnique(t *testing.T) {
	seen := map[string]string{}
	for _, cv := range configRegistry {
		name := cv.tomlSection + "." + cv.tomlKey
		if prev, dup := seen[cv.env]; dup {
			t.Errorf("%s and %s both use %s", prev, name, cv.env)
		}
		seen[cv.env] = name
	}
}

// TestSecretsAreClassified checks that every private value says what it is
// for. The role decides whether `push cloudflare` will send it, so an
// unclassified secret defaults to being treated as an app runtime value — and
// a signing key would be copied into the Worker.
func TestSecretsAreClassified(t *testing.T) {
	for _, cv := range configRegistry {
		if !cv.sensitive() {
			continue
		}
		if cv.role == roleRuntime {
			t.Errorf("%s is secret but unclassified, so push would send it to the Worker",
				cv.env)
		}
		if role, _ := classify(cv.env); role != cv.role {
			t.Errorf("%s: registry says %v, classify says %v", cv.env, cv.role, role)
		}
	}
}

// TestWorkerRejectsCredentialsItCannotUse is the rule that matters most: a
// Worker holding the token that redeploys it turns one compromised Worker into
// the whole account.
func TestWorkerRejectsCredentialsItCannotUse(t *testing.T) {
	for _, cv := range configRegistry {
		if !cv.sensitive() {
			continue
		}
		if withholdReason("cloudflare", cv.env) == "" {
			t.Errorf("%s would be pushed into the Worker runtime", cv.env)
		}
		if withholdReason("github", cv.env) != "" {
			t.Errorf("%s withheld from CI, which has to build and deploy", cv.env)
		}
	}
	// An app's own secret is the one thing a Worker legitimately needs.
	if why := withholdReason("cloudflare", "DATABASE_URL"); why != "" {
		t.Errorf("app runtime secret withheld from the Worker: %s", why)
	}
}

// inConfigDir runs the test in a temporary directory holding one config file,
// since the config functions read from the working directory.
func inConfigDir(t *testing.T, content string) {
	t.Helper()
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, packageConfigFile), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	prev, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.Chdir(prev) })

	// A stray IRGO_* in the developer's shell would otherwise decide the
	// result, since the environment outranks the file.
	for _, cv := range configRegistry {
		if cv.env != "" && os.Getenv(cv.env) != "" && strings.HasPrefix(cv.env, "IRGO_") {
			t.Setenv(cv.env, "")
		}
	}
}

// TestDerivedValuesAreValid is the test that would have caught the Android
// package name: irgo derived com.irgo.irgo-demo from the module path and
// reported it as configured, because nothing checked derived values against
// the rules of the thing consuming them.
//
// Anything irgo works out for itself has to satisfy the same pattern a human
// would be held to.
func TestDerivedValuesAreValid(t *testing.T) {
	modules := []string{
		"github.com/joeblew999/irgo-demo", // the hyphen that started this
		"github.com/example/myapp",
		"example.com/2fast",  // a segment that cannot start a Java identifier
		"myapp",              // bare module name
		"github.com/a/b-c-d", // several hyphens
	}
	cv := *registryFor("reviews", "android_package")
	for _, mp := range modules {
		got := androidPackageFromModulePath(mp)
		if err := validateConfigValue(cv, got); err != nil {
			t.Errorf("module %q derived %q, which irgo would then reject: %v",
				mp, got, err)
		}
	}
}

// TestValidationCatchesTypos checks the patterns actually discriminate. A
// pattern that accepts everything is worse than none: it reads as a check.
func TestValidationCatchesTypos(t *testing.T) {
	cases := []struct {
		section, key string
		good, bad    string
	}{
		{"ios", "team", "9Z237BG9S9", "TOOSHORT"},
		{"reviews", "ios_key_id", "ABCD123456", "abc"},
		{"reviews", "ios_issuer_id", "69a6de70-1f2c-47e3-e053-5b8c7c11a4d1", "not-a-uuid"},
		{"reviews", "ios_app_id", "1234567890", "id1234567890"},
		{"reviews", "android_package", "com.example.myapp", "com.example.my-app"},
		{"windows", "publisher", "CN=Example Ltd", "Example Ltd"},
		{"ios", "export_method", "app-store", "appstore"},
	}
	for _, c := range cases {
		cv := registryFor(c.section, c.key)
		if cv == nil {
			t.Errorf("%s.%s is not in the registry", c.section, c.key)
			continue
		}
		if err := validateConfigValue(*cv, c.good); err != nil {
			t.Errorf("%s.%s rejected a valid value %q: %v", c.section, c.key, c.good, err)
		}
		if err := validateConfigValue(*cv, c.bad); err == nil {
			t.Errorf("%s.%s accepted %q", c.section, c.key, c.bad)
		}
	}
}

// An unset value is not an invalid one — that distinction is what `required`
// carries, and conflating them makes every optional setting an error.
func TestValidationIgnoresEmpty(t *testing.T) {
	for _, cv := range configRegistry {
		if err := validateConfigValue(cv, ""); err != nil {
			t.Errorf("%s reported empty as invalid: %v", cv.env, err)
		}
	}
}

// TestSecretsNeverReachTheCommittedFile is the contradiction this config had
// at its centre: irgo.package.toml is committed, and the wizard wrote every
// answer into it — including the app-specific password it had just told you
// was going somewhere gitignored.
//
// A test rather than care, because the write path is one function away from
// every prompt, and the next setting added will not think about it.
func TestSecretsNeverReachTheCommittedFile(t *testing.T) {
	for _, cv := range configRegistry {
		if cv.tomlSection == "" {
			continue
		}
		t.Run(cv.tomlSection+"."+cv.tomlKey, func(t *testing.T) {
			dir := t.TempDir()
			prev, _ := os.Getwd()
			if err := os.Chdir(dir); err != nil {
				t.Fatal(err)
			}
			t.Cleanup(func() { os.Chdir(prev) })

			const canary = "S3CRET-CANARY"
			if err := writeConfigValueFor(cv, canary); err != nil {
				t.Fatal(err)
			}

			committed, _ := os.ReadFile(packageConfigFile)
			local, _ := os.ReadFile(packageLocalFile)

			if cv.secret {
				if strings.Contains(string(committed), canary) {
					t.Errorf("%s.%s is secret and was written to %s, which is committed",
						cv.tomlSection, cv.tomlKey, packageConfigFile)
				}
				if !strings.Contains(string(local), canary) {
					t.Errorf("%s.%s is secret but did not reach %s",
						cv.tomlSection, cv.tomlKey, packageLocalFile)
				}
				if info, err := os.Stat(packageLocalFile); err == nil {
					if perm := info.Mode().Perm(); perm != 0o600 {
						t.Errorf("%s is %o, readable by others", packageLocalFile, perm)
					}
				}
				return
			}
			if !strings.Contains(string(committed), canary) {
				t.Errorf("%s.%s is not secret but did not reach %s",
					cv.tomlSection, cv.tomlKey, packageConfigFile)
			}
		})
	}
}

// TestTheLocalOverlayIsGitignored checks the assumption the whole split rests
// on. A gitignored file that is not actually gitignored is worse than no
// split at all, because everything downstream trusts it.
func TestTheLocalOverlayIsGitignored(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("templates", ".gitignore"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), packageLocalFile) {
		t.Errorf("scaffolded .gitignore does not list %s, so every secret the "+
			"wizard writes would be committed", packageLocalFile)
	}
}

// TestPathsAreTreatedAsCredentials. A path is not a secret — the name belongs
// in the committed config — but the file it names is, and in CI it travels as
// contents. Getting that wrong sent the Android signing keystore, which cannot
// be reissued, into the Worker runtime.
func TestPathsAreTreatedAsCredentials(t *testing.T) {
	for _, cv := range configRegistry {
		if !cv.path {
			continue
		}
		if !cv.sensitive() {
			t.Errorf("%s names a file but is not treated as sensitive", cv.env)
		}
		if cv.role == roleRuntime {
			t.Errorf("%s names a credential file but is classified as app runtime", cv.env)
		}
		if withholdReason("cloudflare", cv.env) == "" {
			t.Errorf("%s: the file it names would be pushed into the Worker", cv.env)
		}
		if cv.secret {
			t.Errorf("%s is a path; marking it secret hides the filename for no gain "+
				"and keeps it out of the committed config where it belongs", cv.env)
		}
	}
}
