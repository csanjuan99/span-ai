"use client"

import { Button } from "@/components/ui/button"
import { PromptHeading } from "@/components/shell/PromptHeading"

/** The digest is long; twelve characters tell two sealed stories apart. */
export function shortDigest(digest: string): string {
  return digest.slice(0, 12)
}

/** The story markdown, with the seal button while the session is ready and the digest once sealed. */
export function Story({
  story,
  digest,
  sealed,
  busy,
  onSeal,
}: {
  story: string
  digest: string
  sealed: boolean
  busy: boolean
  onSeal: () => void
}) {
  return (
    <section className="flex flex-col gap-2">
      <PromptHeading>story</PromptHeading>
      <pre className="border border-border bg-card p-3 text-sm whitespace-pre-wrap">
        {story}
      </pre>
      {sealed ? (
        <p className="text-sm text-muted-foreground">
          Sealed, digest{" "}
          <span className="text-foreground tabular-nums">
            {shortDigest(digest)}
          </span>
        </p>
      ) : (
        <Button
          type="button"
          className="self-start"
          disabled={busy}
          onClick={onSeal}
        >
          Seal story
        </Button>
      )}
    </section>
  )
}
