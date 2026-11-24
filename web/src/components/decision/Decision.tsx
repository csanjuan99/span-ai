"use client"

import { useState, type FormEvent } from "react"

import { Button } from "@/components/ui/button"
import { Label } from "@/components/ui/label"
import { Textarea } from "@/components/ui/textarea"
import type { DecisionRequest } from "@/lib/types"

/** A question the run cannot go on without. The answer says where the run resumes. */
export function Decision(props: {
  request: DecisionRequest
  note: string
  onDecide: (option: string, text: string) => Promise<void>
}) {
  const { request } = props
  const [option, setOption] = useState(request.recommendation)
  const [text, setText] = useState("")
  const [busy, setBusy] = useState(false)

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
    <form className="flex flex-col gap-4" onSubmit={(event) => void decide(event)}>
      <div className="flex flex-col gap-1">
        <h3 className="text-balance text-base font-medium">{request.title}</h3>
        <p className="text-pretty text-sm text-muted-foreground">{request.happened}</p>
      </div>
      {request.evidence.map((item, index) => (
        <figure className="flex flex-col gap-1" key={`${index}-${item.ref}`}>
          <figcaption className="font-mono text-xs text-muted-foreground">{item.ref}</figcaption>
          <pre className="max-h-60 overflow-auto whitespace-pre-wrap rounded-md border border-border p-3 font-mono text-xs">
            {item.excerpt}
          </pre>
        </figure>
      ))}
      <fieldset className="flex flex-col gap-2">
        <legend className="sr-only">Choose how to go on</legend>
        {request.options.map((choice) => (
          <label
            className="flex cursor-pointer items-start gap-3 rounded-md border border-border p-3 has-[:checked]:border-foreground"
            key={choice.id}
          >
            <input
              type="radio"
              name="decision-option"
              className="mt-1"
              value={choice.id}
              checked={option === choice.id}
              onChange={() => setOption(choice.id)}
            />
            <span className="flex flex-col gap-1">
              <span className="text-sm font-medium">
                {choice.label}
                {choice.id === request.recommendation ? <span className="ml-2 text-xs text-muted-foreground">recommended</span> : null}
              </span>
              <span className="text-pretty text-sm text-muted-foreground">{choice.consequence}</span>
              {choice.reversible ? null : <span className="text-xs text-muted-foreground">This cannot be undone from here.</span>}
            </span>
          </label>
        ))}
      </fieldset>
      <div className="flex flex-col gap-2">
        <Label htmlFor="decision-text">Your words for the agent (optional)</Label>
        <Textarea id="decision-text" value={text} onChange={(event) => setText(event.target.value)} rows={3} />
      </div>
      <div className="flex flex-wrap items-center gap-3">
        <Button type="submit" className="self-start" disabled={busy || option === ""}>
          Choose and continue
        </Button>
        {props.note !== "" ? <p className="text-pretty text-sm">{props.note}</p> : null}
      </div>
    </form>
  )
}
