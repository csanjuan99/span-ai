"use client"

import { useCallback, useEffect, useState } from "react"
import { LinkIcon } from "lucide-react"

import {
  ActionPanel,
  Panel,
  Pill,
  wideColumns,
} from "@/components/shaping/session-ui"
import { Button } from "@/components/ui/button"
import { useNotedAction } from "@/components/shaping/use-noted-action"
import { dayLabel } from "@/lib/clock"
import { reasonText } from "@/lib/guards"
import {
  createShare,
  listShares,
  revokeShare,
  type ShareLink,
} from "@/lib/shape-flows"
import { emptyPanelText, hintText } from "@/components/shaping/styles"

const lasts = [1, 7, 30]

/** Read-only links to a sealed story: make one, copy it, revoke it. */
export function SharePanel({
  id,
  workspace,
  sealed,
}: {
  id: string
  workspace: string
  sealed: boolean
}) {
  const [links, setLinks] = useState<ShareLink[]>([])
  const [days, setDays] = useState(7)
  const [copied, setCopied] = useState("")
  const { note, setNote, busy, run } = useNotedAction()

  // load is the effect's dependency and the refresh the actions below call: a stable identity
  // keeps the effect from re-running on every render.
  const load = useCallback(() => {
    return listShares(id, workspace)
      .then(setLinks)
      .catch((err: unknown) =>
        setNote(reasonText(err, "the links could not be read"))
      )
  }, [id, workspace, setNote])

  useEffect(() => {
    if (sealed) {
      void load()
    }
  }, [load, sealed])

  const pathOf = (token: string) =>
    `${window.location.origin}/shaping/${id}/shared/${token}?workspace=${encodeURIComponent(workspace)}`

  function copy(token: string) {
    void navigator.clipboard
      .writeText(pathOf(token))
      .then(() => setCopied(token))
      .catch(() =>
        setNote("The link could not be copied. Select it and copy by hand.")
      )
  }

  if (!sealed) {
    return (
      <Panel className={emptyPanelText}>
        Only a sealed story can be shared. Seal it first.
      </Panel>
    )
  }
  return (
    <div className={wideColumns}>
      <Panel aria-label="Links" className="flex flex-col gap-3 p-4">
        <h2 className="text-lg font-semibold">Links</h2>
        {links.length === 0 ? (
          <p className={hintText}>No link yet.</p>
        ) : (
          <ul className="flex flex-col gap-2">
            {links.map((link) => (
              <li
                key={link.token}
                className="flex flex-wrap items-center gap-2 rounded-[10px] border border-border p-3 text-[13px]"
              >
                <span className="font-mono">{link.token.slice(0, 8)}…</span>
                <Pill tone={link.revoked ? "mut" : "pass"} small>
                  {link.revoked
                    ? "Revoked"
                    : `until ${dayLabel(link.expiresAt)}`}
                </Pill>
                <span className="text-xs text-muted-foreground">
                  opened {link.opens} {link.opens === 1 ? "time" : "times"}
                </span>
                {link.revoked ? null : (
                  <span className="ml-auto flex gap-2">
                    <Button
                      type="button"
                      size="sm"
                      variant="outline"
                      onClick={() => copy(link.token)}
                    >
                      {copied === link.token ? "Copied" : "Copy link"}
                    </Button>
                    <Button
                      type="button"
                      size="sm"
                      variant="ghost"
                      disabled={busy}
                      aria-label={`Revoke ${link.token.slice(0, 8)}`}
                      onClick={() =>
                        void run(async () => {
                          await revokeShare(id, workspace, link.token)
                          await load()
                        }, "the link was not revoked")
                      }
                    >
                      Revoke
                    </Button>
                  </span>
                )}
              </li>
            ))}
          </ul>
        )}
        {note === "" ? null : (
          <p role="alert" className="text-sm text-destructive">
            {note}
          </p>
        )}
      </Panel>
      <aside className="flex flex-col gap-4 lg:sticky lg:top-4 lg:self-start">
        <ActionPanel
          title="If you make a link"
          icon={LinkIcon}
          rows={[
            {
              label: "Shows",
              body: "The story, its rules and examples, and the digest. Not the questions or the code.",
            },
            {
              label: "Read-only",
              body: "Whoever opens it cannot change anything. It opens with the same access as Span.",
            },
            { label: "Ends", body: "When it expires or when you revoke it." },
          ]}
        >
          <div role="group" aria-label="Lasts" className="flex gap-2">
            {lasts.map((value) => (
              <Button
                key={value}
                type="button"
                size="sm"
                variant={days === value ? "default" : "outline"}
                aria-pressed={days === value}
                onClick={() => setDays(value)}
              >
                {value} {value === 1 ? "day" : "days"}
              </Button>
            ))}
          </div>
          <Button
            type="button"
            disabled={busy}
            onClick={() =>
              void run(async () => {
                await createShare(id, workspace, days)
                await load()
              }, "the link was not made")
            }
          >
            Make a link
          </Button>
        </ActionPanel>
      </aside>
    </div>
  )
}
