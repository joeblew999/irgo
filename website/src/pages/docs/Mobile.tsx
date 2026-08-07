import type { FC } from 'hono/jsx'
import { DocsLayout } from '../../components/DocsLayout'
import { CodeBlock } from '../../components/CodeBlock'
import { Callout } from '../../components/FeatureCard'

export const MobilePage: FC = () => {
  return (
    <DocsLayout title="Mobile Apps" currentPath="/docs/mobile">
      <h1>Mobile Apps</h1>

      <p>
        irgo enables you to build native iOS and Android apps using Go, HTML, and Datastar.
        The mobile architecture uses a "virtual HTTP" approach where requests are handled
        in-process without network sockets.
      </p>

      <h2>How Mobile Mode Works</h2>

      <CodeBlock language="text">
{`┌─────────────────────────────────────────┐
│            Mobile App                    │
│  ┌─────────────────────────────────┐    │
│  │     WebView (Datastar)            │    │
│  │  • HTML rendered by Go templates │    │
│  │  • Requests via irgo:// scheme   │    │
│  └──────────────┬──────────────────┘    │
│                 │                        │
│  ┌──────────────▼──────────────────┐    │
│  │   Native Bridge (Swift/Kotlin)   │    │
│  │  • Intercepts irgo:// requests   │    │
│  │  • Routes to Go via gomobile     │    │
│  └──────────────┬──────────────────┘    │
│                 │                        │
│  ┌──────────────▼──────────────────┐    │
│  │     Go Runtime (gomobile bind)   │    │
│  │  • HTTP router (chi)             │    │
│  │  • Template rendering (templ)    │    │
│  │  • Business logic                │    │
│  └─────────────────────────────────┘    │
└─────────────────────────────────────────┘`}
      </CodeBlock>

      <p>Key characteristics:</p>

      <ul>
        <li><strong>No network sockets</strong> - Requests never leave the device</li>
        <li><strong>In-process execution</strong> - Go handlers run directly in the app process</li>
        <li><strong>Custom URL scheme</strong> - Datastar uses <code>irgo://</code> for requests</li>
        <li><strong>gomobile binding</strong> - Go code compiled to native frameworks</li>
      </ul>

      <h2>Prerequisites</h2>

      <h3>Common Requirements</h3>

      <ul>
        <li>Go</li>
        <li>iOS: Xcode — the one thing irgo cannot install for you</li>
        <li>Android: nothing. <code>irgo app build android</code> fetches the JDK, SDK and NDK itself</li>
      </ul>

      <h3>iOS Development</h3>

      <ul>
        <li>macOS with Xcode installed</li>
        <li>iOS Simulator or physical device</li>
        <li>Apple Developer account (for device deployment)</li>
      </ul>

      <h3>Android Development</h3>

      <ul>
        <li>Android Studio with SDK installed</li>
        <li>Android NDK (install via SDK Manager)</li>
        <li>Android Emulator or physical device</li>
      </ul>

      <h2>Mobile Entry Point</h2>

      <p>
        The mobile entry point uses the <code>!desktop</code> build tag:
      </p>

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

      <h2>iOS Development</h2>

      <h3>Development with Hot Reload</h3>

      <CodeBlock language="bash">
{`# Start iOS Simulator with hot reload
irgo app run ios --dev`}
      </CodeBlock>

      <p>
        This watches for changes in your Go and templ files, rebuilds the framework,
        and refreshes the Simulator automatically.
      </p>

      <h3>Production Build</h3>

      <CodeBlock language="bash">
{`# Build iOS framework
irgo app build ios

# Build and run on Simulator
irgo app run ios`}
      </CodeBlock>

      <h3>Build Output</h3>

      <CodeBlock language="text">
{`build/ios/
└── Irgo.xcframework/
    ├── ios-arm64/
    │   └── Irgo.framework/     # Device framework
    └── ios-arm64_x86_64-simulator/
        └── Irgo.framework/     # Simulator framework`}
      </CodeBlock>

      <h3>Xcode Integration</h3>

      <ol>
        <li>Open your Xcode project in <code>ios/</code></li>
        <li>Drag <code>Irgo.xcframework</code> into your project</li>
        <li>Ensure "Embed & Sign" is selected in Frameworks settings</li>
        <li>Build and run from Xcode</li>
      </ol>

      <h2>Android Development</h2>

      <h3>Development with Hot Reload</h3>

      <CodeBlock language="bash">
{`# Start Android Emulator with hot reload
irgo app run android --dev`}
      </CodeBlock>

      <h3>Production Build</h3>

      <CodeBlock language="bash">
{`# Build Android AAR
irgo app build android

# Build and run on Emulator
irgo app run android`}
      </CodeBlock>

      <h3>Build Output</h3>

      <CodeBlock language="text">
{`build/android/
└── irgo.aar                    # Android Archive`}
      </CodeBlock>

      <h3>Android Studio Integration</h3>

      <ol>
        <li>Open your Android project in <code>android/</code></li>
        <li>Copy <code>irgo.aar</code> to <code>app/libs/</code></li>
        <li>Add to <code>build.gradle</code>: <code>implementation files('libs/irgo.aar')</code></li>
        <li>Build and run from Android Studio</li>
      </ol>

      <h2>The Virtual HTTP Adapter</h2>

      <p>
        The mobile bridge uses an HTTP adapter that executes handlers in memory:
      </p>

      <CodeBlock language="go">
{`// The adapter converts core.Request/Response to net/http
// This happens automatically when you call mobile.SetHandler()

import "github.com/stukennedy/irgo/pkg/adapter"

// The adapter executes handlers without network I/O
adapter := adapter.NewHTTPAdapter(handler)

// Native code calls this with request data
response := adapter.HandleRequest(request)`}
      </CodeBlock>

      <Callout type="info">
        <p>
          The adapter uses special <code>core.Request</code> and <code>core.Response</code> types
          that are compatible with gomobile (which doesn't support maps or slices of custom types).
        </p>
      </Callout>

      <h2>Static Assets on Mobile</h2>

      <p>
        Static files are bundled into the native app and served via the Go runtime:
      </p>

      <CodeBlock language="go">
{`// In your router setup
r := router.New()

// Embed static files
//go:embed static
var staticFS embed.FS

// Serve embedded files
r.StaticFS("/static", http.FS(staticFS))`}
      </CodeBlock>

      <h2>Debugging Mobile Apps</h2>

      <h3>iOS Debugging</h3>

      <ol>
        <li>Open Safari and go to Develop menu</li>
        <li>Select your Simulator or device</li>
        <li>Choose your app's WebView</li>
        <li>Use Safari Web Inspector to debug</li>
      </ol>

      <h3>Android Debugging</h3>

      <ol>
        <li>Open Chrome and go to <code>chrome://inspect</code></li>
        <li>Your app's WebView should appear under "Remote Target"</li>
        <li>Click "inspect" to open DevTools</li>
      </ol>

      <Callout type="tip">
        <p>
          Enable WebView debugging in your native code for production builds if needed.
          For security, disable it in release builds.
        </p>
      </Callout>

      <h2>Mobile vs Desktop Comparison</h2>

      <div class="table-wrapper">
        <table class="comparison-table">
          <thead>
            <tr>
              <th>Aspect</th>
              <th>Mobile</th>
              <th>Desktop</th>
            </tr>
          </thead>
          <tbody>
            <tr>
              <td>HTTP</td>
              <td>Virtual (no sockets)</td>
              <td>Real localhost server</td>
            </tr>
            <tr>
              <td>Bridge</td>
              <td>gomobile + native code</td>
              <td>None (direct HTTP)</td>
            </tr>
            <tr>
              <td>Entry Point</td>
              <td><code>main.go</code></td>
              <td><code>main_desktop.go</code></td>
            </tr>
            <tr>
              <td>Build Tag</td>
              <td><code>!desktop</code></td>
              <td><code>desktop</code></td>
            </tr>
            <tr>
              <td>CGO Required</td>
              <td>No</td>
              <td>Yes (webview)</td>
            </tr>
            <tr>
              <td>Build Tool</td>
              <td>gomobile bind</td>
              <td>go build</td>
            </tr>
          </tbody>
        </table>
      </div>

      <h2>Troubleshooting</h2>

      <h3>gomobile not found</h3>

      <CodeBlock language="bash">
{`go install golang.org/x/mobile/cmd/gomobile@latest
gomobile init`}
      </CodeBlock>

      <h3>Android NDK not found</h3>

      <p>Install via Android Studio SDK Manager, or set:</p>

      <CodeBlock language="bash">
{`export ANDROID_HOME=$HOME/Library/Android/sdk
export ANDROID_NDK_HOME=$ANDROID_HOME/ndk/<version>`}
      </CodeBlock>

      <h3>iOS build fails</h3>

      <p>Ensure Xcode CLI tools are installed and selected:</p>

      <CodeBlock language="bash">
{`xcode-select --install
sudo xcode-select -s /Applications/Xcode.app/Contents/Developer`}
      </CodeBlock>

      <h2>Next Steps</h2>

      <ul>
        <li><a href="/docs/desktop">Desktop Apps</a> - Desktop development</li>
        <li><a href="/docs/deployment">Deployment</a> - App store distribution</li>
        <li><a href="/docs/examples">Examples</a> - Sample mobile applications</li>
      </ul>
    </DocsLayout>
  )
}
