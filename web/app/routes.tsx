// Route definitions for library mode (`createBrowserRouter`).
// The root element is <App /> (from root.tsx) which renders the full
// HTML document (doctype, <html>, <head>, <body>) and exposes the
// Providers + <Outlet/> that every route renders into. The chrome
// (header, nav, footer) lives in <Shell /> as a layout child of App.
//
// Routes are imported eagerly (no lazy splitting) because the SPA bundle
// is small (~800 KB) and eager import keeps the first-paint latency
// minimal at the cost of a single larger JS chunk.

import type { RouteObject } from "react-router";

import App from "./root";
import Shell from "./routes/shell";
import Dashboard from "./routes/_index";
import Tools from "./routes/tools";
import Configure from "./routes/configure";
import Models from "./routes/models";
import Services from "./routes/services";
import Plugins from "./routes/plugins";
import Jobs from "./routes/jobs";
import Logs from "./routes/logs";
import Settings from "./routes/settings";
import NotFound from "./routes/not-found";

export const routes: RouteObject[] = [
  {
    path: "/",
    element: <App />,
    children: [
      {
        element: <Shell />,
        children: [
          {
            index: true,
            element: <Dashboard />,
            handle: { title: "Dashboard" },
          },
          { path: "tools", element: <Tools />, handle: { title: "Tools" } },
          {
            path: "configure",
            element: <Configure />,
            handle: { title: "Configurar" },
          },
          { path: "models", element: <Models />, handle: { title: "Modelos" } },
          {
            path: "services",
            element: <Services />,
            handle: { title: "Servicios" },
          },
          {
            path: "plugins",
            element: <Plugins />,
            handle: { title: "Plugins" },
          },
          { path: "jobs", element: <Jobs />, handle: { title: "Jobs" } },
          { path: "logs", element: <Logs />, handle: { title: "Logs" } },
          {
            path: "settings",
            element: <Settings />,
            handle: { title: "Settings" },
          },
          { path: "*", element: <NotFound />, handle: { title: "404" } },
        ],
      },
    ],
  },
];
