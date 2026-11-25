/** How a stretch of a failing check's output is drawn. */
export type Tone = "fail" | "name" | "muted" | "plain"

/** One stretch of a line, with the tone it is drawn in. */
export interface Segment {
  text: string
  tone: Tone
}

const failingTest = /^(\s*--- FAIL:)(\s+)(\S+)(\s+)(\(\d+(?:\.\d+)?s\))(.*)$/
const verdictWord = /^(FAIL)(\s.*|)$/
const assertion = /^\s+\S+:\d+:/

/**
 * Splits one line of test output into the parts a reader scans for: the failing marker, the
 * test's name and the detail under it. Output that is not a test report stays plain, so a
 * line is never coloured on a guess.
 */
export function highlight(line: string): Segment[] {
  if (line === "") {
    return []
  }
  const test = failingTest.exec(line)
  if (test !== null) {
    const [, marker = "", gap = "", name = "", gap2 = "", timing = "", rest = ""] = test
    return [
      { text: marker, tone: "fail" as const },
      { text: gap, tone: "plain" as const },
      { text: name, tone: "name" as const },
      { text: gap2, tone: "plain" as const },
      { text: timing, tone: "muted" as const },
      ...(rest === "" ? [] : [{ text: rest, tone: "plain" as const }]),
    ]
  }
  if (assertion.test(line)) {
    return [{ text: line, tone: "muted" }]
  }
  const verdict = verdictWord.exec(line)
  if (verdict !== null) {
    const [, word = "", tail = ""] = verdict
    return [{ text: word, tone: "fail" }, ...(tail === "" ? [] : [{ text: tail, tone: "plain" as const }])]
  }
  return [{ text: line, tone: "plain" }]
}
