/** What a problem leaves safe, and the one thing to do about it. */
export function KeptNext({ kept, next }: { kept: string; next: string }) {
  return (
    <dl className="grid gap-3 border-t border-border pt-3 text-sm sm:grid-cols-2">
      <div className="flex flex-col gap-1">
        <dt className="text-xs tracking-wide text-muted-foreground uppercase">
          Kept
        </dt>
        <dd className="text-pretty">
          <span aria-hidden="true" className="mr-1.5 font-mono text-pass">
            +
          </span>
          {kept}
        </dd>
      </div>
      <div className="flex flex-col gap-1">
        <dt className="text-xs tracking-wide text-muted-foreground uppercase">
          Next
        </dt>
        <dd className="text-pretty">{next}</dd>
      </div>
    </dl>
  )
}
