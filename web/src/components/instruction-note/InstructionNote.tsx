"use client"

import { MessageSquareIcon } from "lucide-react"
import type { FormEvent } from "react"

import { ControlBlock, ControlRow } from "@/components/run-controls/ControlRow"
import { stages } from "@/lib/types"
import { Button } from "@/components/ui/button"
import { Label } from "@/components/ui/label"
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select"
import { Textarea } from "@/components/ui/textarea"

/** A note a stage reads when it starts. The stage already running does not see one saved now. */
export function InstructionNote(props: {
  open: boolean
  selected: string
  timing: string
  body: string
  note: string
  onSelect: (stage: string) => void
  onBody: (body: string) => void
  onSave: (event: FormEvent<HTMLFormElement>) => void
  onOpen: () => void
  /** True when the run is held, so the note is read on resume. */
  hold?: boolean
  /** Saves the note and resumes the run in one step. Only given when the run can resume. */
  onSaveAndResume?: () => void
}) {
  if (!props.open) {
    return (
      <ControlRow
        icon={MessageSquareIcon}
        title={`Leave a note for ${props.selected}`}
        caption={props.hold ? "Resume reads this note when this stage starts again." : "The agent reads it on its next run of this stage."}
        onClick={props.onOpen}
      />
    )
  }
  return (
    <ControlBlock>
    <form className="flex flex-col gap-3" onSubmit={props.onSave}>
      <Label htmlFor="instruction-stage">Instruction</Label>
      <p className="text-pretty text-sm text-muted-foreground">{props.timing}</p>
      <Select value={props.selected} onValueChange={props.onSelect}>
        <SelectTrigger id="instruction-stage" className="w-40">
          <SelectValue />
        </SelectTrigger>
        <SelectContent>
          {stages.map((name) => (
            <SelectItem key={name} value={name}>
              {name}
            </SelectItem>
          ))}
        </SelectContent>
      </Select>
      <Textarea
        aria-label="Instruction body"
        value={props.body}
        onChange={(event) => {
          props.onSelect(props.selected)
          props.onBody(event.target.value)
        }}
        rows={6}
      />
      <div className="flex items-center gap-3">
        <Button type="submit" variant="secondary" className="self-start">
          Save instruction
        </Button>
        {props.onSaveAndResume !== undefined ? (
          <Button type="button" className="self-start" onClick={props.onSaveAndResume}>
            Save and resume
          </Button>
        ) : null}
        {props.note !== "" ? <p className="text-pretty text-sm">{props.note}</p> : null}
      </div>
    </form>
    </ControlBlock>
  )
}
