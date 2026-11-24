import { getJSON } from "@/lib/api"

/** The Jira session saved for a folder, as the server answers it. */
export function readJiraSession(folder: string): Promise<unknown> {
  return getJSON(`/jira?path=${encodeURIComponent(folder)}`)
}
