import type { Statement } from "@/lib/types"

/** One line per command the run ran, in ledger order: the stage, the command and how it ended. */
export function commandLog(statements: Statement[]): string[] {
  return statements.flatMap((item) => item.commands.map((command) => `${item.stage}  ${command.command}  exit ${command.exitCode}`))
}
