"use client"

import { ArrowLeftIcon, PlayIcon } from "lucide-react"
import { useState } from "react"

import { sampleBrief } from "@/components/desk/EmptyDesk"
import { DryLook } from "@/components/onboarding/DryLook"
import { Consequences } from "@/components/status-block/Consequences"
import { Panel } from "@/components/onboarding/Frame"
import { Pill } from "@/components/onboarding/Pill"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Textarea } from "@/components/ui/textarea"
import { saveAgent, startRun } from "@/lib/api"
import { reasonText } from "@/lib/guards"
import { designField, designInput } from "@/lib/field-class"

/** Step 3: a sample brief, a look at what each stage does, and the Start button that starts it. */
export function FirstRunStep({
  folder,
  agent,
  onStarted,
  onBack,
  onSkip,
}: {
  folder: string
  agent: string
  onStarted: (id: string) => void
  onBack: () => void
  onSkip: () => void
}) {
  const [title, setTitle] = useState(sampleBrief.title)
  const [brief, setBrief] = useState(sampleBrief.detail)
  const [busy, setBusy] = useState(false)
  const [note, setNote] = useState("")

  async function start() {
    setBusy(true)
    setNote("")
    try {
      if (agent !== "") {
        await saveAgent(folder, agent)
      }
      const form = new FormData()
      form.set("workspace", folder)
      form.set("story", `${title.trim()}\n\n${brief.trim()}`)
      form.set("kind", "infer")
      const started = await startRun(form)
      if (typeof started === "string") {
        setNote(started)
        return
      }
      onStarted(started.id)
    } catch (reason: unknown) {
      setNote(reasonText(reason, "the run could not be started"))
    } finally {
      setBusy(false)
    }
  }

  return (
    <div className="mx-auto flex w-full max-w-[1200px] flex-col gap-6 sm:gap-8">
      <div className="flex flex-col gap-2">
        <Pill tone="acc">Step 3 of 3</Pill>
        <h1 className="text-2xl font-semibold tracking-tight">
          Your first run
        </h1>
        <p className="text-[13px] text-muted-foreground">
          Start with a small brief we filled in. Look at what each stage will do
          first, then press Start.
        </p>
      </div>
      <div className="grid items-start gap-4 lg:grid-cols-[400px_minmax(0,1fr)] lg:gap-6">
        <div className="contents lg:flex lg:min-w-0 lg:flex-col lg:gap-6">
          <div className="order-1 lg:order-none">
            <Panel
              title="Sample brief"
              badge={<Pill tone="mut">Prefilled</Pill>}
            >
              <label className="flex flex-col gap-1.5">
                <span className="text-xs font-medium">Title</span>
                <Input
                  className={designInput}
                  value={title}
                  onChange={(event) => setTitle(event.target.value)}
                />
              </label>
              <label className="flex flex-col gap-1.5">
                <span className="text-xs font-medium">What should change</span>
                <Textarea
                  rows={4}
                  aria-label="Brief"
                  className={designField}
                  value={brief}
                  onChange={(event) => setBrief(event.target.value)}
                />
              </label>
              <p className="text-[13px] text-muted-foreground">
                Runs in <span className="font-mono text-xs">{folder}</span> on a
                new branch named{" "}
                <span className="font-mono text-xs">span/…</span>.
              </p>
            </Panel>
          </div>
          <section className="order-3 flex flex-col gap-4 rounded-[14px] border border-border bg-card p-4 shadow-sm sm:px-5">
            <Consequences
              title="When you press Start"
              icon={PlayIcon}
              iconClassName="text-primary"
              rows={[
                {
                  label: "Changes",
                  body: `Creates a branch for the run. No file changes until build.${
                    agent !== ""
                      ? ` Saves ${agent} as the project's agent in span.yaml.`
                      : ""
                  }`,
                },
                {
                  label: "Where",
                  body: (
                    <span className="font-mono text-xs break-all">
                      {folder}
                    </span>
                  ),
                },
                {
                  label: "Kept",
                  body: "Everything you have now. Nothing is overwritten.",
                },
                {
                  label: "Next",
                  body: "Intake reads the brief. If something is unclear, the run holds and asks you.",
                },
              ]}
            />
            {note !== "" ? (
              <p role="alert" className="text-sm text-pretty text-destructive">
                {note}
              </p>
            ) : null}
            <Button
              className="h-10 w-full"
              disabled={busy || title.trim() === "" || folder.trim() === ""}
              onClick={() => void start()}
            >
              <PlayIcon aria-hidden="true" />
              Start the first run
            </Button>
            <div className="flex flex-wrap gap-3">
              <Button variant="ghost" onClick={onBack}>
                <ArrowLeftIcon aria-hidden="true" />
                Back
              </Button>
              <Button variant="ghost" onClick={onSkip}>
                Skip for now
              </Button>
            </div>
          </section>
        </div>
        <div className="order-2 lg:order-none">
          <DryLook />
        </div>
      </div>
    </div>
  )
}
