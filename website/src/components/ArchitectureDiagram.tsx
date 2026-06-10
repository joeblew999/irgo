import type { FC } from "hono/jsx";

export const ArchitectureDiagram: FC = () => {
  return (
    <div class="arch-diagram">
      <svg
        viewBox="0 0 800 400"
        fill="none"
        xmlns="http://www.w3.org/2000/svg"
        class="arch-diagram__svg"
      >
        <defs>
          {/* Gradients */}
          <linearGradient id="blueGlow" x1="0%" y1="0%" x2="100%" y2="0%">
            <stop offset="0%" stop-color="#3b82f6" stop-opacity="0" />
            <stop offset="50%" stop-color="#3b82f6" stop-opacity="1" />
            <stop offset="100%" stop-color="#3b82f6" stop-opacity="0" />
          </linearGradient>

          <linearGradient id="goGradient" x1="0%" y1="0%" x2="100%" y2="100%">
            <stop offset="0%" stop-color="#00ADD8" />
            <stop offset="100%" stop-color="#007D9C" />
          </linearGradient>

          <linearGradient id="htmlGradient" x1="0%" y1="0%" x2="100%" y2="100%">
            <stop offset="0%" stop-color="#3b82f6" />
            <stop offset="100%" stop-color="#60a5fa" />
          </linearGradient>

          {/* Glow filter */}
          <filter id="glow" x="-50%" y="-50%" width="200%" height="200%">
            <feGaussianBlur stdDeviation="3" result="coloredBlur" />
            <feMerge>
              <feMergeNode in="coloredBlur" />
              <feMergeNode in="SourceGraphic" />
            </feMerge>
          </filter>

          {/* Animated dash pattern */}
          <pattern
            id="flowPattern"
            patternUnits="userSpaceOnUse"
            width="20"
            height="1"
          >
            <rect width="10" height="1" fill="#3b82f6">
              <animate
                attributeName="x"
                from="0"
                to="20"
                dur="0.5s"
                repeatCount="indefinite"
              />
            </rect>
          </pattern>
        </defs>

        {/* Background grid pattern */}
        <pattern
          id="grid"
          width="40"
          height="40"
          patternUnits="userSpaceOnUse"
        >
          <path
            d="M 40 0 L 0 0 0 40"
            fill="none"
            stroke="rgba(255,255,255,0.03)"
            stroke-width="1"
          />
        </pattern>
        <rect width="100%" height="100%" fill="url(#grid)" />

        {/* Go Runtime Box */}
        <g class="arch-node arch-node--go">
          <rect
            x="50"
            y="140"
            width="180"
            height="120"
            rx="12"
            fill="#18181b"
            stroke="url(#goGradient)"
            stroke-width="2"
          />
          <text x="140" y="175" text-anchor="middle" class="arch-node__title">
            Go Runtime
          </text>

          {/* Go gopher simplified icon */}
          <g transform="translate(115, 190)">
            <ellipse cx="25" cy="25" rx="20" ry="18" fill="#00ADD8" />
            <circle cx="18" cy="22" r="4" fill="white" />
            <circle cx="32" cy="22" r="4" fill="white" />
            <circle cx="18" cy="22" r="2" fill="#0a0a0b" />
            <circle cx="32" cy="22" r="2" fill="#0a0a0b" />
            <ellipse cx="25" cy="32" rx="6" ry="3" fill="#0a0a0b" />
          </g>

          <text
            x="140"
            y="245"
            text-anchor="middle"
            class="arch-node__subtitle"
          >
            Handlers + Templates
          </text>
        </g>

        {/* HTML Partials - floating fragments */}
        <g class="arch-node arch-node--html">
          <rect
            x="310"
            y="100"
            width="180"
            height="200"
            rx="12"
            fill="#18181b"
            stroke="url(#htmlGradient)"
            stroke-width="2"
          />
          <text x="400" y="135" text-anchor="middle" class="arch-node__title">
            HTML Partials
          </text>

          {/* Animated HTML fragments */}
          <g class="html-fragments">
            {/* Fragment 1 */}
            <g class="html-fragment html-fragment--1">
              <rect
                x="325"
                y="150"
                width="150"
                height="30"
                rx="4"
                fill="#1f1f23"
                stroke="#3b82f6"
                stroke-width="1"
              />
              <text x="335" y="170" class="arch-code">
                {"<div>"}Hello{"</div>"}
              </text>
            </g>

            {/* Fragment 2 */}
            <g class="html-fragment html-fragment--2">
              <rect
                x="325"
                y="190"
                width="150"
                height="30"
                rx="4"
                fill="#1f1f23"
                stroke="#3b82f6"
                stroke-width="1"
              />
              <text x="335" y="210" class="arch-code">
                {"<button/>"}
              </text>
            </g>

            {/* Fragment 3 */}
            <g class="html-fragment html-fragment--3">
              <rect
                x="325"
                y="230"
                width="150"
                height="30"
                rx="4"
                fill="#1f1f23"
                stroke="#3b82f6"
                stroke-width="1"
              />
              <text x="335" y="250" class="arch-code">
                {"<li>"}Item{"</li>"}
              </text>
            </g>
          </g>

          <text
            x="400"
            y="285"
            text-anchor="middle"
            class="arch-node__subtitle"
          >
            No JS Framework
          </text>
        </g>

        {/* WebView / Browser */}
        <g class="arch-node arch-node--webview">
          <rect
            x="570"
            y="140"
            width="180"
            height="120"
            rx="12"
            fill="#18181b"
            stroke="#52525b"
            stroke-width="2"
          />
          {/* Browser chrome */}
          <rect
            x="570"
            y="140"
            width="180"
            height="24"
            rx="12"
            fill="#1f1f23"
          />
          <circle cx="586" cy="152" r="4" fill="#ff5f57" />
          <circle cx="600" cy="152" r="4" fill="#febc2e" />
          <circle cx="614" cy="152" r="4" fill="#28c840" />

          <text x="660" y="195" text-anchor="middle" class="arch-node__title">
            WebView
          </text>

          {/* Screen content representation */}
          <rect x="585" y="205" width="60" height="8" rx="2" fill="#3b82f6" />
          <rect
            x="585"
            y="218"
            width="150"
            height="6"
            rx="2"
            fill="#52525b"
          />
          <rect
            x="585"
            y="228"
            width="120"
            height="6"
            rx="2"
            fill="#52525b"
          />
        </g>

        {/* Flow arrows - Go to HTML */}
        <g class="arch-flow">
          <path
            d="M 230 200 C 270 200, 270 200, 310 200"
            stroke="#3b82f6"
            stroke-width="2"
            fill="none"
            stroke-dasharray="8 4"
            class="flow-line flow-line--1"
          />
          <polygon points="305,195 315,200 305,205" fill="#3b82f6" />
        </g>

        {/* Flow arrows - HTML to WebView */}
        <g class="arch-flow">
          <path
            d="M 490 200 C 530 200, 530 200, 570 200"
            stroke="#3b82f6"
            stroke-width="2"
            fill="none"
            stroke-dasharray="8 4"
            class="flow-line flow-line--2"
          />
          <polygon points="565,195 575,200 565,205" fill="#3b82f6" />
        </g>

        {/* Data packets flowing */}
        <g class="data-packet data-packet--1">
          <rect
            x="0"
            y="195"
            width="20"
            height="10"
            rx="2"
            fill="#3b82f6"
            filter="url(#glow)"
          >
            <animateMotion
              path="M 240 0 L 310 0"
              dur="2s"
              repeatCount="indefinite"
              begin="0s"
            />
          </rect>
        </g>

        <g class="data-packet data-packet--2">
          <rect
            x="0"
            y="195"
            width="20"
            height="10"
            rx="2"
            fill="#3b82f6"
            filter="url(#glow)"
          >
            <animateMotion
              path="M 500 0 L 570 0"
              dur="2s"
              repeatCount="indefinite"
              begin="0.5s"
            />
          </rect>
        </g>

        {/* Labels */}
        <text x="270" y="180" class="arch-label">
          Render
        </text>
        <text x="530" y="180" class="arch-label">
          Swap
        </text>

        {/* Bottom annotation */}
        <g class="arch-annotation">
          <line
            x1="50"
            y1="330"
            x2="750"
            y2="330"
            stroke="#3b82f6"
            stroke-width="1"
            stroke-dasharray="4 4"
          />
          <text x="400" y="360" text-anchor="middle" class="arch-callout">
            Thin Client: Server renders HTML, client just displays it
          </text>
        </g>

        {/* Datastar badge */}
        <g transform="translate(355, 50)">
          <rect
            x="0"
            y="0"
            width="90"
            height="24"
            rx="12"
            fill="#3b82f6"
            filter="url(#glow)"
          />
          <text x="45" y="17" text-anchor="middle" class="arch-badge">
            Datastar
          </text>
        </g>

        {/* Comparison indicator */}
        <g class="arch-comparison" transform="translate(50, 370)">
          <rect x="0" y="0" width="8" height="8" fill="#22c55e" rx="2" />
          <text x="15" y="8" class="arch-legend">
            irgo: Go renders → HTML partials → DOM update
          </text>
        </g>

        <g class="arch-comparison" transform="translate(400, 370)">
          <rect x="0" y="0" width="8" height="8" fill="#52525b" rx="2" />
          <text x="15" y="8" class="arch-legend">
            Traditional: API → JSON → JS Framework → Virtual DOM → DOM
          </text>
        </g>
      </svg>
    </div>
  );
};

export const MobileArchitectureDiagram: FC = () => {
  return (
    <div class="arch-diagram arch-diagram--mobile">
      <svg
        viewBox="0 0 280 580"
        fill="none"
        xmlns="http://www.w3.org/2000/svg"
        class="arch-diagram__svg"
      >
        <defs>
          <linearGradient id="mobileGoGradient" x1="0%" y1="0%" x2="100%" y2="100%">
            <stop offset="0%" stop-color="#00ADD8" />
            <stop offset="100%" stop-color="#007D9C" />
          </linearGradient>
          <linearGradient id="mobileBridgeGradient" x1="0%" y1="0%" x2="100%" y2="100%">
            <stop offset="0%" stop-color="#f59e0b" />
            <stop offset="100%" stop-color="#d97706" />
          </linearGradient>
          <filter id="mobileGlow" x="-50%" y="-50%" width="200%" height="200%">
            <feGaussianBlur stdDeviation="2" result="coloredBlur" />
            <feMerge>
              <feMergeNode in="coloredBlur" />
              <feMergeNode in="SourceGraphic" />
            </feMerge>
          </filter>
        </defs>

        {/* Background */}
        <rect width="100%" height="100%" fill="transparent" />

        {/* Phone outline - centered horizontally */}
        <rect
          x="40"
          y="30"
          width="200"
          height="430"
          rx="24"
          fill="#18181b"
          stroke="#3b82f6"
          stroke-width="2"
        />
        <rect x="110" y="45" width="60" height="8" rx="4" fill="#1f1f23" />

        {/* WebView Layer */}
        <g class="mobile-layer mobile-layer--webview">
          <rect
            x="55"
            y="70"
            width="170"
            height="100"
            rx="8"
            fill="#1f1f23"
            stroke="#52525b"
            stroke-width="1"
          />
          <text x="140" y="95" text-anchor="middle" class="arch-node__title">
            WebView
          </text>
          <text x="140" y="115" text-anchor="middle" class="arch-node__subtitle">
            HTML + Datastar
          </text>

          {/* Content lines */}
          <rect x="70" y="130" width="80" height="6" rx="2" fill="#3b82f6" />
          <rect x="70" y="142" width="140" height="4" rx="2" fill="#52525b" />
          <rect x="70" y="152" width="100" height="4" rx="2" fill="#52525b" />
        </g>

        {/* Arrow 1: WebView to Bridge */}
        <g class="mobile-flow">
          <path
            d="M 140 170 L 140 210"
            stroke="#3b82f6"
            stroke-width="2"
            stroke-dasharray="4 2"
          />
          <polygon points="135,207 140,217 145,207" fill="#3b82f6" />
        </g>

        {/* Label for first arrow - irgo:// (inside phone, right of arrow) */}
        <g class="arch-label-group">
          <line x1="145" y1="190" x2="165" y2="190" stroke="#3b82f6" stroke-width="1" stroke-dasharray="2 2" opacity="0.5" />
          <text x="170" y="194" class="arch-label--side">
            irgo://
          </text>
        </g>

        {/* Native Bridge Layer */}
        <g class="mobile-layer mobile-layer--bridge">
          <rect
            x="55"
            y="220"
            width="170"
            height="70"
            rx="8"
            fill="#1f1f23"
            stroke="url(#mobileBridgeGradient)"
            stroke-width="1"
          />
          <text x="140" y="250" text-anchor="middle" class="arch-node__title">
            Native Bridge
          </text>
          <text x="140" y="270" text-anchor="middle" class="arch-node__subtitle">
            Swift / Kotlin
          </text>
        </g>

        {/* Arrow 2: Bridge to Go */}
        <g class="mobile-flow">
          <path
            d="M 140 290 L 140 330"
            stroke="#3b82f6"
            stroke-width="2"
            stroke-dasharray="4 2"
          />
          <polygon points="135,327 140,337 145,327" fill="#3b82f6" />
        </g>

        {/* Label for second arrow - In-process (inside phone, right of arrow) */}
        <g class="arch-label-group">
          <line x1="145" y1="310" x2="158" y2="310" stroke="#3b82f6" stroke-width="1" stroke-dasharray="2 2" opacity="0.5" />
          <text x="163" y="314" class="arch-label--side">
            In-process
          </text>
        </g>

        {/* Go Runtime Layer */}
        <g class="mobile-layer mobile-layer--go">
          <rect
            x="55"
            y="340"
            width="170"
            height="100"
            rx="8"
            fill="#1f1f23"
            stroke="url(#mobileGoGradient)"
            stroke-width="2"
          />
          <text x="140" y="370" text-anchor="middle" class="arch-node__title">
            Go Runtime
          </text>
          <text x="140" y="390" text-anchor="middle" class="arch-node__subtitle">
            gomobile bind
          </text>

          {/* Go gopher mini */}
          <g transform="translate(115, 400)">
            <ellipse cx="25" cy="12" rx="12" ry="10" fill="#00ADD8" />
            <circle cx="20" cy="10" r="2" fill="white" />
            <circle cx="30" cy="10" r="2" fill="white" />
          </g>
        </g>

        {/* Animated data flow */}
        <circle r="4" fill="#3b82f6" filter="url(#mobileGlow)">
          <animateMotion
            path="M 140 170 L 140 220 L 140 290 L 140 340"
            dur="2s"
            repeatCount="indefinite"
          />
        </circle>

        {/* Bottom label */}
        <text x="140" y="500" text-anchor="middle" class="arch-callout">
          No network sockets
        </text>
        <text x="140" y="520" text-anchor="middle" class="arch-callout">
          Virtual HTTP in-process
        </text>

        {/* Platform badge */}
        <g transform="translate(90, 545)">
          <rect
            x="0"
            y="0"
            width="100"
            height="24"
            rx="12"
            fill="#1f1f23"
            stroke="#3b82f6"
            stroke-width="1"
          />
          <text x="50" y="17" text-anchor="middle" class="arch-badge">
            iOS / Android
          </text>
        </g>
      </svg>
    </div>
  );
};
