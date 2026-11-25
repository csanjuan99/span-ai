import { describe, expect, it } from "vitest"

import { tidy } from "@/lib/tidy"

describe("tidy", () => {
  it("removes terminal color codes, with or without the escape character", () => {
    expect(tidy("\u001b[0;36mRules file\u001b[0m: AGENTS.md")).toBe("Rules file: AGENTS.md")
    expect(tidy("[0;36mRules file[0m: AGENTS.md")).toBe("Rules file: AGENTS.md")
  })

  it("reads a Go duration as a person would: whole seconds, no fraction", () => {
    expect(tidy("lead time 54m22.887124s")).toBe("lead time 54m 22s")
    expect(tidy("lead time 1h3m5.5s")).toBe("lead time 1h 03m")
    expect(tidy("took 850ms")).toBe("took 850ms")
    expect(tidy("lead time 12.4s")).toBe("lead time 12s")
  })

  it("leaves ordinary text alone", () => {
    expect(tidy("brief accepted")).toBe("brief accepted")
    expect(tidy("array[0] and list[1;2]")).toBe("array[0] and list[1;2]")
  })
})
