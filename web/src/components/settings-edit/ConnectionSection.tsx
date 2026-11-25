"use client"

import { CheckIcon, RefreshCwIcon } from "lucide-react"
import { useState, type FormEvent } from "react"

import { FieldRow } from "@/components/settings-edit/FieldRow"
import { Pill } from "@/components/onboarding/Pill"
import { CommandLine } from "@/components/problem/CommandLine"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import {
  getJSON,
  rememberServeToken,
  serveToken,
  UnauthorizedError,
} from "@/lib/api"
import { touchButton } from "@/lib/button-class"
import { hhmm } from "@/lib/clock"
import { isRecord, reasonText, text } from "@/lib/guards"
import { designInput } from "@/lib/field-class"

/** What the last Test found, so the State row can say it in the board's words. */
type Check =
  | { kind: "none" }
  | { kind: "connected"; ms: number; version: string; at: string }
  | { kind: "refused" }
  | { kind: "unreachable"; reason: string }

function StateRow({ check }: { check: Check }) {
  if (check.kind === "connected") {
    return (
      <div className="flex flex-wrap items-center gap-x-3 gap-y-1">
        <Pill tone="pass">
          <CheckIcon className="size-3.5" aria-hidden="true" />
          Connected
        </Pill>
        <span className="text-xs text-muted-foreground">
          Answered in {check.ms} ms
          {check.version === "" ? "" : ` · Span ${check.version}`} · checked{" "}
          {check.at}
        </span>
      </div>
    )
  }
  if (check.kind === "refused") {
    return <Pill tone="hold">Refused · the server wants an access token</Pill>
  }
  if (check.kind === "unreachable") {
    return (
      <div className="flex flex-wrap items-center gap-x-3 gap-y-1">
        <Pill tone="fail">Unreachable</Pill>
        <span className="text-xs text-muted-foreground">{check.reason}</span>
      </div>
    )
  }
  return <Pill tone="mut">Not checked</Pill>
}

/** The server this page talks to, its state, and the access token kept in this browser tab only. */
export function ConnectionSection() {
  const server = typeof window === "undefined" ? "" : window.location.origin
  const [token, setToken] = useState(() => serveToken())
  const [check, setCheck] = useState<Check>({ kind: "none" })
  const [note, setNote] = useState("")
  const [testing, setTesting] = useState(false)

  async function test() {
    setTesting(true)
    setNote("")
    try {
      const before = performance.now()
      const body = await getJSON("/health")
      setCheck({
        kind: "connected",
        ms: Math.round(performance.now() - before),
        version: isRecord(body) ? text(body.version) : "",
        at: hhmm(new Date()),
      })
    } catch (err: unknown) {
      setCheck(
        err instanceof UnauthorizedError
          ? { kind: "refused" }
          : {
              kind: "unreachable",
              reason: reasonText(err, "the server did not answer"),
            }
      )
    } finally {
      setTesting(false)
    }
  }

  function save(event: FormEvent<HTMLFormElement>) {
    event.preventDefault()
    rememberServeToken(token.trim())
    setNote(
      token.trim() === ""
        ? "Access token cleared."
        : "Access token saved for this tab."
    )
  }

  return (
    <form onSubmit={save}>
      <FieldRow
        label="Server address"
        fallback="the address serving this page"
        changed={false}
        unsaved={false}
      >
        <Input
          aria-label="Server address"
          readOnly
          className={designInput}
          value={server}
        />
      </FieldRow>
      <FieldRow
        label="State"
        fallback="not checked"
        changed={false}
        unsaved={false}
      >
        <div role="status">
          <StateRow check={check} />
        </div>
      </FieldRow>
      <FieldRow
        label="Access token"
        fallback="empty"
        changed={false}
        unsaved={false}
      >
        <Input
          aria-label="Access token"
          type="password"
          autoComplete="off"
          placeholder="Empty on your own machine"
          className={designInput}
          value={token}
          onChange={(event) => setToken(event.target.value)}
        />
        <div className="flex flex-wrap gap-2">
          <Button type="submit" variant="outline" className={touchButton}>
            <CheckIcon aria-hidden="true" />
            Save token
          </Button>
          <Button
            type="button"
            variant="outline"
            className={touchButton}
            disabled={testing}
            onClick={() => void test()}
          >
            <RefreshCwIcon aria-hidden="true" />
            Test
          </Button>
        </div>
        <p className="text-xs text-muted-foreground">
          Kept in this browser tab only. Close the tab and it is gone. Span
          never writes it to a file.
        </p>
        {note !== "" ? (
          <p className="text-sm text-pretty" role="status">
            {note}
          </p>
        ) : null}
      </FieldRow>
      <div className="flex flex-col gap-2 border-t border-border pt-4 text-[13px] leading-5">
        <p>
          A token is only needed for a Span server reached from another machine.
          On your own machine it stays empty. On the server, set{" "}
          <code className="font-mono">SPAN_TOKEN</code> and paste the same value
          here.
        </p>
        <p className="font-medium">If the test fails</p>
        <p>
          <span className="font-semibold text-hold">Refused (401).</span> The
          server wants an access token and none was sent. Paste it above and
          press Test.
        </p>
        <p>
          <span className="font-semibold text-fail">Unreachable.</span> Nothing
          answered at {server}. Your runs are safe. Start Span in a terminal,
          then Test again.
        </p>
        <CommandLine command="span serve" />
      </div>
    </form>
  )
}
