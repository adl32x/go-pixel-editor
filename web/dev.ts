// Dev server: Bun's HTML-import bundler serves index.html (with hot reload
// via `bun --hot dev.ts`), while any /api/* request is proxied to the Go
// backend (`pixel serve`, default port 7788) — the dev-time equivalent of
// the reference project's vite.config.ts `server.proxy`.
import index from "./index.html";

const DEV_PORT = 5173;
const API_TARGET = process.env.PIXEL_API_TARGET ?? "http://localhost:7788";

const server = Bun.serve({
  port: DEV_PORT,
  development: true,
  routes: {
    "/": index,
  },
  async fetch(req) {
    const url = new URL(req.url);
    if (url.pathname.startsWith("/api/")) {
      const upstream = new URL(url.pathname + url.search, API_TARGET);
      const hasBody = req.method !== "GET" && req.method !== "HEAD";
      return fetch(upstream, {
        method: req.method,
        headers: req.headers,
        body: hasBody ? await req.arrayBuffer() : undefined,
      });
    }
    return new Response("Not found", { status: 404 });
  },
});

console.log(`dev server: ${server.url} (proxying /api/* -> ${API_TARGET})`);
