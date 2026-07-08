// Client entry point. Library mode: no SSR / no prerender, so we use
// `createRoot` (NOT `hydrateRoot`) and render the RouterProvider with
// the browser router from `./router`. QueryClient + ThemeProvider wrap
// the entire route tree so every page can use useQuery / useTheme.

import { StrictMode, useState } from "react";
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

function Providers({ children }: { children: React.ReactNode }) {
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