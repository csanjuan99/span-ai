"use client"

import { useEffect, useState } from "react"

import {
  noSummary,
  readSealed,
  readSummary,
  type SealedStory,
  type StartSummary,
} from "@/lib/start-info"

/** What a person needs to see before pressing Start: the saved settings and the stories to start from. */
export function useStartInfo(folder: string, kind: string, base: string) {
  const [summary, setSummary] = useState<StartSummary>(noSummary)
  const [sealed, setSealed] = useState<SealedStory[]>([])
  const [problem, setProblem] = useState("")

  useEffect(() => {
    let closed = false
    const timer = window.setTimeout(() => {
      readSummary(folder, kind, base)
        .then((next) => {
          if (!closed) {
            setSummary(next)
          }
        })
        .catch((err: unknown) => {
          if (!closed) {
            setProblem(err instanceof Error ? err.message : "settings not read")
          }
        })
      readSealed(folder)
        .then((next) => {
          if (!closed) {
            setSealed(next)
          }
        })
        .catch((err: unknown) => {
          if (!closed) {
            setProblem(err instanceof Error ? err.message : "stories not read")
          }
        })
    }, 250)
    return () => {
      closed = true
      window.clearTimeout(timer)
    }
  }, [folder, kind, base])

  return { summary, sealed, problem }
}
