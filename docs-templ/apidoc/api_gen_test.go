package apidoc

import (
	"bytes"
	"encoding/json"
	"flag"
	"go/ast"
	"go/doc"
	"go/parser"
	"go/printer"
	"go/token"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"
)

// The generator, and the test that keeps its output honest.
//
// go/doc lives here rather than in api.go for the same reason chroma lives in
// content's test: this package is compiled into a WebAssembly module served
// from a Cloudflare Worker, and the parser is only needed on a developer's
// machine.

var update = flag.Bool("update", false, "rewrite apidoc/api.json from the framework source")

// documented is the framework surface the reference covers, in page order.
// Adding a package here is all it takes for it to appear.
var documented = []struct{ Dir, ImportPath string }{
	{"../../pkg/router", "github.com/stukennedy/irgo/pkg/router"},
	{"../../pkg/render", "github.com/stukennedy/irgo/pkg/render"},
	{"../../pkg/datastar", "github.com/stukennedy/irgo/pkg/datastar"},
	{"../../pkg/websocket", "github.com/stukennedy/irgo/pkg/websocket"},
	{"../../pkg/adapter", "github.com/stukennedy/irgo/pkg/adapter"},
	{"../../pkg/testing", "github.com/stukennedy/irgo/pkg/testing"},
	{"../../desktop", "github.com/stukennedy/irgo/desktop"},
	{"../../mobile", "github.com/stukennedy/irgo/mobile"},
}

// envRoots are scanned for environment variables. The CLI is included because
// that is where a reader most expects them, and where the stale IRGO_PORT
// entry claimed to be.
var envRoots = []string{"../../pkg", "../../cmd", "../../desktop", "../../mobile"}

func TestEnvDocIsCurrent(t *testing.T) {
	vars, err := scanEnv()
	if err != nil {
		t.Skipf("cannot read the framework source from here: %v", err)
	}
	data, err := json.MarshalIndent(vars, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	data = append(data, '\n')

	if *update {
		if err := os.WriteFile("env.json", data, 0o644); err != nil {
			t.Fatal(err)
		}
		t.Logf("wrote %d environment variables", len(vars))
		return
	}
	got, err := os.ReadFile("env.json")
	if err != nil {
		t.Fatalf("no env.json — run: go test ./apidoc -run TestEnvDocIsCurrent -update")
	}
	if !bytes.Equal(bytes.TrimSpace(got), bytes.TrimSpace(data)) {
		t.Error("env.json no longer matches the source. Regenerate with:\n" +
			"\tgo test ./apidoc -run TestEnvDocIsCurrent -update")
	}
}

// scanEnv finds every os.Getenv and os.LookupEnv call with a literal name.
func scanEnv() ([]EnvVar, error) {
	used := map[string]map[string]bool{}

	for _, root := range envRoots {
		err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
			if err != nil || info.IsDir() || !strings.HasSuffix(path, ".go") ||
				strings.HasSuffix(path, "_test.go") {
				return err
			}
			fset := token.NewFileSet()
			f, err := parser.ParseFile(fset, path, nil, 0)
			if err != nil {
				return nil // a file that will not parse here is not our business
			}
			ast.Inspect(f, func(n ast.Node) bool {
				call, ok := n.(*ast.CallExpr)
				if !ok || len(call.Args) == 0 {
					return true
				}
				sel, ok := call.Fun.(*ast.SelectorExpr)
				if !ok {
					return true
				}
				pkg, ok := sel.X.(*ast.Ident)
				if !ok || pkg.Name != "os" {
					return true
				}
				if sel.Sel.Name != "Getenv" && sel.Sel.Name != "LookupEnv" {
					return true
				}
				lit, ok := call.Args[0].(*ast.BasicLit)
				if !ok || lit.Kind != token.STRING {
					return true
				}
				name := strings.Trim(lit.Value, `"`)
				if name == "" {
					return true
				}
				if used[name] == nil {
					used[name] = map[string]bool{}
				}
				used[name][shortDir(path)] = true
				return true
			})
			return nil
		})
		if err != nil {
			return nil, err
		}
	}

	names := make([]string, 0, len(used))
	for n := range used {
		names = append(names, n)
	}
	sort.Strings(names)

	out := make([]EnvVar, 0, len(names))
	for _, n := range names {
		var where []string
		for d := range used[n] {
			where = append(where, d)
		}
		sort.Strings(where)
		out = append(out, EnvVar{Name: n, Used: where})
	}
	return out, nil
}

// shortDir names the package a file belongs to, relative to the framework root.
func shortDir(path string) string {
	d := filepath.Dir(path)
	d = strings.TrimPrefix(filepath.ToSlash(d), "../../")
	return d
}

func TestAPIDocIsCurrent(t *testing.T) {
	want, err := build()
	if err != nil {
		t.Skipf("cannot read the framework source from here: %v", err)
	}

	data, err := json.MarshalIndent(want, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	data = append(data, '\n')

	if *update {
		if err := os.WriteFile("api.json", data, 0o644); err != nil {
			t.Fatal(err)
		}
		var types, funcs int
		for _, p := range want {
			types += len(p.Types)
			funcs += len(p.Funcs)
			for _, ty := range p.Types {
				funcs += len(ty.Methods) + len(ty.Ctors)
			}
		}
		t.Logf("wrote %d packages, %d types, %d functions", len(want), types, funcs)
		return
	}

	got, err := os.ReadFile("api.json")
	if err != nil {
		t.Fatalf("no api.json — run: go test ./apidoc -run TestAPIDocIsCurrent -update")
	}
	if !bytes.Equal(bytes.TrimSpace(got), bytes.TrimSpace(data)) {
		t.Error("api.json no longer matches the framework source. " +
			"Something was renamed, added or removed. Regenerate with:\n" +
			"\tgo test ./apidoc -run TestAPIDocIsCurrent -update")
	}
}

// TestEveryPackageHasContent guards the quiet failure: a package that parses
// to nothing still produces valid JSON and an empty section on the page.
func TestEveryPackageHasContent(t *testing.T) {
	for _, p := range Packages() {
		if len(p.Types) == 0 && len(p.Funcs) == 0 {
			t.Errorf("%s documents nothing", p.ImportPath)
		}
	}
}

func build() ([]Package, error) {
	var out []Package
	for _, d := range documented {
		p, err := readPackage(d.Dir, d.ImportPath)
		if err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, nil
}

func readPackage(dir, importPath string) (Package, error) {
	fset := token.NewFileSet()
	pkgs, err := parser.ParseDir(fset, dir, func(fi os.FileInfo) bool {
		return !strings.HasSuffix(fi.Name(), "_test.go")
	}, parser.ParseComments)
	if err != nil {
		return Package{}, err
	}

	// A directory can hold more than one package name once build tags are
	// involved. Prefer the one named after the directory, then the
	// lexicographically first.
	want := filepath.Base(strings.TrimSuffix(importPath, "/"))
	names := make([]string, 0, len(pkgs))
	for name := range pkgs {
		if !strings.HasSuffix(name, "_test") {
			names = append(names, name)
		}
	}
	if len(names) == 0 {
		return Package{}, os.ErrNotExist
	}
	sort.Strings(names)
	chosen := pkgs[names[0]]
	for _, n := range names {
		if n == want {
			chosen = pkgs[n]
			break
		}
	}

	files := selectFiles(chosen)
	d, err := doc.NewFromFiles(fset, files, importPath, doc.AllDecls)
	if err != nil {
		return Package{}, err
	}

	out := Package{
		ImportPath: importPath,
		Name:       want,
		Doc:        firstParagraph(d.Doc),
	}

	for _, fn := range d.Funcs {
		if !ast.IsExported(fn.Name) {
			continue
		}
		out.Funcs = append(out.Funcs, toFunc(fset, fn))
	}

	for _, ty := range d.Types {
		if !ast.IsExported(ty.Name) {
			continue
		}
		t := Type{Name: ty.Name, Doc: firstParagraph(ty.Doc)}
		if decl := structDecl(fset, ty); decl != "" {
			t.Decl = decl
		}
		for _, fn := range ty.Funcs {
			if ast.IsExported(fn.Name) {
				t.Ctors = append(t.Ctors, toFunc(fset, fn))
			}
		}
		for _, fn := range ty.Methods {
			if ast.IsExported(fn.Name) {
				t.Methods = append(t.Methods, toFunc(fset, fn))
			}
		}
		if len(t.Ctors) == 0 && len(t.Methods) == 0 && t.Decl == "" {
			continue
		}
		out.Types = append(out.Types, t)
	}
	return out, nil
}

// buildTagRE reads the constraint off a file, which is all this needs: these
// packages gate on a single tag and its negation.
var buildTagRE = regexp.MustCompile(`(?m)^//go:build\s+(!?)([a-zA-Z0-9_]+)\s*$`)

// selectFiles returns the package's files in a fixed order, dropping the
// negative half of a build-tag pair.
//
// desktop declares App.Bind twice: once in webview_desktop.go behind
// //go:build desktop, and once in webview_stub.go behind //go:build !desktop
// saying the method is unavailable. go/doc keeps whichever it reads first, and
// it reads a map, so the generated file flipped between runs and the staleness
// test became a coin toss. Sorting fixes the flapping; dropping the negative
// file documents the implementation rather than the stub, which is what a
// reader building a desktop app needs.
func selectFiles(pkg *ast.Package) []*ast.File {
	type tagged struct {
		name string
		neg  bool
		tag  string
		file *ast.File
	}
	var all []tagged
	positive := map[string]bool{}

	for name, f := range pkg.Files {
		t := tagged{name: name, file: f}
		// go/parser strips the build comment from the AST, so read the line
		// back off disk.
		if data, err := os.ReadFile(name); err == nil {
			if m := buildTagRE.FindSubmatch(data); m != nil {
				t.neg = string(m[1]) == "!"
				t.tag = string(m[2])
			}
		}
		if t.tag != "" && !t.neg {
			positive[t.tag] = true
		}
		all = append(all, t)
	}

	sort.Slice(all, func(i, j int) bool { return all[i].name < all[j].name })

	files := make([]*ast.File, 0, len(all))
	for _, t := range all {
		if t.neg && positive[t.tag] {
			continue // the stub half of a pair
		}
		files = append(files, t.file)
	}
	return files
}

func toFunc(fset *token.FileSet, fn *doc.Func) Func {
	return Func{
		Name:      fn.Name,
		Signature: render(fset, fn.Decl),
		Doc:       firstParagraph(fn.Doc),
	}
}

// render prints a declaration without its body, which is what a reference
// shows: the signature as written, parameter names included.
func render(fset *token.FileSet, decl *ast.FuncDecl) string {
	stripped := *decl
	stripped.Body = nil
	stripped.Doc = nil
	var buf bytes.Buffer
	if err := printer.Fprint(&buf, fset, &stripped); err != nil {
		return decl.Name.Name
	}
	return strings.TrimSpace(buf.String())
}

// structDecl prints a struct's exported fields, and nothing for other kinds.
// The unexported half of a struct is not API and only makes the page longer.
func structDecl(fset *token.FileSet, ty *doc.Type) string {
	for _, spec := range ty.Decl.Specs {
		ts, ok := spec.(*ast.TypeSpec)
		if !ok {
			continue
		}
		st, ok := ts.Type.(*ast.StructType)
		if !ok || st.Fields == nil {
			continue
		}
		var fields []*ast.Field
		for _, f := range st.Fields.List {
			for _, n := range f.Names {
				if ast.IsExported(n.Name) {
					fields = append(fields, f)
					break
				}
			}
		}
		if len(fields) == 0 {
			return ""
		}
		var buf bytes.Buffer
		buf.WriteString("type " + ty.Name + " struct {\n")
		for _, f := range fields {
			var names []string
			for _, n := range f.Names {
				if ast.IsExported(n.Name) {
					names = append(names, n.Name)
				}
			}
			var typ bytes.Buffer
			printer.Fprint(&typ, fset, f.Type)
			buf.WriteString("\t" + strings.Join(names, ", ") + " " + typ.String() + "\n")
		}
		buf.WriteString("}")
		return buf.String()
	}
	return ""
}

// firstParagraph keeps the summary and drops the essay. The reference is an
// index; the prose pages are where the reasoning belongs.
func firstParagraph(doc string) string {
	doc = strings.TrimSpace(doc)
	if doc == "" {
		return ""
	}
	if i := strings.Index(doc, "\n\n"); i >= 0 {
		doc = doc[:i]
	}
	return strings.Join(strings.Fields(doc), " ")
}
