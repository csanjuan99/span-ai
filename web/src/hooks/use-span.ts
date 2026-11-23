import { useCallback, useEffect, useState } from "react"

import { acceptanceView, agentStatus, getJSON, implementationView, liveFrame, snapshot, socketProtocol } from "@/lib/api"
import { agentReviewMs, type TimelineAction, type TimelineFrame } from "@/lib/timeline"
import type { AgentStatus, AcceptanceView, ImplementationView, LiveFrame, Snapshot, Statement } from "@/lib/types"

/** What the span page renders. The socket updates this object. The disk remains the record. */
export interface SpanState {
  span: Snapshot | null
  acceptance: AcceptanceView | null
  implementation: ImplementationView | null
  transcript: string[]
  frames: TimelineFrame[]
  actions: TimelineAction[]
  agent: AgentStatus | null
  agentAt: string
  note: (text: string) => void
  error: string
  connected: boolean
}

function upsert(statements: Statement[], next: Statement): Statement[] {
  const kept = statements.filter(
    (item) => !(item.stage === next.stage && item.iteration === next.iteration && item.digest === next.digest),
  )
  return [...kept, next]
}

/** Loads one run and applies live frames onto it. A refresh replays the sealed record. */
export function useSpan(id: string): SpanState {
  const [span, setSpan] = useState<Snapshot | null>(null)
  const [acceptance, setAcceptance] = useState<AcceptanceView | null>(null)
  const [implementation, setImplementation] = useState<ImplementationView | null>(null)
  const [transcript, setTranscript] = useState<string[]>([])
  const [frames, setFrames] = useState<TimelineFrame[]>([])
  const [actions, setActions] = useState<TimelineAction[]>([])
  const [agent, setAgent] = useState<AgentStatus | null>(null)
  const [agentAt, setAgentAt] = useState("")
  const [tracked, setTracked] = useState(id)
  const [error, setError] = useState("")
  const [connected, setConnected] = useState(false)
  if (tracked !== id) {
    setTracked(id)
    setFrames([])
    setActions([])
    setTranscript([])
    setSpan(null)
    setAgent(null)
    setAgentAt("")
  }
  const note = useCallback((text: string) => {
    setActions((current) => [...current, { at: new Date().toISOString(), text }])
  }, [])
  const remember = useCallback((body: LiveFrame) => {
    setFrames((current) => [...current, { at: new Date().toISOString(), frame: body }])
  }, [])

  useEffect(() => {
    let closed = false
    let seenSHA = ""
    let diffToken = 0
    const protocol = socketProtocol()
    const address = `${location.protocol === "https:" ? "wss" : "ws"}://${location.host}/spans/${id}/live`
    const socket = protocol === "" ? new WebSocket(address) : new WebSocket(address, [protocol])
    const applyDiff = (body: unknown, token: number) => {
      if (!closed && token === diffToken) {
        setImplementation(implementationView(body))
      }
    }
    const loadDiff = (sha: string) => {
      if (sha === "" || sha === seenSHA) {
        return
      }
      seenSHA = sha
      diffToken += 1
      const token = diffToken
      void getJSON(`/spans/${id}/implementation`)
        .then((body) => {
          applyDiff(body, token)
        })
        .catch((err: unknown) => {
          if (!closed) {
            setError(err instanceof Error ? err.message : "the diff could not be read")
          }
        })
    }
    socket.addEventListener("open", () => {
      if (!closed) {
        setConnected(true)
        note("socket open")
      }
    })
    socket.addEventListener("message", (event) => {
      if (closed || typeof event.data !== "string") {
        return
      }
      let raw: unknown
      try {
        raw = JSON.parse(event.data)
      } catch {
        return
      }
      const frame = liveFrame(raw)
      if (frame.type === "replay" && frame.span) {
        setSpan(frame.span)
        setTranscript(frame.transcript ?? [])
        setFrames([])
      }
      if (frame.type === "cursor" && frame.cursor) {
        const next = frame.cursor
        loadDiff(next.gitSHA)
        setSpan((current) => (current ? { ...current, cursor: next } : current))
        remember(frame)
      }
      if (frame.type === "statement" && frame.statement) {
        const next = frame.statement
        loadDiff(next.gitSHA ?? "")
        setSpan((current) => (current ? { ...current, statements: upsert(current.statements, next) } : current))
        remember(frame)
      }
      if (frame.type === "transcript" && frame.lines && (frame.stage === "build" || frame.stage === undefined)) {
        const lines = frame.lines
        setTranscript((current) => [...current, ...lines])
        remember(frame)
      }
      if (frame.type === "gate") {
        remember(frame)
        void getJSON(`/spans/${id}/acceptance`)
          .then((body) => {
            if (!closed) {
              setAcceptance(acceptanceView(body))
            }
          })
          .catch((err: unknown) => {
            if (!closed) {
              setError(err instanceof Error ? err.message : "acceptance could not be read")
            }
          })
      }
    })
    socket.addEventListener("close", () => {
      if (!closed) {
        setConnected(false)
        note("socket closed")
      }
    })
    void Promise.all([
      getJSON(`/spans/${id}`),
      getJSON(`/spans/${id}/implementation`),
      getJSON(`/spans/${id}/acceptance`),
    ])
      .then(([spanBody, diffBody, acceptanceBody]) => {
        if (closed) {
          return
        }
        const loaded = snapshot(spanBody)
        setSpan((current) => current ?? loaded)
        if (seenSHA === "") {
          seenSHA = loaded.cursor.gitSHA
        }
        applyDiff(diffBody, diffToken)
        setAcceptance(acceptanceView(acceptanceBody))
      })
      .catch((err: unknown) => {
        if (!closed) {
          setError(err instanceof Error ? err.message : "the span could not be read")
        }
      })
    return () => {
      closed = true
      socket.close()
    }
  }, [id, note, remember])

  useEffect(() => {
    let gone = false
    const pull = () => {
      void getJSON(`/spans/${id}/agent`)
        .then((body) => {
          if (!gone) {
            setAgent(agentStatus(body))
            setAgentAt(new Date().toISOString())
          }
        })
        .catch(() => undefined)
    }
    pull()
    const timer = window.setInterval(pull, agentReviewMs)
    return () => {
      gone = true
      window.clearInterval(timer)
    }
  }, [id])

  return { span, acceptance, implementation, transcript, frames, actions, agent, agentAt, note, error, connected }
}
