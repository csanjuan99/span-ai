import { afterEach, describe, expect, it, vi } from "vitest"

import {
  archiveShape,
  deleteShape,
  listShapes,
  resumeShape,
  setShapeAgent,
  shapeSession,
} from "@/lib/api"

afterEach(() => vi.unstubAllGlobals())

function stub(body: unknown, status = 200) {
  const fetchMock = vi.fn(() =>
    Promise.resolve(
      new Response(body === null ? null : JSON.stringify(body), { status })
    )
  )
  vi.stubGlobal("fetch", fetchMock)
  return fetchMock
}

describe("shape session api", () => {
  it("reads the derived fields and the failure of a session", () => {
    const got = shapeSession({
      id: "a",
      ref: "SES-A",
      title: "T",
      openQuestions: 2,
      jiraKey: "BILL-1",
      updatedAt: "2026-11-25T10:00:00Z",
      archived: true,
      agent: "codex",
      model: "o3",
      failure: { at: "t", agent: "codex", kind: "usage_limit", message: "limit" },
    })
    expect(got).toMatchObject({
      ref: "SES-A",
      title: "T",
      openQuestions: 2,
      jiraKey: "BILL-1",
      archived: true,
      agent: "codex",
      model: "o3",
      failure: { kind: "usage_limit", message: "limit" },
    })
  })

  it("defaults a session an older server sent without them", () => {
    const got = shapeSession({ id: "a" })
    expect(got.failure).toBeNull()
    expect(got.archived).toBe(false)
    expect(got.openQuestions).toBe(0)
  })

  it("sends only the filters that are set", async () => {
    const fetchMock = stub([])
    await listShapes("/w", { status: "sealed", query: "refund", sort: "created", repo: "", archived: "only" })
    const [url] = fetchMock.mock.calls[0] as unknown as [string]
    const query = new URL(url, "http://x").searchParams
    expect(Object.fromEntries(query)).toEqual({
      workspace: "/w",
      status: "sealed",
      q: "refund",
      sort: "created",
      archived: "only",
    })
  })

  it("calls the routes of resume, agent, archive and delete", async () => {
    const fetchMock = stub({ id: "a" })
    await resumeShape("a", "/w", "codex", "o3")
    await setShapeAgent("a", "/w", "codex", "")
    await archiveShape("a", "/w", false)
    const sent = fetchMock.mock.calls.map((call) => {
      const [url, init] = call as unknown as [string, RequestInit]
      return [url, init.method, init.body]
    })
    expect(sent).toEqual([
      ["/shapes/a/resume", "POST", JSON.stringify({ workspace: "/w", agent: "codex", model: "o3" })],
      ["/shapes/a/agent", "POST", JSON.stringify({ workspace: "/w", agent: "codex", model: "" })],
      ["/shapes/a/archive", "POST", JSON.stringify({ workspace: "/w", archived: false })],
    ])
    const del = stub(null, 204)
    await deleteShape("a", "/w")
    const [url, init] = del.mock.calls[0] as unknown as [string, RequestInit]
    expect([url, init.method]).toEqual(["/shapes/a?workspace=%2Fw", "DELETE"])
  })
})
