"use client"

import { RefusalPanels } from "@/components/start-run/RefusalPanels"
import {
  StartForm,
  type StartFormProps,
} from "@/components/start-run/StartForm"
import { StartingResults } from "@/components/start-run/StartingResults"
import type { StartRefusal } from "@/components/start-run/types"
import type { Started } from "@/lib/api"
import { cn } from "@/lib/utils"

/**
 * Desk start flow: N1–N4 in one panel. The form stays mounted while the panel is closed, so
 * the sample brief can still start a run from its fields.
 */
export function StartRunPanel({
  visible,
  refusal,
  onDismissRefusal,
  started,
  onOpenRun,
  onBack,
  form,
}: {
  visible: boolean
  refusal: StartRefusal | null
  onDismissRefusal: () => void
  started: Started | null
  onOpenRun: (id: string) => void
  /** Leaves the results for the desk. */
  onBack: () => void
  form: StartFormProps
}) {
  return (
    <div hidden={!visible} className={cn(visible && "flex flex-col gap-6")}>
      {visible && refusal !== null ? (
        <RefusalPanels refusal={refusal} onDismiss={onDismissRefusal} />
      ) : null}
      {visible && started !== null ? (
        <StartingResults
          result={started}
          onOpen={onOpenRun}
          onBack={onBack}
        />
      ) : null}
      <StartForm {...form} />
    </div>
  )
}
