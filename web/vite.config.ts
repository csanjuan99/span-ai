import { resolve } from "node:path"
import tailwindcss from "@tailwindcss/vite"
import react from "@vitejs/plugin-react"
import { defineConfig } from "vite"

// The `span serve` the dev page talks to. Override it to point at another server.
const api = process.env.SPAN_DEV_API ?? "http://127.0.0.1:8787"

// A browser navigation to a page that shares its path with an API route (`/metrics`,
// `/spans/:id`) asks for HTML. That one stays with Vite so the page is the dev build, not
// the one embedded in the binary. Fetches and the socket never ask for HTML.
function keepPagesLocal(request: { headers: { accept?: string } }): string | undefined {
  return request.headers.accept?.includes("text/html") ? "/index.html" : undefined
}

const routes = ["/quickstart", "/repos", "/jira", "/agents", "/branches", "/metrics", "/shapes", "/spans"]

// https://vite.dev/config/
export default defineConfig({
  plugins: [react(), tailwindcss()],
  resolve: {
    alias: {
      "@": resolve(import.meta.dirname, "./src"),
    },
  },
  server: {
    proxy: Object.fromEntries(routes.map((path) => [path, { target: api, ws: path === "/spans", bypass: keepPagesLocal }])),
  },
})
