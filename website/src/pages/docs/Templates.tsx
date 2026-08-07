import type { FC } from 'hono/jsx'
import { DocsLayout } from '../../components/DocsLayout'
import { CodeBlock } from '../../components/CodeBlock'
import { Callout } from '../../components/FeatureCard'

export const TemplatesPage: FC = () => {
  return (
    <DocsLayout title="Templ Templates" currentPath="/docs/templates">
      <h1>Templ Templates</h1>

      <p>
        irgo uses <a href="https://templ.guide">templ</a> for type-safe HTML templating.
        Templ compiles to Go code, giving you compile-time error checking and full IDE support.
      </p>

      <h2>Basic Syntax</h2>

      <CodeBlock title="templates/greeting.templ" language="templ">
{`package templates

// Simple component
templ Greeting(name string) {
    <div class="greeting">
        <h1>Hello, { name }!</h1>
    </div>
}`}
      </CodeBlock>

      <p>After running <code>templ generate</code>, this creates <code>greeting_templ.go</code>.</p>

      <h2>Rendering Templates</h2>

      <CodeBlock language="go">
{`import (
    "myapp/templates"
    "github.com/stukennedy/irgo/pkg/render"
)

renderer := render.NewTemplRenderer()

func handler(ctx *router.Context) (string, error) {
    html, err := renderer.Render(templates.Greeting("World"))
    if err != nil {
        return "", err
    }
    return html, nil
}`}
      </CodeBlock>

      <h2>Component Parameters</h2>

      <CodeBlock language="templ">
{`// Multiple parameters
templ UserCard(name string, email string, avatar string) {
    <div class="user-card">
        <img src={ avatar } alt={ name } />
        <h3>{ name }</h3>
        <p>{ email }</p>
    </div>
}

// Struct parameter
templ UserProfile(user User) {
    <div class="profile">
        <h1>{ user.Name }</h1>
        <p>{ user.Email }</p>
        <span>Joined: { user.CreatedAt.Format("Jan 2, 2006") }</span>
    </div>
}`}
      </CodeBlock>

      <h2>Conditionals</h2>

      <CodeBlock language="templ">
{`templ Status(active bool) {
    if active {
        <span class="badge badge-success">Active</span>
    } else {
        <span class="badge badge-danger">Inactive</span>
    }
}

templ UserBadge(user User) {
    <div class="badge">
        if user.IsAdmin {
            <span class="admin">Admin</span>
        } else if user.IsModerator {
            <span class="mod">Moderator</span>
        } else {
            <span class="user">Member</span>
        }
    </div>
}`}
      </CodeBlock>

      <h2>Loops</h2>

      <CodeBlock language="templ">
{`templ UserList(users []User) {
    <ul class="user-list">
        for _, user := range users {
            <li>
                @UserCard(user.Name, user.Email, user.Avatar)
            </li>
        }
    </ul>
}

templ NumberedList(items []string) {
    <ol>
        for i, item := range items {
            <li>{ fmt.Sprintf("%d. %s", i+1, item) }</li>
        }
    </ol>
}`}
      </CodeBlock>

      <h2>Component Composition</h2>

      <p>Use <code>@</code> to include other components:</p>

      <CodeBlock language="templ">
{`templ Layout(title string) {
    <!DOCTYPE html>
    <html>
        <head>
            <title>{ title }</title>
            <link rel="stylesheet" href="/static/css/output.css" />
            <script type="module" src="/static/js/datastar.js"></script>
        </head>
        <body>
            @Header()
            <main>
                { children... }
            </main>
            @Footer()
        </body>
    </html>
}

templ Header() {
    <header class="header">
        <a href="/">Home</a>
        <nav>
            <a href="/about">About</a>
            <a href="/contact">Contact</a>
        </nav>
    </header>
}

templ Footer() {
    <footer class="footer">
        <p>&copy; 2024 My App</p>
    </footer>
}`}
      </CodeBlock>

      <h3>Using Layouts</h3>

      <CodeBlock language="templ">
{`templ HomePage() {
    @Layout("Home") {
        <h1>Welcome!</h1>
        <p>This is the home page.</p>
    }
}

templ AboutPage() {
    @Layout("About") {
        <h1>About Us</h1>
        <p>Learn more about us.</p>
    }
}`}
      </CodeBlock>

      <Callout type="info">
        <p>
          The <code>{`{ children... }`}</code> syntax lets you pass content into a component.
          It's like React's <code>children</code> prop or Vue's slots.
        </p>
      </Callout>

      <h2>Dynamic Attributes</h2>

      <h3>Conditional Attributes</h3>

      <CodeBlock language="templ">
{`// Boolean attribute (checked, disabled, etc.)
templ Checkbox(checked bool) {
    <input type="checkbox" checked?={ checked } />
}

// Dynamic value
templ SelectedOption(value string, selected string) {
    <option value={ value } selected?={ value == selected }>
        { value }
    </option>
}`}
      </CodeBlock>

      <h3>Conditional Classes</h3>

      <CodeBlock language="templ">
{`// Using templ.KV for conditional classes
templ TodoItem(todo Todo) {
    <div class={ "todo-item", templ.KV("completed", todo.Done) }>
        <span class={ templ.KV("line-through", todo.Done) }>
            { todo.Title }
        </span>
    </div>
}

// Multiple conditional classes
templ Button(primary bool, large bool, disabled bool) {
    <button class={
        "btn",
        templ.KV("btn-primary", primary),
        templ.KV("btn-lg", large),
        templ.KV("btn-disabled", disabled),
    } disabled?={ disabled }>
        { children... }
    </button>
}`}
      </CodeBlock>

      <h3>Safe URLs</h3>

      <CodeBlock language="templ">
{`// Use templ.SafeURL for dynamic URLs
templ Link(url string, text string) {
    <a href={ templ.SafeURL(url) }>{ text }</a>
}

// For URLs you construct
templ UserLink(userID string) {
    <a href={ templ.SafeURL("/users/" + userID) }>
        View Profile
    </a>
}`}
      </CodeBlock>

      <h2>Datastar Patterns</h2>

      <h3>Click to Load</h3>

      <CodeBlock language="templ">
{`templ LoadButton() {
    <div data-signals="{loading: false}">
        <button
            data-on:click="$loading = true; @get('/api/data')"
            data-attr:disabled="$loading"
        >
            <span data-show="!$loading">Load Data</span>
            <span data-show="$loading">Loading...</span>
        </button>
        <div id="result"></div>
    </div>
}`}
      </CodeBlock>

      <h3>Form Submission</h3>

      <CodeBlock language="templ">
{`templ TodoForm() {
    <form
        data-signals="{title: ''}"
        data-on:submit__prevent="@post('/todos')"
    >
        <input
            type="text"
            data-bind:title
            placeholder="New todo..."
            required
        />
        <button type="submit">Add</button>
    </form>
    <div id="todo-list"></div>
}`}
      </CodeBlock>

      <h3>Delete with Confirmation</h3>

      <CodeBlock language="templ">
{`templ DeleteButton(id string) {
    <div data-signals="{confirmDelete: false}">
        <button
            data-show="!$confirmDelete"
            data-on:click="$confirmDelete = true"
            class="btn-danger"
        >
            Delete
        </button>
        <div data-show="$confirmDelete">
            <span>Are you sure?</span>
            <button data-on:click={ "@delete('/items/" + id + "')" }>Yes</button>
            <button data-on:click="$confirmDelete = false">No</button>
        </div>
    </div>
}`}
      </CodeBlock>

      <h3>Toggle/Update</h3>

      <CodeBlock language="templ">
{`templ TodoItem(todo Todo) {
    <div id={ "todo-" + todo.ID } class="todo-item">
        <input
            type="checkbox"
            checked?={ todo.Done }
            data-on:click={ "@patch('/todos/" + todo.ID + "')" }
        />
        <span class={ templ.KV("line-through", todo.Done) }>
            { todo.Title }
        </span>
    </div>
}`}
      </CodeBlock>

      <h3>Infinite Scroll</h3>

      <CodeBlock language="templ">
{`templ ItemList(items []Item, page int, hasMore bool) {
    for i, item := range items {
        if i == len(items)-1 && hasMore {
            // Last item triggers loading more
            <div data-intersect={ fmt.Sprintf("@get('/items?page=%d')", page+1) }>
                @ItemCard(item)
            </div>
        } else {
            @ItemCard(item)
        }
    }
}`}
      </CodeBlock>

      <h3>Multiple Updates</h3>

      <CodeBlock language="go">
{`// Update multiple elements with one SSE response
func CreateItem(ctx *router.Context) error {
    item := createItem()
    count := getTotalCount()

    sse := ctx.SSE()

    // Update item list
    sse.PatchTempl(templates.ItemCard(item))

    // Update counter
    sse.PatchTempl(templates.ItemCount(count))

    // Clear input signal
    sse.PatchSignals(map[string]any{"title": ""})

    return nil
}`}
      </CodeBlock>

      <h2>Generating Templates</h2>

      <CodeBlock language="bash">
{`# Generate once
templ generate

# Watch for changes
templ generate --watch

# Or use the irgo CLI
irgo project assets`}
      </CodeBlock>

      <Callout type="tip">
        <p>
          When using <code>irgo server dev</code>, template generation is handled automatically
          via the air configuration.
        </p>
      </Callout>

      <h2>Next Steps</h2>

      <ul>
        <li><a href="/docs/datastar">Datastar Integration</a> - More Datastar patterns and techniques</li>
        <li><a href="/docs/routing">Routing & Handlers</a> - Connect templates to handlers</li>
        <li><a href="https://templ.guide">templ Documentation</a> - Full templ reference</li>
      </ul>
    </DocsLayout>
  )
}
