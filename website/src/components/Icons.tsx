import type { FC } from "hono/jsx";

type IconProps = {
  size?: number;
  class?: string;
};

// Platform Icons
export const IOSIcon: FC<IconProps> = ({ size = 24, class: className }) => (
  <svg
    width={size}
    height={size}
    viewBox="0 0 24 24"
    fill="none"
    class={className}
  >
    <rect
      x="5"
      y="2"
      width="14"
      height="20"
      rx="3"
      stroke="currentColor"
      stroke-width="1.5"
    />
    <circle cx="12" cy="18" r="1.5" fill="currentColor" />
    <line
      x1="9"
      y1="5"
      x2="15"
      y2="5"
      stroke="currentColor"
      stroke-width="1.5"
      stroke-linecap="round"
    />
  </svg>
);

export const AndroidIcon: FC<IconProps> = ({ size = 24, class: className }) => (
  <svg
    width={size}
    height={size}
    viewBox="0 0 24 24"
    fill="none"
    class={className}
  >
    <path
      d="M6 10V17C6 17.5523 6.44772 18 7 18H8V21C8 21.5523 8.44772 22 9 22C9.55228 22 10 21.5523 10 21V18H14V21C14 21.5523 14.4477 22 15 22C15.5523 22 16 21.5523 16 21V18H17C17.5523 18 18 17.5523 18 17V10H6Z"
      stroke="currentColor"
      stroke-width="1.5"
    />
    <path
      d="M4 10V15C4 15.5523 4.44772 16 5 16C5.55228 16 6 15.5523 6 15V10"
      stroke="currentColor"
      stroke-width="1.5"
      stroke-linecap="round"
    />
    <path
      d="M18 10V15C18 15.5523 18.4477 16 19 16C19.5523 16 20 15.5523 20 15V10"
      stroke="currentColor"
      stroke-width="1.5"
      stroke-linecap="round"
    />
    <path
      d="M6 10C6 7.23858 8.23858 5 11 5H13C15.7614 5 18 7.23858 18 10"
      stroke="currentColor"
      stroke-width="1.5"
    />
    <circle cx="9" cy="7.5" r="0.75" fill="currentColor" />
    <circle cx="15" cy="7.5" r="0.75" fill="currentColor" />
    <path
      d="M7 5L8.5 3"
      stroke="currentColor"
      stroke-width="1.5"
      stroke-linecap="round"
    />
    <path
      d="M17 5L15.5 3"
      stroke="currentColor"
      stroke-width="1.5"
      stroke-linecap="round"
    />
  </svg>
);

export const DesktopIcon: FC<IconProps> = ({ size = 24, class: className }) => (
  <svg
    width={size}
    height={size}
    viewBox="0 0 24 24"
    fill="none"
    class={className}
  >
    <rect
      x="2"
      y="3"
      width="20"
      height="14"
      rx="2"
      stroke="currentColor"
      stroke-width="1.5"
    />
    <path
      d="M8 21H16"
      stroke="currentColor"
      stroke-width="1.5"
      stroke-linecap="round"
    />
    <path
      d="M12 17V21"
      stroke="currentColor"
      stroke-width="1.5"
      stroke-linecap="round"
    />
    <circle cx="12" cy="14" r="1" fill="currentColor" />
  </svg>
);

export const WebIcon: FC<IconProps> = ({ size = 24, class: className }) => (
  <svg
    width={size}
    height={size}
    viewBox="0 0 24 24"
    fill="none"
    class={className}
  >
    <circle cx="12" cy="12" r="10" stroke="currentColor" stroke-width="1.5" />
    <ellipse
      cx="12"
      cy="12"
      rx="4"
      ry="10"
      stroke="currentColor"
      stroke-width="1.5"
    />
    <path
      d="M2 12H22"
      stroke="currentColor"
      stroke-width="1.5"
      stroke-linecap="round"
    />
    <path
      d="M4 7H20"
      stroke="currentColor"
      stroke-width="1.5"
      stroke-linecap="round"
    />
    <path
      d="M4 17H20"
      stroke="currentColor"
      stroke-width="1.5"
      stroke-linecap="round"
    />
  </svg>
);

// Feature Icons
export const BoltIcon: FC<IconProps> = ({ size = 24, class: className }) => (
  <svg
    width={size}
    height={size}
    viewBox="0 0 24 24"
    fill="none"
    class={className}
  >
    <path
      d="M13 2L4 14H11L10 22L20 10H13L14 2H13Z"
      stroke="currentColor"
      stroke-width="1.5"
      stroke-linecap="round"
      stroke-linejoin="round"
    />
  </svg>
);

export const RefreshIcon: FC<IconProps> = ({ size = 24, class: className }) => (
  <svg
    width={size}
    height={size}
    viewBox="0 0 24 24"
    fill="none"
    class={className}
  >
    <path
      d="M4 12C4 7.58172 7.58172 4 12 4C15.0736 4 17.7554 5.64421 19.2 8.12132"
      stroke="currentColor"
      stroke-width="1.5"
      stroke-linecap="round"
    />
    <path
      d="M20 12C20 16.4183 16.4183 20 12 20C8.92644 20 6.24461 18.3558 4.8 15.8787"
      stroke="currentColor"
      stroke-width="1.5"
      stroke-linecap="round"
    />
    <path
      d="M15 8H20V3"
      stroke="currentColor"
      stroke-width="1.5"
      stroke-linecap="round"
      stroke-linejoin="round"
    />
    <path
      d="M9 16H4V21"
      stroke="currentColor"
      stroke-width="1.5"
      stroke-linecap="round"
      stroke-linejoin="round"
    />
  </svg>
);

export const CodeBracketIcon: FC<IconProps> = ({
  size = 24,
  class: className,
}) => (
  <svg
    width={size}
    height={size}
    viewBox="0 0 24 24"
    fill="none"
    class={className}
  >
    <path
      d="M16 18L22 12L16 6"
      stroke="currentColor"
      stroke-width="1.5"
      stroke-linecap="round"
      stroke-linejoin="round"
    />
    <path
      d="M8 6L2 12L8 18"
      stroke="currentColor"
      stroke-width="1.5"
      stroke-linecap="round"
      stroke-linejoin="round"
    />
    <path
      d="M14 4L10 20"
      stroke="currentColor"
      stroke-width="1.5"
      stroke-linecap="round"
    />
  </svg>
);

export const CubeIcon: FC<IconProps> = ({ size = 24, class: className }) => (
  <svg
    width={size}
    height={size}
    viewBox="0 0 24 24"
    fill="none"
    class={className}
  >
    <path
      d="M12 2L3 7V17L12 22L21 17V7L12 2Z"
      stroke="currentColor"
      stroke-width="1.5"
      stroke-linejoin="round"
    />
    <path
      d="M12 12L3 7"
      stroke="currentColor"
      stroke-width="1.5"
      stroke-linecap="round"
    />
    <path
      d="M12 12L21 7"
      stroke="currentColor"
      stroke-width="1.5"
      stroke-linecap="round"
    />
    <path
      d="M12 12V22"
      stroke="currentColor"
      stroke-width="1.5"
      stroke-linecap="round"
    />
  </svg>
);

export const FireIcon: FC<IconProps> = ({ size = 24, class: className }) => (
  <svg
    width={size}
    height={size}
    viewBox="0 0 24 24"
    fill="none"
    class={className}
  >
    <path
      d="M12 22C16.4183 22 20 18.4183 20 14C20 11.5 18.5 9 17 7.5C17 9 16 10 15 10C15 7 13 4 10 2C10 5 8 7 6 9C4.5 10.5 4 12.5 4 14C4 18.4183 7.58172 22 12 22Z"
      stroke="currentColor"
      stroke-width="1.5"
      stroke-linejoin="round"
    />
    <path
      d="M12 22C14.2091 22 16 20.2091 16 18C16 16.5 15.2 15.3 14.5 14.5C14.5 15.3 14 16 13 16C13 14.5 12 13 10 12C10 13.5 9 14.5 8 15.5C7.4 16.1 7 17 7 18C7 20.2091 8.79086 22 11 22"
      stroke="currentColor"
      stroke-width="1.5"
      stroke-linejoin="round"
    />
  </svg>
);

export const TerminalIcon: FC<IconProps> = ({
  size = 24,
  class: className,
}) => (
  <svg
    width={size}
    height={size}
    viewBox="0 0 24 24"
    fill="none"
    class={className}
  >
    <rect
      x="3"
      y="4"
      width="18"
      height="16"
      rx="2"
      stroke="currentColor"
      stroke-width="1.5"
    />
    <path
      d="M7 15H13"
      stroke="currentColor"
      stroke-width="1.5"
      stroke-linecap="round"
    />
    <path
      d="M7 9L10 12L7 15"
      stroke="currentColor"
      stroke-width="1.5"
      stroke-linecap="round"
      stroke-linejoin="round"
    />
  </svg>
);

// Icon mapping for easy lookup
export const platformIcons = {
  ios: IOSIcon,
  android: AndroidIcon,
  desktop: DesktopIcon,
  web: WebIcon,
};

export const featureIcons = {
  bolt: BoltIcon,
  refresh: RefreshIcon,
  code: CodeBracketIcon,
  cube: CubeIcon,
  fire: FireIcon,
  terminal: TerminalIcon,
};
