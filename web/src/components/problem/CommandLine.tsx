import { CopyChip } from "@/components/ledger/CopyChip"

/** A command to paste into a terminal, with a copy button. */
export function CommandLine({ command }: { command: string }) {
  return (
    <div className="flex items-center gap-3 rounded-lg bg-surface-code px-3 py-2 font-mono text-sm">
      <span className="text-muted-foreground" aria-hidden="true">
        $
      </span>
      <span className="min-w-0 flex-1 truncate">{command}</span>
      <CopyChip label="command" value={command}>
        Copy
      </CopyChip>
    </div>
  )
}
