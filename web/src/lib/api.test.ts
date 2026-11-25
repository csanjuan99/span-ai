import { afterEach, describe, expect, it, vi } from "vitest"

import {
  implementationView,
  abortConflict,
  agentChoice,
  approveStage,
  getRunLog,
  basePlaceholder,
  branchChoice,
  configForm,
  projectConfig,
  saveConfig,
  branchLine,
  chooseConflict,
  conflictOf,
  hintConflict,
  liveFrame,
  modelList,
  ApiError,
  getJSON,
  answerShape,
  createShape,
  createShapeIssue,
  previewShapeIssue,
  saveModel,
  sealShape,
  shapeSession,
  snapshot,
  startRun,
} from "@/lib/api"

function reply(status: number, body: unknown) {
  vi.stubGlobal(
    "fetch",
    vi.fn(() =>
      Promise.resolve(
        new Response(typeof body === "string" ? body : JSON.stringify(body), {
          status,
        })
      )
    )
  )
}

afterEach(() => {
  vi.unstubAllGlobals()
})

describe("getJSON", () => {
  it("reads a plain-text 404 as not found", async () => {
    reply(404, "404 page not found")
    await expect(getJSON("/spans/missing-run")).rejects.toMatchObject({
      name: "ApiError",
      code: "not_found",
      message: "404 page not found",
    })
  })

  it("reads a gateway failure as unreachable", async () => {
    reply(502, "")
    await expect(getJSON("/spans")).rejects.toMatchObject({
      code: "unreachable",
    })
  })
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
    expect(started.runs.map((run) => `${run.repo}:${run.id}`)).toEqual([
      "api:span-1",
      "web:span-2",
    ])
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
    reply(202, {
      id: "span-1",
      stage: "intake",
      runs: [null, 3, { id: 4 }, { id: "span-2", stage: "intake" }],
    })
    const started = await startRun(new FormData())
    if (typeof started === "string") {
      throw new Error("expected a started run")
    }
    expect(started.runs.map((run) => run.id)).toEqual(["span-2"])
  })

  it("shows the server's busy message, not a bare status", async () => {
    reply(409, {
      error: "busy",
      message: "another run is active in this checkout (pid 4 run span-3)",
    })
    await expect(startRun(new FormData())).rejects.toThrow(
      "another run is active in this checkout"
    )
  })
})

describe("approveStage", () => {
  it("posts the stage and the environment as a form", async () => {
    const fetchSpy = vi.fn(() =>
      Promise.resolve(
        new Response(JSON.stringify({ id: "s", stage: "promote" }), {
          status: 202,
        })
      )
    )
    vi.stubGlobal("fetch", fetchSpy)
    await approveStage("span-1", "promote", "preview")
    const call = fetchSpy.mock.calls[0] as unknown as [string, RequestInit]
    expect(call[0]).toBe("/spans/span-1/approve")
    expect(call[1].method).toBe("POST")
    expect(String(call[1].body)).toBe("stage=promote&environment=preview")
  })

  it("surfaces the server's reason when the run is not held there", async () => {
    reply(409, { error: "invalid", message: "this run is not held at merge" })
    await expect(approveStage("span-1", "merge", "")).rejects.toThrow(
      "this run is not held at merge"
    )
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

function conflictBody(
  over: Record<string, unknown> = {}
): Record<string, unknown> {
  return {
    base: "main",
    baseSHA: "abc",
    state: "choosing",
    note: "two groups",
    groups: [group()],
    ...over,
  }
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
      conflictBody({
        groups: [
          group({
            kind: "submodule",
            options: [
              option({ risk: "tiny", action: "explode" }),
              option({ id: "o2" }),
            ],
          }),
        ],
      })
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
    expect(
      conflictOf(conflictBody({ groups: [group({ options: [] })] }))
    ).toBeUndefined()
    const parsed = conflictOf(
      conflictBody({
        groups: [group({ id: "empty", options: [] }), group({ id: "g2" })],
      })
    )
    expect(parsed?.groups.map((item) => item.id)).toEqual(["g2"])
  })

  it("keeps a proposing conflict without groups: the agent is still reading", () => {
    expect(
      conflictOf(conflictBody({ state: "proposing", groups: [] }))?.state
    ).toBe("proposing")
  })

  it("fills missing fields with empty values and ignores a non-object", () => {
    expect(conflictOf(null)).toBeUndefined()
    expect(conflictOf("conflict")).toBeUndefined()
    const parsed = conflictOf({
      state: "choosing",
      groups: [{ id: "g1", options: [{ id: "o1" }] }],
    })
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
          options: [
            {
              id: "o1",
              title: "",
              summary: "",
              risk: "high",
              action: "keep",
              command: "",
              diff: "",
            },
          ],
        },
      ],
    })
  })

  it("reads a pending pause on the snapshot, absent by default", () => {
    expect(snapshot({ cursor: { id: "s" }, pauseRequested: true }).pauseRequested).toBe(true)
    expect(snapshot({ cursor: { id: "s" } }).pauseRequested).toBeUndefined()
  })

  it("reads the conflict on the cursor and on the live frame", () => {
    const span = snapshot({
      cursor: { id: "s", conflict: conflictBody() },
      statements: [],
    })
    expect(span.cursor.conflict?.base).toBe("main")
    expect(snapshot({ cursor: { id: "s" } }).cursor.conflict).toBeUndefined()
    const frame = liveFrame({
      type: "conflict",
      conflict: conflictBody({ state: "applying" }),
    })
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
    expect(new Headers(init.headers).get("Content-Type")).toBe(
      "application/x-www-form-urlencoded"
    )
  })

  it("sends the hint text and the abort, and surfaces the server message", async () => {
    const fetchMock = vi.fn((url: string, init: RequestInit) => {
      void init
      if (url.endsWith("/abort")) {
        return Promise.resolve(
          new Response(JSON.stringify({ message: "no merge in progress" }), {
            status: 409,
          })
        )
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
    for (const seq of [
      -1,
      1.5,
      Number.NaN,
      Number.POSITIVE_INFINITY,
      "3",
      null,
    ]) {
      expect(liveFrame({ type: "cursor", seq }).seq).toBeUndefined()
    }
  })
})

describe("liveFrame offset", () => {
  it("keeps a whole non-negative offset and drops anything else", () => {
    expect(liveFrame({ type: "transcript", offset: 16 }).offset).toBe(16)
    expect(liveFrame({ type: "transcript", offset: -1 }).offset).toBeUndefined()
    expect(
      liveFrame({ type: "transcript", offset: "16" }).offset
    ).toBeUndefined()
  })
})

describe("branchChoice", () => {
  it("keeps the branch names and the default base", () => {
    expect(
      branchChoice({
        branches: ["dev", "feat/SAL-3", 4, ""],
        default: "feat/SAL-3",
      })
    ).toEqual({
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
    const span = snapshot({
      cursor: { branch: "feat/SAL-4", base: "feat/SAL-3" },
    })
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

describe("models", () => {
  it("reads the model chosen for each CLI and drops empty ones", () => {
    const choice = agentChoice({
      agents: [],
      models: { claude: "opus", cursor: "", pi: 4 },
    })
    expect(choice.models).toEqual({ claude: "opus" })
    expect(agentChoice({}).models).toEqual({})
  })

  it("reads a model list and degrades a bad reply to an empty free-text one", () => {
    expect(
      modelList({
        models: [
          { id: "opus", label: "Opus" },
          { id: "" },
          null,
          { label: "x" },
        ],
        free: true,
        source: "fixed",
      })
    ).toEqual({
      models: [{ id: "opus", label: "Opus" }],
      free: true,
      source: "fixed",
      error: "",
    })
    expect(modelList("nope")).toEqual({
      models: [],
      free: true,
      source: "none",
      error: "",
    })
    expect(modelList({ source: "weird" }).source).toBe("none")
  })

  it("posts the project, the CLI and the model as a form and surfaces the server's reason", async () => {
    const fetchMock = vi.fn((url: string, init: RequestInit) => {
      void url
      void init
      return Promise.resolve(
        new Response(
          JSON.stringify({
            message:
              "Claude: not ready. Gentle AI and its configuration are required",
          }),
          { status: 400 }
        )
      )
    })
    vi.stubGlobal("fetch", fetchMock)
    await expect(saveModel("/p", "claude", "opus")).rejects.toThrow(
      "Gentle AI and its configuration are required"
    )
    const [url, init] = fetchMock.mock.calls[0] ?? ["", {}]
    expect(url).toBe("/agent/model")
    expect(String(init.body)).toBe("project=%2Fp&cli=claude&model=opus")
  })
})

const rawSession = {
  id: "sh-1",
  workspace: "/w",
  repo: "api",
  idea: "idea",
  status: "ready",
  map: {
    rules: [{ id: "R1", text: "rule", source: { kind: "code", ref: "a.go" } }],
    examples: [
      { id: "E1", rule: "R1", text: "ex", source: { kind: "prompt", ref: "" } },
    ],
    questions: [
      {
        id: "Q1",
        text: "q?",
        blocks: "R1",
        assume: "yes",
        status: "open",
        answer: "",
      },
    ],
  },
  split: [{ story: "a", leftOut: "b" }],
  proposals: [{ id: "P1", section: "Users", text: "t", status: "proposed" }],
  story: "# s",
  digest: "abc",
  jira: { key: "PRJ-1", url: "https://j/PRJ-1", status: "Backlog" },
}

describe("shapeSession", () => {
  it("flattens the map and keeps the source of every fact", () => {
    const session = shapeSession(rawSession)
    expect(session.rules[0]?.source).toEqual({ kind: "code", ref: "a.go" })
    expect(session.examples[0]?.rule).toBe("R1")
    expect(session.questions[0]?.status).toBe("open")
    expect(session.jira?.key).toBe("PRJ-1")
  })

  it("treats an empty jira object as no issue and tolerates missing parts", () => {
    const session = shapeSession({ id: "x", jira: {} })
    expect(session.jira).toBeNull()
    expect(session.rules).toEqual([])
    expect(session.status).toBe("interviewing")
  })
})

describe("shape client", () => {
  it("posts the idea and reads the session back", async () => {
    reply(201, rawSession)
    const session = await createShape({
      workspace: "/w",
      repo: "api",
      idea: "idea",
    })
    expect(session.id).toBe("sh-1")
    const call = vi.mocked(fetch).mock.calls[0]
    expect(call?.[0]).toBe("/shapes")
    expect(call?.[1]?.method).toBe("POST")
    expect(JSON.parse(String(call?.[1]?.body))).toEqual({
      workspace: "/w",
      repo: "api",
      idea: "idea",
    })
  })

  it("sends answers and assumptions in one body", async () => {
    reply(200, rawSession)
    await answerShape("sh-1", "/w", [{ question: "Q1", text: "a" }], ["Q2"])
    const call = vi.mocked(fetch).mock.calls[0]
    expect(call?.[0]).toBe("/shapes/sh-1/answers")
    expect(JSON.parse(String(call?.[1]?.body))).toEqual({
      workspace: "/w",
      answers: [{ question: "Q1", text: "a" }],
      assume: ["Q2"],
    })
  })

  it("keeps the error code and message of a refusal", async () => {
    reply(409, { error: "shape_not_ready", message: "open questions: Q1" })
    const failure = await sealShape("sh-1", "/w").catch((err: unknown) => err)
    expect(failure).toBeInstanceOf(ApiError)
    expect(failure).toMatchObject({
      code: "shape_not_ready",
      message: "open questions: Q1",
    })
  })

  it("reads the preview from the query", async () => {
    reply(200, {
      project: "PRJ",
      board: { id: 7, name: "Kanban" },
      backlogStatuses: ["Backlog"],
      summary: "s",
      description: "d",
    })
    const preview = await previewShapeIssue("sh-1", "/w", "PRJ", "7")
    expect(preview.board).toEqual({ id: 7, name: "Kanban" })
    expect(vi.mocked(fetch).mock.calls[0]?.[0]).toBe(
      "/shapes/sh-1/jira/preview?workspace=%2Fw&project=PRJ&board=7"
    )
  })

  it("confirms the Jira creation explicitly", async () => {
    reply(200, rawSession)
    await createShapeIssue("sh-1", "/w", "PRJ", "")
    const body: unknown = JSON.parse(
      String(vi.mocked(fetch).mock.calls[0]?.[1]?.body)
    )
    expect(body).toEqual({ workspace: "/w", project: "PRJ", confirm: true })
  })
})

describe("snapshot test counts", () => {
  const cursorOf = {
    id: "r1",
    stage: "prove",
    iteration: 1,
    hold: true,
    stop: false,
    done: false,
  }
  const read = (commands: unknown[]) =>
    snapshot({
      cursor: cursorOf,
      statements: [
        {
          stage: "prove",
          iteration: 1,
          verdict: "iterate",
          reason: "",
          commands,
        },
      ],
    }).statements[0]?.commands ?? []

  it("keeps the counts a check printed, and reads a field the engine left out as zero", () => {
    const [counted] = read([
      { command: "npm test", exitCode: 1, tests: { passed: 41, failed: 2 } },
    ])
    expect(counted?.tests).toEqual({ passed: 41, failed: 2, skipped: 0 })
  })

  it("leaves the counts out for a check that printed none", () => {
    const [plain] = read([{ command: "go build ./...", exitCode: 0 }])
    expect(plain?.tests).toBeUndefined()
  })
})

describe("projectConfig", () => {
  it("fills the defaults when the project has no policy", () => {
    const config = projectConfig({})
    expect(config.exists).toBe(false)
    expect(config.stages.conflict).toBe("hold")
    expect(config.stages.merge).toBe("auto")
    expect(config.repair).toBe(3)
    expect(config.prove.fast).toEqual([])
  })

  it("reads what the server sent and ignores what is not text", () => {
    const config = projectConfig({
      folder: "/work/api",
      exists: true,
      base: "dev",
      stages: { merge: "hold" },
      prove: { fast: ["go test ./...", 7] },
      qa: "npm test",
      repair: 5,
    })
    expect(config.stages.merge).toBe("hold")
    expect(config.prove.fast).toEqual(["go test ./..."])
    expect(config.repair).toBe(5)
    expect(config.qa).toBe("npm test")
  })
})

describe("configForm", () => {
  it("posts the project and only the keys that change, a list as one line per check", () => {
    const form = configForm("/work/api", {
      merge: "hold",
      fast: ["go vet ./...", "go test ./..."],
      repair: 5,
    })
    expect(Object.fromEntries(form)).toEqual({
      project: "/work/api",
      merge: "hold",
      fast: "go vet ./...\ngo test ./...",
      repair: "5",
    })
  })

  it("posts an emptied list as an empty field, so the server clears it", () => {
    expect(configForm("/work/api", { fast: [] }).get("fast")).toBe("")
  })
})

describe("saveConfig", () => {
  it("reads the changed lines and the settings after the write", async () => {
    reply(200, {
      lines: [{ key: "stages.merge", before: "auto", after: "hold" }],
      config: { exists: true, stages: { merge: "hold" } },
    })
    const saved = await saveConfig("/work/api", { merge: "hold" })
    expect(saved.lines).toEqual([
      { key: "stages.merge", before: "auto", after: "hold" },
    ])
    expect(saved.config.stages.merge).toBe("hold")
  })

  it("throws the server's reason when the file would refuse the value", async () => {
    reply(400, {
      error: "invalid",
      message: 'stages.merge "maybe" is not auto or hold',
    })
    await expect(saveConfig("/work/api", { merge: "maybe" })).rejects.toThrow(
      "is not auto or hold"
    )
  })
})

describe("snapshot", () => {
  const cursor = { id: "a", stage: "prove", startedAt: "2026-11-25T14:08:00Z" }

  it("keeps what a failing check printed and where its full log is", () => {
    const got = snapshot({
      cursor,
      statements: [
        {
          stage: "prove",
          verdict: "hold",
          commands: [
            {
              command: "go test",
              exitCode: 1,
              output: "FAIL x",
              logPath: "iter/1/logs/prove-1.log",
            },
          ],
        },
      ],
    })
    expect(got.statements[0]?.commands[0]).toMatchObject({
      output: "FAIL x",
      logPath: "iter/1/logs/prove-1.log",
    })
  })

  it("reads what the run is about, its folder and when it began", () => {
    const got = snapshot({
      cursor,
      statements: [],
      title: "Register a payment",
      repo: "billing-api",
    })
    expect(got.title).toBe("Register a payment")
    expect(got.repo).toBe("billing-api")
    expect(got.cursor.startedAt).toBe("2026-11-25T14:08:00Z")
  })
})

describe("snapshot agent block", () => {
  it("keeps an agent that cannot work so the page can say what to do", () => {
    const got = snapshot({
      cursor: {
        id: "a",
        agentBlock: {
          kind: "quota",
          agent: "claude-code",
          detail: "usage limit reached",
        },
      },
      statements: [],
    })
    expect(got.cursor.agentBlock).toEqual({
      kind: "quota",
      agent: "claude-code",
      detail: "usage limit reached",
    })
  })

  it("has no block when the cursor names no agent", () => {
    expect(
      snapshot({ cursor: { id: "a" }, statements: [] }).cursor.agentBlock
    ).toBeUndefined()
  })
})

describe("implementationView commit log", () => {
  it("reads the commits the server listed, oldest first", () => {
    const view = implementationView({
      stat: "",
      patch: "",
      truncated: false,
      commits: 2,
      commitLog: [
        { sha: "a".repeat(40), subject: "feat: add the retry loop", time: "2026-11-25T10:00:00Z" },
        { sha: "b".repeat(40), subject: "test: cover it", time: "2026-11-25T10:05:00Z" },
        "not a commit",
      ],
    })
    expect((view.commitLog ?? []).map((commit) => commit.subject)).toEqual(["feat: add the retry loop", "test: cover it"])
  })

  it("reads no list as an empty one", () => {
    expect(implementationView({ stat: "", patch: "", truncated: false }).commitLog).toEqual([])
  })
})

describe("getRunLog", () => {
  it("asks for the log by its run-relative path and reads the view", async () => {
    const fetchSpy = vi.fn(() =>
      Promise.resolve(
        new Response(
          JSON.stringify({ path: "iter/1/logs/prove-1.log", text: "a\nb\n", lines: 2, truncated: true }),
          { status: 200 }
        )
      )
    )
    vi.stubGlobal("fetch", fetchSpy)
    const log = await getRunLog("span-1", "iter/1/logs/prove-1.log")
    const call = fetchSpy.mock.calls[0] as unknown as [string]
    expect(call[0]).toBe("/spans/span-1/log?path=iter%2F1%2Flogs%2Fprove-1.log")
    expect(log).toEqual({ path: "iter/1/logs/prove-1.log", text: "a\nb\n", lines: 2, truncated: true })
  })
})
