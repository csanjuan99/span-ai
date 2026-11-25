import type { Started } from "@/lib/api"

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
  if (typeof value !== "object" || value === null) {
    return null
  }
  const body = value as Record<string, unknown>
  const code = typeof body.code === "string" ? body.code : ""
  const message = typeof body.message === "string" ? body.message : ""
  const number = typeof body.number === "number" ? body.number : 0
  if ((code === "" && message === "") || body.ok === true) {
    return null
  }
  return {
    number,
    code,
    message,
    runId: typeof body.runId === "string" ? body.runId : undefined,
    branch: typeof body.branch === "string" ? body.branch : undefined,
    base: typeof body.base === "string" ? body.base : undefined,
  }
}
