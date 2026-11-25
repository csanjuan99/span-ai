"use client"

import { InfoIcon } from "lucide-react"

import { StartCard } from "@/components/start-run/StartCard"
import { Consequences } from "@/components/status-block/Consequences"
import { FilterChips } from "@/components/list-controls/FilterChips"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import { noteText } from "@/components/shaping/styles"
import { basePlaceholder } from "@/lib/api"
import type { BranchChoice } from "@/lib/types"

const kinds = ["infer", "feat", "fix", "hotfix"] as const

/** What the How step shows and reports back. */
export interface HowProps {
  kind: string
  onKind: (value: string) => void
  base: string
  onBase: (value: string) => void
  branches: BranchChoice
  /** The base the saved settings give a run, so the kinds can say where they branch from. */
  savedBase: string
}

/** N1 step 3: the kind of change and the base branch, both optional. */
export function HowSection({
  kind,
  onKind,
  base,
  onBase,
  branches,
  savedBase,
}: HowProps) {
  const main = savedBase === "" ? branches.defaultBase : savedBase
  return (
    <StartCard
      step={3}
      title="How"
      hint="Optional. The defaults are usually right."
    >
      <div className="flex flex-col gap-1.5">
        <span className="text-xs font-medium">Kind</span>
        <FilterChips
          label="Kind"
          value={kinds.find((item) => item === kind) ?? "infer"}
          onChange={onKind}
          chips={kinds.map((item) => ({ key: item, label: item }))}
        />
        {kind !== "infer" ? (
          <input type="hidden" name="kind" value={kind} />
        ) : null}
      </div>
      <Consequences
        title="What each default does"
        icon={InfoIcon}
        rows={[
          {
            label: "infer",
            body: "Span reads the story and picks feat or fix. It never picks hotfix.",
          },
          {
            label: "feat, fix",
            body: (
              <>
                Branch from <code className="font-mono">{main}</code>. Merge
                asks you.
              </>
            ),
          },
          {
            label: "hotfix",
            body: "Branch from the release branch and merge back to both. Asks you twice.",
          },
        ]}
      />
      <div className="flex flex-col gap-1.5">
        <Label htmlFor="base">Base branch</Label>
        <Input
          id="base"
          name="base"
          list="base-branches"
          className="h-11 md:h-10"
          placeholder={basePlaceholder(kind, branches.defaultBase)}
          value={base}
          onChange={(event) => onBase(event.target.value)}
        />
        <datalist id="base-branches">
          {branches.branches.map((name) => (
            <option key={name} value={name} />
          ))}
        </datalist>
        {branches.branches.length > 0 ? (
          <p className={noteText}>
            Choose from the {branches.branches.length} branches:{" "}
            <span className="font-mono text-xs">
              {branches.branches.slice(0, 4).join(" ")}
            </span>
            .
          </p>
        ) : null}
        <p className="text-[13px]">
          Empty means: first <code className="font-mono">span.yaml</code> base ({" "}
          <code className="font-mono">{main}</code>), then the kind&apos;s
          default above.
        </p>
      </div>
    </StartCard>
  )
}
