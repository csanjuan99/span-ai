import { afterEach, describe, expect, it, vi } from "vitest"

import {
  abortConflict,
  approveStage,
  basePlaceholder,
  branchChoice,
  branchLine,
  chooseConflict,
  conflictOf,
  hintConflict,
  liveFrame,
  snapshot,
  startRun,
} from "@/lib/api"

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

function option(over: Record<string, unknown> = {}): Record<string, unknown> {
  return {
    id: "o1",
    title: "Keep this branch",
    summary: "The branch wins.",
    risk: "low",
    action: "ours",
    command: "",
    diff: "@@ -1 +1 @@\n-a\n+b\n",
    ...over,
  }
}

function group(over: Record<string, unknown> = {}): Record<string, unknown> {
  return {
    id: "g1",
    kind: "content",
    files: ["src/a.ts"],
    summary: "Both sides edited the total.",
    options: [option(), option({ id: "o2", action: "theirs", risk: "medium" })],
    chosen: "",
    ...over,
  }
}

function conflictBody(over: Record<string, unknown> = {}): Record<string, unknown> {
  return { base: "main", baseSHA: "abc", state: "choosing", note: "two groups", groups: [group()], ...over }
}

describe("conflictOf", () => {
  it("reads every field of the contract", () => {
    const parsed = conflictOf(conflictBody())
    expect(parsed).toEqual({
      base: "main",
      baseSHA: "abc",
      state: "choosing",
      note: "two groups",
      groups: [
        {
          id: "g1",
          kind: "content",
          files: ["src/a.ts"],
          summary: "Both sides edited the total.",
          chosen: "",
          options: [
            {
              id: "o1",
              title: "Keep this branch",
              summary: "The branch wins.",
              risk: "low",
              action: "ours",
              command: "",
              diff: "@@ -1 +1 @@\n-a\n+b\n",
            },
            {
              id: "o2",
              title: "Keep this branch",
              summary: "The branch wins.",
              risk: "medium",
              action: "theirs",
              command: "",
              diff: "@@ -1 +1 @@\n-a\n+b\n",
            },
          ],
        },
      ],
    })
  })

  it("degrades unknown enums to the safe reading instead of throwing", () => {
    const parsed = conflictOf(
      conflictBody({ groups: [group({ kind: "submodule", options: [option({ risk: "tiny", action: "explode" }), option({ id: "o2" })] })] }),
    )
    const first = parsed?.groups[0]
    expect(first?.kind).toBe("other")
    expect(first?.options[0]?.risk).toBe("high")
    expect(first?.options[0]?.action).toBe("keep")
  })

  it("drops a choosing conflict with no groups so the page never shows an empty panel", () => {
    expect(conflictOf(conflictBody({ groups: [] }))).toBeUndefined()
    expect(conflictOf(conflictBody({ groups: "nope" }))).toBeUndefined()
  })

  it("drops a group with no options, and the conflict when no group is left", () => {
    expect(conflictOf(conflictBody({ groups: [group({ options: [] })] }))).toBeUndefined()
    const parsed = conflictOf(conflictBody({ groups: [group({ id: "empty", options: [] }), group({ id: "g2" })] }))
    expect(parsed?.groups.map((item) => item.id)).toEqual(["g2"])
  })

  it("keeps a proposing conflict without groups: the agent is still reading", () => {
    expect(conflictOf(conflictBody({ state: "proposing", groups: [] }))?.state).toBe("proposing")
  })

  it("fills missing fields with empty values and ignores a non-object", () => {
    expect(conflictOf(null)).toBeUndefined()
    expect(conflictOf("conflict")).toBeUndefined()
    const parsed = conflictOf({ state: "choosing", groups: [{ id: "g1", options: [{ id: "o1" }] }] })
    expect(parsed).toEqual({
      base: "",
      baseSHA: "",
      state: "choosing",
      note: "",
      groups: [
        {
          id: "g1",
          kind: "other",
          files: [],
          summary: "",
          chosen: "",
          options: [{ id: "o1", title: "", summary: "", risk: "high", action: "keep", command: "", diff: "" }],
        },
      ],
    })
  })

  it("reads the conflict on the cursor and on the live frame", () => {
    const span = snapshot({ cursor: { id: "s", conflict: conflictBody() }, statements: [] })
    expect(span.cursor.conflict?.base).toBe("main")
    expect(snapshot({ cursor: { id: "s" } }).cursor.conflict).toBeUndefined()
    const frame = liveFrame({ type: "conflict", conflict: conflictBody({ state: "applying" }) })
    expect(frame.conflict?.state).toBe("applying")
  })
})

describe("conflict posts", () => {
  it("sends one group/option pair per group, in order, form-encoded", async () => {
    const fetchMock = vi.fn((url: string, init: RequestInit) => {
      void url
      void init
      return Promise.resolve(new Response("", { status: 202 }))
    })
    vi.stubGlobal("fetch", fetchMock)
    await chooseConflict("span 1", [
      { group: "g1", option: "o2" },
      { group: "g2", option: "o1" },
    ])
    const [url, init] = fetchMock.mock.calls[0] ?? ["", {}]
    expect(url).toBe("/spans/span%201/conflict/choose")
    expect(String(init.body)).toBe("group=g1&option=o2&group=g2&option=o1")
    expect(new Headers(init.headers).get("Content-Type")).toBe("application/x-www-form-urlencoded")
  })

  it("sends the hint text and the abort, and surfaces the server message", async () => {
    const fetchMock = vi.fn((url: string, init: RequestInit) => {
      void init
      if (url.endsWith("/abort")) {
        return Promise.resolve(new Response(JSON.stringify({ message: "no merge in progress" }), { status: 409 }))
      }
      return Promise.resolve(new Response("", { status: 202 }))
    })
    vi.stubGlobal("fetch", fetchMock)
    await hintConflict("s", "keep the new total")
    const [url, init] = fetchMock.mock.calls[0] ?? ["", {}]
    expect(url).toBe("/spans/s/conflict/hint")
    expect(String(init.body)).toBe("text=keep+the+new+total")
    await expect(abortConflict("s")).rejects.toThrow("no merge in progress")
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

describe("branchChoice", () => {
  it("keeps the branch names and the default base", () => {
    expect(branchChoice({ branches: ["dev", "feat/SAL-3", 4, ""], default: "feat/SAL-3" })).toEqual({
      branches: ["dev", "feat/SAL-3"],
      defaultBase: "feat/SAL-3",
    })
  })

  it("falls back to dev when the reply says nothing", () => {
    expect(branchChoice(null)).toEqual({ branches: [], defaultBase: "dev" })
  })
})

describe("branchLine", () => {
  it("names the branch and the base it opens into", () => {
    const span = snapshot({ cursor: { branch: "feat/SAL-4", base: "feat/SAL-3" } })
    expect(span.cursor.base).toBe("feat/SAL-3")
    expect(branchLine(span.cursor)).toBe("branch feat/SAL-4 · base feat/SAL-3")
  })

  it("shows the base alone, and nothing for a run with neither", () => {
    expect(branchLine({ branch: "", base: "dev" })).toBe("base dev")
    expect(branchLine({ branch: "", base: "" })).toBe("")
  })
})

describe("basePlaceholder", () => {
  it("shows master for a hotfix and the project default otherwise", () => {
    expect(basePlaceholder("hotfix", "release/2.1")).toBe("master")
    expect(basePlaceholder("feat", "release/2.1")).toBe("release/2.1")
    expect(basePlaceholder("infer", "dev")).toBe("dev")
  })
})
