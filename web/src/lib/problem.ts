import { ApiError } from "@/lib/api"

/** A failure that replaces the whole page: the server cannot be reached, or the run is not there. */
export type Problem = "unreachable" | "notfound"

/** Which page-level problem a failed read is, or null when the caller should report it in place. */
export function problemOf(err: unknown): Problem | null {
  if (!(err instanceof ApiError)) {
    return null
  }
  if (err.code === "unreachable") {
    return "unreachable"
  }
  return err.code === "not_found" ? "notfound" : null
}
