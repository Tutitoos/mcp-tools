// Route definitions for library mode (`createBrowserRouter`).
// Each route maps a URL pattern to a component. Nested routes inherit
// their parent's element, so we wrap the chrome in a single parent
// route that renders the <Shell /> layout.
//
// Routes are imported eagerly (no lazy splitting) because the SPA bundle
// is small (~800 KB) and eager import keeps the first-paint latency
// minimal at the cost of a single larger JS chunk.

import type { RouteObject } from "react-router";

import Shell from "./routes/shell";
import Dashboard from "./routes/_index";
import Tools from "./routes/tools";
import Configure from "./routes/configure";
import Models from "./routes/models";
import Services from "./routes/services";
import Logs from "./routes/logs";
import Settings from "./routes/settings";
import NotFound from "./routes/not-found";

export const routes: RouteObject[] = [
  {
    path: "/",
    element: <Shell />,
    children: [
      { index: true, element: <Dashboard /> },
      { path: "tools", element: <Tools /> },
      { path: "configure", element: <Configure /> },
      { path: "models", element: <Models /> },
      { path: "services", element: <Services /> },
      { path: "logs", element: <Logs /> },
      { path: "settings", element: <Settings /> },
    ],
  },
  { path: "*", element: <NotFound /> },
];