import { getJSON } from "@/lib/api"
import { isRecord } from "@/lib/guards"

/** The Jira session saved for a folder, as the server answers it. */
export function readJiraSession(folder: string): Promise<unknown> {
  return getJSON(`/jira?path=${encodeURIComponent(folder)}`)
}

/** Whether a Jira account is saved for a folder, and which one. */
export interface JiraSession {
  configured: boolean
  email: string
  baseUrl: string
}

/** No account saved. */
export const noSession: JiraSession = {
  configured: false,
  email: "",
  baseUrl: "",
}

/** Reads the `/jira` reply. Anything that is not text or true reads as not set. */
export function jiraSessionOf(value: unknown): JiraSession {
  if (!isRecord(value)) {
    return noSession
  }
  return {
    configured: value.configured === true,
    email: typeof value.email === "string" ? value.email : "",
    baseUrl: typeof value.baseUrl === "string" ? value.baseUrl : "",
  }
}
