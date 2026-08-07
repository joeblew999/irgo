//go:build js && wasm

// Inside a Worker there is nothing to configure: D1 arrives as a binding on
// the environment, so importing the driver is the whole integration.
//
// The file is named for the platform rather than ending in _js.go on purpose —
// a file ending in a GOOS name is compiled only for that GOOS and silently
// excluded everywhere else, which is a trap this repository has fallen into
// more than once. The build constraint above does the work.
package d1

import (
	// Registers the "d1" driver against the Worker's binding.
	_ "github.com/syumai/workers/cloudflare/d1"
)

// checkConfig has nothing to check: a missing binding is reported by the
// driver itself when the database is opened.
func checkConfig(binding string) error { return nil }
