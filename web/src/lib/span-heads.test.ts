import { describe, expect, it } from "vitest"

import { spanHeadsOf } from "@/lib/api"

describe("spanHeadsOf", () => {
  it("reads the label and where each run stands", () => {
    const heads = spanHeadsOf({
      spans: [
        { id: "2026-11-19-0253-87f0", label: "SAL-3-2026-11-19-0253-87f0-salute-hermes", issue: "SAL-3", repo: "salute-hermes", stage: "merge", verdict: "hold" },
        { id: "2026-11-19-0253-1a7d", label: "SAL-3-2026-11-19-0253-1a7d-salute-infra", issue: "SAL-3", repo: "salute-infra", stage: "prove", verdict: "running" },
      ],
    })
    expect(heads).toHaveLength(2)
    expect(heads[0].label).toBe("SAL-3-2026-11-19-0253-87f0-salute-hermes")
    expect(heads[0].repo).toBe("salute-hermes")
    expect(heads[1].verdict).toBe("running")
  })

  it("drops a row with no id and tolerates an empty body", () => {
    expect(spanHeadsOf({ spans: [{ label: "ghost" }] })).toHaveLength(0)
    expect(spanHeadsOf(undefined)).toHaveLength(0)
  })
})
