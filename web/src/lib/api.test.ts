import { afterEach, describe, expect, it, vi } from "vitest"

import { approveStage, liveFrame, startRun } from "@/lib/api"

function reply(status: number, body: unknown) {
  vi.stubGlobal(
    "fetch",
    vi.fn(() => Promise.resolve(new Response(typeof body === "string" ? body : JSON.stringify(body), { status }))),
  )
}

afterEach(() => {
  vi.unstubAllGlobals()
})

describe("startRun", () => {
  it("returns every run a submission started, first one as the main id", async () => {
    reply(202, {
      id: "span-1",
      stage: "intake",
      runs: [
        { id: "span-1", stage: "intake", repo: "api" },
        { id: "span-2", stage: "intake", repo: "web" },
      ],
    })
    const started = await startRun(new FormData())
    expect(typeof started).toBe("object")
    if (typeof started === "string") {
      return
    }
    expect(started.id).toBe("span-1")
    expect(started.runs.map((run) => `${run.repo}:${run.id}`)).toEqual(["api:span-1", "web:span-2"])
    expect(started.partial).toBe(false)
  })

  it("reads a reply from before `runs` existed as one run", async () => {
    reply(202, { id: "span-9", stage: "intake" })
    const started = await startRun(new FormData())
    expect(typeof started).toBe("object")
    if (typeof started === "string") {
      return
    }
    expect(started.runs).toEqual([{ id: "span-9", stage: "intake", repo: "" }])
  })

  it("keeps the partial flag and the note naming the repo that did not start", async () => {
    reply(202, {
      id: "span-1",
      stage: "intake",
      runs: [{ id: "span-1", stage: "intake", repo: "api" }],
      partial: true,
      notes: "repo web did not start: another run is active in this checkout\n",
    })
    const started = await startRun(new FormData())
    if (typeof started === "string") {
      throw new Error("expected a started run")
    }
    expect(started.partial).toBe(true)
    expect(started.notes).toContain("repo web did not start")
  })

  it("ignores run entries that are not runs", async () => {
    reply(202, { id: "span-1", stage: "intake", runs: [null, 3, { id: 4 }, { id: "span-2", stage: "intake" }] })
    const started = await startRun(new FormData())
    if (typeof started === "string") {
      throw new Error("expected a started run")
    }
    expect(started.runs.map((run) => run.id)).toEqual(["span-2"])
  })

  it("shows the server's busy message, not a bare status", async () => {
    reply(409, { error: "busy", message: "another run is active in this checkout (pid 4 run span-3)" })
    await expect(startRun(new FormData())).rejects.toThrow("another run is active in this checkout")
  })
})

describe("approveStage", () => {
  it("posts the stage and the environment as a form", async () => {
    const fetchSpy = vi.fn(() => Promise.resolve(new Response(JSON.stringify({ id: "s", stage: "promote" }), { status: 202 })))
    vi.stubGlobal("fetch", fetchSpy)
    await approveStage("span-1", "promote", "preview")
    const call = fetchSpy.mock.calls[0] as unknown as [string, RequestInit]
    expect(call[0]).toBe("/spans/span-1/approve")
    expect(call[1].method).toBe("POST")
    expect(String(call[1].body)).toBe("stage=promote&environment=preview")
  })

  it("surfaces the server's reason when the run is not held there", async () => {
    reply(409, { error: "invalid", message: "this run is not held at merge" })
    await expect(approveStage("span-1", "merge", "")).rejects.toThrow("this run is not held at merge")
  })
})

describe("liveFrame seq", () => {
  it("keeps a finite non-negative integer seq", () => {
    expect(liveFrame({ type: "cursor", seq: 0 }).seq).toBe(0)
    expect(liveFrame({ type: "cursor", seq: 42 }).seq).toBe(42)
  })

  it("drops a seq that is not a whole non-negative number", () => {
    for (const seq of [-1, 1.5, Number.NaN, Number.POSITIVE_INFINITY, "3", null]) {
      expect(liveFrame({ type: "cursor", seq }).seq).toBeUndefined()
    }
  })
})

describe("liveFrame offset", () => {
  it("keeps a whole non-negative offset and drops anything else", () => {
    expect(liveFrame({ type: "transcript", offset: 16 }).offset).toBe(16)
    expect(liveFrame({ type: "transcript", offset: -1 }).offset).toBeUndefined()
    expect(liveFrame({ type: "transcript", offset: "16" }).offset).toBeUndefined()
  })
})
