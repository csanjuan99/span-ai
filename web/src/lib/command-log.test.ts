import { describe, expect, it } from "vitest"

import { commandLog } from "@/lib/command-log"
import type { Statement } from "@/lib/types"

const statement = (stage: string, commands: Statement["commands"]): Statement => ({
  stage,
  iteration: 1,
  verdict: "advance",
  reason: "",
  commands,
})

describe("commandLog", () => {
  it("lists every command the run ran, in order, with its stage and exit code", () => {
    const lines = commandLog([
      statement("prove", [
        { command: "go build ./...", exitCode: 0 },
        { command: "go test ./...", exitCode: 1 },
      ]),
      statement("qa", [{ command: "npm run e2e", exitCode: 0 }]),
    ])
    expect(lines).toEqual([
      "prove  go build ./...  exit 0",
      "prove  go test ./...  exit 1",
      "qa  npm run e2e  exit 0",
    ])
  })

  it("is empty when no command ran", () => {
    expect(commandLog([statement("frame", [])])).toEqual([])
  })
})
