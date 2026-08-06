// Package d1 gives every irgo target the same database.
//
// Cloudflare D1 is reachable two ways, and which one applies is decided by
// where the code is running rather than by anything the app says:
//
//   - Inside a Worker, D1 is a binding on the environment object. The
//     handler talks to it directly, with no network hop and no credentials.
//   - Anywhere else — the web target on a normal server, a desktop binary,
//     an iOS or Android app — there is no binding, so it goes over
//     Cloudflare's HTTP API.
//
// Both register the same database/sql driver under the name "d1", opened with
// the binding name as the DSN, so this is identical on all of them:
//
//	db, err := sql.Open("d1", "DB")
//
// and the handler that uses it does not know or care which target it was
// compiled for. That is the whole point: one set of handlers, one data layer,
// four platforms.
//
// # Credentials
//
// The HTTP transport needs an API token, and an API token is not something to
// ship inside an application you hand to people — it is account-scoped, and a
// desktop or mobile binary is readable by whoever has it. Use the HTTP
// transport on servers you control. From a desktop or mobile app, talk to your
// own Worker over HTTPS and let it hold the binding; the Worker is the only
// place the credential belongs.
//
// Open reports that distinction rather than leaving it to be discovered.
package d1

import (
	"database/sql"
	"fmt"
	"os"
)

// Binding is the default D1 binding name, matching the convention in
// wrangler.toml.
const Binding = "DB"

// Open returns a *sql.DB for the named binding, using whichever transport this
// build has.
func Open(binding string) (*sql.DB, error) {
	if binding == "" {
		binding = Binding
	}
	if err := checkConfig(binding); err != nil {
		return nil, err
	}
	return sql.Open("d1", binding)
}

// envOr reads the first of the given environment variables that is set.
func envOr(names ...string) string {
	for _, n := range names {
		if v := os.Getenv(n); v != "" {
			return v
		}
	}
	return ""
}

// missing describes configuration that is absent, naming the variables to set
// rather than failing later inside an HTTP request with a 401.
func missing(what string, names ...string) error {
	return fmt.Errorf("d1: %s is not configured — set one of: %v\n"+
		"  On a Worker these are unnecessary: D1 is a binding.\n"+
		"  On a server, set them in the environment.\n"+
		"  In a desktop or mobile app, do not set them at all — an API token\n"+
		"  is account-scoped and readable by anyone holding the binary. Call\n"+
		"  your Worker over HTTPS instead and let it hold the binding.",
		what, names)
}
