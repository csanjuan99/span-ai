"use client"

import { useState } from "react"
import { Link } from "react-router"
import { ArrowRightIcon } from "lucide-react"

import { RadioCard } from "@/components/list-controls/RadioCard"
import { Button } from "@/components/ui/button"
import {
  ActionPanel,
  Panel,
  twoColumns,
  Eyebrow,
} from "@/components/shaping/session-ui"
import { SessionSummary } from "@/components/shaping/SessionSummary"
import type { ShapeSession } from "@/lib/types"
import { headingText } from "@/components/shaping/styles"

type Choice = "split" | "keep"

function Option({
  choice,
  value,
  title,
  body,
  onPick,
}: {
  choice: Choice
  value: Choice
  title: string
  body: string
  onPick: (value: Choice) => void
}) {
  return (
    <RadioCard
      roomy
      name="split-choice"
      checked={choice === value}
      onSelect={() => onPick(value)}
      title={title}
    >
      {body}
    </RadioCard>
  )
}

/**
 * The agent found more than one outcome. The person splits the story, which makes one linked
 * session per story, or keeps it whole. Once split, the children are listed and open.
 */
export function SplitBoard({
  session,
  busy,
  onDecide,
}: {
  session: ShapeSession
  busy: boolean
  onDecide: (decision: Choice) => void
}) {
  const [choice, setChoice] = useState<Choice>("split")
  const children = session.children ?? []
  const count = session.split.length
  const done = children.length > 0
  return (
    <div className={twoColumns}>
      <div className="flex min-w-0 flex-col gap-4">
        <div>
          <h2 className={headingText}>
            {done
              ? `Split into ${children.length} stories`
              : "The agent suggests splitting this story"}
          </h2>
          <p className="text-sm text-muted-foreground">
            {done
              ? "Each one is its own session. They start from this example map, so nothing you answered is asked again."
              : "Smaller stories finish faster and fail in smaller ways. Each one shows what it leaves out."}
          </p>
        </div>
        {done ? null : (
          <fieldset className="flex flex-col gap-3">
            <legend className="sr-only">Split or keep whole</legend>
            <Option
              choice={choice}
              value="split"
              title={`Split into ${count} smaller stories`}
              body="Each can be sealed and run on its own."
              onPick={setChoice}
            />
            <Option
              choice={choice}
              value="keep"
              title="Keep as one story"
              body="Seal it as it is. It is bigger, so a run may hold more often."
              onPick={setChoice}
            />
          </fieldset>
        )}
        <ol className="grid gap-4 md:grid-cols-3">
          {session.split.map((item, index) => (
            <li key={item.story}>
              <Panel className="flex h-full flex-col gap-2 p-4">
                <div className="flex items-center gap-2">
                  <span className="font-mono text-[13px] font-semibold text-primary">
                    {index + 1}
                  </span>
                  <span className="text-sm font-semibold text-pretty">
                    {item.story}
                  </span>
                </div>
                <Eyebrow>Leaves out</Eyebrow>
                <p className="grid grid-cols-[16px_minmax(0,1fr)] gap-2 text-[13px] leading-5">
                  <span
                    aria-hidden="true"
                    className="font-mono font-semibold text-muted-foreground"
                  >
                    −
                  </span>
                  {item.leftOut}
                </p>
                {children[index] !== undefined ? (
                  <Link
                    className="mt-auto text-sm text-primary underline-offset-4 hover:underline"
                    to={`/shaping/${encodeURIComponent(children[index])}?workspace=${encodeURIComponent(session.workspace)}`}
                  >
                    Open this story
                  </Link>
                ) : null}
              </Panel>
            </li>
          ))}
        </ol>
      </div>
      <aside className="flex min-w-0 flex-col gap-4">
        {done ? null : (
          <ActionPanel
            title="If you split"
            icon={ArrowRightIcon}
            rows={[
              {
                label: "Changes",
                body: `Replaces this story with ${count}. The example map stays and is shared.`,
              },
              { label: "Kept", body: "All answers, rules and examples." },
              {
                label: "Next",
                body: "You seal each story in turn. Then run or send to Jira, one by one.",
              },
            ]}
          >
            <Button
              type="button"
              className="h-10 w-full"
              variant={choice === "split" ? "default" : "outline"}
              disabled={busy}
              onClick={() => onDecide("split")}
            >
              {`Split into ${count} stories`}
            </Button>
            <Button
              type="button"
              className="h-10 w-full"
              variant={choice === "keep" ? "default" : "outline"}
              disabled={busy}
              onClick={() => onDecide("keep")}
            >
              {busy ? "Thinking…" : "Keep as one story"}
            </Button>
            <p className="text-xs text-muted-foreground">
              Keeping one story runs the agent once more so it writes the story.
            </p>
          </ActionPanel>
        )}
        <SessionSummary session={session} />
      </aside>
    </div>
  )
}
