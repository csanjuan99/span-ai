import type { Started } from "@/lib/api"

/** The start request narrowed to one repository, from the base given or the one Span picks. */
export function retryForm(
  sent: FormData,
  repo: string,
  base: string
): FormData {
  const data = new FormData()
  for (const [name, value] of sent.entries()) {
    if (name !== "repo" && name !== "base") {
      data.append(name, value)
    }
  }
  data.append("repo", repo)
  if (base !== "") {
    data.append("base", base)
  }
  return data
}

/** The results once a repository that had failed was started again. */
export function withRetried(
  before: Started,
  repo: string,
  retried: Started
): Started {
  const failed = [
    ...before.failed.filter((item) => item.repo !== repo),
    ...retried.failed,
  ]
  return {
    ...before,
    runs: [...before.runs, ...retried.runs],
    failed,
    partial: failed.length > 0,
    notes: "",
  }
}

/** The results with a failed repository left for later. */
export function withSkipped(before: Started, repo: string): Started {
  return {
    ...before,
    failed: before.failed.filter((item) => item.repo !== repo),
    notes: "",
  }
}
