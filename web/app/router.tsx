// router.tsx -- the single browser router used by the SPA. Library
// mode: no SSR, no prerender; the SPA shell lives in web/index.html and
// the React tree is built entirely in the browser.

import { createBrowserRouter } from "react-router";
import { ErrorBoundary } from "./root";
import { routes } from "./routes";




// Attach the root-level ErrorBoundary as a per-route fallback so any
// uncaught error in any route renders the friendly card instead of
// blowing up the whole tree.
const routesWithBoundary = routes.map((r) => ({
  ...r,
  errorElement: <ErrorBoundary error={undefined} />,
}));

export const router = createBrowserRouter(routesWithBoundary);