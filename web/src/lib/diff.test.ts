import { describe, expect, it } from "vitest"

import { filesOf, parseDiff, parseStat, splitPath, totals } from "@/lib/diff"

const patch = [
  "diff --git a/pay.go b/pay.go",
  "index 111..222 100644",
  "--- a/pay.go",
  "+++ b/pay.go",
  "@@ -3,4 +3,5 @@ func Charge() {",
  " keep",
  "-old",
  "+new",
  "+extra",
  " tail",
  "\\ No newline at end of file",
  "diff --git a/README.md b/README.md",
  "new file mode 100644",
  "--- /dev/null",
  "+++ b/README.md",
  "@@ -0,0 +1,1 @@",
  "+hello",
].join("\n")

describe("parseDiff", () => {
  it("numbers each side and counts additions and removals", () => {
    const [file] = parseDiff(patch)
    expect(file?.path).toBe("pay.go")
    expect([file?.added, file?.removed]).toEqual([2, 1])
    const lines = file?.hunks[0]?.lines ?? []
    expect(lines.map((line) => [line.kind, line.oldNo, line.newNo])).toEqual([
      ["ctx", 3, 3],
      ["del", 4, null],
      ["add", null, 4],
      ["add", null, 5],
      ["ctx", 5, 6],
      ["note", null, null],
    ])
    expect(file?.hunks[0]?.section).toBe("func Charge() {")
  })

  it("reads a new file and several files", () => {
    const files = parseDiff(patch)
    expect(files.map((file) => [file.path, file.status])).toEqual([
      ["pay.go", "modified"],
      ["README.md", "added"],
    ])
  })

  it("keeps what it read when the patch is cut", () => {
    expect(parseDiff(patch.slice(0, 120))[0]?.path).toBe("pay.go")
    expect(parseDiff("")).toEqual([])
  })
})

describe("stat helpers", () => {
  it("reads rows and ignores the summary line", () => {
    const rows = parseStat(" pay.go | 3 ++-\n 1 file changed, 2 insertions(+)")
    expect(rows).toEqual([{ path: "pay.go", changes: 3, plus: 2, minus: 1 }])
  })

  it("totals and splits paths", () => {
    expect(totals([{ added: 2, removed: 1 }, { added: 1, removed: 0 }])).toEqual({ files: 2, added: 3, removed: 1 })
    expect(splitPath("a/b/c.go")).toEqual({ dir: "a/b/", name: "c.go" })
    expect(splitPath("c.go")).toEqual({ dir: "", name: "c.go" })
  })
})

describe("filesOf", () => {
  const patch = "diff --git a/pay.go b/pay.go\n--- a/pay.go\n+++ b/pay.go\n@@ -1 +1,2 @@\n a\n+b\n"

  it("holds the files of the patch, then the ones only the stat names", () => {
    const files = filesOf(" pay.go | 1 +\n big.go | 40 ++++++++++++++++++++++++++++++++++++++++\n", patch)
    expect(files.map((file) => file.path)).toEqual(["pay.go", "big.go"])
    expect(files[1]?.hunks).toEqual([])
    expect(files[1]?.status).toBe("modified")
  })

  it("is empty when there is neither a patch nor a stat", () => {
    expect(filesOf("", "")).toEqual([])
  })
})
