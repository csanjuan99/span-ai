import { useCallback, useEffect, useState } from "react"

import { acceptanceView, agentStatus, getJSON, implementationView, liveFrame, snapshot, socketProtocol, UnauthorizedError } from "@/lib/api"
import { reasonText } from "@/lib/guards"
import { problemOf, type Problem } from "@/lib/problem"
import { coveredByReplay, frameGate, reconnectDelay } from "@/lib/reconnect"
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
  /** Why the first read of the run failed, when the whole page has to say so. */
  problem: Problem | null
}

function upsert(statements: Statement[], next: Statement): Statement[] {
  const kept = statements.filter(
    (item) => !(item.stage === next.stage && item.iteration === next.iteration && item.digest === next.digest),
  )
  return [...kept, next]
}

/** Reads one socket message; anything that is not a live frame is noise, not a crash. */
function parseFrame(data: unknown): LiveFrame | null {
  if (typeof data !== "string") {
    return null
  }
  try {
    const raw: unknown = JSON.parse(data)
    return liveFrame(raw)
  } catch {
    return null
  }
}

/** Reads a settled payload, or null when the request or the payload failed. */
function settledView<T>(result: PromiseSettledResult<unknown>, read: (body: unknown) => T): T | null {
  if (result.status === "rejected") {
    return null
  }
  try {
    return read(result.value)
  } catch {
    return null
  }
}

interface LiveHandlers {
  open: () => void
  frame: (frame: LiveFrame) => void
  /**
   * The socket is down; the next attempt starts in `retryMs`. `fresh` is true for the
   * first failure after a healthy connection (or the first try), so a long outage is
   * announced once instead of on every failed attempt.
   */
  down: (retryMs: number, fresh: boolean) => void
}

/** How long a connection must stay up and in order before the backoff starts over. */
const stableMs = 5000

/**
 * Keeps one live socket open for as long as the caller wants it. A close schedules a
 * reconnect with backoff; a seq gap closes the socket and resyncs, because the replay
 * that opens every connection is the only way to recover lost frames. The backoff only
 * starts over after a connection stayed healthy for `stableMs`, and only the first gap in
 * that window resyncs at once, so a server that gaps or drops on every frame is not
 * hammered. Returns the function that stops it for good.
 */
function keepLive(address: string, protocol: string, on: LiveHandlers): () => void {
  let stopped = false
  let attempt = 0
  let lastSeq = 0
  let replayOffset = 0
  let resync = false
  let openedAt = 0
  let healthy = false
  let lastGapAt = Number.NEGATIVE_INFINITY
  let timer: number | undefined
  let socket: WebSocket | null = null
  const schedule = () => {
    const now = Date.now()
    if (healthy && now - openedAt >= stableMs) {
      attempt = 0
    }
    const wait = resync ? 0 : reconnectDelay(attempt)
    const fresh = attempt === 0
    resync = false
    healthy = false
    attempt += 1
    on.down(wait, fresh)
    timer = window.setTimeout(connect, wait)
  }
  const onGap = () => {
    const now = Date.now()
    // The first gap in a window resyncs at once; the next ones wait for the backoff.
    resync = now - lastGapAt >= stableMs
    lastGapAt = now
    healthy = false
    socket?.close()
  }
  // A replay names the new seq. Its offset becomes the transcript baseline whenever it
  // brings a transcript or a span (the page replaces both), or after a restart, when the
  // seq starts over; a bare replay keeps the old baseline.
  const rebase = (frame: LiveFrame) => {
    const seq = frame.seq ?? 0
    const restart = seq < lastSeq
    lastSeq = seq
    if (frame.span !== undefined || frame.transcript !== undefined || restart) {
      replayOffset = frame.offset ?? 0
    }
  }
  const receive = (data: unknown) => {
    const frame = parseFrame(data)
    if (frame === null || resync) {
      return
    }
    const verdict = frameGate(lastSeq, frame)
    if (verdict === "skip") {
      return
    }
    if (verdict === "gap") {
      onGap()
      return
    }
    healthy = true
    if (verdict === "reset") {
      rebase(frame)
      on.frame(frame)
      return
    }
    lastSeq = frame.seq ?? lastSeq
    // Only the replay's offset decides: two flushes may arrive out of order, and moving
    // the baseline on each one would drop the earlier, unseen one.
    if (coveredByReplay(replayOffset, frame)) {
      return
    }
    on.frame(frame)
  }
  function connect() {
    if (stopped) {
      return
    }
    const next = protocol === "" ? new WebSocket(address) : new WebSocket(address, [protocol])
    socket = next
    const current = () => !stopped && socket === next
    next.addEventListener("open", () => {
      if (current()) {
        openedAt = Date.now()
        on.open()
      }
    })
    next.addEventListener("message", (event) => {
      if (current()) {
        receive(event.data)
      }
    })
    next.addEventListener("close", () => {
      if (current()) {
        socket = null
        schedule()
      }
    })
  }
  connect()
  return () => {
    stopped = true
    window.clearTimeout(timer)
    socket?.close()
  }
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
  const [problem, setProblem] = useState<Problem | null>(null)
  if (tracked !== id) {
    setTracked(id)
    setProblem(null)
    setFrames([])
    setActions([])
    setTranscript([])
    setSpan(null)
    setAgent(null)
    setAgentAt("")
  }
  // note and remember are dependencies of the live-stream effect below; a stable identity
  // keeps that effect from tearing down and reopening the stream on every render.
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
    let replays = 0
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
            setError(reasonText(err, "the diff could not be read"))
          }
        })
    }
    // Like the diff, every acceptance read takes a token and only the newest may land:
    // a slow first load must not overwrite the read a later gate frame started.
    let acceptanceToken = 0
    const loadAcceptance = () => {
      acceptanceToken += 1
      const token = acceptanceToken
      void getJSON(`/spans/${id}/acceptance`)
        .then((body) => {
          if (!closed && token === acceptanceToken) {
            setAcceptance(acceptanceView(body))
          }
        })
        .catch((err: unknown) => {
          if (!closed) {
            setError(reasonText(err, "acceptance could not be read"))
          }
        })
    }
    // Span reads take a token too: the first load and a later reload must not cross.
    let spanToken = 0
    const loadSpan = () => {
      spanToken += 1
      const token = spanToken
      void getJSON(`/spans/${id}`)
        .then((body) => {
          if (closed || token !== spanToken) {
            return
          }
          const loaded = snapshot(body)
          setProblem(null)
          setSpan(loaded)
          loadDiff(loaded.cursor.gitSHA)
        })
        .catch((err: unknown) => {
          if (closed) {
            return
          }
          const page = problemOf(err)
          if (page !== null) {
            setProblem(page)
          }
          note(`the run could not be reloaded: ${reasonText(err, "unknown error")}`)
        })
    }
    const apply = (frame: LiveFrame) => {
      if (frame.type === "replay" && !frame.span) {
        // The transcript does not depend on the snapshot the server failed to read.
        if (frame.transcript) {
          setTranscript(frame.transcript)
        }
        loadSpan()
      }
      if (frame.type === "replay" && frame.span) {
        const next = frame.span
        setSpan(next)
        setTranscript(frame.transcript ?? [])
        setFrames([])
        replays += 1
        // A reconnect may have missed cursor and gate frames; the first replay is
        // covered by the first load below.
        if (replays > 1) {
          loadDiff(next.cursor.gitSHA)
          loadAcceptance()
        }
      }
      if (frame.type === "cursor" && frame.cursor) {
        const next = frame.cursor
        loadDiff(next.gitSHA)
        setSpan((current) => (current ? { ...current, cursor: next } : current))
        remember(frame)
      }
      if (frame.type === "conflict") {
        const conflict = frame.conflict
        setSpan((current) => (current ? { ...current, cursor: { ...current.cursor, conflict } } : current))
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
        loadAcceptance()
      }
    }
    const address = `${location.protocol === "https:" ? "wss" : "ws"}://${location.host}/spans/${id}/live`
    const stop = keepLive(address, socketProtocol(), {
      open: () => {
        setConnected(true)
        note("socket open")
      },
      frame: apply,
      down: (retryMs, fresh) => {
        setConnected(false)
        if (fresh) {
          note("socket closed")
          note(retryMs === 0 ? "frames were lost, resyncing" : `reconnecting in ${Math.max(1, Math.ceil(retryMs / 1000))}s`)
        }
      },
    })
    // Each part settles on its own: /implementation 404s before the first commit, and
    // that must not leave the whole page empty. Only the span itself is an error.
    acceptanceToken += 1
    const firstAcceptance = acceptanceToken
    spanToken += 1
    const firstSpan = spanToken
    void Promise.allSettled([
      getJSON(`/spans/${id}`),
      getJSON(`/spans/${id}/implementation`),
      getJSON(`/spans/${id}/acceptance`),
    ]).then(([spanResult, diffResult, acceptanceResult]) => {
      if (closed) {
        return
      }
      if (spanResult.status === "rejected") {
        setError(reasonText(spanResult.reason, "the span could not be read"))
      }
      const loaded = settledView(spanResult, snapshot)
      if (loaded !== null && firstSpan === spanToken) {
        setSpan((current) => current ?? loaded)
        if (seenSHA === "") {
          seenSHA = loaded.cursor.gitSHA
        }
      }
      const diff = settledView(diffResult, implementationView)
      if (diff !== null && diffToken === 0) {
        setImplementation(diff)
      }
      const checks = settledView(acceptanceResult, acceptanceView)
      if (checks !== null && firstAcceptance === acceptanceToken) {
        setAcceptance(checks)
      }
    })
    return () => {
      closed = true
      stop()
    }
  }, [id, note, remember])

  useEffect(() => {
    let gone = false
    let lastFailure = ""
    const pull = () => {
      void getJSON(`/spans/${id}/agent`)
        .then((body) => {
          if (!gone) {
            lastFailure = ""
            setAgent(agentStatus(body))
            setAgentAt(new Date().toISOString())
          }
        })
        .catch((err: unknown) => {
          if (gone) {
            return
          }
          // The poll keeps running; the agent line shows the last good read. A refused token is
          // the page's error, any other failure is one line in the timeline, once per reason.
          if (err instanceof UnauthorizedError) {
            setError(err.message)
            return
          }
          const reason = reasonText(err, "unknown error")
          if (reason !== lastFailure) {
            lastFailure = reason
            note(`the agent status could not be read: ${reason}`)
          }
        })
    }
    pull()
    const timer = window.setInterval(pull, agentReviewMs)
    return () => {
      gone = true
      window.clearInterval(timer)
    }
  }, [id, note])

  return { span, acceptance, implementation, transcript, frames, actions, agent, agentAt, note, error, connected, problem }
}
