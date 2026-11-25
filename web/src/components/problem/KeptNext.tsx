import type { ReactNode } from "react"

import { InfoIcon } from "lucide-react"

function Row({ label, children }: { label: string; children: ReactNode }) {
  return (
    <div className="grid grid-cols-[64px_1fr] gap-2.5 border-t border-border py-2">
      <dt className="pt-0.5 text-xs font-medium text-muted-foreground">
        {label}
      </dt>
      <dd className="text-[13px] leading-5 text-pretty">{children}</dd>
    </div>
  )
}

/** What a problem leaves safe, and the one thing to do about it. */
export function KeptNext({ kept, next }: { kept: string; next: string }) {
  return (
    <div className="rounded-[10px] bg-muted px-3.5 py-3">
      <div className="mb-1 flex items-center gap-2 text-sm font-semibold">
        <InfoIcon className="size-4 text-primary" aria-hidden="true" />
        What to know
      </div>
      <dl>
        <Row label="Kept">
          <span
            aria-hidden="true"
            className="mr-1.5 font-mono font-semibold text-pass"
          >
            +
          </span>
          {kept}
        </Row>
        <Row label="Next">{next}</Row>
      </dl>
    </div>
  )
}
