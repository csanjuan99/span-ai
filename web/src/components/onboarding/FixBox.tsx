"use client"

import { CheckIcon, CopyIcon } from "lucide-react"
import { useState, type ReactNode } from "react"

import { Button } from "@/components/ui/button"

/** A command with a Copy button. When the browser refuses the copy it says so instead of failing quietly. */
export function CommandLine({ command }: { command: string }) {
  const [state, setState] = useState<"idle" | "copied" | "refused">("idle")

  function copy() {
    navigator.clipboard.writeText(command).then(
      () => setState("copied"),
      () => setState("refused")
    )
  }

  return (
    <div className="flex items-center justify-between gap-3 rounded-[10px] border border-border bg-surface-code px-3 py-1.5 font-mono text-[13px]">
      <span className="min-w-0 break-all">
        <span className="text-primary">$ </span>
        {command}
      </span>
      <Button
        type="button"
        size="xs"
        variant="ghost"
        aria-label={`Copy ${command}`}
        onClick={copy}
      >
        {state === "copied" ? <CheckIcon /> : <CopyIcon />}
        {state === "copied"
          ? "Copied"
          : state === "refused"
            ? "Select and copy"
            : "Copy"}
      </Button>
    </div>
  )
}

/** The exact fix under a problem: what to do, and the command when there is one. */
export function FixBox({
  children,
  command,
}: {
  children: ReactNode
  command?: string
}) {
  return (
    <div className="mt-2 flex flex-col gap-1.5">
      <div className="text-xs font-medium text-muted-foreground">Fix</div>
      <p className="text-[13px]">{children}</p>
      {command !== undefined ? <CommandLine command={command} /> : null}
    </div>
  )
}
