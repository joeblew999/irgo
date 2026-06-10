import type { FC } from 'hono/jsx'
import { DocsLayout } from '../../components/DocsLayout'
import { CodeBlock } from '../../components/CodeBlock'
import { Callout } from '../../components/FeatureCard'

export const RoutingPage: FC = () => {
  return (
    <DocsLayout title="Routing & Handlers" currentPath="/docs/routing">
      <h1>Routing & Handlers</h1>

      <p>
        irgo uses a chi-based router with a simplified API designed for Datastar applications.
        Handlers return HTML via SSE rather than writing directly to response writers.
      </p>

      <h2>Basic Routing</h2>

      <CodeBlock language="go">
{`import "github.com/stukennedy/irgo/pkg/router"

r := router.New()

// Basic routes
r.GET("/", HomeHandler)
r.POST("/users", CreateUserHandler)
r.PUT("/users/{id}", UpdateUserHandler)
r.DELETE("/users/{id}", DeleteUserHandler)
r.PATCH("/users/{id}", PatchUserHandler)`}
      </CodeBlock>

      <h2>Datastar SSE Handlers</h2>

      <p>
        Datastar handlers return <code>error</code> and use <code>ctx.SSE()</code>
        to stream HTML updates via Server-Sent Events:
      </p>

      <CodeBlock language="go">
{`// Page handlers return (string, error) for full page renders
func HomeHandler(ctx *router.Context) (string, error) {
    return renderer.Render(templates.HomePage())
}

// Datastar handlers return error and use SSE
func UserHandler(ctx *router.Context) error {
    id := ctx.Param("id")
    user, err := db.GetUser(id)
    if err != nil {
        return ctx.NotFound("User not found")
    }
    return ctx.SSE().PatchTempl(templates.UserProfile(user))
}`}
      </CodeBlock>

      <h2>URL Parameters</h2>

      <p>Use curly braces for URL parameters:</p>

      <CodeBlock language="go">
{`// Single parameter
r.GET("/users/{id}", func(ctx *router.Context) (string, error) {
    id := ctx.Param("id")
    return fmt.Sprintf("User ID: %s", id), nil
})

// Multiple parameters
r.GET("/posts/{postID}/comments/{commentID}", func(ctx *router.Context) (string, error) {
    postID := ctx.Param("postID")
    commentID := ctx.Param("commentID")
    return fmt.Sprintf("Post %s, Comment %s", postID, commentID), nil
})`}
      </CodeBlock>

      <h2>Query Parameters</h2>

      <CodeBlock language="go">
{`// GET /search?q=hello&limit=10
r.GET("/search", func(ctx *router.Context) (string, error) {
    query := ctx.Query("q")
    limit := ctx.Query("limit")

    if query == "" {
        return "", ctx.BadRequest("Query parameter 'q' is required")
    }

    results := search(query, limit)
    return renderer.Render(templates.SearchResults(results))
})`}
      </CodeBlock>

      <h2>Form Data</h2>

      <CodeBlock language="go">
{`r.POST("/users", func(ctx *router.Context) (string, error) {
    name := ctx.FormValue("name")
    email := ctx.FormValue("email")

    if name == "" || email == "" {
        return "", ctx.BadRequest("Name and email are required")
    }

    user := db.CreateUser(name, email)
    return renderer.Render(templates.UserCard(user))
})`}
      </CodeBlock>

      <h2>Route Groups</h2>

      <p>Organize related routes with groups:</p>

      <CodeBlock language="go">
{`r.Route("/api", func(r *router.Router) {
    // All routes here are prefixed with /api
    r.GET("/users", ListUsers)
    r.POST("/users", CreateUser)

    r.Route("/admin", func(r *router.Router) {
        // /api/admin/...
        r.GET("/stats", AdminStats)
        r.POST("/reset", AdminReset)
    })
})`}
      </CodeBlock>

      <h2>Context API</h2>

      <p>The <code>router.Context</code> provides methods for handling requests and responses:</p>

      <h3>Input Methods</h3>

      <CodeBlock language="go">
{`func handler(ctx *router.Context) (string, error) {
    // URL parameters
    id := ctx.Param("id")

    // Query string parameters
    query := ctx.Query("q")

    // Form values
    name := ctx.FormValue("name")

    // Request headers
    token := ctx.Header("Authorization")

    // Access underlying request
    req := ctx.Request

    return "", nil
}`}
      </CodeBlock>

      <h3>Datastar Detection & Signals</h3>

      <CodeBlock language="go">
{`func handler(ctx *router.Context) error {
    // Check if request is from Datastar
    if ctx.IsDatastar() {
        // Handle as SSE response
        return ctx.SSE().PatchTempl(templates.UserCard(user))
    }
    // Return full page for non-Datastar requests
    return ctx.HTML(renderer.Render(templates.UserPage(user)))
}

// Read signals from Datastar request
var signals struct {
    Name  string \`json:"name"\`
    Email string \`json:"email"\`
}
ctx.ReadSignals(&signals)`}
      </CodeBlock>

      <h3>Response Methods</h3>

      <CodeBlock language="go">
{`// HTML responses
ctx.HTML("<div>Hello</div>")
ctx.HTMLStatus(201, "<div>Created</div>")

// JSON responses
ctx.JSON(user)
ctx.JSONStatus(201, user)

// Redirects (Datastar-aware)
ctx.Redirect("/new-location")

// No content
ctx.NoContent()`}
      </CodeBlock>

      <h3>Error Responses</h3>

      <CodeBlock language="go">
{`// Generic error
ctx.Error(err)
ctx.ErrorStatus(500, "Internal error")

// Common HTTP errors
ctx.NotFound("Resource not found")
ctx.BadRequest("Invalid input")
ctx.Unauthorized("Please log in")
ctx.Forbidden("Access denied")`}
      </CodeBlock>

      <h3>SSE Response Methods</h3>

      <p>Control Datastar behavior with SSE response methods:</p>

      <CodeBlock language="go">
{`func handler(ctx *router.Context) error {
    sse := ctx.SSE()

    // Morph HTML into DOM (by element ID in the template)
    sse.PatchTempl(templates.UserCard(user))

    // Update client-side signals
    sse.PatchSignals(map[string]any{"name": "", "loading": false})

    // Remove elements from DOM
    sse.RemoveElements("#old-item")

    // Redirect to new URL
    sse.Redirect("/users/123")

    // Log to browser console
    sse.Console("info", "Update complete")

    return nil
}`}
      </CodeBlock>

      <Callout type="tip" title="Multiple Updates">
        <p>
          With Datastar, you can call multiple <code>PatchTempl</code> methods
          in a single handler to update multiple elements. Each element is
          identified by its ID in the template.
        </p>
      </Callout>

      <h2>Static Files</h2>

      <CodeBlock language="go">
{`// Serve static files from a directory
r.Static("/static", http.Dir("static"))

// Or use http.FileServer directly
mux := http.NewServeMux()
mux.Handle("/static/", http.StripPrefix("/static/",
    http.FileServer(http.Dir("static"))))
mux.Handle("/", r.Handler())`}
      </CodeBlock>

      <h2>Middleware</h2>

      <p>irgo includes common middleware:</p>

      <CodeBlock language="go">
{`import "github.com/stukennedy/irgo/pkg/router/middleware"

r := router.New()

// Detect Datastar requests (sets ctx.IsDatastar())
r.Use(middleware.DatastarRequestMiddleware)

// Wrap non-Datastar responses in layout
r.Use(middleware.LayoutWrapper(templates.Layout))

// Prevent caching
r.Use(middleware.NoCacheMiddleware)

// CORS support
r.Use(middleware.CORSMiddleware)

// Require Datastar for routes
r.Route("/api", func(r *router.Router) {
    r.Use(middleware.RequireDatastar)
    r.DSGet("/user-card", UserCardHandler)
})`}
      </CodeBlock>

      <h2>Getting the HTTP Handler</h2>

      <p>Convert the router to a standard <code>http.Handler</code>:</p>

      <CodeBlock language="go">
{`r := router.New()
// ... register routes ...

// Get the http.Handler
handler := r.Handler()

// Use with http.ListenAndServe
http.ListenAndServe(":8080", handler)

// Or with desktop.New
app := desktop.New(handler, config)`}
      </CodeBlock>

      <h2>Next Steps</h2>

      <ul>
        <li><a href="/docs/templates">Templ Templates</a> - Learn the templating syntax</li>
        <li><a href="/docs/datastar">Datastar Integration</a> - Datastar patterns and best practices</li>
        <li><a href="/docs/testing">Testing</a> - Test your handlers</li>
      </ul>
    </DocsLayout>
  )
}
