import type { FC } from "hono/jsx";
import { DocsLayout } from "../../components/DocsLayout";
import { CodeBlock } from "../../components/CodeBlock";
import { Callout } from "../../components/FeatureCard";

export const ProjectStructurePage: FC = () => {
  return (
    <DocsLayout title="Project Structure" currentPath="/docs/project-structure">
      <h1>Project Structure</h1>

      <p>
        An irgo project follows a clean, conventional structure that separates
        concerns while keeping related code together.
      </p>

      <h2>Directory Layout</h2>

      <CodeBlock language="text">
        {`myapp/
├── main.go              # Mobile/web entry point (build tag: !desktop)
├── main_desktop.go      # Desktop entry point (build tag: desktop)
├── go.mod               # Go module definition
├── .air.toml            # Air hot reload configuration
├── package.json         # Node dependencies (Tailwind 4)
│
├── app/
│   └── app.go           # Router setup and app configuration
│
├── handlers/
│   └── handlers.go      # HTTP handlers (business logic)
│
├── templates/
│   ├── layout.templ     # Base HTML layout
│   ├── home.templ       # Home page template
│   └── components.templ # Reusable components
│
├── static/
│   ├── css/
│   │   ├── input.css    # Tailwind source
│   │   └── output.css   # Generated CSS
│   └── js/
│       └── datastar.js  # Datastar library (downloaded automatically)
│
├── mobile/
│   └── mobile.go        # Mobile bridge setup
│
├── ios/                 # iOS Xcode project
├── android/             # Android project
│
└── build/
    ├── ios/             # Built iOS framework
    ├── android/         # Built Android AAR
    └── desktop/         # Built desktop apps
        ├── macos/       # macOS .app bundle
        ├── windows/     # Windows .exe
        └── linux/       # Linux binary`}
      </CodeBlock>

      <h2>Entry Points</h2>

      <p>irgo uses Go build tags to separate platform-specific entry points:</p>

      <h3>Mobile/Web Entry Point</h3>

      <CodeBlock title="main.go" language="go">
        {`//go:build !desktop

package main

import (
    "fmt"
    "log"
    "net/http"
    "os"

    "myapp/app"
    "github.com/stukennedy/irgo/mobile"
)

func main() {
    // If run with "serve" argument, start web dev server
    if len(os.Args) > 1 && os.Args[1] == "serve" {
        runDevServer()
        return
    }
    // Otherwise, initialize for mobile
    initMobile()
}

func initMobile() {
    mobile.Initialize()
    r := app.NewRouter()
    mobile.SetHandler(r.Handler())
    fmt.Println("Mobile app initialized")
}

func runDevServer() {
    r := app.NewRouter()
    mux := http.NewServeMux()
    mux.Handle("/static/", http.StripPrefix("/static/",
        http.FileServer(http.Dir("static"))))
    mux.Handle("/", r.Handler())

    fmt.Println("Dev server at http://localhost:8080")
    log.Fatal(http.ListenAndServe(":8080", mux))
}`}
      </CodeBlock>

      <h3>Desktop Entry Point</h3>

      <CodeBlock title="main_desktop.go" language="go">
        {`//go:build desktop

package main

import (
    "flag"
    "fmt"
    "net/http"

    "myapp/app"
    "github.com/stukennedy/irgo/desktop"
)

func main() {
    devMode := flag.Bool("dev", false, "Enable devtools")
    flag.Parse()

    r := app.NewRouter()

    mux := http.NewServeMux()
    staticDir := desktop.FindStaticDir()
    mux.Handle("/static/", http.StripPrefix("/static/",
        http.FileServer(http.Dir(staticDir))))
    mux.Handle("/", r.Handler())

    config := desktop.DefaultConfig()
    config.Title = "My App"
    config.Debug = *devMode

    desktopApp := desktop.New(mux, config)

    fmt.Println("Starting desktop app...")
    if err := desktopApp.Run(); err != nil {
        fmt.Printf("Error: %v\\n", err)
    }
}`}
      </CodeBlock>

      <Callout type="info" title="Build Tags">
        <p>
          The <code>//go:build !desktop</code> and{" "}
          <code>//go:build desktop</code> tags ensure only the correct entry
          point is compiled for each platform. The irgo CLI handles this
          automatically.
        </p>
      </Callout>

      <h2>App Configuration</h2>

      <p>
        The <code>app/</code> directory contains your router setup and
        application configuration:
      </p>

      <CodeBlock title="app/app.go" language="go">
        {`package app

import (
    "myapp/handlers"
    "github.com/stukennedy/irgo/pkg/render"
    "github.com/stukennedy/irgo/pkg/router"
)

func NewRouter() *router.Router {
    r := router.New()
    renderer := render.NewTemplRenderer()

    // Mount handlers
    handlers.Mount(r, renderer)

    return r
}`}
      </CodeBlock>

      <h2>Handlers</h2>

      <p>
        The <code>handlers/</code> directory contains your HTTP handlers. Group
        related handlers into separate files:
      </p>

      <CodeBlock title="handlers/handlers.go" language="go">
        {`package handlers

import (
    "myapp/templates"
    "github.com/stukennedy/irgo/pkg/render"
    "github.com/stukennedy/irgo/pkg/router"
)

var renderer *render.TemplRenderer

func Mount(r *router.Router, rend *render.TemplRenderer) {
    renderer = rend

    // Pages
    r.GET("/", HomePage)
    r.GET("/about", AboutPage)

    // API routes
    r.Route("/api", func(r *router.Router) {
        r.GET("/users", ListUsers)
        r.POST("/users", CreateUser)
    })
}`}
      </CodeBlock>

      <h2>Templates</h2>

      <p>
        The <code>templates/</code> directory contains your templ templates.
        Organize by feature or page:
      </p>

      <CodeBlock language="text">
        {`templates/
├── layout.templ       # Base HTML structure
├── home.templ         # Home page
├── about.templ        # About page
├── components/
│   ├── nav.templ      # Navigation component
│   ├── footer.templ   # Footer component
│   └── card.templ     # Reusable card component
└── users/
    ├── list.templ     # User list page
    ├── form.templ     # User form
    └── item.templ     # Single user item`}
      </CodeBlock>

      <h2>Static Assets</h2>

      <p>
        The <code>static/</code> directory serves static files like CSS,
        JavaScript, and images:
      </p>

      <CodeBlock language="text">
        {`static/
├── css/
│   ├── input.css      # Tailwind source file
│   └── output.css     # Generated CSS (git-ignored)
├── js/
│   ├── datastar.js    # Datastar library (downloaded by irgo new)
│   └── app.js         # Custom JavaScript (if needed)
└── images/
    └── logo.png       # Static images`}
      </CodeBlock>

      <Callout type="tip">
        <p>
          The <code>output.css</code> is generated by Tailwind during build. Add
          it to
          <code>.gitignore</code> and generate it as part of your build process.
        </p>
      </Callout>

      <h2>Build Output</h2>

      <p>
        Built artifacts are placed in the <code>build/</code> directory:
      </p>

      <CodeBlock language="text">
        {`build/
├── ios/
│   └── Irgo.xcframework/    # iOS framework for Xcode
├── android/
│   └── irgo.aar             # Android archive
└── desktop/
    ├── macos/
    │   └── MyApp.app/       # macOS application bundle
    ├── windows/
    │   └── MyApp.exe        # Windows executable
    └── linux/
        └── myapp            # Linux binary`}
      </CodeBlock>

      <h2>Next Steps</h2>

      <ul>
        <li>
          <a href="/docs/routing">Routing & Handlers</a> - Deep dive into the
          router API
        </li>
        <li>
          <a href="/docs/templates">Templ Templates</a> - Learn template syntax
          and patterns
        </li>
        <li>
          <a href="/docs/desktop">Desktop Apps</a> - Desktop-specific
          configuration
        </li>
      </ul>
    </DocsLayout>
  );
};
