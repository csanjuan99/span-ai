import {
  ActivityIcon,
  ClipboardListIcon,
  LockIcon,
  SplitIcon,
  type LucideIcon,
} from "lucide-react"

import { Pill, type PillTone } from "@/components/onboarding/Pill"
import { stateLabels, type ShapeState } from "@/lib/shape-sessions"

const tone: Record<ShapeState, { tone: PillTone; Icon: LucideIcon }> = {
  interviewing: { tone: "acc", Icon: ActivityIcon },
  ready: { tone: "hold", Icon: ClipboardListIcon },
  split: { tone: "hold", Icon: SplitIcon },
  sealed: { tone: "pass", Icon: LockIcon },
}

/** The state of a session as a pill: its icon and its word. */
export function StatePill({ state }: { state: ShapeState }) {
  const { tone: colour, Icon } = tone[state]
  return (
    <Pill tone={colour}>
      <Icon aria-hidden="true" className="size-3" />
      {stateLabels[state]}
    </Pill>
  )
}
