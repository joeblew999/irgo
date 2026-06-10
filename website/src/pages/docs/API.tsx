import type { FC } from 'hono/jsx'
import { DocsLayout } from '../../components/DocsLayout'
import { CodeBlock } from '../../components/CodeBlock'

export const APIPage: FC = () => {
  return (
    <DocsLayout title="API Reference" currentPath="/docs/api">
      <h1>API Reference</h1>

      <p>
        Complete API reference for irgo packages.
      </p>

      <h2>pkg/router</h2>

      <h3>Router</h3>

      <CodeBlock language="go">
{`// Create new router
r := router.New()

// HTTP methods
r.GET(path string, handler FragmentHandler)
r.POST(path string, handler FragmentHandler)
r.PUT(path string, handler FragmentHandler)
r.DELETE(path string, handler FragmentHandler)
r.PATCH(path string, handler FragmentHandler)

// Route groups
r.Route(pattern string, fn func(r *Router))

// Static files
r.Static(path string, root http.FileSystem)

// Middleware
r.Use(middleware ...func(http.Handler) http.Handler)

// Get http.Handler
r.Handler() http.Handler`}
      </CodeBlock>

      <h3>Context</h3>

      <CodeBlock language="go">
{`type Context struct {
    Request        *http.Request
    ResponseWriter http.ResponseWriter
}

// Input
ctx.Param(key string) string
ctx.Query(key string) string
ctx.FormValue(key string) string
ctx.Header(key string) string

// Datastar detection & signals
ctx.IsDatastar() bool
ctx.ReadSignals(v any) error
ctx.SSE() *datastar.SSE

// HTML responses
ctx.HTML(html string) error
ctx.HTMLStatus(status int, html string) error

// JSON responses
ctx.JSON(data any) error
ctx.JSONStatus(status int, data any) error

// Errors
ctx.Error(err error) error
ctx.ErrorStatus(status int, msg string) error
ctx.NotFound(msg string) error
ctx.BadRequest(msg string) error
ctx.Unauthorized(msg string) error
ctx.Forbidden(msg string) error

// Redirects
ctx.Redirect(url string) error

// No content
ctx.NoContent() error

// SSE response methods (via ctx.SSE())
sse.PatchTempl(c templ.Component) error
sse.PatchElements(id string, html string) error
sse.PatchSignals(signals map[string]any) error
sse.RemoveElements(selector string) error
sse.Redirect(url string) error
sse.Console(level string, msg string) error`}
      </CodeBlock>

      <h3>Handler Types</h3>

      <CodeBlock language="go">
{`// Page handler (full page renders)
type PageHandler func(ctx *Context) (string, error)

// Datastar handler (SSE responses)
type DatastarHandler func(ctx *Context) error

// Register Datastar handlers
r.DSGet(path string, handler DatastarHandler)
r.DSPost(path string, handler DatastarHandler)
r.DSPut(path string, handler DatastarHandler)
r.DSPatch(path string, handler DatastarHandler)
r.DSDelete(path string, handler DatastarHandler)`}
      </CodeBlock>

      <h2>pkg/render</h2>

      <h3>TemplRenderer</h3>

      <CodeBlock language="go">
{`// Create renderer
renderer := render.NewTemplRenderer()

// Render templ component to string
html, err := renderer.Render(component templ.Component) (string, error)`}
      </CodeBlock>

      <h2>pkg/desktop</h2>

      <h3>Config</h3>

      <CodeBlock language="go">
{`type Config struct {
    Title     string // Window title
    Width     int    // Window width
    Height    int    // Window height
    Resizable bool   // Allow resize
    Debug     bool   // Enable devtools
    Port      int    // Server port (0 = auto)
}

// Default configuration
config := desktop.DefaultConfig()`}
      </CodeBlock>

      <h3>App</h3>

      <CodeBlock language="go">
{`// Create desktop app
app := desktop.New(handler http.Handler, config Config)

// Run (blocks until window closed)
err := app.Run()`}
      </CodeBlock>

      <h3>Utilities</h3>

      <CodeBlock language="go">
{`// Find static files directory
dir := desktop.FindStaticDir() string

// Find resource path
path := desktop.FindResourcePath(name string) string`}
      </CodeBlock>

      <h2>pkg/mobile</h2>

      <h3>Bridge</h3>

      <CodeBlock language="go">
{`// Initialize mobile bridge
mobile.Initialize()

// Set HTTP handler
mobile.SetHandler(handler http.Handler)`}
      </CodeBlock>

      <h2>pkg/adapter</h2>

      <h3>HTTPAdapter</h3>

      <CodeBlock language="go">
{`// Create adapter for virtual HTTP
adapter := adapter.NewHTTPAdapter(handler http.Handler)

// Handle request (called from native code)
response := adapter.HandleRequest(request *core.Request) *core.Response`}
      </CodeBlock>

      <h2>pkg/datastar</h2>

      <h3>SSE</h3>

      <CodeBlock language="go">
{`// Create SSE writer
sse := datastar.NewSSE(w http.ResponseWriter, r *http.Request)

// Patch templ component into DOM
sse.PatchTempl(c templ.Component) error

// Patch raw HTML into element
sse.PatchElements(id string, html string) error

// Update client-side signals
sse.PatchSignals(signals map[string]any) error

// Remove elements from DOM
sse.RemoveElements(selector string) error

// Redirect to URL
sse.Redirect(url string) error

// Log to browser console
sse.Console(level string, msg string) error`}
      </CodeBlock>

      <h3>Hub (Server Push)</h3>

      <CodeBlock language="go">
{`// Create hub
hub := datastar.NewHub()

// Run hub (in goroutine)
go hub.Run()

// Handle SSE connections
hub.HandleSSE(w http.ResponseWriter, r *http.Request, opts ...Option)

// Broadcast to all
hub.Broadcast(events ...datastar.Event)

// Broadcast to room
hub.BroadcastToRoom(room string, events ...datastar.Event)

// Send to session
hub.SendToSession(sessionID string, events ...datastar.Event)`}
      </CodeBlock>

      <h3>Options</h3>

      <CodeBlock language="go">
{`// Set room
datastar.WithRoom(room string)

// Set session ID
datastar.WithSessionID(id string)`}
      </CodeBlock>

      <h2>pkg/testing</h2>

      <h3>Client</h3>

      <CodeBlock language="go">
{`// Create test client
client := testing.NewClient(handler http.Handler)

// HTTP methods
client.Get(path string) *Response
client.GetWithHeaders(path string, headers map[string]string) *Response
client.PostForm(path string, data map[string]string) *Response
client.PostJSON(path string, data any) *Response
client.Put(path string, data map[string]string) *Response
client.Patch(path string, data map[string]string) *Response
client.Delete(path string) *Response

// Datastar testing
client.Datastar() *Client  // Returns client with Datastar headers`}
      </CodeBlock>

      <h3>Response</h3>

      <CodeBlock language="go">
{`type Response struct {
    StatusCode int
    Headers    http.Header
}

// Get body
resp.Body() string

// Assertions
resp.AssertOK(t *testing.T)
resp.AssertStatus(t *testing.T, status int)
resp.AssertRedirect(t *testing.T, url string)
resp.AssertContains(t *testing.T, text string)
resp.AssertNotContains(t *testing.T, text string)
resp.AssertHeader(t *testing.T, key, value string)

// SSE assertions
resp.AssertSSEEvent(t *testing.T, eventType string)
resp.ParseSSEEvents() []SSEEvent`}
      </CodeBlock>

      <h2>Next Steps</h2>

      <ul>
        <li><a href="/docs/routing">Routing & Handlers</a> - Usage examples</li>
        <li><a href="/docs/templates">Templ Templates</a> - Template syntax</li>
        <li><a href="https://github.com/stukennedy/irgo">GitHub</a> - Source code</li>
      </ul>
    </DocsLayout>
  )
}
