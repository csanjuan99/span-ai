/** A question the run is waiting on. Notices from Jira are not questions. */
export function waitingQuestions(cursor: {
  questions: string[]
  reason: string
  hold: boolean
  stop: boolean
} | null): string[] {
  if (cursor === null || (!cursor.hold && !cursor.stop)) {
    return []
  }
  if (cursor.questions.length > 0) {
    return cursor.questions
  }
  const prefix = "open questions: "
  if (!cursor.reason.startsWith(prefix)) {
    return []
  }
  return cursor.reason
    .slice(prefix.length)
    .split("; ")
    .map((part) => part.trim())
    .filter((part) => part !== "" && !notice(part))
}

function notice(part: string): boolean {
  return part === "jira account is not configured" || part.startsWith("the Jira comment was not posted") || part.startsWith("commented and moved")
}
