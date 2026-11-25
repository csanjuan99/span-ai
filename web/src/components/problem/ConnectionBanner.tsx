import { CheckIcon, LoaderCircleIcon } from "lucide-react"

import type { ConnectionNotice } from "@/hooks/use-connection-notice"

/** A calm line under the header while the live connection is down, and a short one when it returns. */
export function ConnectionBanner({
  notice,
  lastSeen,
}: {
  notice: ConnectionNotice
  lastSeen: string
}) {
  if (notice === "idle") {
    return null
  }
  if (notice === "back") {
    return (
      <p
        role="status"
        className="flex items-center gap-2 rounded-lg bg-pass-soft px-3 py-2 text-sm text-pass"
      >
        <CheckIcon aria-hidden="true" className="size-4" />
        Back online. Updated just now.
      </p>
    )
  }
  return (
    <p
      role="status"
      className="flex items-center gap-2 rounded-lg bg-hold-soft px-3 py-2 text-sm text-hold"
    >
      <LoaderCircleIcon
        aria-hidden="true"
        className="size-4 animate-spin motion-reduce:animate-none"
      />
      <span>
        Reconnecting to Span… Your run keeps going on your machine.
        {lastSeen !== "" ? ` Last updated ${lastSeen}.` : ""}
      </span>
    </p>
  )
}
