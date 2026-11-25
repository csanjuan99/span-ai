import type { Statement } from "@/lib/types"

/**
 * Whether a run finished without ever sealing its build. Only a finished run with a ledger can
 * say so: while the run goes on build may still be sealed, and an empty ledger says nothing.
 */
export function unsealedBuild(done: boolean, statements: Statement[]): boolean {
  if (!done || statements.length === 0) {
    return false
  }
  return !statements.some((item) => item.stage === "build" && item.verdict === "advance")
}
