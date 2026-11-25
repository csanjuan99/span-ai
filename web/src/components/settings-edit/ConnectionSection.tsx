"use client"

import { useState, type FormEvent } from "react"

import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import { getJSON, rememberServeToken, serveToken } from "@/lib/api"
import { reasonText } from "@/lib/guards"

/** Server URL, health check, and access token kept in this browser tab only. */
export function ConnectionSection() {
  const [server, setServer] = useState(() =>
    typeof window === "undefined" ? "" : window.location.origin
  )
  const [token, setToken] = useState(() => serveToken())
  const [state, setState] = useState("unknown")
  const [note, setNote] = useState("")
  const [testing, setTesting] = useState(false)

  async function test() {
    setTesting(true)
    setNote("")
    try {
      const before = performance.now()
      await getJSON("/health")
      const ms = Math.round(performance.now() - before)
      setState(`connected · ${ms} ms`)
    } catch (err: unknown) {
      setState("unreachable")
      setNote(reasonText(err, "the server did not answer"))
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
    <form className="flex flex-col gap-4" onSubmit={save}>
      <div className="flex flex-col gap-2">
        <Label htmlFor="settings-server">Server</Label>
        <Input
          id="settings-server"
          value={server}
          onChange={(event) => setServer(event.target.value)}
          className="max-w-md"
        />
        <p className="text-[13px] text-muted-foreground">State: {state}</p>
      </div>
      <div className="flex flex-col gap-2">
        <Label htmlFor="settings-token">Access token</Label>
        <Input
          id="settings-token"
          type="password"
          autoComplete="off"
          className="max-w-md"
          value={token}
          onChange={(event) => setToken(event.target.value)}
        />
        <p className="text-[13px] text-muted-foreground">
          Kept in this browser tab only. Set <code>SPAN_TOKEN</code> on the
          server and pass the same value here for remote use.
        </p>
      </div>
      <div className="flex flex-wrap gap-2">
        <Button type="submit" variant="secondary" className="min-h-11 min-w-11">
          Save
        </Button>
        <Button
          type="button"
          variant="outline"
          className="min-h-11 min-w-11"
          disabled={testing}
          onClick={() => void test()}
        >
          Test
        </Button>
      </div>
      {note !== "" ? (
        <p className="text-sm text-pretty" role="status">
          {note}
        </p>
      ) : null}
    </form>
  )
}
