import { isRecord } from "@/lib/guards"
import type { SessionProgress } from "@/lib/shape-session"

/** What the person decided for a question on this page, before it is sent. */
export type Staged = { kind: "assume" } | { kind: "answer"; text: string }

const stagedKey = (id: string) => `span:shape:${id}:staged`
const progressKey = (id: string) => `span:shape:${id}:progress`

/** The stored value for a key; a blocked or empty storage reads as nothing. */
function stored(key: string): unknown {
  try {
    const raw = localStorage.getItem(key)
    return raw === null ? null : JSON.parse(raw)
  } catch {
    return null
  }
}

/** Writes or clears a key. Without storage the page simply forgets on reload. */
function keep(key: string, value: unknown): boolean {
  try {
    if (value === null) {
      localStorage.removeItem(key)
    } else {
      localStorage.setItem(key, JSON.stringify(value))
    }
    return true
  } catch {
    return false
  }
}

/** One stored decision, or nothing when it is not a decision this page makes. */
function staged(value: unknown): Staged | null {
  if (!isRecord(value)) {
    return null
  }
  if (value.kind === "assume") {
    return { kind: "assume" }
  }
  return value.kind === "answer" && typeof value.text === "string"
    ? { kind: "answer", text: value.text }
    : null
}

/** The decisions saved for a session's open questions, so a reload keeps them. */
export function loadStaged(id: string): Record<string, Staged> {
  const value = stored(stagedKey(id))
  if (!isRecord(value)) {
    return {}
  }
  const entries = Object.entries(value).flatMap(([question, item]) => {
    const decision = staged(item)
    return decision === null ? [] : [[question, decision] as const]
  })
  return Object.fromEntries(entries)
}

/** Saves the decisions of a session; an empty set removes them. */
export function saveStaged(id: string, decisions: Record<string, Staged>) {
  return keep(
    stagedKey(id),
    Object.keys(decisions).length > 0 ? decisions : null
  )
}

/** How far the person got on a session: the map read and the context settled. */
export function loadProgress(id: string): SessionProgress {
  const value = stored(progressKey(id))
  if (!isRecord(value)) {
    return {}
  }
  return {
    ...(value.reviewed === true ? { reviewed: true } : {}),
    ...(value.contextDone === true ? { contextDone: true } : {}),
  }
}

/** Saves how far the person got; the Jira preview is never kept. */
export function saveProgress(id: string, progress: SessionProgress) {
  const kept = {
    ...(progress.reviewed === true ? { reviewed: true } : {}),
    ...(progress.contextDone === true ? { contextDone: true } : {}),
  }
  return keep(progressKey(id), Object.keys(kept).length > 0 ? kept : null)
}
