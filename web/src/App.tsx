import type { ReactNode } from "react"
import { BrowserRouter, Route, Routes, useLocation } from "react-router"

import { ErrorBoundary } from "@/components/error-boundary/ErrorBoundary"
import { Desk } from "@/pages/Desk"
import { Metrics } from "@/pages/Metrics"
import { SpanPage } from "@/pages/SpanPage"

/** One page behind its own boundary; moving to another route clears a caught fault. */
function Guarded({ children }: { children: ReactNode }) {
  const location = useLocation()
  return <ErrorBoundary resetKey={location.pathname}>{children}</ErrorBoundary>
}

/** Desk at the root, one span page for a run. */
export function App() {
  return (
    <BrowserRouter>
      <Routes>
        <Route
          path="/"
          element={
            <Guarded>
              <Desk />
            </Guarded>
          }
        />
        <Route
          path="/metrics"
          element={
            <Guarded>
              <Metrics />
            </Guarded>
          }
        />
        <Route
          path="/spans/:id"
          element={
            <Guarded>
              <SpanPage />
            </Guarded>
          }
        />
      </Routes>
    </BrowserRouter>
  )
}
