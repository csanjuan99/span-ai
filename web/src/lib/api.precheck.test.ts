import { afterEach, describe, expect, it, vi } from "vitest"

import { getPrecheck } from "@/lib/api"

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

describe("getPrecheck", () => {
  const refusal = {
    number: 15,
    code: "base_missing",
    message: 'branch "develop" is not in ws',
  }

  it("returns the refusal a blocked start answers with, whatever its status", async () => {
    for (const status of [400, 401, 409]) {
      reply(status, refusal)
      await expect(
        getPrecheck("/start/precheck?base=develop")
      ).resolves.toEqual(refusal)
    }
  })

  it("returns the body of a start nothing blocks", async () => {
    reply(200, { ok: true })
    await expect(getPrecheck("/start/precheck")).resolves.toEqual({ ok: true })
  })

  it("still fails, with the code its status stands for, when the answer is not a refusal", async () => {
    reply(500, { message: "the checkout could not be read" })
    await expect(getPrecheck("/start/precheck")).rejects.toMatchObject({
      name: "ApiError",
      message: "the checkout could not be read",
    })
    reply(404, "404 page not found")
    await expect(getPrecheck("/start/precheck")).rejects.toMatchObject({
      code: "not_found",
    })
    reply(401, "no token")
    await expect(getPrecheck("/start/precheck")).rejects.toMatchObject({
      name: "UnauthorizedError",
    })
  })
})
