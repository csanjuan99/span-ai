import { useEffect, useState } from "react"

import { folderProbeOf, getJSON } from "@/lib/api"
import { reasonText } from "@/lib/guards"
import type { FolderProbe } from "@/lib/types"

/** What is known about the folder: not asked yet, being asked, answered, or unanswerable. */
export type Probe =
  | { state: "idle" }
  | { state: "checking" }
  | { state: "found"; folder: FolderProbe }
  | { state: "failed"; reason: string }

/** Asks the server about a folder a moment after the person stops typing, and again on `again`. */
export function useFolderProbe(folder: string, again: number): Probe {
  const [result, setResult] = useState<Probe>({ state: "idle" })
  const path = folder.trim()

  useEffect(() => {
    if (path === "") {
      return
    }
    let live = true
    const timer = window.setTimeout(() => {
      setResult({ state: "checking" })
      getJSON(`/onboarding/folder?path=${encodeURIComponent(path)}`)
        .then((body) => {
          if (live) {
            setResult({ state: "found", folder: folderProbeOf(body) })
          }
        })
        .catch((reason: unknown) => {
          if (live) {
            setResult({
              state: "failed",
              reason: reasonText(reason, "the folder could not be read"),
            })
          }
        })
    }, 250)
    return () => {
      live = false
      window.clearTimeout(timer)
    }
  }, [path, again])

  return path === "" ? { state: "idle" } : result
}

/** Whether the folder is a git repository Span can work in. */
export function usable(probe: Probe): boolean {
  return probe.state === "found" && probe.folder.exists && probe.folder.git
}
