import { describe, expect, it } from "vitest"

import { ApiError } from "@/lib/api"
import { problemOf } from "@/lib/problem"

describe("problemOf", () => {
  it("knows a server that did not answer", () => {
    expect(problemOf(new ApiError("unreachable", "Failed to fetch"))).toBe(
      "unreachable"
    )
  })

  it("knows a run that is not on this desk", () => {
    expect(problemOf(new ApiError("not_found", "run not found"))).toBe(
      "notfound"
    )
  })

  it("leaves every other failure to the caller", () => {
    expect(problemOf(new ApiError("invalid", "bad"))).toBeNull()
    expect(problemOf(new Error("boom"))).toBeNull()
    expect(problemOf("text")).toBeNull()
  })
})
