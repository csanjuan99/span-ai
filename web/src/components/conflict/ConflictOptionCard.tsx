"use client"

import { cn } from "@/lib/utils"

import { actionLabel, riskLabel } from "@/lib/conflict"
import type { ConflictOption, ConflictRisk } from "@/lib/types"
import { checkedCard, roomyRadio } from "@/components/list-controls/RadioCard"
import { Badge } from "@/components/ui/badge"

/** Each risk differs in words and in weight, so it never reads from color alone. */
const riskStyle: Record<
  ConflictRisk,
  { variant: "outline" | "secondary" | "destructive"; className: string }
> = {
  low: { variant: "outline", className: "" },
  medium: { variant: "secondary", className: "" },
  high: { variant: "destructive", className: "font-semibold uppercase" },
}

/** The risk of one option as a badge. */
export function RiskBadge({ risk }: { risk: ConflictRisk }) {
  const style = riskStyle[risk]
  return (
    <Badge variant={style.variant} className={style.className}>
      {riskLabel(risk)}
    </Badge>
  )
}

/**
 * One option of a group: a native radio inside a label, so arrow keys, focus and the
 * radio role come from the browser. The regenerate command sits outside the label so
 * reading it does not change the choice. What the option writes shows under the group.
 */
export function ConflictOptionCard({
  name,
  option,
  checked,
  onChoose,
}: {
  name: string
  option: ConflictOption
  checked: boolean
  onChoose: (option: string) => void
}) {
  return (
    <li
      className={cn(
        "flex flex-col gap-2 rounded-[10px] border border-input bg-card p-3",
        checked && checkedCard
      )}
    >
      <label className="flex cursor-pointer items-start gap-2.5">
        <input
          type="radio"
          name={name}
          value={option.id}
          checked={checked}
          onChange={() => onChoose(option.id)}
          className={roomyRadio}
        />
        <span className="flex flex-col gap-0.5">
          <span className="text-sm font-semibold">{option.title}</span>
          {option.summary !== "" ? (
            <span className="text-xs text-pretty text-muted-foreground">
              {option.summary}
            </span>
          ) : null}
        </span>
      </label>
      <div className="flex flex-wrap items-center gap-2">
        <RiskBadge risk={option.risk} />
        <span className="text-xs text-muted-foreground">
          {actionLabel(option.action)}
        </span>
      </div>
      {option.action === "regenerate" && option.command !== "" ? (
        <div className="flex flex-col gap-1">
          <pre className="overflow-x-auto rounded-md border border-border bg-muted p-2 font-mono text-xs">
            <code>{option.command}</code>
          </pre>
          <p className="text-xs text-muted-foreground">
            Runs only after you apply
          </p>
        </div>
      ) : null}
    </li>
  )
}
