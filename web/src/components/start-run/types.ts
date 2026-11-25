import type { Started } from "@/lib/api"
import { isRecord, text } from "@/lib/guards"

/** Coded refusal from GET /start/precheck or a failed start. */
export interface StartRefusal {
  number: number
  code: string
  message: string
  runId?: string
  branch?: string
  base?: string
}

export interface StartRunState {
  started: Started | null
  refusal: StartRefusal | null
  runNote: string
  starting: boolean
}

/** The refusal the desk shows when the server turned the access token away. */
export function tokenRefusal(err: Error): StartRefusal {
  return { number: 18, code: "token_refused", message: err.message }
}

export function refusalOf(value: unknown): StartRefusal | null {
  if (!isRecord(value)) {
    return null
  }
  const code = text(value.code)
  const message = text(value.message)
  if ((code === "" && message === "") || value.ok === true) {
    return null
  }
  return {
    number: typeof value.number === "number" ? value.number : 0,
    code,
    message,
    runId: text(value.runId) || undefined,
    branch: text(value.branch) || undefined,
    base: text(value.base) || undefined,
  }
}

/** A branch that exists to offer when the base is missing: main, then master, then the first other. */
export function suggestBranch(branches: string[], missing: string): string {
  const others = branches.filter((name) => name !== missing)
  return (
    others.find((name) => name === "main") ??
    others.find((name) => name === "master") ??
    others[0] ??
    ""
  )
}
