"use client"

import { useState } from "react"

import { CheckLine } from "@/components/onboarding/CheckLine"
import { FixBox } from "@/components/onboarding/FixBox"
import { Panel } from "@/components/onboarding/Frame"
import { Pill } from "@/components/onboarding/Pill"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { checkJiraAccount, saveJiraAccount } from "@/lib/api"
import { reasonText } from "@/lib/guards"
import type { JiraSession } from "@/lib/jira-session"
import type { JiraCheck } from "@/lib/types"
import { designInput } from "@/lib/field-class"

/** The optional Jira account: the site, the email and a token, tested before it is saved. */
export function JiraPanel({
  folder,
  session,
  onSaved,
  onSkip,
}: {
  folder: string
  session: JiraSession
  onSaved: () => void
  onSkip: () => void
}) {
  const [url, setUrl] = useState(session.baseUrl)
  const [email, setEmail] = useState(session.email)
  const [token, setToken] = useState("")
  const [result, setResult] = useState<JiraCheck | null>(null)
  const [note, setNote] = useState("")
  const [busy, setBusy] = useState(false)

  async function test() {
    setBusy(true)
    setNote("")
    try {
      const account = { url, email, token }
      const checked = await checkJiraAccount(folder, account)
      setResult(checked)
      if (checked.status === "ok") {
        await saveJiraAccount(folder, account)
        setToken("")
        onSaved()
      }
    } catch (reason: unknown) {
      setNote(reasonText(reason, "the account could not be tested"))
    } finally {
      setBusy(false)
    }
  }

  return (
    <Panel
      title="Jira"
      badge={<Pill tone="mut">Optional</Pill>}
      lead="Lets Span read stories and move them when a run finishes. Skip it and paste a brief by hand instead."
    >
      {session.configured ? (
        <ul>
          <CheckLine tone="pass">
            <b className="text-pass">Connected.</b> {session.email}{" "}
            <span className="font-mono text-xs">{session.baseUrl}</span>
          </CheckLine>
        </ul>
      ) : null}
      <label className="flex flex-col gap-1.5">
        <span className="text-xs font-medium">Jira site</span>
        <Input
          className={designInput}
          placeholder="https://acme.atlassian.net"
          value={url}
          onChange={(event) => setUrl(event.target.value)}
        />
      </label>
      <label className="flex flex-col gap-1.5">
        <span className="text-xs font-medium">Email</span>
        <Input
          className={designInput}
          type="email"
          value={email}
          onChange={(event) => setEmail(event.target.value)}
        />
      </label>
      <label className="flex flex-col gap-1.5">
        <span className="text-xs font-medium">API token</span>
        <Input
          className={designInput}
          type="password"
          autoComplete="off"
          placeholder={
            session.configured ? "Saved. Leave blank to keep it." : ""
          }
          value={token}
          onChange={(event) => setToken(event.target.value)}
        />
      </label>
      {result !== null && result.status !== "ok" ? (
        <ul>
          <CheckLine
            tone="fail"
            fix={
              result.status === "unauthorized" ? (
                <FixBox>
                  Create a new token at{" "}
                  <span className="font-mono text-xs">
                    id.atlassian.com/manage-profile/security/api-tokens
                  </span>
                  , then paste it above.
                </FixBox>
              ) : null
            }
          >
            <b className="text-fail">
              {result.status === "unauthorized"
                ? "Rejected."
                : result.status === "unreachable"
                  ? "Not reachable."
                  : "Not saved."}
            </b>{" "}
            {result.message}
          </CheckLine>
        </ul>
      ) : null}
      {result?.status === "ok" ? (
        <ul>
          <CheckLine tone="pass">
            <b className="text-pass">Accepted and saved.</b> Jira answered.
          </CheckLine>
        </ul>
      ) : null}
      {note !== "" ? (
        <p role="alert" className="text-sm text-destructive">
          {note}
        </p>
      ) : null}
      <div className="flex flex-wrap items-center gap-3">
        <Button
          type="button"
          variant="outline"
          disabled={busy || url.trim() === ""}
          onClick={() => void test()}
        >
          Test and save
        </Button>
        <Button type="button" variant="ghost" onClick={onSkip}>
          Skip Jira
        </Button>
      </div>
    </Panel>
  )
}
