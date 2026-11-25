"use client"

import { useState, type FormEvent } from "react"

import { ArrowRight } from "lucide-react"

import { RadioCard } from "@/components/list-controls/RadioCard"
import { Consequences } from "@/components/status-block/Consequences"
import { whereKeptRows } from "@/components/status-block/whereKept"
import { Button } from "@/components/ui/button"
import { wideButton } from "@/lib/button-class"
import { Label } from "@/components/ui/label"
import { Textarea } from "@/components/ui/textarea"
import { optionLetter } from "@/lib/option-letter"
import type { DecisionOption, DecisionRequest } from "@/lib/types"

/** What choosing an option does next, in words, from the action the option carries. */
function nextLine(choice: DecisionOption, stage: string): string {
  if (choice.then === "stop") {
    return "The run ends. The branch and the changes stay as they are."
  }
  if (choice.then === "retry") {
    return `Span runs ${stage} again without the agent.`
  }
  return `The ${stage} stage goes on with your answer as guidance.`
}

function Tradeoff({ choice }: { choice: DecisionOption }) {
  if (!choice.gain && !choice.cost) {
    return null
  }
  return (
    <span className="grid grid-cols-[44px_1fr] gap-x-2 gap-y-0.5 text-xs">
      {choice.gain ? (
        <>
          <span className="text-muted-foreground">Gain</span>
          <span>{choice.gain}</span>
        </>
      ) : null}
      {choice.cost ? (
        <>
          <span className="text-muted-foreground">Cost</span>
          <span>{choice.cost}</span>
        </>
      ) : null}
    </span>
  )
}

/**
 * A question the run cannot go on without. The options sit on the left; on the right the box
 * says what the chosen one does, then the action. `where` and `kept` fill the box's rows when
 * the page knows them.
 */
export function Decision(props: {
  request: DecisionRequest
  note: string
  where?: string
  kept?: string
  onDecide: (option: string, text: string) => Promise<void>
}) {
  const { request } = props
  const [option, setOption] = useState(request.recommendation)
  const [text, setText] = useState("")
  const [busy, setBusy] = useState(false)
  const picked = request.options.find((choice) => choice.id === option)

  async function decide(event: FormEvent<HTMLFormElement>) {
    event.preventDefault()
    setBusy(true)
    try {
      await props.onDecide(option, text)
    } finally {
      setBusy(false)
    }
  }

  return (
    <form
      className="flex flex-col gap-6 lg:grid lg:grid-cols-[minmax(0,1fr)_300px]"
      onSubmit={(event) => void decide(event)}
    >
      <div className="flex min-w-0 flex-col gap-4">
        <div className="flex flex-col gap-2">
          <h3 className="text-lg font-semibold text-balance">
            {request.title}
          </h3>
          <p className="text-sm text-pretty text-muted-foreground">
            {request.happened}
          </p>
        </div>
        {request.evidence.map((item, index) => (
          <figure className="flex flex-col gap-1" key={`${index}-${item.ref}`}>
            <figcaption className="font-mono text-xs text-muted-foreground">
              {item.ref}
            </figcaption>
            <pre className="max-h-60 overflow-auto rounded-[10px] border border-border bg-muted p-3 font-mono text-xs whitespace-pre-wrap">
              {item.excerpt}
            </pre>
          </figure>
        ))}
        <fieldset className="flex flex-col gap-3">
          <legend className="sr-only">Choose how to go on</legend>
          {request.options.map((choice, at) => (
            <RadioCard
              key={choice.id}
              name="decision-option"
              value={choice.id}
              checked={option === choice.id}
              onSelect={() => setOption(choice.id)}
              roomy
              title={
                <>
                  {optionLetter(at)} · {choice.label}
                  {choice.id === request.recommendation ? (
                    <span className="rounded-full bg-accent px-2 py-0.5 text-xs font-medium text-primary">
                      recommended
                    </span>
                  ) : null}
                </>
              }
            >
              <span className="text-sm text-pretty text-muted-foreground">
                {choice.consequence}
              </span>
              <Tradeoff choice={choice} />
              {choice.reversible ? null : (
                <span className="text-xs text-muted-foreground">
                  This cannot be undone from here.
                </span>
              )}
            </RadioCard>
          ))}
        </fieldset>
      </div>
      <div className="flex min-w-0 flex-col gap-4">
        {picked === undefined ? null : (
          <Consequences
            title={`If you choose ${optionLetter(request.options.indexOf(picked))}`}
            icon={ArrowRight}
            rows={[
              { label: "Changes", body: picked.consequence },
              ...whereKeptRows(props.where, props.kept),
              { label: "Next", body: nextLine(picked, request.stage) },
            ]}
          />
        )}
        <Button
          type="submit"
          className={wideButton}
          disabled={busy || option === ""}
        >
          Choose and continue
        </Button>
        {props.note !== "" ? (
          <p className="text-sm text-pretty">{props.note}</p>
        ) : null}
        <div className="flex flex-col gap-2">
          <Label
            htmlFor="decision-text"
            className="text-xs font-medium text-muted-foreground"
          >
            Your words for the agent (optional)
          </Label>
          <Textarea
            id="decision-text"
            value={text}
            onChange={(event) => setText(event.target.value)}
            rows={3}
          />
        </div>
      </div>
    </form>
  )
}
