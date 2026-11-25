import { CircleAlert, Hand } from "lucide-react"
import type { ReactNode } from "react"

import { StatusBlock } from "@/components/status-block/StatusBlock"
import type { Needs } from "@/lib/needs-you"

/**
 * The one block above the tabs that holds whatever the run waits on. The header names it in
 * words and color together; the panels the page already has (questions, decision, failed
 * check, conflict, approval) sit inside it, so nothing the person must act on hides in a tab.
 * `aside` is the right column: what the action will do, and the action itself.
 */
export function NeedsYou({
  needs,
  title,
  where,
  aside,
  children,
}: {
  needs: Needs
  /** The band's label; the needs' own label when empty. */
  title?: string
  where: string
  aside?: ReactNode
  children: ReactNode
}) {
  return (
    <StatusBlock
      label="Needs you"
      tone={needs.tone}
      icon={needs.tone === "fail" ? CircleAlert : Hand}
      title={title === undefined || title === "" ? needs.label : title}
      where={where}
      strong
      aside={aside}
    >
      {children}
    </StatusBlock>
  )
}
