package content

import (
	"reflect"
	"regexp"
	"sort"
	"strings"
	"testing"

	"github.com/stukennedy/irgo/desktop"
	"github.com/stukennedy/irgo/pkg/adapter"
	dsstar "github.com/stukennedy/irgo/pkg/datastar"
	irgorender "github.com/stukennedy/irgo/pkg/render"
	"github.com/stukennedy/irgo/pkg/router"
	irgotest "github.com/stukennedy/irgo/pkg/testing"
	"github.com/stukennedy/irgo/pkg/websocket"
)

// The documentation is checked against the framework it documents.
//
// The CLI reference cannot be wrong: it is rendered from what the binary
// declares. Everything else on this site is prose someone typed, and prose
// drifts — the ported pages arrived documenting ctx.Unauthorized,
// sse.PatchElements, r.StaticFS and a datastar.Hub, none of which exist. Every
// one of those reads as plausible and none of them compile.
//
// So the samples are read back and every call on a framework type is looked up
// by reflection. This cannot check that a sample is *good*, only that it is
// possible, which is the failure that was actually shipping.

// receivers maps the variable names the samples use to the type they stand
// for. A call on anything else — db, todos, signals — is application code and
// is skipped.
var receivers = map[string]reflect.Type{
	"ctx":        reflect.TypeOf(&router.Context{}),
	"c":          reflect.TypeOf(&router.Context{}),
	"sse":        reflect.TypeOf(&dsstar.SSE{}),
	"r":          reflect.TypeOf(&router.Router{}),
	"resp":       reflect.TypeOf(&irgotest.Response{}),
	"client":     reflect.TypeOf(&irgotest.Client{}),
	"hub":        reflect.TypeOf(&websocket.Hub{}),
	"renderer":   reflect.TypeOf(&irgorender.TemplRenderer{}),
	"adapter":    reflect.TypeOf(&adapter.HTTPAdapter{}),
	"desktopApp": reflect.TypeOf(&desktop.App{}),
}

// packages maps a package qualifier to the functions it really exports.
var packages = map[string]map[string]bool{
	"router":    {"New": true, "NewWithoutMiddleware": true, "NewContext": true, "IsDatastarRequest": true, "CORSMiddleware": true, "NoCacheMiddleware": true, "RequireDatastar": true, "DatastarRequestMiddleware": true, "SecretValidationMiddleware": true, "StrictOriginMiddleware": true, "WebSocketSecretMiddleware": true},
	"desktop":   {"New": true, "NewWithHub": true, "DefaultConfig": true, "FindStaticDir": true, "FindResourcePath": true, "GenerateSecret": true, "SetupMenu": true, "StaticFS": true, "Config": true},
	"mobile":    {"Initialize": true, "SetHandler": true, "GetHub": true, "IsReady": true, "Shutdown": true, "ClearCookies": true, "HandleRequest": true, "RenderInitialPage": true, "SetStateDir": true},
	"render":    {"NewTemplRenderer": true, "New": true, "RenderComponent": true, "MustRenderComponent": true, "TemplHandler": true, "DefaultFuncs": true},
	"adapter":   {"NewHTTPAdapter": true, "NewHandlerFuncAdapter": true},
	"websocket": {"NewHub": true, "HTMLEnvelope": true, "JSONEnvelope": true, "NewEnvelope": true, "ReplyEnvelope": true, "SwapEnvelope": true},
	"datastar":  {"NewSSE": true, "ReadSignals": true, "RenderTempl": true},
}

// callRE finds `receiver.Method(` in a sample.
var callRE = regexp.MustCompile(`\b([a-zA-Z_][a-zA-Z0-9_]*)\.([A-Z][a-zA-Z0-9_]*)\(`)

// TestSamplesOnlyCallRealMethods is the guard the API page never had.
func TestSamplesOnlyCallRealMethods(t *testing.T) {
	bad := map[string][]string{}

	check := func(where, src string) {
		for _, m := range callRE.FindAllStringSubmatch(src, -1) {
			recv, method := m[1], m[2]

			// A name can be both, and legitimately: `adapter :=
			// adapter.NewHTTPAdapter(h)` shadows the package with a value of
			// its own type. Accept the call if either reading resolves.
			typ, isRecv := receivers[recv]
			fns, isPkg := packages[recv]
			if !isRecv && !isPkg {
				continue
			}
			if isRecv {
				if _, found := typ.MethodByName(method); found {
					continue
				}
			}
			if isPkg && fns[method] {
				continue
			}
			switch {
			case isRecv && isPkg:
				bad[where] = append(bad[where],
					recv+"."+method+" — neither a method on "+typ.String()+
						" nor exported by package "+recv)
			case isRecv:
				bad[where] = append(bad[where],
					recv+"."+method+" — no such method on "+typ.String())
			default:
				bad[where] = append(bad[where],
					recv+"."+method+" — not exported by package "+recv)
			}
		}
	}

	for _, p := range Pages {
		for _, b := range p.Blocks {
			if b.Kind == KindCode {
				check(p.Slug, b.Text)
			}
		}
	}
	for name, s := range Samples {
		check("sample:"+name, s.Text)
	}

	if len(bad) == 0 {
		return
	}
	keys := make([]string, 0, len(bad))
	for k := range bad {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		seen := map[string]bool{}
		for _, msg := range bad[k] {
			if seen[msg] {
				continue
			}
			seen[msg] = true
			t.Errorf("%s: %s", k, msg)
		}
	}
}

// returnRE finds a call used as a return value, with or without a leading
// empty string for the (string, error) handler shape.
var returnRE = regexp.MustCompile(`return (?:"",\s*)?([a-zA-Z_][a-zA-Z0-9_]*)\.([A-Z][a-zA-Z0-9_]*)\(`)

// TestReturnedCallsReturnSomething catches the other half of the drift.
//
// ctx.NotFound, ctx.BadRequest and ctx.HTML all return nothing, and the ported
// samples used every one of them as `return ctx.NotFound("...")`. That is a
// compile error, and it reads perfectly.
func TestReturnedCallsReturnSomething(t *testing.T) {
	check := func(where, src string) {
		for _, m := range returnRE.FindAllStringSubmatch(src, -1) {
			recv, method := m[1], m[2]
			typ, ok := receivers[recv]
			if !ok {
				continue
			}
			fn, found := typ.MethodByName(method)
			if !found {
				continue // reported by the test above
			}
			if fn.Type.NumOut() == 0 {
				t.Errorf("%s: `return %s.%s(...)` — %s returns nothing",
					where, recv, method, method)
			}
		}
	}
	for _, p := range Pages {
		for _, b := range p.Blocks {
			if b.Kind == KindCode {
				check(p.Slug, b.Text)
			}
		}
	}
	for name, s := range Samples {
		check("sample:"+name, s.Text)
	}
}

// TestContextFieldsAreReal — the API page listed the Context struct by hand,
// and named a field that does not exist.
func TestContextFieldsAreReal(t *testing.T) {
	typ := reflect.TypeOf(router.Context{})
	for _, page := range Pages {
		for _, b := range page.Blocks {
			if b.Kind != KindCode || !strings.Contains(b.Text, "type Context struct") {
				continue
			}
			for _, line := range strings.Split(b.Text, "\n") {
				fields := strings.Fields(line)
				if len(fields) != 2 || !isExportedName(fields[0]) {
					continue
				}
				if _, ok := typ.FieldByName(fields[0]); !ok {
					t.Errorf("%s: Context has no field %s", page.Slug, fields[0])
				}
			}
		}
	}
}

func isExportedName(s string) bool {
	return s != "" && s[0] >= 'A' && s[0] <= 'Z'
}
