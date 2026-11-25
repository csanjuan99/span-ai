import {
  cleanup,
  fireEvent,
  render,
  screen,
  waitFor,
} from "@testing-library/react"
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest"

import {
  Implementation,
  type RunFacts,
} from "@/components/implementation/Implementation"
import { changeOf } from "@/lib/change"
import { parseFeature } from "@/lib/featuredoc"
import type {
  Audit,
  CommitDiff,
  CommitStat,
  FileEntry,
  Reasoning,
} from "@/lib/implementation-api"
import type { Cursor } from "@/lib/types"

const api = vi.hoisted(() => ({
  readFiles: vi.fn(),
  readCommits: vi.fn(),
  readCommitDiff: vi.fn(),
  readReasoning: vi.fn(),
  readAudit: vi.fn(),
  readFileDiff: vi.fn(),
  markViewed: vi.fn(),
  checkDecisions: vi.fn(),
}))

vi.mock("@/lib/implementation-api", () => api)

const cursor: Cursor = {
  id: "r1",
  stage: "inspect",
  iteration: 1,
  hold: false,
  stop: false,
  done: false,
  reason: "",
  gitSHA: "f2f2080984371ad259b88ce5a89d6149822dacb6",
  changeURL: "",
  branch: "feat/x",
  base: "dev",
  workspace: "/w",
  issue: "",
  questions: [],
}

const run: RunFacts = {
  cursor,
  statements: [],
  agents: {},
  feature: null,
  journal: { events: [], intact: true, brokenAt: 0 },
}

const patch = [
  "diff --git a/pay.go b/pay.go",
  "--- a/pay.go",
  "+++ b/pay.go",
  "@@ -1,1 +1,2 @@",
  " a",
  "+charge",
].join("\n")

const file = (over: Partial<FileEntry> = {}): FileEntry => ({
  path: "pay.go",
  from: "",
  status: "M",
  added: 1,
  removed: 0,
  binary: false,
  generated: false,
  large: false,
  viewed: false,
  ...over,
})

const commit = (over: Partial<CommitStat> = {}): CommitStat => ({
  sha: "91d4be7aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
  subject: "Test retry timing",
  time: "2026-11-25T14:47:00Z",
  parent: "7b02d9e0000000000000000000000000000000000",
  files: 2,
  added: 65,
  removed: 5,
  renameOnly: false,
  ...over,
})

const reasoning = (over: Partial<Reasoning> = {}): Reasoning => ({
  title: "Pay",
  specs: [{ id: "S1", text: "A failed delivery is retried." }],
  tasks: [{ text: "Add the scheduler", done: true, commit: "7b02d9e" }],
  decisions: [
    {
      id: "D1",
      what: "Retry on 429 only",
      why: "other codes will not succeed",
      source: "",
      files: ["pay.go"],
      assumed: true,
      checked: false,
    },
  ],
  unchecked: 1,
  recorded: true,
  ...over,
})

const audit = (over: Partial<Audit> = {}): Audit => ({
  entries: [
    {
      seq: 1,
      time: "2026-11-25T14:08:00Z",
      actor: "person",
      what: "started",
      hash: "a3f1aaaa",
      prev: "",
      want: "",
    },
    {
      seq: 2,
      time: "2026-11-25T14:10:00Z",
      actor: "span",
      what: "sealed intake",
      hash: "8c02bbbb",
      prev: "a3f1aaaa",
      want: "a3f1aaaa",
    },
  ],
  intact: true,
  brokenAt: 0,
  head: "8c02bbbb",
  ...over,
})

/** Renders the tab the way the page does: the change is built once and handed in. */
function view(
  over: {
    stat?: string
    patch?: string
    truncated?: boolean
    run?: RunFacts
  } = {}
) {
  const facts = over.run ?? run
  const truncated = over.truncated ?? false
  const change = changeOf({
    stat: over.stat ?? "",
    patch: over.patch ?? "",
    truncated,
    statements: facts.statements,
    cursor: facts.cursor,
    feature: facts.feature,
  })
  return <Implementation change={change} truncated={truncated} run={facts} />
}

describe("Implementation", () => {
  beforeEach(() => {
    api.readFiles.mockResolvedValue([file()])
    api.readCommits.mockResolvedValue([commit()])
    api.readReasoning.mockResolvedValue(reasoning())
    api.readAudit.mockResolvedValue(audit())
    api.markViewed.mockResolvedValue(undefined)
  })
  afterEach(() => {
    cleanup()
    vi.clearAllMocks()
  })

  it("leads with the size of the change and lists the files", () => {
    render(view({ stat: " pay.go | 1 +", patch }))
    expect(screen.getByRole("region", { name: "Summary" })).toBeTruthy()
    expect(screen.getByText("A 0 · M 1 · R 0 · D 0")).toBeTruthy()
    expect(screen.getByRole("button", { name: "pay.go" })).toBeTruthy()
    expect(screen.getByText("0 of 1 viewed")).toBeTruthy()
  })

  it("lists a file the cut patch did not reach and says the patch was cut", () => {
    render(
      view({ stat: " pay.go | 1 +\n docs.md | 4 ++--", patch, truncated: true })
    )
    expect(screen.getByRole("button", { name: "docs.md" })).toBeTruthy()
    expect(screen.getByText(/The patch was cut at 256 KiB/)).toBeTruthy()
  })

  it("says so when there is nothing yet", () => {
    render(view())
    expect(screen.getByText("No commit on this run yet.")).toBeTruthy()
    expect(screen.getByText("Nothing to assess yet")).toBeTruthy()
  })

  it("keeps the verdict and names a risk with the file it points at", () => {
    const risky = ["diff --git a/.env b/.env", "@@ -0,0 +1,1 @@", "+X=1"].join(
      "\n"
    )
    render(view({ stat: " .env | 1 +", patch: risky }))
    expect(screen.getByText("high risk")).toBeTruthy()
    expect(screen.getByText("Touches an environment file")).toBeTruthy()
  })

  it("opens a file from the list into the diff, with its line numbers and a way to the next file", () => {
    render(view({ stat: " pay.go | 1 +", patch }))
    fireEvent.click(screen.getByRole("button", { name: "pay.go" }))
    expect(screen.getByRole("region", { name: "Diff of pay.go" })).toBeTruthy()
    expect(screen.getByText("charge")).toBeTruthy()
    expect(screen.getByText("File 1 of 1")).toBeTruthy()
    expect(
      (screen.getByRole("button", { name: "Next file" }) as HTMLButtonElement)
        .disabled
    ).toBe(true)
  })

  it("saves a viewed mark on the run and takes it back when the save fails", async () => {
    api.markViewed.mockRejectedValueOnce(new Error("disk full"))
    render(view({ stat: " pay.go | 1 +", patch }))
    const box = screen.getByRole("checkbox", {
      name: "Viewed pay.go",
    }) as HTMLInputElement
    fireEvent.click(box)
    expect(api.markViewed).toHaveBeenCalledWith("r1", "pay.go", true)
    await waitFor(() => expect(screen.getByText("disk full")).toBeTruthy())
    expect(
      (
        screen.getByRole("checkbox", {
          name: "Viewed pay.go",
        }) as HTMLInputElement
      ).checked
    ).toBe(false)
  })

  it("holds back a generated file until the reader asks for it", async () => {
    api.readFiles.mockResolvedValue([file({ generated: true })])
    api.readFileDiff.mockResolvedValue({
      ...file({ generated: true }),
      commit: "",
      patch,
      truncated: false,
      hunks: 1,
    })
    render(view({ stat: " pay.go | 1 +", patch }))
    await waitFor(() => expect(api.readFiles).toHaveBeenCalled())
    fireEvent.click(screen.getByRole("button", { name: "pay.go" }))
    await waitFor(() => expect(screen.getByText("Generated")).toBeTruthy())
    expect(screen.queryByText("charge")).toBeNull()
    fireEvent.click(
      screen.getByRole("button", { name: "Show the diff anyway" })
    )
    await waitFor(() => expect(screen.getByText("charge")).toBeTruthy())
    expect(api.readFileDiff).toHaveBeenCalledWith("r1", "pay.go", {
      force: true,
    })
  })

  it("lists the commits with their sizes and reads one commit on its own", async () => {
    const detail: CommitDiff = {
      parent: "7b02d9e",
      patch,
      truncated: false,
      files: [file()],
    }
    api.readCommitDiff.mockResolvedValue(detail)
    render(view({ stat: " pay.go | 1 +", patch }))
    fireEvent.click(screen.getByRole("tab", { name: "Commits" }))
    await waitFor(() =>
      expect(screen.getByText("Test retry timing")).toBeTruthy()
    )
    expect(screen.getByText("+65")).toBeTruthy()
    fireEvent.click(screen.getByRole("button", { name: /91d4be7/ }))
    await waitFor(() =>
      expect(api.readCommitDiff).toHaveBeenCalledWith("r1", commit().sha)
    )
    await waitFor(() =>
      expect(screen.getByText(/Only this commit/)).toBeTruthy()
    )
  })

  it("shows the agent's decisions as assumed and marks them checked", async () => {
    api.checkDecisions.mockResolvedValue(
      reasoning({
        unchecked: 0,
        decisions: [
          {
            id: "D1",
            what: "Retry on 429 only",
            why: "other codes will not succeed",
            source: "",
            files: [],
            assumed: true,
            checked: true,
          },
        ],
      })
    )
    render(
      view({
        stat: " pay.go | 1 +",
        patch,
        run: { ...run, feature: parseFeature("# T") },
      })
    )
    fireEvent.click(screen.getByRole("tab", { name: "Reasoning" }))
    await waitFor(() =>
      expect(screen.getByText("Retry on 429 only")).toBeTruthy()
    )
    expect(screen.getByText("A failed delivery is retried.")).toBeTruthy()
    expect(screen.getByText("Assumed: check before you approve")).toBeTruthy()
    fireEvent.click(
      screen.getByRole("button", { name: "Mark 1 assumption as checked" })
    )
    await waitFor(() =>
      expect(api.checkDecisions).toHaveBeenCalledWith("r1", ["D1"])
    )
    await waitFor(() =>
      expect(
        (
          screen.getByRole("checkbox", {
            name: "I checked D1",
          }) as HTMLInputElement
        ).checked
      ).toBe(true)
    )
  })

  it("tells the reader what the agent did not record instead of staying silent", async () => {
    api.readReasoning.mockResolvedValue(
      reasoning({ decisions: [], unchecked: 0 })
    )
    render(view({ stat: " pay.go | 1 +", patch }))
    fireEvent.click(screen.getByRole("tab", { name: "Reasoning" }))
    await waitFor(() =>
      expect(
        screen.getByText("The agent recorded no decisions for this run")
      ).toBeTruthy()
    )
  })

  it("says the chain holds, or where it broke", async () => {
    render(view({ stat: " pay.go | 1 +", patch }))
    fireEvent.click(screen.getByRole("tab", { name: "Audit" }))
    await waitFor(() => expect(screen.getByText("Chain intact")).toBeTruthy())
    cleanup()
    api.readAudit.mockResolvedValue(audit({ intact: false, brokenAt: 2 }))
    render(view({ stat: " pay.go | 1 +", patch }))
    fireEvent.click(screen.getByRole("tab", { name: "Audit" }))
    await waitFor(() =>
      expect(screen.getByText(/Audit chain broken · entry 2 of 2/)).toBeTruthy()
    )
    expect(screen.getByText("Broken here")).toBeTruthy()
  })

  it("shows the route and the command of each action, closed until opened", () => {
    render(view({ stat: " pay.go | 1 +", patch }))
    expect(screen.getByText("API / CLI")).toBeTruthy()
    expect(
      screen.getByText(/\.\/span implementation r1 --files --json/)
    ).toBeTruthy()
  })
})
