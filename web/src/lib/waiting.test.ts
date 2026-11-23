import { describe, expect, it } from "vitest"

import { waitingQuestions } from "@/lib/waiting"

const cursor = {
  questions: [] as string[],
  reason: "",
  hold: false,
  stop: false,
}

describe("waitingQuestions", () => {
  it("uses the questions stored on the cursor", () => {
    expect(waitingQuestions({ ...cursor, hold: true, questions: ["Which permission atom is Pago.Ver?"] })).toEqual([
      "Which permission atom is Pago.Ver?",
    ])
  })

  it("reads an older reason and drops the jira notice", () => {
    const reason =
      "open questions: Which permission atom is Pago.Ver?; Acceptance criteria are missing.; jira account is not configured"
    expect(waitingQuestions({ ...cursor, stop: true, reason })).toEqual([
      "Which permission atom is Pago.Ver?",
      "Acceptance criteria are missing.",
    ])
  })

  it("leaves a real stop without questions", () => {
    expect(waitingQuestions({ ...cursor, stop: true, reason: "out of scope" })).toEqual([])
  })
})
