import {
  Links,
  Meta,
  Outlet,
  Scripts,
  ScrollRestoration,
  isRouteErrorResponse,
  useRouteError,
} from "react-router";
import { ThemeProvider } from "next-themes";
import { Toaster } from "sonner";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { useState } from "react";
import type { LinksFunction } from "react-router";
import "./app.css";

type LinkDescriptor = {
  rel: string;
  href: string;
  crossOrigin?: "anonymous" | "use-credentials" | "" | undefined;
};

const preconnectGoogle: LinkDescriptor = {
  rel: "preconnect",
  href: "https://fonts.googleapis.com",
};

const preconnectGstatic: LinkDescriptor = {
  rel: "preconnect",
  href: "https://fonts.gstatic.com",
  crossOrigin: "anonymous",
};

const geistStylesheet: LinkDescriptor = {
  rel: "stylesheet",
  href: "https://fonts.googleapis.com/css2?family=Geist:wght@300;400;500;600;700&family=Geist+Mono:wght@400;500;600&display=swap",
};

export const links: LinksFunction = () => [
  preconnectGoogle,
  preconnectGstatic,
  geistStylesheet,
];

export function Layout({ children }: { children: React.ReactNode }) {
  return (
    <html lang="es" suppressHydrationWarning>
      <head>
        <meta charSet="utf-8" />
        <meta name="viewport" content="width=device-width, initial-scale=1" />
        <Meta />
        <Links />
      </head>
      <body className="min-h-screen bg-background font-sans antialiased">
        <ThemeProvider
          attribute="class"
          defaultTheme="dark"
          enableSystem
          disableTransitionOnChange={false}
        >
          <div className="relative isolate">
            <div className="gradient-mesh pointer-events-none fixed inset-0 -z-10" />
            {children}
            <Toaster richColors position="top-right" />
          </div>
        </ThemeProvider>
        <ScrollRestoration />
        <Scripts />
      </body>
    </html>
  );
}

export default function App() {
  const [queryClient] = useState(
    () =>
      new QueryClient({
        defaultOptions: {
          queries: {
            staleTime: 2_000,
            refetchOnWindowFocus: false,
          },
        },
      }),
  );

  return (
    <QueryClientProvider client={queryClient}>
      <Outlet />
    </QueryClientProvider>
  );
}

export function ErrorBoundary() {
  const error = useRouteError();
  let title = "Algo salió mal";
  let detail = "Error inesperado.";
  if (isRouteErrorResponse(error)) {
    title = `${error.status} ${error.statusText}`;
    detail = typeof error.data === "string" ? error.data : detail;
  } else if (error instanceof Error) {
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