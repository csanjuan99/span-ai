/** True for a plain object. A reply from the server is untrusted until it passes this. */
export function isRecord(value: unknown): value is Record<string, unknown> {
  return typeof value === "object" && value !== null
}

/** The first allowed value equal to the input, or the fallback. Narrows without a cast. */
export function oneOf<T extends string>(
  value: unknown,
  allowed: readonly T[],
  fallback: T
): T {
  return allowed.find((item) => item === value) ?? fallback
}

/** A count read from a reply: a finite number above zero, rounded down, else zero. */
export function whole(value: unknown): number {
  return typeof value === "number" && Number.isFinite(value) && value > 0
    ? Math.floor(value)
    : 0
}

/** The entries of an array reply that are strings; anything else is an empty list. */
export function strings(value: unknown): string[] {
  return Array.isArray(value)
    ? value.filter((item): item is string => typeof item === "string")
    : []
}

/** The message of a thrown error, or the fallback when something else was thrown. */
export function reasonText(reason: unknown, fallback: string): string {
  return reason instanceof Error ? reason.message : fallback
}

/** The entries of an array reply that are objects; anything else is an empty list. */
export function rows(value: unknown): Record<string, unknown>[] {
  return Array.isArray(value) ? value.filter(isRecord) : []
}

/** A string field of an untrusted reply; anything else reads as empty. */
export function text(value: unknown): string {
  return typeof value === "string" ? value : ""
}
