import { useState, type FormEvent } from "react"

import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"

/**
 * Settings that belong to this browser and this server, not to a story: the access token a
 * remote server asks for, and the Jira session of the folder. Closed by default so the desk
 * stays about the story, and opened by the desk when the server refuses a request.
 */
export function Settings({
  open,
  onOpenChange,
  token,
  needsToken,
  onSaveToken,
  jira,
  onCloseJira,
}: {
  open: boolean
  onOpenChange: (open: boolean) => void
  token: string
  /** The server refused a request for lack of a token. */
  needsToken: boolean
  onSaveToken: (value: string) => void
  /** The Jira session of the folder, or "" when there is none. */
  jira: string
  onCloseJira: () => void
}) {
  const [value, setValue] = useState(token)

  function submit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault()
    onSaveToken(value)
  }

  return (
    <section className="flex flex-col gap-3 border-t border-border pt-6" aria-label="Settings">
      <h2 className="text-balance text-sm">
        <span className="text-primary">$ </span>settings
        <Button type="button" variant="link" size="xs" aria-expanded={open} onClick={() => onOpenChange(!open)}>
          {open ? "Hide" : "Show"}
        </Button>
      </h2>
      {open ? (
        <div className="flex flex-col gap-6">
          <form className="flex flex-col gap-2" onSubmit={submit}>
            <Label htmlFor="serve-token">Access token</Label>
            <p className="text-pretty text-sm text-muted-foreground">
              Only for a Span server reached from another machine. A server started with <code>SPAN_TOKEN</code> refuses every request
              without it. On your own machine, leave it empty.
            </p>
            {needsToken ? <p className="text-pretty text-sm text-hold">This server refused the last request. Add its token and try again.</p> : null}
            <div className="flex flex-wrap items-end gap-2">
              <Input
                id="serve-token"
                type="password"
                autoComplete="off"
                className="w-64"
                value={value}
                onChange={(event) => setValue(event.target.value)}
              />
              <Button type="submit" variant="secondary">
                Save token
              </Button>
            </div>
            <p className="text-pretty text-xs text-muted-foreground">Kept in this browser tab only.</p>
          </form>
          <div className="flex flex-col gap-2">
            <span className="text-sm">Jira session</span>
            {jira === "" ? (
              <p className="text-pretty text-sm text-muted-foreground">No session for this folder. Connect one in setup.</p>
            ) : (
              <>
                <p className="text-pretty text-sm">{jira}</p>
                <Button type="button" variant="secondary" className="self-start" onClick={onCloseJira}>
                  Close session
                </Button>
              </>
            )}
          </div>
        </div>
      ) : null}
    </section>
  )
}
