import { useEffect, useState } from "react"

import { getJSON } from "@/lib/api"

/** One line of a run's journal. */
export interface JournalEvent {
  seq: number
  time: string
  actor: string
  type: string
  detail: Record<string, string>
}

/** The journal and whether its hash chain still holds. */
export interface JournalState {
  events: JournalEvent[]
  intact: boolean
  brokenAt: number
}

const empty: JournalState = { events: [], intact: true, brokenAt: 0 }

function text(value: unknown): string {
  return typeof value === "string" ? value : ""
}

function eventOf(value: unknown): JournalEvent | null {
  if (typeof value !== "object" || value === null) {
    return null
  }
  const record: Record<string, unknown> = { ...value }
  const detail: Record<string, string> = {}
  if (typeof record.detail === "object" && record.detail !== null) {
    for (const [key, item] of Object.entries(record.detail)) {
      detail[key] = text(item)
    }
  }
  return {
    seq: typeof record.seq === "number" ? record.seq : 0,
    time: text(record.time),
    actor: text(record.actor),
    type: text(record.type),
    detail,
  }
}

/** Reads a journal payload. Anything that is not the expected shape reads as an empty journal. */
export function journalState(body: unknown): JournalState {
  if (typeof body !== "object" || body === null) {
    return empty
  }
  const record: Record<string, unknown> = { ...body }
  const events = Array.isArray(record.events) ? record.events.map(eventOf).filter((item): item is JournalEvent => item !== null) : []
  return {
    events,
    intact: record.intact !== false,
    brokenAt: typeof record.brokenAt === "number" ? record.brokenAt : 0,
  }
}

/** Reads the run's journal, again when the run moves on to another statement. */
export function useEvents(id: string, moved: number): JournalState {
  const [state, setState] = useState<JournalState>(empty)
  useEffect(() => {
    let closed = false
    void getJSON(`/spans/${id}/events`)
      .then((body) => {
        if (!closed) {
          setState(journalState(body))
        }
      })
      .catch(() => {
        if (!closed) {
          setState(empty)
        }
      })
    return () => {
      closed = true
    }
  }, [id, moved])
  return state
}
