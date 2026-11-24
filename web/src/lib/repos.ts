import { isRecord } from "@/lib/guards"

/** The repository names a `/repos` reply lists. The reply is untrusted input. */
export function repoNames(value: unknown): string[] {
  if (!isRecord(value) || !Array.isArray(value.repos)) {
    return []
  }
  return value.repos
    .filter(isRecord)
    .flatMap((item) =>
      typeof item.name === "string" && item.name !== "" ? [item.name] : []
    )
}
