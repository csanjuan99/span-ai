"use client"

import { useState } from "react"
import { cn } from "cn"

import { CodeBlock } from "@/components/shaping/session-ui"
import { TabStrip } from "@/components/shaping/TabStrip"

type View = "markdown" | "preview"

function Heading({ line }: { line: string }) {
  const level = line.length - line.replace(/^#+/, "").length
  return (
    <span
      className={cn(level === 1 ? "text-xl" : "text-base", "font-semibold")}
    >
      {line.slice(level).trim()}
    </span>
  )
}

/** The story as it reads once rendered: headings, bullets and paragraphs. */
function Rendered({ story }: { story: string }) {
  return (
    <div className="flex flex-col gap-1.5 rounded-[10px] border border-border p-4 text-sm leading-6">
      {story.split("\n").map((line, index) => {
        if (line.trim() === "") {
          return <div key={index} className="h-1" />
        }
        if (line.startsWith("#")) {
          return <Heading key={index} line={line} />
        }
        if (/^\s*[-*] /.test(line)) {
          return (
            <p key={index} className="pl-4 -indent-3">
              <span aria-hidden="true">· </span>
              {line.replace(/^\s*[-*] /, "")}
            </p>
          )
        }
        return <p key={index}>{line}</p>
      })}
    </div>
  )
}

/** The markdown of the story, one line each; the heading marks stand out. */
function Source({ story }: { story: string }) {
  return (
    <CodeBlock label="Story markdown">
      {story.split("\n").map((line, index) => {
        const mark = /^#+/.exec(line)?.[0] ?? ""
        return (
          <div key={index} className="min-h-5 px-3.5 whitespace-pre-wrap">
            <span className="text-primary">{mark}</span>
            {line.slice(mark.length)}
          </div>
        )
      })}
    </CodeBlock>
  )
}

const views: { value: View; label: string }[] = [
  { value: "markdown", label: "Markdown" },
  { value: "preview", label: "Preview" },
]

/** The story, as markdown or rendered. It never edits: a story changes only through the interview. */
export function StoryView({ story }: { story: string }) {
  const [view, setView] = useState<View>("markdown")
  return (
    <div className="flex flex-col gap-3">
      <TabStrip
        label="Story view"
        value={view}
        options={views}
        onChange={setView}
        className="max-w-[520px]"
      />
      {view === "markdown" ? (
        <Source story={story} />
      ) : (
        <Rendered story={story} />
      )}
    </div>
  )
}
