import { describe, expect, it } from "vitest"

import { unsealedBuild } from "@/lib/unsealed"
import type { Statement } from "@/lib/types"

const sealed = (stage: string): Statement => ({ stage, iteration: 1, verdict: "advance", reason: "", commands: [] })

describe("unsealedBuild", () => {
  it("is true only for a finished run whose ledger has no sealed build", () => {
    expect(unsealedBuild(true, [sealed("intake"), sealed("frame")])).toBe(true)
  })

  it("is false when build was sealed", () => {
    expect(unsealedBuild(true, [sealed("intake"), sealed("build")])).toBe(false)
  })

  it("is false for a run that is still going, since build may yet be sealed", () => {
    expect(unsealedBuild(false, [sealed("intake")])).toBe(false)
  })

  it("is false when the ledger is empty, since nothing says what happened", () => {
    expect(unsealedBuild(true, [])).toBe(false)
  })
})
