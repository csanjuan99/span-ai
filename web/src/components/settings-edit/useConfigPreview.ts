"use client"

import { useEffect, useState } from "react"

import { previewConfig } from "@/lib/api"
import { reasonText } from "@/lib/guards"
import type { ConfigEdits, ConfigEffect, ConfigLine } from "@/lib/types"

/** How long typing pauses before the preview asks the server what would change. */
const settle = 300

/** The lines a save would change, or why the file would refuse them. Nothing is written. */
export interface ConfigPreview {
  lines: ConfigLine[]
  effects: ConfigEffect[]
  error: string
}

const none: ConfigPreview = { lines: [], effects: [], error: "" }

/** Asks the server what saving `edits` would change, once typing has paused. */
export function useConfigPreview(
  project: string,
  edits: ConfigEdits,
  active: boolean
): ConfigPreview {
  const [preview, setPreview] = useState<ConfigPreview>(none)
  const key = JSON.stringify(edits)

  useEffect(() => {
    if (!active) {
      return
    }
    let live = true
    const timer = window.setTimeout(() => {
      void previewConfig(project, JSON.parse(key) as ConfigEdits)
        .then((reply) => {
          if (live) {
            setPreview({ ...reply, error: "" })
          }
        })
        .catch((err: unknown) => {
          if (live) {
            setPreview({
              lines: [],
              effects: [],
              error: reasonText(err, "The change could not be checked"),
            })
          }
        })
    }, settle)
    return () => {
      live = false
      window.clearTimeout(timer)
    }
  }, [project, key, active])

  return active ? preview : none
}
