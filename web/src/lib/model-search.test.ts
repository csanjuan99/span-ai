import { describe, expect, it } from "vitest"

import { searchModels } from "@/lib/model-search"

const models = [
  { id: "auto", label: "Auto (default)" },
  { id: "gpt-5.3-codex-high", label: "Codex 5.3 High" },
  { id: "claude-opus-5-thinking-high", label: "Claude Opus 5 1M Thinking" },
]

describe("searchModels", () => {
  it("keeps every model for an empty query, whatever is already chosen", () => {
    expect(searchModels(models, "")).toEqual(models)
    expect(searchModels(models, "   ")).toEqual(models)
  })

  it("matches the id or the label, every word, in any order and case", () => {
    expect(searchModels(models, "HIGH codex").map((m) => m.id)).toEqual(["gpt-5.3-codex-high"])
    expect(searchModels(models, "opus").map((m) => m.id)).toEqual(["claude-opus-5-thinking-high"])
    expect(searchModels(models, "default").map((m) => m.id)).toEqual(["auto"])
  })

  it("returns nothing when no model matches", () => {
    expect(searchModels(models, "zzz")).toEqual([])
  })
})
