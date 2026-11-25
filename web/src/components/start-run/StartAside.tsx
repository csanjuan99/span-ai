"use client"

import { PlayIcon, SparklesIcon, WandSparklesIcon } from "lucide-react"
import { Link } from "react-router"

import { ActionPanel, Panel } from "@/components/shaping/session-ui"
import { noteText } from "@/components/shaping/styles"
import { Consequences } from "@/components/status-block/Consequences"
import { Button } from "@/components/ui/button"
import { wideButton } from "@/lib/button-class"
import type { StartSummary } from "@/lib/start-info"
import { shortStoryWords, wordCount } from "@/lib/start-info"

/** What the side column shows and reports back. */
export interface AsideProps {
  summary: StartSummary
  /** How many runs Start will create: one per selected repository. */
  runs: number
  story: string
  /** Whether the story is typed in the form, the only source that can be shaped first. */
  shapeable: boolean
  onShapeFirst: () => void
}

function SettingsLink({ section }: { section: string }) {
  return (
    <Link
      className="block text-primary underline-offset-4 hover:underline"
      to="/settings"
    >
      Settings · {section}
    </Link>
  )
}

function approvals(stages: Record<string, string>): string {
  const waits = Object.entries(stages)
    .filter(([, approval]) => approval === "hold")
    .map(([name]) => name)
  const others = Object.keys(stages).length - waits.length
  if (waits.length === 0) {
    return "No stage waits for you. All of them run on their own."
  }
  return `${waits.join(" and ")} wait for your yes. The other ${others} stages run on their own.`
}

/** N1 side column: what the saved settings will do, what Start does, and the short-story nudge. */
export function StartAside({
  summary,
  runs,
  story,
  shapeable,
  onShapeFirst,
}: AsideProps) {
  const base = summary.base === "" ? "the base branch" : summary.base
  const words = wordCount(story)
  return (
    <aside className="flex flex-col gap-4" aria-label="Before you start">
      <Panel className="flex flex-col gap-3 p-4">
        <Consequences
          title="What will happen"
          icon={SparklesIcon}
          rows={[
            {
              label: "Agents",
              body: (
                <>
                  {summary.chosenAgent === ""
                    ? summary.agentReason
                    : `${summary.chosenAgent} writes the stages.`}
                  <SettingsLink section="Agents" />
                </>
              ),
            },
            {
              label: "Approve",
              body: (
                <>
                  {approvals(summary.stages)}
                  <SettingsLink section="Approvals" />
                </>
              ),
            },
            {
              label: "Proof",
              body: (
                <>
                  {summary.checks.length === 0 ? (
                    "No fast check is set, so nothing runs before review."
                  ) : (
                    <>
                      <code className="font-mono text-xs">
                        {summary.checks.join(", ")}
                      </code>{" "}
                      must pass before review.
                    </>
                  )}
                  <SettingsLink section="Checks" />
                </>
              ),
            },
            {
              label: "Branch",
              body: (
                <>
                  Named from the story, from{" "}
                  <code className="font-mono text-xs">{base}</code>.
                  <SettingsLink section="Project" />
                </>
              ),
            },
          ]}
        />
        <p className={noteText}>
          Read from your saved settings. Unsaved changes in Settings are not
          used until you save them.
        </p>
      </Panel>
      {shapeable && words > 0 && words < shortStoryWords ? (
        <Panel
          aria-label="This story is short"
          className="flex flex-col gap-2 border-primary bg-accent p-4"
        >
          <h3 className="text-sm font-semibold">This story is short</h3>
          <p className="text-[13px] text-pretty">
            {words} words, no acceptance criteria. Span can ask a few questions
            first and write a story you both agree on.
          </p>
          <Button
            type="button"
            variant="outline"
            className={wideButton}
            onClick={onShapeFirst}
          >
            <WandSparklesIcon aria-hidden="true" />
            Shape it first
          </Button>
          <p className={noteText}>Opens Shape a story with this text as the idea.</p>
        </Panel>
      ) : null}
      <ActionPanel
        title="When you press Start"
        icon={PlayIcon}
        rows={[
          {
            label: "Creates",
            body:
              runs > 1
                ? `${runs} branches, then runs intake.`
                : "One branch, then runs intake.",
          },
          {
            label: "Changes",
            body: `Nothing on ${base} and none of your files.`,
          },
          {
            label: "Next",
            body: "The run page opens. Intake asks you only if the story is unclear.",
          },
        ]}
      >
        <Button type="submit" className={wideButton}>
          <PlayIcon aria-hidden="true" />
          {runs > 1 ? `Start ${runs} runs` : "Start run"}
        </Button>
      </ActionPanel>
    </aside>
  )
}
