// router.tsx -- route table factory for library mode + SSR.
//
// The runtime entry on the client is HydratedRouter (entry.client.tsx),
// which reads the server-injected context; on the server, entry.server.tsx
// uses createStaticHandler/createStaticRouter directly and never touches
// this file. This factory remains so unit tests or scripts can build a
// browser router for the same routes in isolation.

import { createBrowserRouter, type RouteObject } from "react-router";
import { ErrorBoundary } from "./root";
import { routes } from "./routes";

// Attach the root-level ErrorBoundary as a per-route fallback so any
// uncaught error in any route renders the friendly card instead of
// blowing up the whole tree.
const routesWithBoundary: RouteObject[] = routes.map((r) => ({
  ...r,
  errorElement: <ErrorBoundary />,
}));

export function createRouter() {
  return createBrowserRouter(routesWithBoundary);
}