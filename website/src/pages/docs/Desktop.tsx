import type { FC } from 'hono/jsx'
import { DocsLayout } from '../../components/DocsLayout'
import { CodeBlock } from '../../components/CodeBlock'
import { Callout } from '../../components/FeatureCard'

export const DesktopPage: FC = () => {
  return (
    <DocsLayout title="Desktop Apps" currentPath="/docs/desktop">
      <h1>Desktop Apps</h1>

      <p>
        irgo's desktop mode creates native applications for macOS, Windows, and Linux using
        a real HTTP server and native webview window.
      </p>

      <h2>How Desktop Mode Works</h2>

      <p>Unlike mobile mode (which uses virtual HTTP), desktop mode:</p>

      <ol>
        <li>Starts a Go HTTP server on an auto-selected localhost port</li>
        <li>Opens a native window with an embedded browser engine (WebKit on macOS, Chromium on Windows/Linux)</li>
        <li>The webview navigates to the localhost URL</li>
      </ol>

      <CodeBlock language="text">
{`┌─────────────────────────────────────────┐
│           Desktop App                    │
│  ┌─────────────────────────────────┐    │
│  │    Native Webview Window         │    │
│  │  (Chromium/WebKit)               │    │
│  │  Navigates to localhost:PORT     │    │
│  └──────────────┬──────────────────┘    │
│                 │ HTTP                   │
│  ┌──────────────▼──────────────────┐    │
│  │    Go HTTP Server                │    │
│  │  • Page Routes → HTML            │    │
│  │  • API Routes → JSON             │    │
│  │  • Static Assets                 │    │
│  └─────────────────────────────────┘    │
└─────────────────────────────────────────┘`}
      </CodeBlock>

      <p>
        This architecture means your app works exactly like the web dev server,
        but packaged as a native desktop application.
      </p>

      <h2>Prerequisites</h2>

      <p>Desktop builds require CGO and a C compiler:</p>

      <ul>
        <li><strong>macOS:</strong> Xcode Command Line Tools (<code>xcode-select --install</code>)</li>
        <li><strong>Windows:</strong> MinGW-w64 or similar</li>
        <li><strong>Linux:</strong> GCC and WebKit2GTK (<code>apt install build-essential libwebkit2gtk-4.0-dev</code>)</li>
      </ul>

      <Callout type="warning">
        <p>
          CGO must be enabled. If you see errors about CGO_ENABLED=0, ensure you have
          a C compiler installed and set <code>CGO_ENABLED=1</code>.
        </p>
      </Callout>

      <h2>Desktop Entry Point</h2>

      <p>
        Desktop apps use a separate entry point with the <code>desktop</code> build tag:
      </p>

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

    // Create router
    r := app.NewRouter()

    // Set up HTTP handler with static files
    mux := http.NewServeMux()
    staticDir := desktop.FindStaticDir()
    mux.Handle("/static/", http.StripPrefix("/static/",
        http.FileServer(http.Dir(staticDir))))
    mux.Handle("/", r.Handler())

    // Configure desktop app
    config := desktop.DefaultConfig()
    config.Title = "My App"
    config.Debug = *devMode

    // Create and run
    desktopApp := desktop.New(mux, config)

    fmt.Println("Starting desktop app...")
    if err := desktopApp.Run(); err != nil {
        fmt.Printf("Error: %v\\n", err)
    }
}`}
      </CodeBlock>

      <h2>Configuration</h2>

      <CodeBlock language="go">
{`config := desktop.Config{
    Title:     "My App",     // Window title
    Width:     1024,         // Initial window width
    Height:    768,          // Initial window height
    Resizable: true,         // Allow window resizing
    Debug:     false,        // Enable browser devtools
    Port:      0,            // 0 = auto-select available port
}

// Or start with defaults
config := desktop.DefaultConfig()
config.Title = "My App"
config.Debug = true`}
      </CodeBlock>

      <h3>Configuration Options</h3>

      <div class="table-wrapper">
        <table class="comparison-table">
          <thead>
            <tr>
              <th>Option</th>
              <th>Type</th>
              <th>Default</th>
              <th>Description</th>
            </tr>
          </thead>
          <tbody>
            <tr>
              <td><code>Title</code></td>
              <td>string</td>
              <td>"Irgo App"</td>
              <td>Window title bar text</td>
            </tr>
            <tr>
              <td><code>Width</code></td>
              <td>int</td>
              <td>1024</td>
              <td>Initial window width in pixels</td>
            </tr>
            <tr>
              <td><code>Height</code></td>
              <td>int</td>
              <td>768</td>
              <td>Initial window height in pixels</td>
            </tr>
            <tr>
              <td><code>Resizable</code></td>
              <td>bool</td>
              <td>true</td>
              <td>Allow window resizing</td>
            </tr>
            <tr>
              <td><code>Debug</code></td>
              <td>bool</td>
              <td>false</td>
              <td>Enable browser devtools (right-click inspect)</td>
            </tr>
            <tr>
              <td><code>Port</code></td>
              <td>int</td>
              <td>0</td>
              <td>Server port (0 = auto-select)</td>
            </tr>
          </tbody>
        </table>
      </div>

      <h2>Running Desktop Apps</h2>

      <CodeBlock language="bash">
{`# Run directly (compiles and runs)
irgo run desktop

# With devtools enabled (for debugging)
irgo run desktop --dev`}
      </CodeBlock>

      <h2>Building Desktop Apps</h2>

      <CodeBlock language="bash">
{`# Build for current platform
irgo build desktop

# Build for specific platform
irgo build desktop macos     # Creates build/desktop/macos/MyApp.app
irgo build desktop windows   # Creates build/desktop/windows/MyApp.exe
irgo build desktop linux     # Creates build/desktop/linux/myapp`}
      </CodeBlock>

      <h3>Build Output</h3>

      <CodeBlock language="text">
{`build/desktop/
├── macos/
│   └── MyApp.app/           # macOS application bundle
│       └── Contents/
│           ├── MacOS/
│           │   └── myapp    # Executable
│           ├── Resources/
│           │   └── static/  # Static assets
│           └── Info.plist   # App metadata
├── windows/
│   └── MyApp.exe            # Windows executable
└── linux/
    └── myapp                # Linux binary`}
      </CodeBlock>

      <h2>Utility Functions</h2>

      <CodeBlock language="go">
{`import "github.com/stukennedy/irgo/desktop"

// Find static files directory
// Works both in dev (./static) and bundled app
staticDir := desktop.FindStaticDir()

// Find bundled resources
// Looks in app bundle on macOS, same dir as exe on Windows/Linux
resourcePath := desktop.FindResourcePath("config.json")`}
      </CodeBlock>

      <h2>Desktop vs Mobile Comparison</h2>

      <div class="table-wrapper">
        <table class="comparison-table">
          <thead>
            <tr>
              <th>Aspect</th>
              <th>Desktop</th>
              <th>Mobile</th>
            </tr>
          </thead>
          <tbody>
            <tr>
              <td>HTTP</td>
              <td>Real localhost server</td>
              <td>Virtual (no sockets)</td>
            </tr>
            <tr>
              <td>Bridge</td>
              <td>None (direct HTTP)</td>
              <td>gomobile + native code</td>
            </tr>
            <tr>
              <td>Entry Point</td>
              <td><code>main_desktop.go</code></td>
              <td><code>main.go</code></td>
            </tr>
            <tr>
              <td>Build Tag</td>
              <td><code>desktop</code></td>
              <td><code>!desktop</code></td>
            </tr>
            <tr>
              <td>CGO Required</td>
              <td>Yes (webview)</td>
              <td>No</td>
            </tr>
            <tr>
              <td>DevTools</td>
              <td>Native browser devtools</td>
              <td>Safari/Chrome remote debugging</td>
            </tr>
          </tbody>
        </table>
      </div>

      <h2>Troubleshooting</h2>

      <h3>CGO Errors</h3>

      <p>If you see "CGO_ENABLED=0" errors:</p>

      <CodeBlock language="bash">
{`# Check CGO is enabled
go env CGO_ENABLED

# Set it if needed
export CGO_ENABLED=1

# Make sure you have a C compiler
# macOS
xcode-select --install

# Linux
sudo apt install build-essential`}
      </CodeBlock>

      <h3>Webview Not Showing (Linux)</h3>

      <CodeBlock language="bash">
{`# Install WebKit2GTK
sudo apt install libwebkit2gtk-4.0-dev`}
      </CodeBlock>

      <h3>Static Files Not Found</h3>

      <p>
        Use <code>desktop.FindStaticDir()</code> to locate static files.
        It handles both development and bundled app scenarios.
      </p>

      <h2>Next Steps</h2>

      <ul>
        <li><a href="/docs/mobile">Mobile Apps</a> - iOS and Android development</li>
        <li><a href="/docs/deployment">Deployment</a> - Packaging and distribution</li>
        <li><a href="/docs/examples">Examples</a> - Sample desktop applications</li>
      </ul>
    </DocsLayout>
  )
}
