import { CopyChip } from "@/components/ledger/CopyChip"

/** A command to paste into a terminal, with a copy button. */
export function CommandLine({ command }: { command: string }) {
  return (
    <div className="rounded-[10px] border border-border bg-surface-code py-1.5 font-mono text-[13px] leading-5">
      <div className="flex items-center justify-between gap-4 px-3.5">
        <span className="min-w-0 truncate">
          <span className="text-code-keyword" aria-hidden="true">
            $
          </span>{" "}
          {command}
        </span>
        <CopyChip label="command" value={command}>
          Copy
        </CopyChip>
      </div>
    </div>
  )
}
