import type { FC } from "hono/jsx";
import { Layout } from "../components/Layout";
import { CodeBlock } from "../components/CodeBlock";

export const DemoPage: FC = () => {
  return (
    <Layout
      title="Interactive Demo - irgo"
      description="Experience server-driven reactivity with irgo and Datastar"
    >
      {/* Animated Background */}
      <div class="demo-bg">
        <div class="demo-bg__gradient demo-bg__gradient--1"></div>
        <div class="demo-bg__gradient demo-bg__gradient--2"></div>
        <div class="demo-bg__gradient demo-bg__gradient--3"></div>
        <div class="demo-bg__grid"></div>
      </div>

      {/* Hero Section */}
      <section class="demo-hero">
        <div class="container">
          <div class="demo-hero__badge">
            <span class="demo-hero__badge-dot"></span>
            Interactive Demo
          </div>
          <h1 class="demo-hero__title">
            <span class="demo-hero__title-line">Server-Driven</span>
            <span class="demo-hero__title-accent">Reactivity</span>
          </h1>
          <p class="demo-hero__subtitle">
            Watch how irgo + Datastar creates seamless, real-time interactions.
            <br />
            All updates flow from Go handlers via SSE streams.
          </p>

          {/* Animated Data Flow Visualization */}
          <div class="demo-flow">
            <DataFlowDiagram />
          </div>
        </div>
      </section>

      {/* Interactive Counter Demo */}
      <section class="demo-section">
        <div class="container">
          <div class="demo-card demo-card--interactive">
            <div class="demo-card__glow"></div>
            <div class="demo-card__content">
              <div class="demo-card__header">
                <h2>Live Counter</h2>
                <span class="demo-card__tag">SSE Response</span>
              </div>
              <p class="demo-card__description">
                Click the button to trigger a Datastar action. The server
                responds with an SSE stream that morphs the DOM.
              </p>

              {/* Counter UI */}
              <div class="counter-demo">
                <div class="counter-demo__display">
                  <div class="counter-demo__value" id="counter-value">
                    <span class="counter-demo__number">0</span>
                  </div>
                  <div class="counter-demo__label">Current Count</div>
                </div>
                <div class="counter-demo__actions">
                  <button
                    class="counter-demo__btn counter-demo__btn--decrement"
                    onclick="decrementCounter()"
                  >
                    <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="3">
                      <path d="M5 12h14" />
                    </svg>
                  </button>
                  <button
                    class="counter-demo__btn counter-demo__btn--increment"
                    onclick="incrementCounter()"
                  >
                    <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="3">
                      <path d="M12 5v14M5 12h14" />
                    </svg>
                  </button>
                </div>
              </div>

              {/* SSE Event Log */}
              <div class="sse-log">
                <div class="sse-log__header">
                  <span class="sse-log__indicator"></span>
                  SSE Event Stream
                </div>
                <div class="sse-log__content" id="sse-log">
                  <div class="sse-log__entry sse-log__entry--info">
                    Waiting for events...
                  </div>
                </div>
              </div>
            </div>
          </div>

          {/* Code Example */}
          <div class="demo-code-split">
            <div class="demo-code-panel">
              <div class="demo-code-panel__label">Go Handler</div>
              <CodeBlock language="go">
{`func Increment(ctx *router.Context) error {
    count++
    return ctx.SSE().PatchTempl(
        templates.Counter(count),
    )
}`}
              </CodeBlock>
            </div>
            <div class="demo-code-panel">
              <div class="demo-code-panel__label">Template</div>
              <CodeBlock language="templ">
{`templ Counter(count int) {
    <div id="counter-value">
        <span>{ strconv.Itoa(count) }</span>
    </div>
}`}
              </CodeBlock>
            </div>
          </div>
        </div>
      </section>

      {/* Morphing Demo */}
      <section class="demo-section demo-section--dark">
        <div class="container">
          <div class="demo-showcase">
            <div class="demo-showcase__visual">
              <MorphingShapeDemo />
            </div>
            <div class="demo-showcase__content">
              <h2>DOM Morphing</h2>
              <p>
                Datastar uses Idiomorph to intelligently update the DOM.
                Elements morph smoothly instead of being replaced, preserving
                scroll position, focus, and CSS animations.
              </p>
              <div class="demo-feature-list">
                <div class="demo-feature-item">
                  <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                    <path d="M22 11.08V12a10 10 0 1 1-5.93-9.14" />
                    <polyline points="22 4 12 14.01 9 11.01" />
                  </svg>
                  <span>Preserves element state</span>
                </div>
                <div class="demo-feature-item">
                  <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                    <path d="M22 11.08V12a10 10 0 1 1-5.93-9.14" />
                    <polyline points="22 4 12 14.01 9 11.01" />
                  </svg>
                  <span>Smooth CSS transitions</span>
                </div>
                <div class="demo-feature-item">
                  <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                    <path d="M22 11.08V12a10 10 0 1 1-5.93-9.14" />
                    <polyline points="22 4 12 14.01 9 11.01" />
                  </svg>
                  <span>No page flicker</span>
                </div>
              </div>
            </div>
          </div>
        </div>
      </section>

      {/* Signals Demo */}
      <section class="demo-section">
        <div class="container">
          <div class="demo-showcase demo-showcase--reverse">
            <div class="demo-showcase__visual">
              <SignalsDemo />
            </div>
            <div class="demo-showcase__content">
              <h2>Reactive Signals</h2>
              <p>
                Datastar signals provide client-side reactivity that syncs with
                server state. Two-way binding keeps your UI and server in perfect
                harmony.
              </p>
              <CodeBlock language="templ">
{`<div data-signals="{name: '', greeting: ''}">
    <input data-bind:name />
    <button data-on:click="@post('/greet')">
        Say Hello
    </button>
    <p data-text="$greeting"></p>
</div>`}
              </CodeBlock>
            </div>
          </div>
        </div>
      </section>

      {/* Real-time Updates Demo */}
      <section class="demo-section demo-section--accent">
        <div class="container">
          <div class="demo-realtime">
            <div class="demo-realtime__header">
              <h2>Real-Time Updates</h2>
              <p>
                Server push via SSE hub. Perfect for live dashboards,
                notifications, and collaborative features.
              </p>
            </div>

            <div class="demo-realtime__grid">
              <RealtimeCard
                title="Live Dashboard"
                icon="chart"
                description="Stream metrics and stats to connected clients"
              />
              <RealtimeCard
                title="Notifications"
                icon="bell"
                description="Push alerts instantly to specific users"
              />
              <RealtimeCard
                title="Collaborative"
                icon="users"
                description="Real-time sync across multiple sessions"
              />
              <RealtimeCard
                title="Chat"
                icon="message"
                description="Build chat rooms with just Go and HTML"
              />
            </div>

            <div class="demo-realtime__code">
              <CodeBlock language="go">
{`// Broadcast to all clients in a room
hub.BroadcastToRoom("dashboard",
    datastar.PatchElements(html),
    datastar.PatchSignals(map[string]any{
        "lastUpdate": time.Now().Format(time.RFC3339),
    }),
)`}
              </CodeBlock>
            </div>
          </div>
        </div>
      </section>

      {/* CTA Section */}
      <section class="demo-cta">
        <div class="container">
          <div class="demo-cta__content">
            <h2>Ready to Build?</h2>
            <p>
              Create your first irgo app in under a minute. Full cross-platform
              support included.
            </p>
            <div class="demo-cta__actions">
              <a href="/docs/getting-started" class="btn btn--primary btn--lg">
                Get Started
                <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                  <path d="M5 12h14M12 5l7 7-7 7" />
                </svg>
              </a>
              <a href="/docs/examples" class="btn btn--secondary btn--lg">
                View Examples
              </a>
            </div>
          </div>
        </div>
      </section>

      {/* Demo Script */}
      <script dangerouslySetInnerHTML={{
        __html: `
          (function() {
            let count = 0;
            let initialized = false;

            function init() {
              if (initialized) return;
              initialized = true;

              // Morphing animation - all paths have 24 points for smooth transitions
              const shapes = document.querySelectorAll('.morph-shape');
              let morphIndex = 0;
              const morphPaths = [
                // Triangle (8 points per edge, sharp corners)
                'M50,10 L55,20 L60,30 L65,40 L70,50 L75,60 L80,70 L85,80 L77,80 L69,80 L61,80 L50,80 L39,80 L31,80 L23,80 L15,80 L20,70 L25,60 L30,50 L35,40 L40,30 L45,20 L50,10 L50,10',
                // Square (6 points per side)
                'M15,15 L29,15 L43,15 L57,15 L71,15 L85,15 L85,29 L85,43 L85,57 L85,71 L85,85 L71,85 L57,85 L43,85 L29,85 L15,85 L15,71 L15,57 L15,43 L15,29 L15,15 L15,15 L15,15 L15,15',
                // Circle (24 points evenly distributed at 15° intervals)
                'M50,8 L62,10 L73,15 L82,23 L89,34 L93,46 L93,58 L89,70 L82,81 L73,89 L62,94 L50,96 L38,94 L27,89 L18,81 L11,70 L7,58 L7,46 L11,34 L18,23 L27,15 L38,10 L50,8 L50,8'
              ];

              if (shapes.length > 0) {
                setInterval(() => {
                  morphIndex = (morphIndex + 1) % morphPaths.length;
                  shapes.forEach(shape => {
                    // Use CSS d property for smooth transitions (modern browsers)
                    shape.style.d = 'path("' + morphPaths[morphIndex] + '")';
                  });
                }, 2000);
              }

              // Signals demo animation
              const signalBars = document.querySelectorAll('.signal-bar');
              if (signalBars.length > 0) {
                setInterval(() => {
                  signalBars.forEach((bar) => {
                    const height = 20 + Math.random() * 60;
                    bar.style.height = height + '%';
                  });
                }, 1000);
              }
            }

            // Counter functions - defined globally for onclick handlers
            window.incrementCounter = function() {
              const logContainer = document.getElementById('sse-log');
              const counterEl = document.querySelector('.counter-demo__number');

              if (logContainer) {
                const entry = document.createElement('div');
                entry.className = 'sse-log__entry sse-log__entry--request';
                entry.innerHTML = '<span class="sse-log__time">' + new Date().toLocaleTimeString() + '</span> POST /api/counter/increment';
                logContainer.insertBefore(entry, logContainer.firstChild);
                while (logContainer.children.length > 5) {
                  logContainer.removeChild(logContainer.lastChild);
                }
              }

              setTimeout(() => {
                count++;

                if (logContainer) {
                  const entry = document.createElement('div');
                  entry.className = 'sse-log__entry sse-log__entry--sse';
                  entry.innerHTML = '<span class="sse-log__time">' + new Date().toLocaleTimeString() + '</span> event: datastar-patch-elements';
                  logContainer.insertBefore(entry, logContainer.firstChild);
                  while (logContainer.children.length > 5) {
                    logContainer.removeChild(logContainer.lastChild);
                  }
                }

                if (counterEl) {
                  counterEl.style.transform = 'scale(1.3)';
                  counterEl.style.filter = 'brightness(1.5) hue-rotate(-20deg)';
                  setTimeout(() => {
                    counterEl.innerText = count.toString();
                    counterEl.style.transform = 'scale(1)';
                    counterEl.style.filter = '';
                  }, 100);
                }
              }, 150);
            };

            window.decrementCounter = function() {
              const logContainer = document.getElementById('sse-log');
              const counterEl = document.querySelector('.counter-demo__number');

              if (logContainer) {
                const entry = document.createElement('div');
                entry.className = 'sse-log__entry sse-log__entry--request';
                entry.innerHTML = '<span class="sse-log__time">' + new Date().toLocaleTimeString() + '</span> POST /api/counter/decrement';
                logContainer.insertBefore(entry, logContainer.firstChild);
                while (logContainer.children.length > 5) {
                  logContainer.removeChild(logContainer.lastChild);
                }
              }

              setTimeout(() => {
                count = Math.max(0, count - 1);

                if (logContainer) {
                  const entry = document.createElement('div');
                  entry.className = 'sse-log__entry sse-log__entry--sse';
                  entry.innerHTML = '<span class="sse-log__time">' + new Date().toLocaleTimeString() + '</span> event: datastar-patch-elements';
                  logContainer.insertBefore(entry, logContainer.firstChild);
                  while (logContainer.children.length > 5) {
                    logContainer.removeChild(logContainer.lastChild);
                  }
                }

                if (counterEl) {
                  counterEl.style.transform = 'scale(1.3)';
                  counterEl.style.filter = 'brightness(1.2) hue-rotate(150deg)';
                  setTimeout(() => {
                    counterEl.innerText = count.toString();
                    counterEl.style.transform = 'scale(1)';
                    counterEl.style.filter = '';
                  }, 100);
                }
              }, 150);
            };

            // Initialize when DOM is ready
            if (document.readyState === 'loading') {
              document.addEventListener('DOMContentLoaded', init);
            } else {
              init();
            }
          })();
        `
      }} />
    </Layout>
  );
};

// Data Flow Diagram Component
const DataFlowDiagram: FC = () => {
  return (
    <svg class="demo-flow__svg" viewBox="0 0 800 200" fill="none">
      {/* Connection Lines */}
      <defs>
        <linearGradient id="flowGradient" x1="0%" y1="0%" x2="100%" y2="0%">
          <stop offset="0%" stop-color="var(--accent-primary)" stop-opacity="0" />
          <stop offset="50%" stop-color="var(--accent-primary)" />
          <stop offset="100%" stop-color="var(--accent-primary)" stop-opacity="0" />
        </linearGradient>
        <filter id="glow">
          <feGaussianBlur stdDeviation="3" result="coloredBlur" />
          <feMerge>
            <feMergeNode in="coloredBlur" />
            <feMergeNode in="SourceGraphic" />
          </feMerge>
        </filter>
      </defs>

      {/* Flow Lines */}
      <path
        class="demo-flow__line"
        d="M180 100 L300 100"
        stroke="url(#flowGradient)"
        stroke-width="2"
        stroke-dasharray="8 4"
      />
      <path
        class="demo-flow__line demo-flow__line--delay-1"
        d="M380 100 L500 100"
        stroke="url(#flowGradient)"
        stroke-width="2"
        stroke-dasharray="8 4"
      />
      <path
        class="demo-flow__line demo-flow__line--delay-2"
        d="M580 100 L700 100"
        stroke="url(#flowGradient)"
        stroke-width="2"
        stroke-dasharray="8 4"
      />

      {/* Data Packets */}
      <circle class="demo-flow__packet" cx="240" cy="100" r="6" fill="var(--accent-primary)" filter="url(#glow)">
        <animate attributeName="cx" values="180;300;180" dur="2s" repeatCount="indefinite" />
        <animate attributeName="opacity" values="0;1;1;0" dur="2s" repeatCount="indefinite" />
      </circle>
      <circle class="demo-flow__packet" cx="440" cy="100" r="6" fill="var(--accent-primary)" filter="url(#glow)">
        <animate attributeName="cx" values="380;500;380" dur="2s" repeatCount="indefinite" begin="0.5s" />
        <animate attributeName="opacity" values="0;1;1;0" dur="2s" repeatCount="indefinite" begin="0.5s" />
      </circle>
      <circle class="demo-flow__packet" cx="640" cy="100" r="6" fill="var(--accent-primary)" filter="url(#glow)">
        <animate attributeName="cx" values="580;700;580" dur="2s" repeatCount="indefinite" begin="1s" />
        <animate attributeName="opacity" values="0;1;1;0" dur="2s" repeatCount="indefinite" begin="1s" />
      </circle>

      {/* Node: User Action */}
      <g class="demo-flow__node">
        <rect x="60" y="60" width="120" height="80" rx="12" fill="var(--bg-tertiary)" stroke="var(--border-default)" stroke-width="1" />
        <text x="120" y="95" text-anchor="middle" fill="var(--text-primary)" font-size="14" font-weight="600">User Action</text>
        <text x="120" y="115" text-anchor="middle" fill="var(--text-tertiary)" font-size="11">data-on:click</text>
      </g>

      {/* Node: Go Handler */}
      <g class="demo-flow__node demo-flow__node--delay-1">
        <rect x="280" y="60" width="120" height="80" rx="12" fill="var(--bg-tertiary)" stroke="var(--accent-primary)" stroke-width="2" />
        <text x="340" y="95" text-anchor="middle" fill="var(--accent-primary)" font-size="14" font-weight="600">Go Handler</text>
        <text x="340" y="115" text-anchor="middle" fill="var(--text-tertiary)" font-size="11">ctx.SSE()</text>
      </g>

      {/* Node: SSE Stream */}
      <g class="demo-flow__node demo-flow__node--delay-2">
        <rect x="480" y="60" width="120" height="80" rx="12" fill="var(--bg-tertiary)" stroke="var(--border-default)" stroke-width="1" />
        <text x="540" y="95" text-anchor="middle" fill="var(--text-primary)" font-size="14" font-weight="600">SSE Stream</text>
        <text x="540" y="115" text-anchor="middle" fill="var(--text-tertiary)" font-size="11">text/event-stream</text>
      </g>

      {/* Node: DOM Update */}
      <g class="demo-flow__node demo-flow__node--delay-3">
        <rect x="680" y="60" width="120" height="80" rx="12" fill="var(--bg-tertiary)" stroke="var(--success)" stroke-width="2" />
        <text x="740" y="95" text-anchor="middle" fill="var(--success)" font-size="14" font-weight="600">DOM Morph</text>
        <text x="740" y="115" text-anchor="middle" fill="var(--text-tertiary)" font-size="11">Idiomorph</text>
      </g>

      {/* Labels */}
      <text x="240" y="165" text-anchor="middle" fill="var(--text-muted)" font-size="10">POST request</text>
      <text x="440" y="165" text-anchor="middle" fill="var(--text-muted)" font-size="10">HTML response</text>
      <text x="640" y="165" text-anchor="middle" fill="var(--text-muted)" font-size="10">Patch elements</text>
    </svg>
  );
};

// Morphing Shape Demo
const MorphingShapeDemo: FC = () => {
  return (
    <div class="morph-demo">
      <svg class="morph-demo__svg" viewBox="0 0 100 100">
        <defs>
          <linearGradient id="morphGradient" x1="0%" y1="0%" x2="100%" y2="100%">
            <stop offset="0%" stop-color="var(--accent-primary)" />
            <stop offset="100%" stop-color="#8b5cf6" />
          </linearGradient>
          <filter id="morphGlow">
            <feGaussianBlur stdDeviation="4" result="coloredBlur" />
            <feMerge>
              <feMergeNode in="coloredBlur" />
              <feMergeNode in="SourceGraphic" />
            </feMerge>
          </filter>
        </defs>
        <path
          class="morph-shape"
          d="M50,10 L55,20 L60,30 L65,40 L70,50 L75,60 L80,70 L85,80 L77,80 L69,80 L61,80 L50,80 L39,80 L31,80 L23,80 L15,80 L20,70 L25,60 L30,50 L35,40 L40,30 L45,20 L50,10 L50,10"
          fill="url(#morphGradient)"
          filter="url(#morphGlow)"
          opacity="0.9"
        />
      </svg>
      <div class="morph-demo__label">Shape morphs smoothly</div>
    </div>
  );
};

// Signals Demo
const SignalsDemo: FC = () => {
  return (
    <div class="signals-demo">
      <div class="signals-demo__bars">
        <div class="signal-bar" style="height: 30%"></div>
        <div class="signal-bar" style="height: 60%"></div>
        <div class="signal-bar" style="height: 45%"></div>
        <div class="signal-bar" style="height: 80%"></div>
        <div class="signal-bar" style="height: 35%"></div>
        <div class="signal-bar" style="height: 55%"></div>
        <div class="signal-bar" style="height: 70%"></div>
        <div class="signal-bar" style="height: 40%"></div>
      </div>
      <div class="signals-demo__label">
        <span class="signals-demo__indicator"></span>
        Signals syncing...
      </div>
    </div>
  );
};

// Realtime Card Component
const RealtimeCard: FC<{
  title: string;
  icon: string;
  description: string;
}> = ({ title, icon, description }) => {
  const icons: Record<string, any> = {
    chart: (
      <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
        <path d="M3 3v18h18" />
        <path d="m19 9-5 5-4-4-3 3" />
      </svg>
    ),
    bell: (
      <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
        <path d="M18 8A6 6 0 0 0 6 8c0 7-3 9-3 9h18s-3-2-3-9" />
        <path d="M13.73 21a2 2 0 0 1-3.46 0" />
      </svg>
    ),
    users: (
      <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
        <path d="M17 21v-2a4 4 0 0 0-4-4H5a4 4 0 0 0-4 4v2" />
        <circle cx="9" cy="7" r="4" />
        <path d="M23 21v-2a4 4 0 0 0-3-3.87" />
        <path d="M16 3.13a4 4 0 0 1 0 7.75" />
      </svg>
    ),
    message: (
      <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
        <path d="M21 15a2 2 0 0 1-2 2H7l-4 4V5a2 2 0 0 1 2-2h14a2 2 0 0 1 2 2z" />
      </svg>
    ),
  };

  return (
    <div class="realtime-card">
      <div class="realtime-card__icon">{icons[icon]}</div>
      <h3 class="realtime-card__title">{title}</h3>
      <p class="realtime-card__description">{description}</p>
      <div class="realtime-card__pulse"></div>
    </div>
  );
};
