"use client"

import { useEffect, useState } from "react"
import { ActivityIcon } from "lucide-react"

import { IdChip } from "@/components/list-controls/IdChip"
import { Panel, Pill } from "@/components/shaping/session-ui"
import { Button } from "@/components/ui/button"
import { clockSeconds } from "@/lib/clock"
import { getActivity, type Activity } from "@/lib/shape-flows"
import { hintText, monoCaption, wrapRow } from "@/components/shaping/styles"

const every = 5000

/**
 * What each agent did last, newest first. It asks the server every few seconds; Pause stops
 * the page, not the agents. Only steps already saved show: Span cannot see what an agent is
 * reading right now.
 */
export function ActivityFeed({ workspace }: { workspace: string }) {
  const [feed, setFeed] = useState<Activity[]>([])
  const [paused, setPaused] = useState(false)
  const [note, setNote] = useState("")

  useEffect(() => {
    if (paused) {
      return
    }
    let live = true
    function load() {
      getActivity(workspace)
        .then((steps) => {
          if (live) {
            setFeed(steps)
            setNote("")
          }
        })
        .catch(() => {
          if (live) {
            setNote("The feed could not be read. It will try again.")
          }
        })
    }
    load()
    const timer = window.setInterval(load, every)
    return () => {
      live = false
      window.clearInterval(timer)
    }
  }, [workspace, paused])

  return (
    <Panel aria-label="Live activity" className="flex flex-col gap-3 p-4">
      <div className={wrapRow}>
        <ActivityIcon aria-hidden="true" className="size-4" />
        <span className="text-[13px] font-semibold">
          What each agent is doing
        </span>
        <Pill tone={paused ? "mut" : "acc"} small>
          {paused ? "paused" : "live"}
        </Pill>
      </div>
      {feed.length === 0 ? (
        <p className={hintText}>No agent has done anything yet.</p>
      ) : (
        <ol className="flex flex-col gap-1.5">
          {feed.map((step) => (
            <li
              key={`${step.at}-${step.session}-${step.kind}`}
              className="grid grid-cols-[64px_84px_minmax(0,1fr)] items-baseline gap-2 text-[13px]"
            >
              <span className={monoCaption}>{clockSeconds(step.at)}</span>
              <IdChip>{step.ref}</IdChip>
              <span className="min-w-0 text-pretty">
                <span aria-hidden="true" className="text-primary">
                  {">"}{" "}
                </span>
                <span className="font-mono">{step.actor}</span> {step.text}
              </span>
            </li>
          ))}
        </ol>
      )}
      {note === "" ? null : (
        <p role="status" className="text-xs text-hold">
          {note}
        </p>
      )}
      <div>
        <Button
          type="button"
          variant="ghost"
          size="sm"
          onClick={() => setPaused((now) => !now)}
        >
          {paused ? "Resume feed" : "Pause feed"}
        </Button>
      </div>
      <p className="text-xs text-muted-foreground">
        Only steps the agents already saved show here. Pausing stops the page,
        not the agents.
      </p>
    </Panel>
  )
}
