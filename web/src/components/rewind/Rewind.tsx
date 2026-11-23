import { useState } from "react"

import { rewindRun } from "@/lib/api"
import { stages } from "@/lib/types"
import { Button } from "@/components/ui/button"
import { Label } from "@/components/ui/label"
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select"
import { Textarea } from "@/components/ui/textarea"

/** intake only writes the brief. A restart starts at a stage that does work. */
const restartable = stages.filter((name) => name !== "intake")

/**
 * Restart the run at a stage, with the instruction that stage reads when it starts.
 * A new iteration keeps the earlier attempt on disk, and the repair budget starts fresh.
 */
export function Rewind({ id, current }: { id: string; current: string }) {
  const [stage, setStage] = useState(() => (current === "" || current === "intake" ? "build" : current))
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
      setMessage(err instanceof Error ? err.message : "the run was not restarted")
    } finally {
      setBusy(false)
    }
  }

  return (
    <div className="flex flex-col gap-2 border border-border p-3">
      <Label htmlFor="rewind-stage">Restart from a stage</Label>
      <p className="text-pretty text-sm text-muted-foreground">
        When what the branch produced is not what the handoff asked for, start again at the stage that has to change.
        The attempt so far stays on disk as its own iteration, and the instruction below is what that stage reads.
      </p>
      <Select value={stage} onValueChange={setStage}>
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
      />
      <div className="flex flex-wrap items-center gap-3">
        <Button type="button" variant="secondary" disabled={busy} onClick={() => void restart()}>
          Start again at {stage}
        </Button>
        {message !== "" ? <p className="text-pretty text-sm">{message}</p> : null}
      </div>
    </div>
  )
}
