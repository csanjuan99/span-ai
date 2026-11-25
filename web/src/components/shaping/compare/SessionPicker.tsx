"use client"

import { CheckField } from "@/components/list-controls/CheckField"
import { IdChip } from "@/components/list-controls/IdChip"
import { StatePill } from "@/components/shape-entry/StatePill"
import { Panel } from "@/components/shaping/session-ui"
import { SelectField } from "@/components/settings-edit/SelectField"
import {
  cardBody,
  headingText,
  hintText,
  titleText,
} from "@/components/shaping/styles"
import { sessionRef, sessionTitle } from "@/lib/shape-sessions"
import type { ShapeSession } from "@/lib/types"

const most = 5

function anchorRepo(sessions: ShapeSession[], chosen: string[]): string | null {
  const first = sessions.find((session) => chosen.includes(session.id))
  if (first === undefined) {
    return null
  }
  return first.repo
}

function sharesRepo(session: ShapeSession, anchor: string | null): boolean {
  if (anchor === null) {
    return true
  }
  return session.repo === anchor
}

function repoLine(anchor: string | null): string {
  if (anchor === null) {
    return "Pick 2 to 5 from one repository."
  }
  const name = anchor === "" ? "this folder" : anchor
  return `Same repo only: ${name}. Pick 2 to 5.`
}

/**
 * The sessions a comparison can include. At most five, and only from one repository.
 * Checking one calls `onChange` with the next set of ids.
 */
export function SessionPicker({
  sessions,
  chosen,
  onChange,
}: {
  sessions: ShapeSession[]
  chosen: string[]
  onChange: (next: string[]) => void
}) {
  const anchor = anchorRepo(sessions, chosen)
  const full = chosen.length >= most

  function toggle(id: string, on: boolean) {
    if (!on) {
      onChange(chosen.filter((item) => item !== id))
      return
    }
    if (full || chosen.includes(id)) {
      return
    }
    const session = sessions.find((item) => item.id === id)
    if (session === undefined || !sharesRepo(session, anchor)) {
      return
    }
    onChange([...chosen, id])
  }

  const addable = sessions.filter(
    (session) =>
      !chosen.includes(session.id) && sharesRepo(session, anchor) && !full
  )

  return (
    <Panel aria-label="Sessions to compare" className={cardBody}>
      <div className="flex flex-wrap items-center gap-2">
        <h2 className={headingText}>Sessions to compare</h2>
        <p className={hintText}>{repoLine(anchor)}</p>
        <span className="ml-auto">
          <SelectField
            label="Add session"
            wrapperClassName="max-w-48"
            value=""
            disabled={addable.length === 0}
            onChange={(id) => toggle(id, true)}
            options={[
              { value: "", label: "Add session" },
              ...addable.map((session) => ({
                value: session.id,
                label: `${sessionRef(session)} · ${sessionTitle(session)}`,
              })),
            ]}
          />
        </span>
      </div>
      {sessions.length === 0 ? (
        <p className={hintText}>This folder has no sessions to compare.</p>
      ) : (
        <ul className="flex flex-col gap-2">
          {sessions.map((session) => {
            const on = chosen.includes(session.id)
            const blocked = !on && (full || !sharesRepo(session, anchor))
            return (
              <li
                key={session.id}
                className="flex flex-wrap items-center gap-3 border-t border-border pt-2 first:border-t-0 first:pt-0"
              >
                <CheckField
                  className="flex min-w-0 flex-1 items-start gap-2"
                  checked={on}
                  disabled={blocked}
                  ariaLabel={`Compare ${sessionRef(session)}`}
                  onChange={(next) => toggle(session.id, next)}
                  label={
                    <span className="flex min-w-0 flex-col items-start gap-0.5">
                      <span className={titleText}>{sessionTitle(session)}</span>
                      <IdChip>{sessionRef(session)}</IdChip>
                    </span>
                  }
                />
                <StatePill state={session.status} />
              </li>
            )
          })}
        </ul>
      )}
    </Panel>
  )
}
