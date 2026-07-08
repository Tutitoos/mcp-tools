// Root helpers used by routes. In library mode there's no App shell
// component requirement (the framework's <Layout> export is gone); the
// QueryClient + ThemeProvider live in entry.client.tsx and wrap the
// entire <RouterProvider />.
//
// This file is kept for shared meta/links + ErrorBoundary so routes can
// stay small. ErrorBoundary is wired through createBrowserRouter's
// `errorElement` per-route (or globally on the router) — see router.tsx.

import type { ReactNode } from "react";
import type { LinksFunction, MetaFunction } from "react-router";
import "./app.css";

export const links: LinksFunction = () => [
  { rel: "preconnect", href: "https://fonts.googleapis.com" },
  { rel: "preconnect", href: "https://fonts.gstatic.com", crossOrigin: "anonymous" },
  {
    rel: "stylesheet",
    href: "https://fonts.googleapis.com/css2?family=Geist:wght@300;400;500;600;700&family=Geist+Mono:wght@400;500;600&display=swap",
  },
];

export const meta: MetaFunction = () => [
  { title: "mcp-tools · admin panel" },
  { name: "description", content: "Web admin panel auto-hospedado para el stack MCP" },
  { name: "viewport", content: "width=device-width, initial-scale=1" },
  { charSet: "utf-8" },
];

export function ErrorBoundary({ error }: { error: unknown }) {
  let title = "Algo salió mal";
  let detail = "Error inesperado.";
  if (error instanceof Error) {
    detail = error.message;
  }
  return (
    <div className="flex min-h-screen items-center justify-center p-6">
      <div className="card-vc max-w-lg p-6">
        <h1 className="text-xl font-semibold">{title}</h1>
        <p className="mt-2 text-sm text-muted-foreground">{detail}</p>
      </div>
    </div>
  );
}

// keep ReactNode imported for downstream consumers; silences unused-
// import linters in tools that read this file's symbol exports.
export type { ReactNode };