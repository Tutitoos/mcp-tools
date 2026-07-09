// Client entry point. With SSR enabled the server has already rendered
// the matched route content into <div id="root"> inside the document
// template, so we hydrate just that subtree. createBrowserRouter + the
// routes used on the server give us the same routing tree; the SSR's
// __staticRouterHydrationData tag is replayed as the initial state.

import { StrictMode, startTransition } from "react";
import { hydrateRoot } from "react-dom/client";
import {
  createBrowserRouter,
  RouterProvider,
} from "react-router";
import { ErrorBoundary } from "./root";
import { routes } from "./routes";

const routesWithBoundary = routes.map((r) => ({
  ...r,
  errorElement: <ErrorBoundary />,
}));

declare global {
  interface Window {
    __staticRouterHydrationData?: unknown;
  }
}

// hydrationData is the union react-router accepts; the global field is
// declared `unknown` and the SSR tag only ever sets loaderData/actionData/
// errors, so the runtime shape always fits.
const router = createBrowserRouter(routesWithBoundary, {
  hydrationData: window.__staticRouterHydrationData as Parameters<
    typeof createBrowserRouter
  >[1] extends infer Opt
    ? Opt extends { hydrationData?: infer H }
      ? H
      : never
    : never,
});

const root = document.getElementById("root");
if (!root) {
  throw new Error("mcp-tools web: missing #root mount point in index.html");
}

startTransition(() => {
  hydrateRoot(
    root,
    <StrictMode>
      <RouterProvider router={router} />
    </StrictMode>,
  );
});