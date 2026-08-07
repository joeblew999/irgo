import type { FC } from 'hono/jsx'
import { DocsLayout } from '../../components/DocsLayout'
import { CodeBlock } from '../../components/CodeBlock'
import { Callout } from '../../components/FeatureCard'

export const ExamplesPage: FC = () => {
  return (
    <DocsLayout title="Examples" currentPath="/docs/examples">
      <h1>Examples</h1>

      <p>
        Learn irgo through complete, working examples. Each example demonstrates
        key patterns and can be used as a starting point for your own apps.
      </p>

      <h2>Todo App</h2>

      <p>
        The classic todo app demonstrates CRUD operations, form handling, and Datastar interactions.
      </p>

      <h3>Handler</h3>

      <CodeBlock title="handlers/todos.go" language="go">
{`package handlers

import (
    "fmt"
    "myapp/templates"
    "github.com/stukennedy/irgo/pkg/router"
)

type Todo struct {
    ID    string
    Title string
    Done  bool
}

var todos = []Todo{}
var nextID = 1

func ListTodos(ctx *router.Context) error {
    return ctx.SSE().PatchTempl(templates.TodoList(todos))
}

func CreateTodo(ctx *router.Context) error {
    var signals struct {
        Title string \`json:"title"\`
    }
    ctx.ReadSignals(&signals)

    if signals.Title == "" {
        return ctx.BadRequest("Title is required")
    }

    todo := Todo{
        ID:    fmt.Sprintf("%d", nextID),
        Title: signals.Title,
        Done:  false,
    }
    nextID++
    todos = append(todos, todo)

    sse := ctx.SSE()
    sse.PatchTempl(templates.TodoItem(todo))
    sse.PatchSignals(map[string]any{"title": ""}) // Clear input
    return nil
}

func ToggleTodo(ctx *router.Context) error {
    id := ctx.Param("id")
    for i := range todos {
        if todos[i].ID == id {
            todos[i].Done = !todos[i].Done
            return ctx.SSE().PatchTempl(templates.TodoItem(todos[i]))
        }
    }
    return ctx.NotFound("Todo not found")
}

func DeleteTodo(ctx *router.Context) error {
    id := ctx.Param("id")
    for i := range todos {
        if todos[i].ID == id {
            todos = append(todos[:i], todos[i+1:]...)
            return ctx.SSE().RemoveElements("#todo-" + id)
        }
    }
    return ctx.NotFound("Todo not found")
}`}
      </CodeBlock>

      <h3>Templates</h3>

      <CodeBlock title="templates/todos.templ" language="templ">
{`package templates

templ TodoPage() {
    @Layout("Todos") {
        <div class="container">
            <h1>My Todos</h1>
            @TodoForm()
            <div id="todo-list">
                for _, todo := range todos {
                    @TodoItem(todo)
                }
            </div>
        </div>
    }
}

templ TodoForm() {
    <form
        data-signals="{title: ''}"
        data-on:submit__prevent="@post('/todos')"
        class="todo-form"
    >
        <input
            type="text"
            data-bind:title
            placeholder="What needs to be done?"
            required
            autofocus
        />
        <button type="submit">Add</button>
    </form>
}

templ TodoItem(todo Todo) {
    <div id={ "todo-" + todo.ID } class="todo-item">
        <input
            type="checkbox"
            checked?={ todo.Done }
            data-on:click={ "@patch('/todos/" + todo.ID + "')" }
        />
        <span class={ templ.KV("completed", todo.Done) }>
            { todo.Title }
        </span>
        <button
            data-on:click={ "@delete('/todos/" + todo.ID + "')" }
            class="delete-btn"
        >
            x
        </button>
    </div>
}

templ TodoList(todos []Todo) {
    for _, todo := range todos {
        @TodoItem(todo)
    }
    if len(todos) == 0 {
        <p class="empty-state">No todos yet. Add one above!</p>
    }
}`}
      </CodeBlock>

      <h3>Routes</h3>

      <CodeBlock title="app/app.go" language="go">
{`func NewRouter() *router.Router {
    r := router.New()

    r.GET("/", handlers.TodoPage)      // Full page render
    r.DSPost("/todos", handlers.CreateTodo)
    r.DSPatch("/todos/{id}", handlers.ToggleTodo)
    r.DSDelete("/todos/{id}", handlers.DeleteTodo)

    return r
}`}
      </CodeBlock>

      <h2>Search with Debounce</h2>

      <p>
        Real-time search with debounced input using Datastar modifiers.
      </p>

      <CodeBlock title="templates/search.templ" language="templ">
{`templ SearchPage() {
    @Layout("Search") {
        <div class="search-container" data-signals="{query: '', loading: false}">
            <input
                type="text"
                data-bind:query
                data-on:keyup__debounce.300ms="$loading = true; @get('/search')"
                placeholder="Search users..."
            />
            <span data-show="$loading" class="spinner">Searching...</span>
            <div id="results"></div>
        </div>
    }
}

templ SearchResults(users []User, query string) {
    if len(users) == 0 {
        <p class="no-results">No users found for "{ query }"</p>
    } else {
        <ul class="user-list">
            for _, user := range users {
                @UserCard(user)
            }
        </ul>
    }
}`}
      </CodeBlock>

      <CodeBlock title="handlers/search.go" language="go">
{`func Search(ctx *router.Context) error {
    var signals struct {
        Query string \`json:"query"\`
    }
    ctx.ReadSignals(&signals)

    if signals.Query == "" {
        return nil
    }

    users := searchUsers(signals.Query)

    sse := ctx.SSE()
    sse.PatchTempl(templates.SearchResults(users, signals.Query))
    sse.PatchSignals(map[string]any{"loading": false})
    return nil
}`}
      </CodeBlock>

      <h2>Modal Dialog</h2>

      <p>
        A reusable modal pattern using Datastar signals for state.
      </p>

      <CodeBlock title="templates/modal.templ" language="templ">
{`templ ModalContainer() {
    <div data-signals="{showModal: false, modalAction: ''}">
        <div id="modal-container"></div>
    </div>
}

templ ConfirmModal(title string, message string, action string) {
    <div class="modal-backdrop" data-show="$showModal" data-on:click="$showModal = false">
        <div class="modal" data-on:click__stop="">
            <h2>{ title }</h2>
            <p>{ message }</p>
            <div class="modal-actions">
                <button
                    data-on:click={ fmt.Sprintf("@post('%s'); $showModal = false", action) }
                    class="btn-danger"
                >
                    Confirm
                </button>
                <button
                    data-on:click="$showModal = false"
                    class="btn-secondary"
                >
                    Cancel
                </button>
            </div>
        </div>
    </div>
}

templ DeleteUserButton(userID string) {
    <button
        data-on:click={ fmt.Sprintf("$modalAction = '/users/%s'; $showModal = true", userID) }
        class="btn-danger"
    >
        Delete User
    </button>
}`}
      </CodeBlock>

      <CodeBlock title="handlers/modal.go" language="go">
{`func DeleteUser(ctx *router.Context) error {
    userID := ctx.Param("id")
    deleteUser(userID)

    sse := ctx.SSE()
    sse.RemoveElements("#user-" + userID)
    return nil
}`}
      </CodeBlock>

      <h2>Infinite Scroll</h2>

      <p>
        Load more items as the user scrolls using Datastar's intersection observer.
      </p>

      <CodeBlock title="templates/feed.templ" language="templ">
{`templ Feed(posts []Post, page int, hasMore bool) {
    <div class="feed">
        for i, post := range posts {
            if i == len(posts)-1 && hasMore {
                // Last item triggers loading more
                <div data-intersect={ fmt.Sprintf("@get('/feed?page=%d')", page+1) }>
                    @PostCard(post)
                </div>
            } else {
                @PostCard(post)
            }
        }
        if !hasMore && len(posts) > 0 {
            <p class="end-of-feed">You've reached the end!</p>
        }
    </div>
}

templ PostCard(post Post) {
    <article class="post-card">
        <h3>{ post.Title }</h3>
        <p>{ post.Excerpt }</p>
        <time>{ post.CreatedAt.Format("Jan 2, 2006") }</time>
    </article>
}`}
      </CodeBlock>

      <h2>Tabs</h2>

      <p>
        Tab navigation using Datastar signals for active state.
      </p>

      <CodeBlock title="templates/tabs.templ" language="templ">
{`templ TabsPage() {
    @Layout("Dashboard") {
        <div class="tabs-container" data-signals="{activeTab: 'overview'}">
            <nav class="tab-nav">
                <button
                    data-class:active="$activeTab === 'overview'"
                    data-on:click="$activeTab = 'overview'; @get('/dashboard/overview')"
                >
                    Overview
                </button>
                <button
                    data-class:active="$activeTab === 'analytics'"
                    data-on:click="$activeTab = 'analytics'; @get('/dashboard/analytics')"
                >
                    Analytics
                </button>
                <button
                    data-class:active="$activeTab === 'settings'"
                    data-on:click="$activeTab = 'settings'; @get('/dashboard/settings')"
                >
                    Settings
                </button>
            </nav>
            <div id="tab-content">
                <!-- Content loads here -->
            </div>
        </div>
    }
}

templ OverviewTab() {
    <div id="tab-content" class="tab-panel">
        <h2>Overview</h2>
        <p>Your dashboard overview content here.</p>
    </div>
}`}
      </CodeBlock>

      <h2>Running the Examples</h2>

      <p>
        Create a new irgo project and try these patterns:
      </p>

      <CodeBlock language="bash">
{`irgo project new myapp
cd myapp

go mod tidy

irgo server dev`}
      </CodeBlock>

      <h2>Next Steps</h2>

      <ul>
        <li><a href="/docs/getting-started">Getting Started</a> - Create your own project</li>
        <li><a href="/docs/datastar">Datastar Integration</a> - More Datastar patterns</li>
        <li><a href="https://github.com/stukennedy/irgo">GitHub</a> - Browse the source code</li>
      </ul>
    </DocsLayout>
  )
}
