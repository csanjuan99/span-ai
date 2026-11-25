"use client"

import { Undo2Icon } from "lucide-react"
import { useState } from "react"

import { ControlBlock, ControlRow } from "@/components/run-controls/ControlRow"
import { rewindRun } from "@/lib/api"
import { stages } from "@/lib/types"
import { Button } from "@/components/ui/button"
import { Label } from "@/components/ui/label"
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select"
import { Textarea } from "@/components/ui/textarea"

/** intake only writes the brief. A restart starts at a stage that does work. */
const restartable = stages.filter((name) => name !== "intake")

/**
 * Restart the run at a stage, with the instruction that stage reads when it starts.
 * A new iteration keeps the earlier attempt on disk, and the repair budget starts fresh.
 */
export function Rewind({
  id,
  current,
  blocked = "",
  startOpen = true,
  requested = "",
}: {
  id: string
  current: string
  blocked?: string
  /** False shows one row that opens the form; the run page uses it. */
  startOpen?: boolean
  /** A stage the page asks to restart at, for example from the failure's "Restart from build…". */
  requested?: string
}) {
  const [open, setOpen] = useState(startOpen || requested !== "")
  const [stage, setStage] = useState(() =>
    requested !== ""
      ? requested
      : current === "" || current === "intake"
        ? "build"
        : current
  )
  // A request that arrives later opens the form on that stage; adjusted while rendering, not in an effect.
  const [seen, setSeen] = useState(requested)
  if (seen !== requested) {
    setSeen(requested)
    if (requested !== "") {
      setOpen(true)
      setStage(requested)
    }
  }
  const [note, setNote] = useState("")
  const [message, setMessage] = useState("")
  const [busy, setBusy] = useState(false)

  async function restart() {
    setBusy(true)
    setMessage("")
    try {
      await rewindRun(id, stage, note)
      setMessage(`The run will start again at ${stage}. Resume it to continue.`)
    } catch (err: unknown) {
      setMessage(
        err instanceof Error ? err.message : "the run was not restarted"
      )
    } finally {
      setBusy(false)
    }
  }

  if (!open) {
    return (
      <ControlRow
        icon={Undo2Icon}
        title="Restart from a stage…"
        caption="Start again at the stage that has to change. The attempt so far stays on disk."
        onClick={() => setOpen(true)}
      />
    )
  }

  return (
    <ControlBlock>
      <Label htmlFor="rewind-stage">Restart from a stage</Label>
      <p className="text-sm text-pretty text-muted-foreground">
        When what the branch produced is not what the handoff asked for, start
        again at the stage that has to change. The attempt so far stays on disk
        as its own iteration, and the instruction below is what that stage
        reads.
      </p>
      <Select value={stage} onValueChange={setStage} disabled={blocked !== ""}>
        <SelectTrigger id="rewind-stage" className="w-40">
          <SelectValue />
        </SelectTrigger>
        <SelectContent>
          {restartable.map((name) => (
            <SelectItem key={name} value={name}>
              {name}
            </SelectItem>
          ))}
        </SelectContent>
      </Select>
      <Textarea
        aria-label="Instruction for the restarted stage"
        placeholder="For example: use the code from the handoff, do not rewrite the module."
        value={note}
        onChange={(event) => setNote(event.target.value)}
        rows={4}
        disabled={blocked !== ""}
      />
      <div className="flex flex-wrap items-center gap-3">
        <Button
          type="button"
          variant="secondary"
          disabled={busy || blocked !== ""}
          onClick={() => void restart()}
        >
          Start again at {stage}
        </Button>
        {message !== "" ? (
          <p className="text-sm text-pretty">{message}</p>
        ) : null}
        {blocked !== "" ? (
          <p className="text-sm text-pretty text-muted-foreground">{blocked}</p>
        ) : null}
      </div>
    </ControlBlock>
  )
}
