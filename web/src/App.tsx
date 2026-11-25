"use client"

import type { ReactNode } from "react"
import {
  BrowserRouter,
  Navigate,
  Route,
  Routes,
  useLocation,
} from "react-router"

import { CommandPalette } from "@/components/help"
import { StartUp } from "@/components/startup/StartUp"
import { ErrorBoundary } from "@/components/error-boundary/ErrorBoundary"
import { needsOnboarding } from "@/lib/onboarding"
import { Desk } from "@/pages/Desk"
import { Metrics } from "@/pages/Metrics"
import { Onboarding } from "@/pages/Onboarding"
import { ProjectSettings } from "@/pages/ProjectSettings"
import { ShapeSession } from "@/pages/ShapeSession"
import { Shaping } from "@/pages/Shaping"
import { SpanPage } from "@/pages/SpanPage"
import { Summary } from "@/pages/Summary"

/** One page behind its own boundary; moving to another route clears a caught fault. */
function Guarded({ children }: { children: ReactNode }) {
  const location = useLocation()
  return <ErrorBoundary resetKey={location.pathname}>{children}</ErrorBoundary>
}

/** A first visit lands on the welcome page; a person who skipped or finished it goes straight to the desk. */
function FirstVisit({ children }: { children: ReactNode }) {
  const location = useLocation()
  if (
    needsOnboarding() &&
    (location.state as { sample?: boolean } | null)?.sample !== true
  ) {
    return <Navigate to="/welcome" replace />
  }
  return children
}

/** Desk at the root, one span page for a run. */
export function App() {
  return (
    <BrowserRouter>
      <StartUp />
      <CommandPalette />
      <Routes>
        <Route
          path="/"
          element={
            <Guarded>
              <FirstVisit>
                <Desk />
              </FirstVisit>
            </Guarded>
          }
        />
        <Route
          path="/welcome"
          element={
            <Guarded>
              <Onboarding />
            </Guarded>
          }
        />
        <Route
          path="/settings"
          element={
            <Guarded>
              <ProjectSettings />
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
          path="/shaping"
          element={
            <Guarded>
              <Shaping />
            </Guarded>
          }
        />
        <Route
          path="/shaping/:id"
          element={
            <Guarded>
              <ShapeSession />
            </Guarded>
          }
        />
        <Route
          path="/spans/:id/summary"
          element={
            <Guarded>
              <Summary />
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
