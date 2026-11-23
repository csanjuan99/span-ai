import { useEffect, useState } from "react"

import { getJSON } from "@/lib/api"
import { parseFeature, type FeatureReport } from "@/lib/featuredoc"

/** What the feature document looks like to the page: loaded, or not there yet. */
export interface FeatureState {
  feature: FeatureReport | null
  truncated: boolean
}

function markdownOf(body: unknown): { markdown: string; truncated: boolean } {
  if (typeof body !== "object" || body === null) {
    return { markdown: "", truncated: false }
  }
  const record: Record<string, unknown> = { ...body }
  return {
    markdown: typeof record.markdown === "string" ? record.markdown : "",
    truncated: record.truncated === true,
  }
}

/**
 * Reads the run's feature document and reads it again when the commit moves, which is when
 * the build stage has had a chance to write to it. A missing document is a null feature.
 */
export function useFeature(id: string, commit: string): FeatureState {
  const [state, setState] = useState<FeatureState>({ feature: null, truncated: false })
  useEffect(() => {
    let closed = false
    void getJSON(`/spans/${id}/feature`)
      .then((body) => {
        if (closed) {
          return
        }
        const { markdown, truncated } = markdownOf(body)
        setState({ feature: markdown === "" ? null : parseFeature(markdown), truncated })
      })
      .catch(() => {
        if (!closed) {
          setState({ feature: null, truncated: false })
        }
      })
    return () => {
      closed = true
    }
  }, [id, commit])
  return state
}
