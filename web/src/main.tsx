import { StrictMode } from "react"
import { createRoot } from "react-dom/client"

import "./index.css"
import { App } from "@/App"
import { ErrorBoundary } from "@/components/error-boundary/ErrorBoundary"
import { ThemeProvider } from "@/components/theme-provider"

const root = document.getElementById("root")
if (!root) {
  throw new Error("root is missing")
}

createRoot(root).render(
  <StrictMode>
    <ThemeProvider>
      <ErrorBoundary>
        <App />
      </ErrorBoundary>
    </ThemeProvider>
  </StrictMode>,
)
