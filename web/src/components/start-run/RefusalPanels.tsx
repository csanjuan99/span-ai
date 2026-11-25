"use client"

import { Link } from "react-router"

import { Button } from "@/components/ui/button"
import type { StartRefusal } from "@/components/start-run/types"

const titles: Record<number, string> = {
  6: "A run for this story already exists",
  8: "Jira is not connected",
  14: "This folder is not a git repository",
  15: "That base branch is missing",
  16: "This checkout is busy",
  17: "No agent is ready",
  18: "The access token was refused",
}

/** N4: coded refusals before a run exists. */
export function RefusalPanels({
  refusal,
  onDismiss,
}: {
  refusal: StartRefusal
  onDismiss: () => void
}) {
  const title = titles[refusal.number] ?? "Span refused to start"
  return (
    <section
      aria-label="Refusal"
      className="flex flex-col gap-3 rounded-[14px] border border-hold/40 bg-hold/10 p-4"
    >
      <h2 className="text-lg font-semibold">{title}</h2>
      <p className="text-sm text-pretty">{refusal.message}</p>
      {refusal.runId !== undefined && refusal.runId !== "" ? (
        <p className="text-sm">
          Run{" "}
          <Link
            className="text-primary underline-offset-4 hover:underline"
            to={`/spans/${refusal.runId}`}
          >
            {refusal.runId}
          </Link>
        </p>
      ) : null}
      <MotionRow refusal={refusal} />
      <Button
        type="button"
        variant="secondary"
        className="min-h-11 self-start"
        onClick={onDismiss}
      >
        Back to the form
      </Button>
    </section>
  )
}

function MotionRow({ refusal }: { refusal: StartRefusal }) {
  if (refusal.number === 8) {
    return (
      <p className="text-sm text-muted-foreground">
        Connect Jira in the form below or under Settings.
      </p>
    )
  }
  if (refusal.number === 18) {
    return (
      <p className="text-sm text-muted-foreground">
        Add the server access token under{" "}
        <Link
          className="text-primary underline-offset-4 hover:underline"
          to="/settings"
        >
          Settings
        </Link>
        .
      </p>
    )
  }
  return null
}
