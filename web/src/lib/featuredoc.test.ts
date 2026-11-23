import { describe, expect, it } from "vitest"

import { parseFeature, splitDecision } from "@/lib/featuredoc"

const doc = [
  "# Pay with the stored card",
  "",
  "## Specs",
  "- S1: **charge** the card",
  "",
  "## Tasks",
  "- [x] T1 charge",
  "- [ ] T2 refund",
  "",
  "## Log",
  "L1 quoted brief",
  "- Decision: keep amounts in cents because floats drift",
  "Decision: no retry",
  "- **Risk:** double charge if the webhook repeats (low, an idempotency test covers it)",
  "Risk: none found",
].join("\n")

describe("parseFeature", () => {
  it("reads the title, specs and tasks", () => {
    const got = parseFeature(doc)
    expect(got.title).toBe("Pay with the stored card")
    expect(got.specs).toEqual(["S1: charge the card"])
    expect(got.tasks).toEqual([
      { text: "T1 charge", done: true },
      { text: "T2 refund", done: false },
    ])
    expect(got.hasLog).toBe(true)
  })

  it("reads decisions with and without a reason", () => {
    expect(parseFeature(doc).decisions).toEqual([
      { what: "keep amounts in cents", why: "floats drift" },
      { what: "no retry", why: "" },
    ])
  })

  it("reads risks and tells none-found apart", () => {
    expect(parseFeature(doc).risks).toEqual([
      { text: "double charge if the webhook repeats (low, an idempotency test covers it)", none: false },
      { text: "none found", none: true },
    ])
  })

  it("gives empty lists for a document it does not understand", () => {
    expect(parseFeature("")).toEqual({ title: "", specs: [], tasks: [], decisions: [], risks: [], hasLog: false })
  })

  it("splits a decision on the first because only", () => {
    expect(splitDecision("a because b because c")).toEqual({ what: "a", why: "b because c" })
  })
})
