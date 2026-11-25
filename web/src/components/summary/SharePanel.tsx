"use client"

import { CheckIcon, CopyIcon, UploadIcon } from "lucide-react"
import { useState } from "react"

import { Button } from "@/components/ui/button"
import { sections, toMarkdown, type Section, type Summary } from "@/lib/summary"

const names: Record<Section, string> = {
  files: "Files changed",
  tests: "Tests",
  risks: "Risks",
  links: "Links",
}

function size(text: string): string {
  const bytes = new TextEncoder().encode(text).length
  return bytes < 1024 ? `${bytes} B` : `${(bytes / 1024).toFixed(1)} KB`
}

/** Pick what goes in, see the markdown it makes, and copy it. */
export function SharePanel({ summary }: { summary: Summary }) {
  const [include, setInclude] = useState<ReadonlySet<Section>>(
    new Set(sections)
  )
  const [note, setNote] = useState<"" | "copied" | "refused">("")
  const text = toMarkdown(summary, include)

  function toggle(section: Section) {
    const next = new Set(include)
    if (!next.delete(section)) {
      next.add(section)
    }
    setInclude(next)
    setNote("")
  }

  async function copy() {
    try {
      await navigator.clipboard.writeText(text)
      setNote("copied")
    } catch {
      setNote("refused")
    }
  }

  return (
    <aside
      aria-label="Share"
      className="flex flex-col gap-3 rounded-[14px] border border-border bg-card p-4 shadow-sm sm:p-5 lg:sticky lg:top-4"
    >
      <div className="flex items-center gap-2">
        <UploadIcon aria-hidden="true" className="size-4" />
        <h2 className="text-lg font-semibold">Share this summary</h2>
      </div>
      <p className="text-[13px] text-muted-foreground">
        Paste it into a pull request, a ticket or a chat. It has no transcripts,
        logs or file contents.
      </p>
      <div className="flex flex-col gap-1">
        {sections.map((section) => (
          <label
            key={section}
            className="flex min-h-8 items-center gap-2 text-[13px]"
          >
            <input
              type="checkbox"
              className="size-[18px] accent-primary"
              checked={include.has(section)}
              onChange={() => toggle(section)}
            />
            {names[section]}
          </label>
        ))}
      </div>
      <pre
        aria-label="Markdown preview"
        tabIndex={0}
        className="max-h-80 overflow-auto rounded-[10px] border border-border bg-surface-code p-3 font-mono text-xs leading-5 whitespace-pre"
      >
        {text}
      </pre>
      <Button type="button" className="h-10 w-full" onClick={() => void copy()}>
        {note === "copied" ? (
          <CheckIcon aria-hidden="true" />
        ) : (
          <CopyIcon aria-hidden="true" />
        )}
        Copy as markdown
      </Button>
      {note === "copied" ? (
        <p
          role="status"
          className="rounded-[10px] bg-pass-soft px-3 py-2 text-[13px] text-pass"
        >
          <b>Copied.</b> {size(text)} of markdown is on your clipboard.
        </p>
      ) : null}
      {note === "refused" ? (
        <p
          role="status"
          className="rounded-[10px] bg-hold-soft px-3 py-2 text-[13px] text-hold"
        >
          The clipboard refused. Select the preview and copy it by hand.
        </p>
      ) : null}
    </aside>
  )
}
