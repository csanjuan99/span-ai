"use client"

import { LockIcon } from "lucide-react"
import { Link } from "react-router"

import { FilterChips } from "@/components/list-controls/FilterChips"
import { IdChip } from "@/components/list-controls/IdChip"
import { RadioCard } from "@/components/list-controls/RadioCard"
import { JiraLookup, type JiraProps } from "@/components/start-run/JiraLookup"
import { StartCard } from "@/components/start-run/StartCard"
import { Pill } from "@/components/shaping/session-ui"
import { noteText } from "@/components/shaping/styles"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import { Textarea } from "@/components/ui/textarea"
import { wordCount, type SealedStory } from "@/lib/start-info"

/** Where the story of a run comes from. */
export type StartSource = "jira" | "text" | "files" | "shape" | "sample"

/** A sealed story that arrived from Shape a story, so the source cannot change. */
export interface LockedStory {
  id: string
  title: string
  workspace: string
}

/** What the What step shows and reports back. */
export interface WhatProps {
  source: StartSource
  onSource: (value: StartSource) => void
  jira: JiraProps
  story: string
  onStory: (value: string) => void
  sealed: SealedStory[]
  shapeId: string
  onShape: (id: string) => void
  locked: LockedStory | null
  /** The sample brief is offered only from the empty desk. */
  sample: { title: string; detail: string } | null
}

function SourceBlock({ what }: { what: WhatProps }) {
  const { source, story } = what
  if (source === "jira") {
    return <JiraLookup {...what.jira} />
  }
  if (source === "text") {
    return (
      <div className="flex flex-col gap-1.5">
        <Label htmlFor="story">Story text</Label>
        <Textarea
          id="story"
          name="story"
          rows={6}
          value={story}
          onChange={(event) => what.onStory(event.target.value)}
        />
        <p className={noteText}>{wordCount(story)} words.</p>
      </div>
    )
  }
  if (source === "files") {
    return (
      <div className="flex flex-col gap-1.5">
        <Label htmlFor="file">Handoff files</Label>
        <Input
          id="file"
          name="file"
          type="file"
          multiple
          className="h-auto max-w-full py-2"
        />
        <p className={noteText}>Up to 5 files, 200 KB each, read as the story.</p>
      </div>
    )
  }
  if (source === "sample" && what.sample !== null) {
    return (
      <div className="flex flex-col gap-1 rounded-[10px] border border-border bg-surface-code p-4">
        <span className="text-xs text-muted-foreground">Sample brief</span>
        <h3 className="text-sm font-semibold">{what.sample.title}</h3>
        <p className="text-[13px] text-muted-foreground">{what.sample.detail}</p>
      </div>
    )
  }
  return <ShapedStories what={what} />
}

function ShapedStories({ what }: { what: WhatProps }) {
  if (what.sealed.length === 0) {
    return (
      <p className={noteText}>
        No sealed story in this folder yet. Shape one first, then start a run
        from it.
      </p>
    )
  }
  return (
    <fieldset className="flex flex-col gap-2">
      <legend className="sr-only">Sealed stories</legend>
      {what.sealed.map((item) => (
        <RadioCard
          key={item.id}
          name="shape"
          value={item.id}
          checked={what.shapeId === item.id}
          onSelect={() => what.onShape(item.id)}
          title={item.title}
        >
          <IdChip>{item.id.slice(-6)}</IdChip>
        </RadioCard>
      ))}
    </fieldset>
  )
}

/** N1 step 2: which story the run works on, and where it comes from. */
export function WhatSection({ what }: { what: WhatProps }) {
  const { locked, source } = what
  if (locked !== null) {
    return (
      <StartCard
        step={2}
        title="What"
        hint="Which story the run works on."
        badge={
          <Pill tone="mut" small>
            <LockIcon aria-hidden="true" className="size-3" />
            Came from Shape a story
          </Pill>
        }
      >
        <div className="flex flex-col gap-2 rounded-[10px] border border-border bg-surface-code p-4">
          <div className="flex flex-wrap items-center gap-2">
            <IdChip>{locked.id.slice(-6)}</IdChip>
            <Pill tone="pass" small>
              Sealed
            </Pill>
          </div>
          <h3 className="text-sm font-semibold">{locked.title}</h3>
          <Link
            className="text-[13px] text-primary underline-offset-4 hover:underline"
            to={`/shaping/${encodeURIComponent(locked.id)}?workspace=${encodeURIComponent(locked.workspace)}`}
          >
            Read the story in Shape a story
          </Link>
        </div>
        <p className={noteText}>
          Locked because you came from Shape a story. To use another source, go
          back to the session.
        </p>
        <input type="hidden" name="story" value={what.story} />
      </StartCard>
    )
  }
  const chips: { key: StartSource; label: string }[] = [
    { key: "jira", label: "Jira story key" },
    { key: "text", label: "Pasted text" },
    { key: "files", label: "Handoff files" },
    { key: "shape", label: "A shaped story" },
  ]
  if (what.sample !== null) {
    chips.push({ key: "sample", label: "Sample brief" })
  }
  return (
    <StartCard step={2} title="What" hint="Which story the run works on.">
      <FilterChips
        label="Story source"
        chips={chips}
        value={source}
        onChange={what.onSource}
      />
      <SourceBlock what={what} />
      {source === "text" ? null : (
        <input type="hidden" name="story" value={what.story} />
      )}
    </StartCard>
  )
}
