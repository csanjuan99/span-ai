import { afterEach, describe, expect, it, vi } from "vitest"

import { getJSON, UnauthorizedError } from "@/lib/api"

afterEach(() => {
  vi.unstubAllGlobals()
})

describe("a server that needs its token", () => {
  it("is reported as such, not as a body that was not JSON", async () => {
    vi.stubGlobal("fetch", () => Promise.resolve(new Response("unauthorized\n", { status: 401 })))
    await expect(getJSON("/spans")).rejects.toBeInstanceOf(UnauthorizedError)
  })

  it("leaves other failures with the server's own message", async () => {
    vi.stubGlobal("fetch", () => Promise.resolve(new Response(JSON.stringify({ message: "no such span" }), { status: 404 })))
    await expect(getJSON("/spans/x")).rejects.toThrow("no such span")
  })
})
