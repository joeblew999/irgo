import { Hono } from "hono";

// Pages
import { HomePage } from "./pages/Home";
import { DemoPage } from "./pages/Demo";
import { IntroductionPage } from "./pages/docs/Introduction";
import { GettingStartedPage } from "./pages/docs/GettingStarted";
import { ProjectStructurePage } from "./pages/docs/ProjectStructure";
import { RoutingPage } from "./pages/docs/Routing";
import { TemplatesPage } from "./pages/docs/Templates";
import { DatastarPage } from "./pages/docs/Datastar";
import { DesktopPage } from "./pages/docs/Desktop";
import { MobilePage } from "./pages/docs/Mobile";
import { WebPage } from "./pages/docs/Web";
import { WebSocketsPage } from "./pages/docs/WebSockets";
import { TestingPage } from "./pages/docs/Testing";
import { DeploymentPage } from "./pages/docs/Deployment";
import { ExamplesPage } from "./pages/docs/Examples";
import { CLIPage } from "./pages/docs/CLI";
import { APIPage } from "./pages/docs/API";

const app = new Hono();

// ═══════════════════════════════════════════════════════════════════════════
// Home
// ═══════════════════════════════════════════════════════════════════════════
app.get("/", (c) => {
  return c.html(<HomePage />);
});

// ═══════════════════════════════════════════════════════════════════════════
// Interactive Demo
// ═══════════════════════════════════════════════════════════════════════════
app.get("/demo", (c) => {
  return c.html(<DemoPage />);
});

// ═══════════════════════════════════════════════════════════════════════════
// Documentation
// ═══════════════════════════════════════════════════════════════════════════

// Getting Started
app.get("/docs", (c) => {
  return c.html(<IntroductionPage />);
});

app.get("/docs/getting-started", (c) => {
  return c.html(<GettingStartedPage />);
});

app.get("/docs/project-structure", (c) => {
  return c.html(<ProjectStructurePage />);
});

// Core Concepts
app.get("/docs/routing", (c) => {
  return c.html(<RoutingPage />);
});

app.get("/docs/templates", (c) => {
  return c.html(<TemplatesPage />);
});

app.get("/docs/datastar", (c) => {
  return c.html(<DatastarPage />);
});

// Platforms
app.get("/docs/desktop", (c) => {
  return c.html(<DesktopPage />);
});

app.get("/docs/mobile", (c) => {
  return c.html(<MobilePage />);
});

app.get("/docs/web", (c) => {
  return c.html(<WebPage />);
});

// Advanced
app.get("/docs/websockets", (c) => {
  return c.html(<WebSocketsPage />);
});

app.get("/docs/testing", (c) => {
  return c.html(<TestingPage />);
});

app.get("/docs/deployment", (c) => {
  return c.html(<DeploymentPage />);
});

// Resources
app.get("/docs/examples", (c) => {
  return c.html(<ExamplesPage />);
});

app.get("/docs/cli", (c) => {
  return c.html(<CLIPage />);
});

app.get("/docs/api", (c) => {
  return c.html(<APIPage />);
});

export default app;
