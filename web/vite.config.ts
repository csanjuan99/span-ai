import { resolve } from "node:path"
import tailwindcss from "@tailwindcss/vite"
import react from "@vitejs/plugin-react"
import { defineConfig } from "vite"

// https://vite.dev/config/
export default defineConfig({
  plugins: [react(), tailwindcss()],
  resolve: {
    alias: {
      "@": resolve(import.meta.dirname, "./src"),
    },
  },
  server: {
    proxy: {
      "/quickstart": "http://127.0.0.1:8787",
      "/repos": "http://127.0.0.1:8787",
      "/jira": "http://127.0.0.1:8787",
      "/spans": {
        target: "http://127.0.0.1:8787",
        ws: true,
      },
    },
  },
})
