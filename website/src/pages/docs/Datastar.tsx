import type { FC } from 'hono/jsx'
import { DocsLayout } from '../../components/DocsLayout'
import { CodeBlock } from '../../components/CodeBlock'
import { Callout } from '../../components/FeatureCard'

export const DatastarPage: FC = () => {
  return (
    <DocsLayout title="Datastar Integration" currentPath="/docs/datastar">
      <h1>Datastar Integration</h1>

      <p>
        Datastar is at the heart of irgo's interactivity model. It uses Server-Sent Events (SSE)
        and reactive signals to deliver real-time updates from your Go handlers to the browser.
      </p>

      <h2>The Hypermedia Pattern</h2>

      <p>The traditional SPA approach:</p>
      <CodeBlock language="text">
{`User Click → JavaScript → API Call → JSON Response → JavaScript → DOM Update`}
      </CodeBlock>

      <p>The Datastar/irgo approach:</p>
      <CodeBlock language="text">
{`User Click → Datastar Request → Go Handler → SSE Stream → DOM Morph`}
      </CodeBlock>

      <p>
        Your Go handlers use SSE to stream HTML updates. Datastar morphs them into the DOM.
        No client-side rendering, no state synchronization, no JavaScript frameworks.
      </p>

      <h2>Core Datastar Attributes</h2>

      <h3>Making Requests</h3>

      <CodeBlock language="html">
{`<!-- GET request -->
<button data-on:click="@get('/api/users')">Load Users</button>

<!-- POST request -->
<button data-on:click="@post('/api/users')">Create User</button>

<!-- PUT request -->
<button data-on:click="@put('/api/users/123')">Update</button>

<!-- DELETE request -->
<button data-on:click="@delete('/api/users/123')">Delete</button>

<!-- PATCH request -->
<button data-on:click="@patch('/api/users/123')">Patch</button>`}
      </CodeBlock>

      <h3>Signals (Reactive State)</h3>

      <p>
        Datastar uses signals for client-side reactive state. Define them with{" "}
        <code>data-signals</code>:
      </p>

      <CodeBlock language="html">
{`<!-- Initialize signals -->
<div data-signals="{name: '', count: 0, items: []}">
    <!-- Two-way binding -->
    <input type="text" data-bind:name placeholder="Enter name" />

    <!-- Display signal value -->
    <span data-text="$name"></span>

    <!-- Use in expressions -->
    <span data-text="'Count: ' + $count"></span>
</div>`}
      </CodeBlock>

      <h3>Binding and Display</h3>

      <CodeBlock language="html">
{`<!-- Two-way input binding -->
<input data-bind:email type="email" />

<!-- Checkbox binding -->
<input type="checkbox" data-bind:isActive />

<!-- Display text -->
<span data-text="$username"></span>

<!-- Conditional display -->
<div data-show="$isLoggedIn">Welcome back!</div>

<!-- Dynamic classes -->
<div data-class:active="$isSelected">Item</div>
<div data-class:hidden="!$visible">Content</div>`}
      </CodeBlock>

      <h3>Event Handling</h3>

      <CodeBlock language="html">
{`<!-- Click events -->
<button data-on:click="@post('/api/action')">Submit</button>

<!-- With signal values (sent automatically) -->
<button data-on:click="@post('/api/save')">Save</button>

<!-- Update signals on click -->
<button data-on:click="$count++">Increment</button>

<!-- Keyboard events -->
<input data-on:keyup="@get('/search')" />

<!-- With modifiers -->
<input data-on:keyup__debounce.300ms="@get('/search')" />
<form data-on:submit__prevent="@post('/submit')">

<!-- Mouse events -->
<div data-on:mousemove__throttle.100ms="@get('/track')">
    Track mouse
</div>`}
      </CodeBlock>

      <h3>Event Modifiers</h3>

      <CodeBlock language="html">
{`<!-- Debounce (wait for pause in events) -->
<input data-on:keyup__debounce.300ms="@get('/search')" />

<!-- Throttle (limit event frequency) -->
<div data-on:scroll__throttle.100ms="@get('/load-more')">

<!-- Prevent default -->
<form data-on:submit__prevent="@post('/submit')">

<!-- Stop propagation -->
<button data-on:click__stop="@delete('/item')">

<!-- Once (fire only once) -->
<div data-on:click__once="@get('/init')">

<!-- Combine modifiers -->
<input data-on:keyup__debounce.500ms__prevent="@get('/search')" />`}
      </CodeBlock>

      <h2>Go Handler Patterns</h2>

      <h3>Basic SSE Response</h3>

      <CodeBlock language="go">
{`// Datastar handlers return error (not string, error)
func CreateUser(ctx *router.Context) error {
    var signals struct {
        Name  string \`json:"name"\`
        Email string \`json:"email"\`
    }
    ctx.ReadSignals(&signals)

    user := createUser(signals.Name, signals.Email)

    sse := ctx.SSE()
    return sse.PatchTempl(templates.UserCard(user))
}`}
      </CodeBlock>

      <h3>Multiple Updates</h3>

      <CodeBlock language="go">
{`func ToggleTodo(ctx *router.Context) error {
    id := ctx.Param("id")
    todo := toggleTodo(id)
    count := getTodoCount()

    sse := ctx.SSE()

    // Update the todo item
    sse.PatchTempl(templates.TodoItem(todo))

    // Update the counter (different element)
    sse.PatchTempl(templates.TodoCount(count))

    return nil
}`}
      </CodeBlock>

      <h3>Updating Signals from Server</h3>

      <CodeBlock language="go">
{`func SubmitForm(ctx *router.Context) error {
    var signals struct {
        Title string \`json:"title"\`
    }
    ctx.ReadSignals(&signals)

    // Process the form...
    item := createItem(signals.Title)

    sse := ctx.SSE()

    // Add the new item to DOM
    sse.PatchTempl(templates.ItemCard(item))

    // Clear the input by updating the signal
    sse.PatchSignals(map[string]any{"title": ""})

    return nil
}`}
      </CodeBlock>

      <h3>Removing Elements</h3>

      <CodeBlock language="go">
{`func DeleteItem(ctx *router.Context) error {
    id := ctx.Param("id")
    deleteItem(id)

    sse := ctx.SSE()

    // Remove the element from DOM
    return sse.RemoveElements("#item-" + id)
}`}
      </CodeBlock>

      <h3>Redirects</h3>

      <CodeBlock language="go">
{`func CreateProject(ctx *router.Context) error {
    var signals struct {
        Name string \`json:"name"\`
    }
    ctx.ReadSignals(&signals)

    project := createProject(signals.Name)

    sse := ctx.SSE()
    return sse.Redirect("/projects/" + project.ID)
}`}
      </CodeBlock>

      <h2>Template Patterns</h2>

      <h3>Basic Form</h3>

      <CodeBlock language="templ">
{`templ CreateForm() {
    <form data-signals="{title: ''}" data-on:submit__prevent="@post('/items')">
        <input
            type="text"
            data-bind:title
            placeholder="Enter title"
            required
        />
        <button type="submit">Create</button>
    </form>
    <div id="items-list"></div>
}`}
      </CodeBlock>

      <h3>Todo Item with Actions</h3>

      <CodeBlock language="templ">
{`templ TodoItem(todo Todo) {
    <div id={ "todo-" + todo.ID } class="todo-item">
        <input
            type="checkbox"
            checked?={ todo.Done }
            data-on:click={ "@patch('/todos/" + todo.ID + "')" }
        />
        <span class={ templ.KV("completed", todo.Done) }>
            { todo.Title }
        </span>
        <button data-on:click={ "@delete('/todos/" + todo.ID + "')" }>
            Delete
        </button>
    </div>
}`}
      </CodeBlock>

      <h3>Search with Debounce</h3>

      <CodeBlock language="templ">
{`templ SearchBox() {
    <div data-signals="{query: ''}">
        <input
            type="text"
            data-bind:query
            data-on:keyup__debounce.300ms="@get('/search')"
            placeholder="Search..."
        />
        <div id="search-results"></div>
    </div>
}`}
      </CodeBlock>

      <h3>Loading Indicator</h3>

      <CodeBlock language="templ">
{`templ LoadButton() {
    <div data-signals="{loading: false}">
        <button
            data-on:click="$loading = true; @get('/data').then(() => $loading = false)"
            data-attr:disabled="$loading"
        >
            <span data-show="!$loading">Load Data</span>
            <span data-show="$loading">Loading...</span>
        </button>
    </div>
}`}
      </CodeBlock>

      <Callout type="tip">
        <p>
          With Datastar, you can use signals for UI state (loading, modals, tabs)
          while keeping business data on the server. This hybrid approach gives you
          the best of both worlds.
        </p>
      </Callout>

      <h2>Common Patterns</h2>

      <h3>Modal Dialog</h3>

      <CodeBlock language="templ">
{`templ ModalTrigger() {
    <div data-signals="{showModal: false}">
        <button data-on:click="$showModal = true">
            Open Modal
        </button>

        <div class="modal-backdrop" data-show="$showModal" data-on:click="$showModal = false">
            <div class="modal" data-on:click__stop="">
                <h2>Confirm Action</h2>
                <p>Are you sure?</p>
                <button data-on:click="@post('/confirm'); $showModal = false">
                    Confirm
                </button>
                <button data-on:click="$showModal = false">
                    Cancel
                </button>
            </div>
        </div>
    </div>
}`}
      </CodeBlock>

      <h3>Tabs</h3>

      <CodeBlock language="templ">
{`templ Tabs() {
    <div data-signals="{activeTab: 'overview'}">
        <nav class="tab-nav">
            <button
                data-class:active="$activeTab === 'overview'"
                data-on:click="$activeTab = 'overview'; @get('/tabs/overview')"
            >
                Overview
            </button>
            <button
                data-class:active="$activeTab === 'settings'"
                data-on:click="$activeTab = 'settings'; @get('/tabs/settings')"
            >
                Settings
            </button>
        </nav>
        <div id="tab-content">
            <!-- Content loads here -->
        </div>
    </div>
}`}
      </CodeBlock>

      <h3>Infinite Scroll</h3>

      <CodeBlock language="templ">
{`templ FeedItem(item Item, isLast bool, nextPage int) {
    if isLast {
        <div
            id={ "item-" + item.ID }
            data-intersect={ fmt.Sprintf("@get('/feed?page=%d')", nextPage) }
        >
            @ItemCard(item)
        </div>
    } else {
        <div id={ "item-" + item.ID }>
            @ItemCard(item)
        </div>
    }
}`}
      </CodeBlock>

      <h2>SSE Response Events</h2>

      <p>
        The Datastar Go SDK provides these SSE response methods:
      </p>

      <div class="table-wrapper">
        <table class="comparison-table">
          <thead>
            <tr>
              <th>Method</th>
              <th>Description</th>
            </tr>
          </thead>
          <tbody>
            <tr>
              <td><code>PatchTempl(c)</code></td>
              <td>Morph templ component into DOM</td>
            </tr>
            <tr>
              <td><code>PatchElements(id, html)</code></td>
              <td>Morph raw HTML into element</td>
            </tr>
            <tr>
              <td><code>PatchSignals(map)</code></td>
              <td>Update client-side signals</td>
            </tr>
            <tr>
              <td><code>RemoveElements(selector)</code></td>
              <td>Remove elements from DOM</td>
            </tr>
            <tr>
              <td><code>Redirect(url)</code></td>
              <td>Navigate to new URL</td>
            </tr>
            <tr>
              <td><code>Console(level, msg)</code></td>
              <td>Log to browser console</td>
            </tr>
          </tbody>
        </table>
      </div>

      <h2>Initialization</h2>

      <p>
        Use <code>data-init</code> to run code when an element is added to the DOM:
      </p>

      <CodeBlock language="html">
{`<!-- Load data on page load -->
<div data-init="@get('/api/initial-data')">
    Loading...
</div>

<!-- Initialize signals and fetch -->
<div
    data-signals="{items: []}"
    data-init="@get('/api/items')"
>
    <div id="items-container"></div>
</div>`}
      </CodeBlock>

      <h2>Next Steps</h2>

      <ul>
        <li><a href="/docs/templates">Templ Templates</a> - Template syntax reference</li>
        <li><a href="/docs/routing">Routing & Handlers</a> - Handler patterns</li>
        <li><a href="https://data-star.dev">Datastar Documentation</a> - Full Datastar reference</li>
      </ul>
    </DocsLayout>
  )
}
