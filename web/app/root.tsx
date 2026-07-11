// Root App component used by both SSR (via entry.server.tsx) and client
// hydration (via entry.client.tsx). The component renders only the
// <body> content (chrome + outlet). The <html>/<head> shell lives in
// web/index.html (and the Go-side ssrHandler wraps the SSR output in
// the same template at runtime). This avoids hydration mismatches on
// the <head> — Vite's index.html transform injects CSS links + module
// script tags that must NOT be duplicated by App.
//
// Theme handling:
//   The page is permanently dark. The static class in web/index.html prevents
//   a light first paint, and ThemeProvider keeps the dark class stable.
//   There is no user-selectable light/system theme.

import { useState } from "react";
import { Outlet, useRouteError } from "react-router";
import {
  QueryCache,
  QueryClient,
  QueryClientProvider,
} from "@tanstack/react-query";
import { ThemeProvider } from "next-themes";
import { Toaster, toast } from "sonner";
import { TooltipProvider } from "~/components/ui/tooltip";

import "./app.css";

function Providers({ children }: { children: React.ReactNode }) {
  const [queryClient] = useState(() => {
    const queryCache = new QueryCache({
      onError: (error, query) => {
        // Only toast fetch-loop failures for queries that have never
        // succeeded OR after the second consecutive failure — avoids
        // spamming when the server briefly hiccups.
        if (query.state.data !== undefined && query.state.fetchFailureCount < 2)
          return;
        const msg = error instanceof Error ? error.message : String(error);
        toast.error("Actualización en segundo plano falló", {
          description: `${String(query.queryKey.at(0) ?? "query")} · ${msg}`,
          id: `qerror-${String(query.queryKey.at(0) ?? "query")}`, // dedupe
        });
      },
    });
    return new QueryClient({
      queryCache,
      defaultOptions: {
        queries: {
          staleTime: 2_000,
          refetchOnWindowFocus: false,
        },
      },
    });
  });
  return (
    <ThemeProvider
      attribute="class"
      defaultTheme="dark"
      forcedTheme="dark"
      enableSystem={false}
    >
      <QueryClientProvider client={queryClient}>
        <TooltipProvider delayDuration={200}>
          <div className="relative isolate">
            {children}
            <Toaster richColors position="top-right" />
          </div>
        </TooltipProvider>
      </QueryClientProvider>
    </ThemeProvider>
  );
}

export default function App() {
  return (
    <Providers>
      <Outlet />
    </Providers>
  );
}

export function ErrorBoundary() {
  const error = useRouteError();
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
