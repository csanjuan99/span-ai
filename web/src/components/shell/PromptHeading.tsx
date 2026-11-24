import type { ReactNode } from "react"

import { cn } from "@/lib/utils"

/** A heading that reads as a shell prompt: the page title is the large one, a section the small one. */
export function PromptHeading({
  as: Tag = "h2",
  children,
}: {
  as?: "h1" | "h2"
  children: ReactNode
}) {
  return (
    <Tag className={cn("text-balance", Tag === "h1" ? "text-xl" : "text-sm")}>
      <span className="text-primary">$ </span>
      {children}
    </Tag>
  )
}
