// `irgo secrets` — what this project needs, and getting it where it is needed.
//
// The secrets live in one place: the keychain, declared per repo in fnox.toml.
// Everything else is a copy that has to be pushed there and kept in step —
// GitHub Actions cannot read a laptop's keychain, and neither can a Cloudflare
// Worker at runtime.
//
// Pushing was the missing half. Reading fnox got a deploy working from a
// laptop; CI still needed someone to remember `gh secret set` for each name in
// each repo, and nothing said when they had drifted.
//
// Values are never printed and never passed as arguments — both `gh secret
// set` and `wrangler secret put` read from stdin, so a secret does not reach
// the shell history, the process list, or this program's output.
package main

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	"github.com/stukennedy/irgo/pkg/secrets"
)

func runSecrets(args []string) error {
	env := envFlag(args)
	switch verb(args) {
	case "list", "":
		return secretsList(env)
	case "status":
		return secretsStatus(env)
	case "push":
		return secretsPush(args, env)
	}
	return fmt.Errorf("unknown: irgo secrets %s", verb(args))
}

// envFlag reads --env, falling back to IRGO_ENV.
//
// The variable matters more than the flag: a CI job sets it once for the whole
// workflow, and every irgo command in that job then resolves the right values
// without each step remembering to pass it.
func envFlag(args []string) string {
	for i, a := range args {
		if v, ok := strings.CutPrefix(a, "--env="); ok {
			return v
		}
		if a == "--env" && i+1 < len(args) {
			return args[i+1]
		}
	}
	return os.Getenv("IRGO_ENV")
}

// verb is the first non-flag argument.
func verb(args []string) string {
	for _, a := range args {
		if !strings.HasPrefix(a, "-") {
			return a
		}
	}
	return ""
}

// secretsList reports what this project declares and whether each resolves.
//
// Names and status only. A command that prints secret values is a command
// someone will run in a shared terminal.
func secretsList(env string) error {
	cfg, ok, err := secrets.Load(".")
	if err != nil {
		return err
	}
	if !ok {
		fmt.Printf("No %s here — this project declares no secrets.\n", secrets.ConfigName)
		return nil
	}

	if env != "" && !cfg.HasEnv(env) {
		return fmt.Errorf("%s declares no environment %q%s", cfg.Path, env,
			envHint(cfg))
	}
	_, skipped, err := secrets.Apply(".", env)
	if err != nil {
		return err
	}
	cfg = cfg.For(env)

	w := 0
	for _, sec := range cfg.Secrets {
		if len(sec.Name) > w {
			w = len(sec.Name)
		}
	}

	// Grouped by what each is for. A flat list reads as though an Apple
	// signing key and a database URL are the same kind of thing, and the
	// whole point of the grouping is that they go to different places.
	type row struct{ name, status string }
	groups := map[string][]row{}
	var order []string
	for _, sec := range cfg.Secrets {
		role, targets := classify(sec.Name)
		head := role.String()
		if len(targets) > 0 {
			head += " — " + strings.Join(targets, ", ")
		}
		if _, seen := groups[head]; !seen {
			order = append(order, head)
		}
		status := "ok        from " + sec.Provider
		if skipped[sec.Name] != nil {
			status = fmt.Sprintf("missing   %v", skipped[sec.Name])
		}
		groups[head] = append(groups[head], row{sec.Name, status})
	}

	head := "Declared in " + cfg.Path
	if env != "" {
		head += ", environment " + env
	}
	fmt.Println(head + ":")
	for _, head := range order {
		fmt.Printf("\n  %s\n", head)
		for _, r := range groups[head] {
			fmt.Printf("    %-*s  %s\n", w, r.name, r.status)
		}
	}
	printKnownSecrets(cfg)

	if len(cfg.EnvNames) > 0 {
		fmt.Println()
		fmt.Printf("Environments declared here: %s\n", strings.Join(cfg.EnvNames, ", "))
		fmt.Println("  irgo secrets list --env " + cfg.EnvNames[0])
	}

	fmt.Println()
	fmt.Println("Push them where they are also needed:")
	fmt.Println("  irgo secrets push github      # GitHub Actions")
	fmt.Println("  irgo secrets push cloudflare  # Worker runtime")
	if len(cfg.EnvNames) > 0 {
		fmt.Println("  ... --env " + cfg.EnvNames[0] + "     # that environment's values")
	}
	return nil
}

// envHint names what is actually declared, since a typo here is otherwise
// indistinguishable from an environment nobody has set up yet.
func envHint(cfg *secrets.Config) string {
	if len(cfg.EnvNames) == 0 {
		return " (none are)"
	}
	return " — declared: " + strings.Join(cfg.EnvNames, ", ")
}

// ---------------------------------------------------------------------------
// What a secret is FOR, which decides where it may be copied.
//
// fnox.toml is a flat list of names, but the things it names are not alike. An
// Apple signing key, a Cloudflare API token and the app's own database URL want
// to end up in three different places, and copying any of them to the wrong one
// is a real mistake — `push cloudflare` used to put the token that can redeploy
// the Worker inside that same Worker.
//
// Rather than a second list saying which is which, this joins against the store
// registry irgo already has: storeConfigValues knows every IRGO_* value, which
// target needs it, and whether it is private. A declared secret that matches one
// is that target's signing credential. Everything else is the app's own.
// ---------------------------------------------------------------------------

type secretRole int

const (
	roleRuntime secretRole = iota // the app needs it while running
	roleDeploy                    // it performs a deploy
	roleSigning                   // it signs a build for a store
)

// classify says what a declared secret is for, and which targets need it.
//
// A lookup in the registry, not a guess: irgo already knows every IRGO_* and
// deploy credential, which target needs it and what it is for. This used to
// match on a CLOUDFLARE_ name prefix, which is a heuristic standing in for a
// fact already on hand — and one that quietly misfiled anything named
// differently.
//
// A name the registry does not carry is the app's own runtime secret. irgo
// cannot know a project's database URL, and should not pretend otherwise.
func classify(name string) (secretRole, []string) {
	if cv := registryForEnv(name); cv != nil && cv.sensitive() {
		return cv.role, cv.targets
	}
	return roleRuntime, nil
}

func (r secretRole) String() string {
	switch r {
	case roleDeploy:
		return "deploy credential"
	case roleSigning:
		return "signing credential"
	}
	return "app runtime"
}

// printKnownSecrets reports the credentials irgo knows about that this project
// has not declared, grouped by the target that needs them.
//
// The registry already carries which values are private, what each is for and
// which target needs it, because `app package setup` has to know not to echo a
// password back. Nobody should have to copy that into fnox.toml by hand and
// keep the two in step.
//
// Only what is missing is shown. A project that ships to one store should not
// have to read about the other three.
func printKnownSecrets(cfg *secrets.Config) {
	declared := map[string]bool{}
	for _, s := range cfg.Secrets {
		declared[s.Name] = true
	}

	byTarget := map[string][]configValue{}
	for _, cv := range configRegistry {
		if !cv.sensitive() || cv.env == "" || declared[cv.env] {
			continue
		}
		if os.Getenv(cv.env) != "" {
			continue // supplied some other way, and irgo is not the boss of that
		}
		for _, t := range cv.targets {
			byTarget[t] = append(byTarget[t], cv)
		}
	}
	if len(byTarget) == 0 {
		return
	}

	var first string
	fmt.Println()
	fmt.Println("Credentials irgo knows about, not declared here:")
	for _, target := range configStores {
		vals := byTarget[target]
		if len(vals) == 0 {
			continue
		}
		fmt.Printf("\n  %s\n", target)
		for _, cv := range vals {
			fmt.Printf("    %-28s %s\n", cv.env, cv.display)
			if first == "" {
				first = cv.env
			}
		}
	}
	fmt.Println()
	fmt.Println("Only needed if this project ships to that target. To use one, put it in")
	fmt.Printf("the keychain and add a line to %s:\n", secrets.ConfigName)
	fmt.Printf("  fnox set -p keychain %s\n", first)
	fmt.Printf("  %s = { provider = \"keychain\", value = \"%s\" }\n", first, first)
}

func secretsPush(args []string, env string) error {
	target := ""
	for _, a := range args[1:] {
		if !strings.HasPrefix(a, "-") && a != "push" {
			target = a
			break
		}
	}
	switch target {
	case "github", "cloudflare":
	default:
		return fmt.Errorf("push where? irgo secrets push github|cloudflare")
	}

	cfg, ok, err := secrets.Load(".")
	if err != nil || !ok {
		return fmt.Errorf("no %s here, so there is nothing to push", secrets.ConfigName)
	}
	if _, _, err := secrets.Apply(".", env); err != nil {
		return err
	}

	// Resolve everything before sending anything: a push that fails halfway
	// leaves a destination holding some of the new values and some of the old,
	// which is worse than not starting.
	values := map[string]string{}
	var missing, withheld []string
	for _, s := range cfg.Secrets {
		if why := withholdReason(target, s.Name); why != "" {
			withheld = append(withheld, fmt.Sprintf("  %-28s %s", s.Name, why))
			continue
		}
		v := os.Getenv(s.Name)
		if v == "" {
			missing = append(missing, s.Name)
			continue
		}
		// A value that is set but malformed should not be copied anywhere.
		// Pushing it propagates the typo to every destination, and each one
		// discovers it separately, late, in its own confusing way.
		if cv := registryForEnv(s.Name); cv != nil {
			if err := validateConfigValue(*cv, v); err != nil {
				return fmt.Errorf("%s: %w\n  fix it before pushing it everywhere", s.Name, err)
			}
		}
		// A file-valued credential is a path here and has to arrive at the
		// other end as contents, under the name the runner will look for.
		if cv := registryForEnv(s.Name); cv != nil && cv.path {
			raw, err := os.ReadFile(expandHome(v))
			if err != nil {
				return fmt.Errorf("%s names a file that cannot be read: %w", s.Name, err)
			}
			values[s.Name+base64Suffix] = base64.StdEncoding.EncodeToString(raw)
			continue
		}
		values[s.Name] = v
	}
	if len(missing) > 0 {
		sort.Strings(missing)
		return fmt.Errorf("these do not resolve locally, so they cannot be pushed: %s\n"+
			"  set them first:  fnox set -p keychain <NAME>", strings.Join(missing, ", "))
	}

	if len(withheld) > 0 {
		sort.Strings(withheld)
		fmt.Println("Not pushed, because this destination must not hold them:")
		for _, line := range withheld {
			fmt.Println(line)
		}
		fmt.Println()
	}
	if len(values) == 0 {
		return fmt.Errorf("nothing here belongs on %s", target)
	}

	names := make([]string, 0, len(values))
	for n := range values {
		names = append(names, n)
	}
	sort.Strings(names)

	where, err := pushTargetDescription(target, env)
	if err != nil {
		return err
	}
	fmt.Printf("Push %d secret(s) to %s:\n\n", len(names), where)
	for _, n := range names {
		fmt.Printf("  %s\n", n)
	}
	fmt.Println()
	fmt.Println("Values are read from the keychain and piped in — never printed,")
	fmt.Println("never passed as arguments.")

	if !confirm(fmt.Sprintf("Push to %s?", where), hasFlag(args, "--yes", "-y")) {
		fmt.Println("Nothing was pushed.")
		return nil
	}

	for _, n := range names {
		if err := pushOne(target, n, values[n], env); err != nil {
			return fmt.Errorf("pushing %s: %w", n, err)
		}
		fmt.Printf("  %s: pushed\n", n)
	}
	return nil
}

// pushTargetDescription names the destination, and fails early when the tool
// for it is missing rather than after resolving secrets.
func pushTargetDescription(target, env string) (string, error) {
	switch target {
	case "github":
		if _, err := exec.LookPath("gh"); err != nil {
			return "", fmt.Errorf("gh is not installed — https://cli.github.com")
		}
		out, err := exec.Command("gh", "repo", "view", "--json", "nameWithOwner",
			"-q", ".nameWithOwner").Output()
		if err != nil {
			return "", fmt.Errorf("not a GitHub repository, or gh is not logged in: %w", err)
		}
		where := "GitHub Actions on " + strings.TrimSpace(string(out))
		if env != "" {
			where += ", environment " + env
		}
		return where, nil
	case "cloudflare":
		name := cloudflareWorkerName()
		if name == "" {
			return "", fmt.Errorf("no wrangler.toml here, so there is no Worker to push to")
		}
		if env != "" {
			return "the " + name + " Worker, environment " + env, nil
		}
		return "the " + name + " Worker", nil
	}
	return "", fmt.Errorf("unknown target %q", target)
}

// pushOne sends a single secret, on stdin.
func pushOne(target, name, value, env string) error {
	var cmd *exec.Cmd
	switch target {
	case "github":
		// A GitHub environment is a real scope with its own protection
		// rules, which is the point: staging values should not be reachable
		// from a job that deploys production.
		gh := []string{"secret", "set", name}
		if env != "" {
			gh = append(gh, "--env", env)
		}
		cmd = exec.Command("gh", gh...)
	case "cloudflare":
		node, err := nodeBin(true)
		if err != nil {
			return err
		}
		wrangler := []string{"--yes", "wrangler@4", "secret", "put", name}
		if env != "" {
			wrangler = append(wrangler, "--env", env)
		}
		cmd = npxCommand(node, wrangler...)
	}
	cmd.Stdin = strings.NewReader(value)
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// currentEnv is the deployment environment this invocation is for, set once at
// dispatch so every command resolves the same values.
var currentEnv string

// applySecrets resolves whatever the repository's fnox.toml declares, into the
// environment, before any command reads it.
//
// This runs for every command, and that is the point. Store credentials are
// resolved by `app package` through IRGO_* environment variables; secrets are
// resolved from the keychain into those same variables. If only the deploy
// path applied them, then declaring IRGO_APPLE_PASSWORD in fnox.toml would
// satisfy `irgo secrets list` and be invisible to `irgo app package macos` —
// two commands disagreeing about whether the same credential exists. It did
// exactly that until this moved here.
//
// So: config says which values a build needs and where a human finds them;
// secrets say where the machine gets the private ones. They meet in the
// environment, and everything downstream reads one chain.
//
// Names already in the environment are left alone, so CI, which supplies them
// as repository secrets, behaves exactly as before. Nothing here prints a
// value; a secret that will not resolve is named, with the reason.
func applySecrets() {
	// Install what the config actually asks for, when it asks for it — the
	// same lazily-provisioned deal as templ and Tailwind. A project whose
	// secrets are in a keychain never sees sops mentioned.
	if cfg, ok, err := secrets.Load("."); err == nil && ok && cfg.Uses("sops") {
		if err := ensureGoTool("sops"); err != nil {
			fmt.Printf("Note: %v\n", err)
		}
	}

	applied, skipped, err := secrets.Apply(".", currentEnv)
	if err != nil {
		fmt.Printf("Note: %v\n", err)
		return
	}
	if len(applied) > 0 {
		fmt.Printf("Resolved from %s: %s\n", secrets.ConfigName, strings.Join(applied, ", "))
	}
	for name, why := range skipped {
		fmt.Printf("Note: %s — %v\n", name, why)
	}

	// After the environment is populated, since a file-valued credential can
	// arrive either as a path from the keychain or as contents from CI.
	materialiseFileSecrets()
}

// withholdReason says why a secret must not go to a destination, empty when it
// may.
//
// GitHub Actions builds and deploys, so it legitimately needs everything. A
// Cloudflare Worker is only the running app: giving it the token that can
// redeploy it, or an Apple signing key it can never use, widens what a single
// compromised Worker is worth for no gain.
func withholdReason(target, name string) string {
	if target != "cloudflare" {
		return ""
	}
	switch role, targets := classify(name); role {
	case roleDeploy:
		return "deploy credential — CI needs it, the Worker must not hold it"
	case roleSigning:
		return "signs " + strings.Join(targets, "/") + " builds — nothing a Worker can use"
	}
	return ""
}

// materialiseFileSecrets turns <ENV>_BASE64 into a real file and points <ENV>
// at it.
//
// Four of the credentials irgo needs are files, not strings: the Android
// keystore, the Windows PFX, the App Store Connect .p8 and the Play service
// account JSON. On a laptop the setting is a path and that is the end of it.
// CI has no filesystem to point at, so the file has to travel as its contents,
// and something has to write it back out.
//
// That something used to be the workflow, in shell, for the keystore only —
// which is why the other three had no way into CI at all. Doing it here means
// every file-valued credential works the same way, and a workflow is only ever
// a mapping of names.
//
// The decoded file is 0600 in the process's own temp dir. On a runner that
// disappears with the runner; locally this only triggers if you set a _BASE64
// variable yourself.
func materialiseFileSecrets() {
	for _, cv := range configRegistry {
		if !cv.path || cv.env == "" || os.Getenv(cv.env) != "" {
			continue
		}
		encoded := os.Getenv(cv.env + base64Suffix)
		if encoded == "" {
			continue
		}
		raw, err := base64.StdEncoding.DecodeString(strings.TrimSpace(encoded))
		if err != nil {
			fmt.Printf("Note: %s%s is not valid base64 — %v\n", cv.env, base64Suffix, err)
			continue
		}
		f, err := os.CreateTemp("", "irgo-*"+filepath.Ext(cv.display))
		if err != nil {
			fmt.Printf("Note: %s: %v\n", cv.env, err)
			continue
		}
		if _, err := f.Write(raw); err != nil {
			f.Close()
			fmt.Printf("Note: %s: %v\n", cv.env, err)
			continue
		}
		f.Close()
		if err := os.Chmod(f.Name(), 0o600); err != nil {
			fmt.Printf("Note: %s: %v\n", cv.env, err)
			continue
		}
		os.Setenv(cv.env, f.Name())
	}
}

// base64Suffix is the convention for shipping a file-valued credential through
// an environment that can only carry strings.
const base64Suffix = "_BASE64"

// ---------------------------------------------------------------------------
// Drift.
//
// Pushing is a copy, and a copy goes stale. A secret rotated in the keychain
// but not pushed, a secret removed from fnox.toml but still sitting in the
// repository, a repository that was never pushed to at all — none of these are
// visible from either end, and all of them fail at the worst moment: a release
// build, on a tag, in front of everyone.
//
// What can honestly be reported is names, not values. Both destinations are
// write-only by design: GitHub will say a secret exists and when it was last
// updated, never what it is, and neither will Cloudflare. So this reports
// presence and age, and says plainly that agreement of names is not agreement
// of values.
// ---------------------------------------------------------------------------

func secretsStatus(env string) error {
	cfg, ok, err := secrets.Load(".")
	if err != nil {
		return err
	}
	if !ok {
		fmt.Printf("No %s here — this project declares no secrets.\n", secrets.ConfigName)
		return nil
	}
	if env != "" && !cfg.HasEnv(env) {
		return fmt.Errorf("%s declares no environment %q%s", cfg.Path, env, envHint(cfg))
	}
	cfg = cfg.For(env)

	for _, target := range []string{"github", "cloudflare"} {
		remote, err := remoteSecretNames(target, env)
		if err != nil {
			fmt.Printf("── %s ──\n  %v\n\n", target, err)
			continue
		}

		fmt.Printf("── %s ──\n", target)
		declared := map[string]bool{}
		var pushed, notPushed []string
		for _, s := range cfg.Secrets {
			if withholdReason(target, s.Name) != "" {
				continue // not this destination's business
			}
			name := s.Name
			if cv := registryForEnv(s.Name); cv != nil && cv.path {
				name += base64Suffix
			}
			declared[name] = true
			if when, there := remote[name]; there {
				pushed = append(pushed, fmt.Sprintf("  ok       %-30s set %s", name, when))
			} else {
				notPushed = append(notPushed, fmt.Sprintf("  MISSING  %-30s never pushed", name))
			}
		}

		// A name at the destination that nothing declares any more. It cannot
		// be read, so nobody can tell whether it still matters — which is
		// exactly why it should be named rather than left to accumulate.
		var orphans []string
		for name, when := range remote {
			if !declared[name] && looksLikeIrgoSecret(name) {
				orphans = append(orphans, fmt.Sprintf("  ORPHAN   %-30s set %s, declared nowhere", name, when))
			}
		}

		sort.Strings(pushed)
		sort.Strings(notPushed)
		sort.Strings(orphans)
		for _, line := range append(append(notPushed, orphans...), pushed...) {
			fmt.Println(line)
		}
		if len(pushed)+len(notPushed)+len(orphans) == 0 {
			fmt.Println("  nothing to compare")
		}
		fmt.Println()
	}

	fmt.Println("Names only. Both destinations are write-only, so a secret being present")
	fmt.Println("is not the same as it being current — if you have rotated one, push it.")
	return nil
}

// looksLikeIrgoSecret keeps the orphan report to names irgo could plausibly
// own. A repository has secrets for other things, and calling those orphans
// would train everyone to ignore the report.
func looksLikeIrgoSecret(name string) bool {
	return strings.HasPrefix(name, "IRGO_") || strings.HasPrefix(name, "CLOUDFLARE_")
}

// remoteSecretNames asks a destination what it holds, and when each was set.
func remoteSecretNames(target, env string) (map[string]string, error) {
	switch target {
	case "github":
		if _, err := exec.LookPath("gh"); err != nil {
			return nil, fmt.Errorf("gh is not installed, so this cannot be checked")
		}
		args := []string{"secret", "list", "--json", "name,updatedAt"}
		if env != "" {
			args = append(args, "--env", env)
		}
		out, err := exec.Command("gh", args...).Output()
		if err != nil {
			return nil, fmt.Errorf("not a GitHub repository, or gh is not logged in")
		}
		var rows []struct{ Name, UpdatedAt string }
		if err := json.Unmarshal(out, &rows); err != nil {
			return nil, err
		}
		m := map[string]string{}
		for _, r := range rows {
			when := r.UpdatedAt
			if len(when) >= 10 {
				when = when[:10]
			}
			m[r.Name] = when
		}
		return m, nil

	case "cloudflare":
		if cloudflareWorkerName() == "" {
			return nil, fmt.Errorf("no wrangler.toml here, so there is no Worker to check")
		}
		node, err := nodeBin(false)
		if err != nil {
			return nil, fmt.Errorf("node is not available, so this cannot be checked")
		}
		args := []string{"--yes", "wrangler@4", "secret", "list"}
		if env != "" {
			args = append(args, "--env", env)
		}
		out, err := npxCommand(node, args...).Output()
		if err != nil {
			return nil, fmt.Errorf("wrangler could not list secrets — is CLOUDFLARE_API_TOKEN set?")
		}
		// wrangler prints a JSON array, sometimes after a banner.
		start := strings.Index(string(out), "[")
		if start < 0 {
			return map[string]string{}, nil
		}
		var rows []struct{ Name string }
		if err := json.Unmarshal(out[start:], &rows); err != nil {
			return nil, err
		}
		m := map[string]string{}
		for _, r := range rows {
			m[r.Name] = "—"
		}
		return m, nil
	}
	return nil, fmt.Errorf("unknown target %q", target)
}
