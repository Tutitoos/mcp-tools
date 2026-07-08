// Client entry point. Library mode: no SSR / no prerender, so we use
// `createRoot` (NOT `hydrateRoot`) and render the RouterProvider with
// the browser router from `./router`. QueryClient + ThemeProvider wrap
// the entire route tree so every page can use useQuery / useTheme.

import { StrictMode, useEffect, useState } from "react";
import { createRoot } from "react-dom/client";
import { RouterProvider } from "react-router";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { ThemeProvider } from "next-themes";
import { Toaster } from "sonner";

import { router } from "./router";

const root = document.getElementById("root");
if (!root) {
  throw new Error("mcp-tools web: missing #root mount point in index.html");
}

// AuthGate listens for the `mcp-tools:unauthorized` event that the API
// client fires on 401, and redirects the browser to /setup. We use
// window.location rather than react-router's navigate because this
// component lives ABOVE <RouterProvider /> in the tree and navigate()
// requires router context. A full-page nav also clears in-flight
// queries, which is what we want (the token is gone, every query will
// 401 until the user re-authenticates).
function AuthGate({ children }: { children: React.ReactNode }) {
  useEffect(() => {
    function onUnauth() {
      const at = window.location.pathname;
      if (at !== "/setup") window.location.replace("/setup");
    }
    window.addEventListener("mcp-tools:unauthorized", onUnauth);
    return () => window.removeEventListener("mcp-tools:unauthorized", onUnauth);
  }, []);
  return <>{children}</>;
}

function Providers({ children }: { children: React.ReactNode }) {
  const [queryClient] = useState(
    () =>
      new QueryClient({
        defaultOptions: {
          queries: {
            staleTime: 2_000,
            refetchOnWindowFocus: false,
            // 401s are terminal: the API client already cleared the
            // token and emitted mcp-tools:unauthorized. Retrying just
            // spams the network and floods the console. 403 is treated
            // the same way (token forbidden). Other statuses (5xx,
            // network error) get the default 3 retries.
            retry: (failureCount, error) => {
              if (error && typeof error === "object" && "status" in error) {
                const status = (error as { status: number }).status;
                if (status === 401 || status === 403) return false;
              }
              return failureCount < 3;
            },
          },
        },
      }),
  );
  return (
    <QueryClientProvider client={queryClient}>
      <ThemeProvider
        attribute="class"
        defaultTheme="dark"
        enableSystem
        disableTransitionOnChange={false}
      >
        <div className="relative isolate">
          <div className="gradient-mesh pointer-events-none fixed inset-0 -z-10" />
          <AuthGate>{children}</AuthGate>
          <Toaster richColors position="top-right" />
        </div>
      </ThemeProvider>
    </QueryClientProvider>
  );
}

createRoot(root).render(
  <StrictMode>
    <Providers>
      <RouterProvider router={router} />
    </Providers>
  </StrictMode>,
);