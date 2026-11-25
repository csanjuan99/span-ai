import type { SpanHead } from "@/lib/types"

/** A run as the desk lists it, with every field filled so a test names only what it cares about. */
export function head(over: Partial<SpanHead> & { id: string }): SpanHead {
  return {
    label: "",
    issue: "",
    repo: "",
    stage: "build",
    verdict: "running",
    title: "",
    branch: "",
    reason: "",
    updatedAt: "",
    startedAt: "",
    need: "",
    count: 0,
    ...over,
  }
}
