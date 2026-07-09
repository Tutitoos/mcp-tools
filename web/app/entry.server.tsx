// Server entry for the SSR build. The Vite SSR target compiles this
// file (and its dep graph) into web/build/server/index.js.
//
// Two CLI modes are dispatched on process.argv:
//
//   node <bundle> --serve    (sidecar — long-lived HTTP server on
//                             127.0.0.1:<random>. The Go parent
//                             holds a keep-alive HTTP client and
//                             POSTs each request to /render. This
//                             replaces the old per-request `exec`
//                             model that cost ~8.5s per hit.)
//   node <bundle> <url>      (one-shot — render the URL once and
//                             print to stdout. Kept for external
//                             smoke tests that invoke the bundle
//                             directly without going through Go.)
//
// We render the document body here; the Go side splices it into the
// canonical shell template (build/client/index.html) so the client
// hydrates only the body, not the full <html>.

import { createServer, type IncomingMessage, type ServerResponse } from "node:http";

import { renderToString } from "react-dom/server";
import {
  createStaticHandler,
  createStaticRouter,
  StaticRouterProvider,
} from "react-router";

import { routes } from "./routes";

export default async function render(url: string): Promise<string> {
  const handler = createStaticHandler(routes);
  // Synthetic origin keeps the request inert; only the path is read by
  // the router. Using a non-routable host name avoids leaking the value
  // into logs.
  const request = new Request(`http://internal${url}`);
  const context = await handler.query(request);
  if (context instanceof Response) {
    // 404 or redirect from the route table; let the SPA fallback handle it
    // (we return an empty body so the Go side serves index.html).
    return "";
  }
  const router = createStaticRouter(handler.dataRoutes, context);
  // Return only the <body> content; the Go-side ssrHandler injects it
  // into the canonical document template (build/client/index.html) so
  // the client only hydrates the body. renderToString emits whatever
  // the matched route tree renders. hydrate={false} suppresses the
  // __staticRouterHydrationData <script> tag StaticRouterProvider would
  // otherwise append — the client reads the same data via the
  // hydrationData prop on createBrowserRouter, so emitting a <script>
  // here would cause a hydration mismatch.
  return renderToString(
    <StaticRouterProvider router={router} context={context} hydrate={false} />,
  );
}

function readBody(req: IncomingMessage): Promise<string> {
  const chunks: Buffer[] = [];
  return new Promise<string>((resolve, reject) => {
    req.on("data", (chunk: Buffer) => chunks.push(chunk));
    req.on("end", () => resolve(Buffer.concat(chunks).toString("utf8")));
    req.on("error", (err: Error) => reject(err));
  });
}

function send(res: ServerResponse, status: number, contentType: string, body: string): void {
  res.statusCode = status;
  res.setHeader("content-type", contentType);
  res.end(body);
}

// Sidecar mode: `node <bundle> --serve`. Listens on 127.0.0.1:0 and
// answers POST /render with text/html. The Go parent parses the
// single "READY <address>" line we print on startup to learn our port,
// then keeps a keep-alive HTTP client pointed at us. Per-request
// socket closes are normal (keep-alive reuses) and do NOT kill us —
// the parent kills us explicitly via Process.Kill in Close().
async function startSidecar(): Promise<void> {
  const server = createServer(async (req, res) => {
    try {
      if (req.method !== "POST" || req.url !== "/render") {
        send(res, 404, "text/plain; charset=utf-8", "not found");
        return;
      }
      const url = (await readBody(req)) || "/";
      try {
        const html = await render(url);
        send(res, 200, "text/html; charset=utf-8", html);
      } catch (err: unknown) {
        const msg = err instanceof Error ? err.message : String(err);
        send(res, 500, "text/plain; charset=utf-8", `ssr: ${msg}`);
      }
    } catch (err: unknown) {
      // Defensive: any unexpected error above the per-request try still
      // produces a valid HTTP response instead of a half-written socket.
      const msg = err instanceof Error ? err.message : String(err);
      try {
        send(res, 500, "text/plain; charset=utf-8", `ssr: ${msg}`);
      } catch {
        // socket already closed; nothing to do
      }
    }
  });
  await new Promise<void>((resolve, reject) => {
    const onError = (err: Error) => {
      server.off("listening", onListening);
      reject(err);
    };
    const onListening = () => {
      server.off("error", onError);
      resolve();
    };
    server.once("error", onError);
    server.once("listening", onListening);
    server.listen(0, "127.0.0.1");
  });
  const addr = server.address();
  if (!addr || typeof addr === "string") {
    throw new Error("ssr: sidecar could not determine listening address");
  }
  // Single handshake line so the Go parent can parse it deterministically
  // (see newSSREngine in internal/web/ssr.go). Keep this line byte-for-byte
  // stable: any change must come with a matching change in the Go parser.
  process.stdout.write(`READY 127.0.0.1:${addr.port}\n`);
}

// CLI entry: when this file is loaded as the script entry,
// import.meta.url matches the path Node was invoked with. That works
// regardless of where the file lives on disk (the embed extracts to a
// temp dir, so a regex against the filename is fragile).
const isCLI =
  typeof process !== "undefined" &&
  process.argv[1] !== undefined &&
  import.meta.url === `file://${process.argv[1]}`;
if (isCLI) {
  if (process.argv.includes("--serve")) {
    startSidecar().catch((err: unknown) => {
      const msg = err instanceof Error ? err.message : String(err);
      process.stderr.write(`ssr: sidecar failed to start: ${msg}\n`);
      process.exit(1);
    });
  } else {
    const url = process.argv[2] || "/";
    render(url)
      .then((html) => {
        process.stdout.write(html);
      })
      .catch((err: unknown) => {
        const msg = err instanceof Error ? err.message : String(err);
        process.stderr.write(`ssr: ${msg}\n`);
        process.exit(1);
      });
  }
}
