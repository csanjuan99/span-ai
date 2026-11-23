import { CircleCheckIcon, CircleXIcon, CirclePauseIcon, GitCommitHorizontalIcon, GitPullRequestIcon } from "lucide-react"

import { CommandBlock } from "@/components/ledger/CommandBlock"
import { CopyChip } from "@/components/ledger/CopyChip"
import { changeLabel, rounds, shortHash, toneOf, type Tone } from "@/lib/ledger"
import { clockOf } from "@/lib/timeline"
import type { Statement } from "@/lib/types"

const toneText: Record<Tone, string> = {
  pass: "text-pass",
  hold: "text-hold",
  stop: "text-destructive",
}

function VerdictIcon({ tone }: { tone: Tone }) {
  const className = `size-4 shrink-0 ${toneText[tone]}`
  if (tone === "stop") {
    return <CircleXIcon aria-hidden="true" className={className} />
  }
  return tone === "hold" ? (
    <CirclePauseIcon aria-hidden="true" className={className} />
  ) : (
    <CircleCheckIcon aria-hidden="true" className={className} />
  )
}

/** One sealed statement: the stage and its verdict, why, and the evidence it carries. */
function Entry({ item }: { item: Statement }) {
  const tone = toneOf(item.verdict)
  const clock = clockOf(item.time ?? "")
  return (
    <li className="flex flex-col gap-1.5 py-3">
      <div className="flex flex-wrap items-center gap-x-2.5 gap-y-1">
        <VerdictIcon tone={tone} />
        <span className="font-medium">{item.stage}</span>
        <span className={`text-sm ${toneText[tone]}`}>{item.verdict}</span>
        {item.weight ? <span className="text-xs text-muted-foreground">{item.weight}</span> : null}
        {clock !== "" ? <span className="ml-auto text-xs text-muted-foreground tabular-nums">{clock}</span> : null}
      </div>
      <p className="pl-6.5 text-pretty">{item.reason}</p>
      {item.inconclusive ? (
        <p className="pl-6.5 text-sm text-hold">The same command passed and failed on this commit.</p>
      ) : null}
      <div className="flex flex-wrap items-center gap-1.5 pl-6.5">
        {item.gitSHA ? (
          <CopyChip label="commit" value={item.gitSHA}>
            <GitCommitHorizontalIcon aria-hidden="true" className="size-3" />
            {shortHash(item.gitSHA)}
          </CopyChip>
        ) : null}
        {item.changeURL ? (
          <a
            href={item.changeURL}
            className="inline-flex items-center gap-1.5 rounded-md border border-border px-1.5 py-0.5 font-mono text-xs text-primary transition-colors hover:border-primary/50 focus-visible:border-ring focus-visible:ring-3 focus-visible:ring-ring/50 focus-visible:outline-none"
          >
            <GitPullRequestIcon aria-hidden="true" className="size-3" />
            {changeLabel(item.changeURL)}
          </a>
        ) : null}
        {item.parentDigest ? (
          <CopyChip label="parent digest" value={item.parentDigest} className="opacity-70">
            parent {shortHash(item.parentDigest)}
          </CopyChip>
        ) : null}
      </div>
      <div className="pl-6.5">
        <CommandBlock commands={item.commands} />
      </div>
    </li>
  )
}

/** The sealed statements of a run, one round per iteration, each stage with its evidence. */
export function Ledger({ statements }: { statements: Statement[] }) {
  if (statements.length === 0) {
    return <p className="text-pretty text-muted-foreground">No record yet. It appears when this stage finishes.</p>
  }
  const groups = rounds(statements)
  return (
    <div className="flex flex-col gap-6">
      {groups.map((group) => (
        <section key={group.number} aria-label={`Round ${group.number}`}>
          <h3 className="flex items-center gap-3 border-b border-border pb-1.5 text-sm text-muted-foreground">
            <span className="text-primary" aria-hidden="true">
              $
            </span>
            <span className="tabular-nums">round {group.number}</span>
            <span className="tabular-nums">
              {group.items.length} {group.items.length === 1 ? "record" : "records"}
            </span>
          </h3>
          <ol className="flex flex-col divide-y divide-border">
            {group.items.map((item) => (
              <Entry key={`${item.stage}-${item.iteration}-${item.digest ?? item.reason}`} item={item} />
            ))}
          </ol>
        </section>
      ))}
    </div>
  )
}
