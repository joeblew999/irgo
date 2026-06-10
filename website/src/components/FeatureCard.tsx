import type { FC } from "hono/jsx";

type FeatureCardProps = {
  icon: any;
  title: string;
  description: string;
};

export const FeatureCard: FC<FeatureCardProps> = ({
  icon,
  title,
  description,
}) => {
  return (
    <div class="feature-card">
      <div class="feature-card__icon">{icon}</div>
      <h3 class="feature-card__title">{title}</h3>
      <p class="feature-card__description">{description}</p>
    </div>
  );
};

type PlatformCardProps = {
  icon: any;
  name: string;
  description: string;
};

export const PlatformCard: FC<PlatformCardProps> = ({
  icon,
  name,
  description,
}) => {
  return (
    <div class="platform-card">
      <div class="platform-card__icon">{icon}</div>
      <h3 class="platform-card__name">{name}</h3>
      <p class="platform-card__desc">{description}</p>
    </div>
  );
};

type CalloutProps = {
  type?: "info" | "warning" | "tip";
  title?: string;
  children: any;
};

export const Callout: FC<CalloutProps> = ({
  type = "info",
  title,
  children,
}) => {
  const titles: Record<string, string> = {
    info: "Note",
    warning: "Warning",
    tip: "Tip",
  };

  return (
    <div class={`callout callout--${type}`}>
      <div class="callout__title">{title || titles[type]}</div>
      <div class="callout__content">{children}</div>
    </div>
  );
};
