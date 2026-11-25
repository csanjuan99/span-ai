"use client"

import { useEffect, useState } from "react"
import { FileDownIcon } from "lucide-react"

import {
  ActionPanel,
  CodeBlock,
  Panel,
  wideColumns,
} from "@/components/shaping/session-ui"
import { CheckField } from "@/components/list-controls/CheckField"
import { RadioCards } from "@/components/shaping/RadioCard"
import { Button } from "@/components/ui/button"
import { reasonText } from "@/lib/guards"
import { exportText, type ExportChoice } from "@/lib/shape-flows"
import { emptyPanelText, mutedLabelText } from "@/components/shaping/styles"

const extras = [
  ["sources", "Sources", "Where each rule and example comes from."],
  ["assumptions", "Assumptions", "What the story was sealed with."],
  ["digest", "Digest", "The sha256 of the story, to prove it did not change."],
] as const

const formats = [
  ["md", "Markdown", "The story as written. One file, readable anywhere."],
  ["json", "JSON", "Rules, examples and story as data."],
] as const

/** Export of a sealed story: format and extras, a preview of the file, and the save. */
export function ExportPanel({
  id,
  workspace,
  sealed,
}: {
  id: string
  workspace: string
  sealed: boolean
}) {
  const [choice, setChoice] = useState<ExportChoice>({
    format: "md",
    sources: false,
    assumptions: false,
    digest: true,
  })
  const [preview, setPreview] = useState("")
  const [note, setNote] = useState("")

  useEffect(() => {
    if (!sealed) {
      return
    }
    let live = true
    void exportText(id, workspace, choice)
      .then((body) => live && (setPreview(body), setNote("")))
      .catch(
        (err: unknown) =>
          live && setNote(reasonText(err, "the export could not be read"))
      )
    return () => {
      live = false
    }
  }, [id, workspace, choice, sealed])

  function save() {
    const type = choice.format === "md" ? "text/markdown" : "application/json"
    const url = URL.createObjectURL(new Blob([preview], { type }))
    const link = document.createElement("a")
    link.href = url
    link.download = choice.format === "md" ? "story.md" : "story.json"
    link.click()
    URL.revokeObjectURL(url)
  }

  if (!sealed) {
    return (
      <Panel className={emptyPanelText}>
        Only a sealed story can be exported. Seal it first.
      </Panel>
    )
  }
  return (
    <div className={wideColumns}>
      <div className="flex min-w-0 flex-col gap-4">
        <Panel className="flex flex-col gap-4 p-4">
          <RadioCards
            legend="Format"
            name="format"
            value={choice.format}
            options={formats}
            onChange={(format) => setChoice((now) => ({ ...now, format }))}
          />
          <fieldset className="flex flex-col gap-2">
            <legend className={mutedLabelText}>Include</legend>
            {extras.map(([key, name, hint]) => (
              <CheckField
                key={key}
                className="flex items-start gap-2 text-[13px]"
                inputClassName="mt-1"
                checked={choice[key]}
                onChange={(next) =>
                  setChoice((now) => ({ ...now, [key]: next }))
                }
                label={
                  <span>
                    <b>{name}</b>{" "}
                    <span className="text-muted-foreground">{hint}</span>
                  </span>
                }
              />
            ))}
          </fieldset>
          {note === "" ? null : (
            <p role="alert" className="text-sm text-destructive">
              {note}
            </p>
          )}
        </Panel>
        <Panel aria-label="Preview" className="flex flex-col gap-2 p-4">
          <div className="text-[13px] font-semibold">Preview</div>
          <CodeBlock label="Export preview">
            {preview.split("\n").map((line, index) => (
              <div key={index} className="px-3.5 whitespace-pre-wrap">
                {line === "" ? " " : line}
              </div>
            ))}
          </CodeBlock>
        </Panel>
      </div>
      <aside className="flex flex-col gap-4 lg:sticky lg:top-4 lg:self-start">
        <ActionPanel
          title="If you export"
          icon={FileDownIcon}
          rows={[
            {
              label: "Changes",
              body: "Nothing in Span. The file is written on your machine.",
            },
            {
              label: "Kept",
              body: "The sealed story and its digest stay as they are.",
            },
            {
              label: "Next",
              body: "Share the file, or attach it to the Jira story.",
            },
          ]}
        >
          <Button type="button" disabled={preview === ""} onClick={save}>
            Save {choice.format === "md" ? "story.md" : "story.json"}
          </Button>
        </ActionPanel>
      </aside>
    </div>
  )
}
