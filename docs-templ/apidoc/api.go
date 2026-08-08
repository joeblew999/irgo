// Package apidoc is the API reference, generated from the framework's source.
//
// The old site's API page was written by hand, and by the time it was ported
// it documented ctx.Unauthorized, ctx.Forbidden, sse.PatchElements,
// sse.RemoveElements, sse.Console, r.StaticFS, client.GetWithHeaders,
// resp.ParseSSEEvents and a Context field called ResponseWriter — none of
// which exist. Every one reads as plausible. None of them compile.
//
// So it is generated, like the CLI reference next to it: go/doc reads the
// framework packages and writes api.json, and a test fails when the file no
// longer matches the source. A page that cannot be written by hand cannot
// describe a method that was renamed three releases ago.
//
// Regenerate with:  go test ./apidoc -run TestAPIDocIsCurrent -update
package apidoc

//go:generate go test . -run TestAPIDocIsCurrent -update
//go:generate go test . -run TestEnvDocIsCurrent -update

import (
	_ "embed"
	"encoding/json"
	"strings"
)

//go:embed api.json
var apiJSON []byte

// Package is one documented framework package.
type Package struct {
	// ImportPath is what you write in an import statement.
	ImportPath string `json:"import"`
	// Name is the identifier the package is referred to by.
	Name string `json:"name"`
	// Doc is the package comment's first paragraph.
	Doc string `json:"doc,omitempty"`
	// Funcs are the package-level functions.
	Funcs []Func `json:"funcs,omitempty"`
	// Types are the exported types, each with its own constructors and methods.
	Types []Type `json:"types,omitempty"`
}

// Type is an exported type.
type Type struct {
	Name string `json:"name"`
	Doc  string `json:"doc,omitempty"`
	// Decl is the type declaration, for structs with exported fields.
	Decl string `json:"decl,omitempty"`
	// Ctors are the functions that return this type.
	Ctors []Func `json:"ctors,omitempty"`
	// Methods are the methods on the type.
	Methods []Func `json:"methods,omitempty"`
}

// Func is a function or method.
type Func struct {
	Name string `json:"name"`
	// Signature is the declaration as written, with parameter names.
	Signature string `json:"sig"`
	Doc       string `json:"doc,omitempty"`
}

// EnvVar is an environment variable the framework reads.
type EnvVar struct {
	Name string `json:"name"`
	// Used lists the packages that read it.
	Used []string `json:"used"`
}

//go:embed env.json
var envJSON []byte

// Env is every environment variable the framework actually reads.
//
// The old site's CLI page listed IRGO_PORT, which nothing has ever read. This
// list is scanned out of the source, so it cannot invent one — or miss one.
func Env() []EnvVar {
	var vars []EnvVar
	if err := json.Unmarshal(envJSON, &vars); err != nil {
		panic("docs: apidoc/env.json does not parse: " + err.Error())
	}
	return vars
}

// Own reports whether this is a variable irgo defines, rather than one it
// reads from the surrounding toolchain. Derived from the name so that nothing
// has to be kept in a list by hand.
func (e EnvVar) Own() bool { return strings.HasPrefix(e.Name, "IRGO_") }

// Packages is every documented package, in the order the page lists them.
func Packages() []Package {
	var pkgs []Package
	if err := json.Unmarshal(apiJSON, &pkgs); err != nil {
		panic("docs: apidoc/api.json does not parse: " + err.Error())
	}
	return pkgs
}

// Summary is a doc comment reduced to its first sentence, for the places a
// page shows one line rather than a paragraph.
func Summary(doc string) string {
	doc = strings.TrimSpace(doc)
	if doc == "" {
		return ""
	}
	if i := strings.Index(doc, ". "); i >= 0 {
		return doc[:i+1]
	}
	if line, _, ok := strings.Cut(doc, "\n"); ok {
		return line
	}
	return doc
}
