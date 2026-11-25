"use client"

import type { ReactNode } from "react"

import { CommandLine } from "@/components/problem/CommandLine"

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
