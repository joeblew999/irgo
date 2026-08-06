import type { FC } from "hono/jsx";
import { DocsLayout } from "../../components/DocsLayout";
import { CodeBlock } from "../../components/CodeBlock";
import { Callout } from "../../components/FeatureCard";

export const GettingStartedPage: FC = () => {
  return (
    <DocsLayout title="Getting Started" currentPath="/docs/getting-started">
      <h1>Getting Started</h1>

      <p>
        This guide will walk you through creating your first irgo application,
        running it in development mode, and building it for different platforms.
      </p>

      <h2>Install the CLI</h2>

      <p>Install the irgo CLI using Go:</p>

      <CodeBlock language="bash">
        {`go install github.com/stukennedy/irgo/cmd/irgo@latest`}
      </CodeBlock>

      <p>Or build from source:</p>

      <CodeBlock language="bash">
        {`git clone https://github.com/stukennedy/irgo.git
cd irgo/cmd/irgo
go install .`}
      </CodeBlock>

      <p>Verify the installation:</p>

      <CodeBlock language="bash">{`irgo version`}</CodeBlock>

      <h2>Create a New Project</h2>

      <CodeBlock language="bash">
        {`irgo project new myapp
cd myapp
go mod tidy`}
      </CodeBlock>

      <p>This creates a new project with the following structure:</p>

      <CodeBlock language="text">
        {`myapp/
├── main.go              # Mobile/web entry point
├── main_desktop.go      # Desktop entry point
├── go.mod               # Go module definition
├── .air.toml            # Hot reload configuration
├── app/
│   └── app.go           # Router setup
├── handlers/
│   └── handlers.go      # HTTP handlers
├── templates/
│   ├── layout.templ     # Base HTML layout
│   ├── home.templ       # Home page template
│   └── components.templ # Reusable components
└── static/
    ├── css/
    │   ├── input.css    # Tailwind source
    │   └── output.css   # Generated CSS
    └── js/
        └── datastar.js  # Datastar library (downloaded automatically)`}
      </CodeBlock>

      <h2>Development Mode</h2>

      <h3>Web Development (Hot Reload)</h3>

      <CodeBlock language="bash">{`irgo server dev`}</CodeBlock>

      <p>
        This starts a development server at <code>http://localhost:8080</code>{" "}
        with hot reload. Edit your Go or templ files and see changes instantly.
      </p>

      <h3>Desktop Development</h3>

      <CodeBlock language="bash">
        {`irgo app run desktop         # Run as desktop app
irgo app run desktop --dev   # With browser devtools enabled`}
      </CodeBlock>

      <h3>iOS Development</h3>

      <CodeBlock language="bash">
        {`irgo app run ios --dev       # Hot reload with iOS Simulator
irgo app run ios             # Production build`}
      </CodeBlock>

      <h3>Android Development</h3>

      <CodeBlock language="bash">
        {`irgo app run android --dev   # Hot reload with Android Emulator
irgo app run android         # Production build`}
      </CodeBlock>

      <h2>Building for Production</h2>

      <h3>Desktop</h3>

      <CodeBlock language="bash">
        {`irgo app build desktop           # Build for current platform
irgo app build desktop macos     # Build macOS .app bundle
irgo app build desktop windows   # Build Windows .exe
irgo app build desktop linux     # Build Linux binary`}
      </CodeBlock>

      <h3>Mobile</h3>

      <CodeBlock language="bash">
        {`irgo app build ios               # Build iOS framework
irgo app build android           # Build Android AAR
irgo app build all               # Build for all mobile platforms`}
      </CodeBlock>

      <h2>Your First Handler</h2>

      <p>Let's create a simple counter to understand the irgo pattern.</p>

      <h3>1. Create the Handler</h3>

      <CodeBlock title="handlers/counter.go" language="go">
        {`package handlers

import (
    "myapp/templates"
    "github.com/stukennedy/irgo/pkg/router"
)

var count = 0

func GetCounter(ctx *router.Context) error {
    return ctx.SSE().PatchTempl(templates.Counter(count))
}

func IncrementCounter(ctx *router.Context) error {
    count++
    return ctx.SSE().PatchTempl(templates.Counter(count))
}

func DecrementCounter(ctx *router.Context) error {
    count--
    return ctx.SSE().PatchTempl(templates.Counter(count))
}`}
      </CodeBlock>

      <h3>2. Create the Template</h3>

      <CodeBlock title="templates/counter.templ" language="templ">
        {`package templates

import "fmt"

templ Counter(count int) {
    <div id="counter" class="counter">
        <h2>Count: { fmt.Sprintf("%d", count) }</h2>
        <div class="buttons">
            <button data-on:click="@post('/counter/decrement')">-</button>
            <button data-on:click="@post('/counter/increment')">+</button>
        </div>
    </div>
}`}
      </CodeBlock>

      <h3>3. Register the Routes</h3>

      <CodeBlock title="app/app.go" language="go">
        {`package app

import (
    "myapp/handlers"
    "github.com/stukennedy/irgo/pkg/router"
)

func NewRouter() *router.Router {
    r := router.New()

    // Counter routes (DS prefix for Datastar SSE handlers)
    r.DSGet("/counter", handlers.GetCounter)
    r.DSPost("/counter/increment", handlers.IncrementCounter)
    r.DSPost("/counter/decrement", handlers.DecrementCounter)

    return r
}`}
      </CodeBlock>

      <h3>4. Generate Templates and Run</h3>

      <CodeBlock language="bash">
        {`irgo project assets      # Generate _templ.go files
irgo server dev        # Start development server`}
      </CodeBlock>

      <p>
        Visit <code>http://localhost:8080/counter</code> - you now have a
        working counter with Datastar-powered interactions!
      </p>

      <Callout type="tip">
        <p>
          Notice how the handler uses SSE to stream HTML updates. Datastar
          morphs the <code>#counter</code> element with the new HTML. This is
          the hypermedia pattern at work.
        </p>
      </Callout>

      <h2>Next Steps</h2>

      <ul>
        <li>
          <a href="/docs/project-structure">Project Structure</a> - Understand
          how irgo projects are organized
        </li>
        <li>
          <a href="/docs/routing">Routing & Handlers</a> - Learn about the
          router and context API
        </li>
        <li>
          <a href="/docs/templates">Templ Templates</a> - Master the templating
          language
        </li>
        <li>
          <a href="/docs/datastar">Datastar Integration</a> - Explore Datastar
          patterns in irgo
        </li>
      </ul>
    </DocsLayout>
  );
};
