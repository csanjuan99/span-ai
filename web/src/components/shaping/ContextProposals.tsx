"use client"

import { useState } from "react"

import { Button } from "@/components/ui/button"
import { Checkbox } from "@/components/ui/checkbox"
import type { ShapeProposal } from "@/lib/types"
import { PromptHeading } from "@/components/shell/PromptHeading"

/** Additions to span/context/product.md. Nothing is written until the person ticks it. */
export function ContextProposals({
  proposals,
  busy,
  onAccept,
}: {
  proposals: ShapeProposal[]
  busy: boolean
  onAccept: (ids: string[]) => void
}) {
  const [ticked, setTicked] = useState<string[]>([])
  const waiting = proposals.filter((proposal) => proposal.status === "proposed")
  if (waiting.length === 0) {
    return null
  }
  return (
    <section className="flex flex-col gap-2">
      <PromptHeading>product context</PromptHeading>
      <p className="text-sm text-pretty text-muted-foreground">
        The agent suggests these for product.md. Only the ones you tick are
        written.
      </p>
      {waiting.map((proposal) => (
        <label key={proposal.id} className="flex items-start gap-2 text-sm">
          <Checkbox
            checked={ticked.includes(proposal.id)}
            onCheckedChange={(checked) =>
              setTicked((current) =>
                checked === true
                  ? [...current, proposal.id]
                  : current.filter((id) => id !== proposal.id)
              )
            }
          />
          <span>
            <span className="text-muted-foreground">{proposal.section}: </span>
            {proposal.text}
          </span>
        </label>
      ))}
      <Button
        type="button"
        variant="secondary"
        className="self-start"
        disabled={busy || ticked.length === 0}
        onClick={() => onAccept(ticked)}
      >
        Add to product context
      </Button>
    </section>
  )
}
