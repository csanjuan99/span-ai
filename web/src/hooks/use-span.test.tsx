import { act, cleanup, renderHook } from "@testing-library/react"
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest"

import { agentReviewMs } from "@/lib/timeline"
import { useSpan } from "@/hooks/use-span"

type Listener = (event: { data?: unknown }) => void

/** Stands in for the browser socket: the test opens, feeds and closes it by hand. */
class FakeSocket {
  static made: FakeSocket[] = []
  readonly url: string
  closed = false
  private listeners = new Map<string, Listener[]>()

  constructor(url: string) {
    this.url = url
    FakeSocket.made.push(this)
  }

  addEventListener(type: string, listener: Listener): void {
    this.listeners.set(type, [...(this.listeners.get(type) ?? []), listener])
  }

  close(): void {
    if (this.closed) {
      return
    }
    this.closed = true
    this.emit("close", {})
  }

  emit(type: string, event: { data?: unknown }): void {
    for (const listener of this.listeners.get(type) ?? []) {
      listener(event)
    }
  }

  open(): void {
    this.emit("open", {})
  }

  send(frame: unknown): void {
    this.emit("message", { data: typeof frame === "string" ? frame : JSON.stringify(frame) })
  }

  /** The server went away: the browser fires close without the page asking. */
  drop(): void {
    this.closed = true
    this.emit("close", {})
  }
}

function latest(): FakeSocket {
  const socket = FakeSocket.made.at(-1)
  if (socket === undefined) {
    throw new Error("no socket was opened")
  }
  return socket
}

const cursor = (stage: string, gitSHA = "") => ({ id: "s1", stage, gitSHA })
const span = (stage: string) => ({ cursor: cursor(stage), statements: [], acceptance: [] })

let failImplementation = false
let failAgent = false
/** Lets one test hold a response back to deliver it out of order. */
let intercept: ((url: string) => Promise<Response> | undefined) | null = null

function respond(url: string): Response {
  if (url.endsWith("/implementation")) {
    return failImplementation
      ? new Response(JSON.stringify({ message: "no commit yet" }), { status: 404 })
      : new Response(JSON.stringify({ stat: "1 file", patch: "", truncated: false }), { status: 200 })
  }
  if (url.endsWith("/acceptance")) {
    return new Response(JSON.stringify({ skipped: true, lines: [] }), { status: 200 })
  }
  if (url.endsWith("/agent")) {
    if (failAgent) {
      return new Response(JSON.stringify({ message: "agent registry down" }), { status: 500 })
    }
    return new Response(JSON.stringify({ id: "s1", running: false }), { status: 200 })
  }
  return new Response(JSON.stringify(span("frame")), { status: 200 })
}

beforeEach(() => {
  vi.useFakeTimers()
  FakeSocket.made = []
  failImplementation = false
  failAgent = false
  intercept = null
  vi.stubGlobal("WebSocket", FakeSocket)
  vi.stubGlobal(
    "fetch",
    vi.fn((input: string) => intercept?.(input) ?? Promise.resolve(respond(input))),
  )
})

afterEach(() => {
  cleanup()
  vi.useRealTimers()
  vi.unstubAllGlobals()
})

async function settle(): Promise<void> {
  await act(async () => {
    await vi.advanceTimersByTimeAsync(0)
  })
}

describe("useSpan live socket", () => {
  it("reconnects with backoff after the server closes the socket", async () => {
    const { result } = renderHook(() => useSpan("s1"))
    await settle()
    act(() => latest().open())
    expect(result.current.connected).toBe(true)
    act(() => latest().drop())
    expect(result.current.connected).toBe(false)
    expect(result.current.actions.some((item) => item.text === "socket closed")).toBe(true)
    expect(FakeSocket.made).toHaveLength(1)
    await act(async () => {
      await vi.advanceTimersByTimeAsync(399)
    })
    expect(FakeSocket.made).toHaveLength(1)
    await act(async () => {
      await vi.advanceTimersByTimeAsync(201)
    })
    expect(FakeSocket.made).toHaveLength(2)
    // The second attempt fails before opening: the wait doubles.
    act(() => latest().drop())
    await act(async () => {
      await vi.advanceTimersByTimeAsync(799)
    })
    expect(FakeSocket.made).toHaveLength(2)
    await act(async () => {
      await vi.advanceTimersByTimeAsync(401)
    })
    expect(FakeSocket.made).toHaveLength(3)
    act(() => latest().open())
    expect(result.current.connected).toBe(true)
  })

  it("stops reconnecting after unmount", async () => {
    const { unmount } = renderHook(() => useSpan("s1"))
    await settle()
    act(() => latest().open())
    act(() => latest().drop())
    unmount()
    await act(async () => {
      await vi.advanceTimersByTimeAsync(60000)
    })
    expect(FakeSocket.made).toHaveLength(1)
  })

  it("closes its socket on unmount and does not reopen it", async () => {
    const { unmount } = renderHook(() => useSpan("s1"))
    await settle()
    const socket = latest()
    act(() => socket.open())
    unmount()
    expect(socket.closed).toBe(true)
    await act(async () => {
      await vi.advanceTimersByTimeAsync(60000)
    })
    expect(FakeSocket.made).toHaveLength(1)
  })

  it("skips duplicates and resyncs at once on a gap", async () => {
    const { result } = renderHook(() => useSpan("s1"))
    await settle()
    const first = latest()
    act(() => first.open())
    act(() => first.send({ type: "replay", seq: 4, span: span("build"), transcript: [] }))
    act(() => first.send({ type: "transcript", stage: "build", lines: ["old"], seq: 4 }))
    act(() => first.send({ type: "transcript", stage: "build", lines: ["new"], seq: 5 }))
    expect(result.current.transcript).toEqual(["new"])
    act(() => first.send({ type: "transcript", stage: "build", lines: ["lost"], seq: 9 }))
    expect(result.current.transcript).toEqual(["new"])
    expect(first.closed).toBe(true)
    await act(async () => {
      await vi.advanceTimersByTimeAsync(0)
    })
    expect(FakeSocket.made).toHaveLength(2)
    const second = latest()
    act(() => second.open())
    act(() => second.send({ type: "replay", seq: 9, span: span("qa"), transcript: ["new", "lost"] }))
    act(() => second.send({ type: "transcript", stage: "build", lines: ["after"], seq: 10 }))
    expect(result.current.span?.cursor.stage).toBe("qa")
    expect(result.current.transcript).toEqual(["new", "lost", "after"])
    expect(result.current.actions.some((item) => item.text === "frames were lost, resyncing")).toBe(true)
    // The resync replay may follow a missed gate frame, so acceptance is read again.
    const acceptanceReads = vi.mocked(fetch).mock.calls.filter(([url]) => String(url).endsWith("/acceptance"))
    expect(acceptanceReads).toHaveLength(2)
  })

  it("ignores a frame that is not a live frame", async () => {
    const { result } = renderHook(() => useSpan("s1"))
    await settle()
    const socket = latest()
    act(() => socket.open())
    expect(() => {
      act(() => socket.send("42"))
      act(() => socket.send("not json"))
      act(() => socket.send("null"))
    }).not.toThrow()
    act(() => socket.send({ type: "transcript", stage: "build", lines: ["kept"] }))
    expect(result.current.transcript).toEqual(["kept"])
    expect(result.current.connected).toBe(true)
  })
})

describe("useSpan resync", () => {
  it("shows a line once when the replay already holds it", async () => {
    const { result } = renderHook(() => useSpan("s1"))
    await settle()
    const socket = latest()
    act(() => socket.open())
    act(() => socket.send({ type: "replay", seq: 0, offset: 16, span: span("build"), transcript: ["compiling pkg A"] }))
    act(() => socket.send({ type: "transcript", stage: "build", lines: ["compiling pkg A"], seq: 1, offset: 16 }))
    act(() => socket.send({ type: "transcript", stage: "build", lines: ["compiling pkg B"], seq: 2, offset: 32 }))
    expect(result.current.transcript).toEqual(["compiling pkg A", "compiling pkg B"])
  })

  it("takes a replay without a span as a baseline and keeps the page", async () => {
    let spanReads = 0
    intercept = (url) => {
      if (!url.endsWith("/spans/s1")) {
        return undefined
      }
      spanReads += 1
      return spanReads === 1 ? undefined : Promise.resolve(new Response(JSON.stringify({ message: "gone" }), { status: 404 }))
    }
    const { result } = renderHook(() => useSpan("s1"))
    await settle()
    const first = latest()
    act(() => first.open())
    act(() => first.send({ type: "replay", seq: 7, span: span("build"), transcript: ["kept"] }))
    act(() => first.drop())
    await act(async () => {
      await vi.advanceTimersByTimeAsync(1000)
    })
    const second = latest()
    act(() => second.open())
    // The server restarted and cannot read the run: its counter starts over.
    act(() => second.send({ type: "replay", seq: 0 }))
    act(() => second.send({ type: "transcript", stage: "build", lines: ["fresh"], seq: 1 }))
    await settle()
    // The refresh it asked for failed: the page keeps what it had.
    expect(spanReads).toBe(2)
    expect(result.current.span?.cursor.stage).toBe("build")
    expect(result.current.transcript).toEqual(["kept", "fresh"])
    expect(result.current.error).toBe("")
    expect(FakeSocket.made).toHaveLength(2)
  })

  it("keeps its baseline on a span-less replay that is not a restart, and reloads the span", async () => {
    const { result } = renderHook(() => useSpan("s1"))
    await settle()
    const first = latest()
    act(() => first.open())
    act(() => first.send({ type: "replay", seq: 7, offset: 16, span: span("build"), transcript: ["kept"] }))
    act(() => first.drop())
    await act(async () => {
      await vi.advanceTimersByTimeAsync(1000)
    })
    const second = latest()
    act(() => second.open())
    act(() => second.send({ type: "replay", seq: 9 }))
    await settle()
    // GET /spans/s1 answers the "frame" snapshot: the missed frames are recovered from it.
    expect(result.current.span?.cursor.stage).toBe("frame")
    const spanReads = vi.mocked(fetch).mock.calls.filter(([url]) => String(url).endsWith("/spans/s1"))
    expect(spanReads).toHaveLength(2)
    // The transcript baseline stayed at 16 and the seq moved up to 9.
    act(() => second.send({ type: "transcript", stage: "build", lines: ["kept"], seq: 10, offset: 16 }))
    act(() => second.send({ type: "transcript", stage: "build", lines: ["next"], seq: 11, offset: 21 }))
    expect(result.current.transcript).toEqual(["kept", "next"])
    expect(FakeSocket.made).toHaveLength(2)
  })

  it("takes the transcript of a span-less replay so lines published while away are kept", async () => {
    const { result } = renderHook(() => useSpan("s1"))
    await settle()
    const first = latest()
    act(() => first.open())
    act(() => first.send({ type: "replay", seq: 1, offset: 2, span: span("build"), transcript: ["a"] }))
    act(() => first.drop())
    await act(async () => {
      await vi.advanceTimersByTimeAsync(1000)
    })
    const second = latest()
    act(() => second.open())
    act(() => second.send({ type: "replay", seq: 2, offset: 4, transcript: ["a", "b"] }))
    act(() => second.send({ type: "transcript", stage: "build", lines: ["c"], seq: 3, offset: 6 }))
    expect(result.current.transcript).toEqual(["a", "b", "c"])
  })

  it("counts a frame the server emptied so the next one is not a gap", async () => {
    const { result } = renderHook(() => useSpan("s1"))
    await settle()
    const socket = latest()
    act(() => socket.open())
    act(() => socket.send({ type: "replay", seq: 0, offset: 2, span: span("build"), transcript: ["a"] }))
    act(() => socket.send({ type: "transcript", stage: "build", seq: 1, offset: 2 }))
    act(() => socket.send({ type: "transcript", stage: "build", lines: ["b"], seq: 2, offset: 4 }))
    expect(result.current.transcript).toEqual(["a", "b"])
    expect(socket.closed).toBe(false)
    expect(FakeSocket.made).toHaveLength(1)
  })

  it("never drops an unseen line when two flushes arrive out of order", async () => {
    const { result } = renderHook(() => useSpan("s1"))
    await settle()
    const socket = latest()
    act(() => socket.open())
    act(() => socket.send({ type: "replay", seq: 0, offset: 4, span: span("build"), transcript: ["old"] }))
    act(() => socket.send({ type: "transcript", stage: "build", lines: ["B"], seq: 1, offset: 17 }))
    act(() => socket.send({ type: "transcript", stage: "build", lines: ["AAAAAAAAAA"], seq: 2, offset: 15 }))
    expect(result.current.transcript).toEqual(["old", "B", "AAAAAAAAAA"])
  })

  it("does not reconnect faster than the backoff when every frame is a gap", async () => {
    renderHook(() => useSpan("s1"))
    await settle()
    const served = new Set<FakeSocket>()
    for (let step = 0; step < 100; step += 1) {
      for (const socket of FakeSocket.made) {
        if (served.has(socket)) {
          continue
        }
        served.add(socket)
        act(() => socket.open())
        act(() => socket.send({ type: "replay", seq: 0, span: span("build") }))
        act(() => socket.send({ type: "cursor", cursor: cursor("build"), seq: 5 }))
      }
      await act(async () => {
        await vi.advanceTimersByTimeAsync(100)
      })
    }
    // Ten seconds: 0 ms, then about 1 s, 2 s and 4 s apart, then 8 s.
    expect(FakeSocket.made.length).toBeLessThanOrEqual(6)
    expect(FakeSocket.made.length).toBeGreaterThanOrEqual(3)
  })

  it("resets the backoff once a connection stayed healthy", async () => {
    renderHook(() => useSpan("s1"))
    await settle()
    for (let round = 0; round < 3; round += 1) {
      act(() => latest().drop())
      await act(async () => {
        await vi.advanceTimersByTimeAsync(10000)
      })
    }
    const socket = latest()
    act(() => socket.open())
    act(() => socket.send({ type: "replay", seq: 0, span: span("build") }))
    await act(async () => {
      await vi.advanceTimersByTimeAsync(5000)
    })
    const before = FakeSocket.made.length
    act(() => socket.drop())
    await act(async () => {
      await vi.advanceTimersByTimeAsync(600)
    })
    expect(FakeSocket.made.length).toBe(before + 1)
  })

  it("never lets an older acceptance reply overwrite a newer one", async () => {
    let releaseFirst: (response: Response) => void = () => undefined
    let acceptanceCalls = 0
    intercept = (url) => {
      if (!url.endsWith("/acceptance")) {
        return undefined
      }
      acceptanceCalls += 1
      if (acceptanceCalls === 1) {
        return new Promise((resolve) => {
          releaseFirst = resolve
        })
      }
      return Promise.resolve(new Response(JSON.stringify({ verdict: "pass", lines: [] }), { status: 200 }))
    }
    const { result } = renderHook(() => useSpan("s1"))
    await settle()
    const socket = latest()
    act(() => socket.open())
    act(() => socket.send({ type: "gate", gate: { verdict: "pass" } }))
    await settle()
    expect(result.current.acceptance?.verdict).toBe("pass")
    releaseFirst(new Response(JSON.stringify({ skipped: true, lines: [] }), { status: 200 }))
    await settle()
    await settle()
    expect(result.current.acceptance?.verdict).toBe("pass")
    expect(result.current.acceptance?.skipped).toBe(false)
  })
})

describe("useSpan first load", () => {
  it("keeps the span when the implementation cannot be read", async () => {
    failImplementation = true
    const { result } = renderHook(() => useSpan("s1"))
    await settle()
    await settle()
    expect(result.current.span?.cursor.stage).toBe("frame")
    expect(result.current.acceptance?.skipped).toBe(true)
    expect(result.current.implementation).toBeNull()
    expect(result.current.error).toBe("")
  })
})

describe("useSpan agent poll", () => {
  it("records a failed poll once, keeps polling, and recovers", async () => {
    failAgent = true
    const { result } = renderHook(() => useSpan("s1"))
    await settle()
    const failures = () => result.current.actions.filter((line) => line.text.includes("the agent status could not be read: agent registry down"))
    expect(failures()).toHaveLength(1)
    await act(async () => {
      await vi.advanceTimersByTimeAsync(agentReviewMs)
    })
    expect(failures()).toHaveLength(1)
    failAgent = false
    await act(async () => {
      await vi.advanceTimersByTimeAsync(agentReviewMs)
    })
    expect(result.current.agent).not.toBeNull()
  })
})
