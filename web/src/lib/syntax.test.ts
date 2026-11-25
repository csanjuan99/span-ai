import { describe, expect, it } from "vitest"

import { languageOf, tokenize } from "@/lib/syntax"

describe("tokenize", () => {
  it("colours keywords, calls, strings and numbers in Go", () => {
    const tokens = tokenize("return c.retry.Schedule(d, 429) // retried", "go")
    expect(tokens.find((t) => t.text === "return")?.kind).toBe("keyword")
    expect(tokens.find((t) => t.text === "Schedule")?.kind).toBe("function")
    expect(tokens.find((t) => t.text === "429")?.kind).toBe("number")
    expect(tokens.at(-1)).toEqual({ kind: "comment", text: "// retried" })
  })

  it("keeps a comment marker inside a string as part of the string", () => {
    const tokens = tokenize('url := "http://x" // real comment', "go")
    expect(tokens.find((t) => t.kind === "string")?.text).toBe('"http://x"')
    expect(tokens.at(-1)?.text).toBe("// real comment")
  })

  it("gives back the same text it was given", () => {
    const line = '  if err != nil { return fmt.Errorf("send %s: %w", id, err) }'
    expect(
      tokenize(line, "go")
        .map((t) => t.text)
        .join("")
    ).toBe(line)
  })

  it("leaves a file with no known language plain", () => {
    expect(tokenize("# heading", languageOf("README.md"))).toEqual([
      { kind: "plain", text: "# heading" },
    ])
    expect(languageOf("a/b/client.go")).toBe("go")
  })
})
