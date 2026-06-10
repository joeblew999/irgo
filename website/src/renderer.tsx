// This file is kept for compatibility but we're using full page components
// that handle their own HTML structure

import { jsxRenderer } from "hono/jsx-renderer";

export const renderer = jsxRenderer(({ children }) => {
  return <>{children}</>;
});
