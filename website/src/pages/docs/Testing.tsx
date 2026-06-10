import type { FC } from 'hono/jsx'
import { DocsLayout } from '../../components/DocsLayout'
import { CodeBlock } from '../../components/CodeBlock'

export const TestingPage: FC = () => {
  return (
    <DocsLayout title="Testing" currentPath="/docs/testing">
      <h1>Testing</h1>

      <p>
        irgo provides testing utilities to help you test your handlers and templates.
      </p>

      <h2>Test Client</h2>

      <p>
        The <code>testing</code> package provides an HTTP client for testing handlers:
      </p>

      <CodeBlock language="go">
{`import (
    "testing"
    irgotest "github.com/stukennedy/irgo/pkg/testing"
)

func TestListTodos(t *testing.T) {
    r := app.NewRouter()
    client := irgotest.NewClient(r.Handler())

    resp := client.Get("/todos")

    resp.AssertOK(t)
    resp.AssertContains(t, "My Todos")
}`}
      </CodeBlock>

      <h2>HTTP Methods</h2>

      <CodeBlock language="go">
{`client := irgotest.NewClient(handler)

// GET request
resp := client.Get("/users")

// GET with query parameters
resp := client.Get("/search?q=hello")

// POST with form data
resp := client.PostForm("/users", map[string]string{
    "name":  "John",
    "email": "john@example.com",
})

// POST with JSON
resp := client.PostJSON("/api/users", map[string]any{
    "name":  "John",
    "email": "john@example.com",
})

// PUT request
resp := client.Put("/users/123", map[string]string{
    "name": "Jane",
})

// PATCH request
resp := client.Patch("/users/123", map[string]string{
    "name": "Jane",
})

// DELETE request
resp := client.Delete("/users/123")`}
      </CodeBlock>

      <h2>Assertions</h2>

      <CodeBlock language="go">
{`resp := client.Get("/users")

// Status assertions
resp.AssertOK(t)                    // 200
resp.AssertStatus(t, 201)           // Specific status
resp.AssertRedirect(t, "/login")    // 3xx redirect

// Content assertions
resp.AssertContains(t, "Welcome")
resp.AssertNotContains(t, "Error")

// Header assertions
resp.AssertHeader(t, "Content-Type", "text/html")

// Get response data
body := resp.Body()           // string
status := resp.StatusCode     // int`}
      </CodeBlock>

      <h2>Testing Datastar SSE Requests</h2>

      <CodeBlock language="go">
{`func TestDatastarRequest(t *testing.T) {
    client := irgotest.NewClient(handler)

    // Simulate Datastar request
    resp := client.Datastar().Get("/users")

    resp.AssertOK(t)
    resp.AssertHeader(t, "Content-Type", "text/event-stream")

    // Parse SSE events
    events := resp.ParseSSEEvents()
    if len(events) == 0 {
        t.Error("expected SSE events in response")
    }

    // Check for patch event
    resp.AssertSSEEvent(t, "datastar-patch-elements")
}`}
      </CodeBlock>

      <h2>Testing Handlers Directly</h2>

      <CodeBlock language="go">
{`func TestCreateUser(t *testing.T) {
    // Create mock context
    ctx := irgotest.NewContext()
    ctx.SetFormValue("name", "John")
    ctx.SetFormValue("email", "john@example.com")

    // Call handler
    html, err := handlers.CreateUser(ctx)

    if err != nil {
        t.Fatalf("unexpected error: %v", err)
    }

    if !strings.Contains(html, "John") {
        t.Error("expected response to contain user name")
    }
}`}
      </CodeBlock>

      <h2>Testing Templates</h2>

      <CodeBlock language="go">
{`func TestUserCard(t *testing.T) {
    user := User{
        ID:    "123",
        Name:  "John",
        Email: "john@example.com",
    }

    renderer := render.NewTemplRenderer()
    html, err := renderer.Render(templates.UserCard(user))

    if err != nil {
        t.Fatalf("render error: %v", err)
    }

    if !strings.Contains(html, "John") {
        t.Error("expected card to contain user name")
    }

    if !strings.Contains(html, "john@example.com") {
        t.Error("expected card to contain email")
    }
}`}
      </CodeBlock>

      <h2>Running Tests</h2>

      <CodeBlock language="bash">
{`# Run all tests
go test ./...

# Run with verbose output
go test -v ./...

# Run specific package
go test ./handlers/...

# Run with coverage
go test -cover ./...`}
      </CodeBlock>

      <h2>Example Test File</h2>

      <CodeBlock title="handlers/todos_test.go" language="go">
{`package handlers

import (
    "testing"
    "myapp/app"
    irgotest "github.com/stukennedy/irgo/pkg/testing"
)

func TestTodoWorkflow(t *testing.T) {
    r := app.NewRouter()
    client := irgotest.NewClient(r.Handler())

    // List todos (empty)
    resp := client.Get("/todos")
    resp.AssertOK(t)
    resp.AssertContains(t, "No todos yet")

    // Create todo
    resp = client.PostForm("/todos", map[string]string{
        "title": "Buy groceries",
    })
    resp.AssertOK(t)
    resp.AssertContains(t, "Buy groceries")

    // Toggle todo
    resp = client.PostForm("/todos/1/toggle", nil)
    resp.AssertOK(t)
    resp.AssertContains(t, "completed")

    // Delete todo
    resp = client.Delete("/todos/1")
    resp.AssertOK(t)
}`}
      </CodeBlock>

      <h2>Next Steps</h2>

      <ul>
        <li><a href="/docs/routing">Routing & Handlers</a> - Handler API reference</li>
        <li><a href="/docs/templates">Templ Templates</a> - Template testing</li>
      </ul>
    </DocsLayout>
  )
}
