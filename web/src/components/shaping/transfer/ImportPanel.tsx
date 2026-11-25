"use client"

import { useState } from "react"
import { useNavigate } from "react-router"
import { DownloadIcon } from "lucide-react"

import {
  ActionPanel,
  CodeBlock,
  Panel,
  Pill,
  wideColumns,
} from "@/components/shaping/session-ui"
import { RadioCards } from "@/components/list-controls/RadioCard"
import { SelectField } from "@/components/settings-edit/SelectField"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { useBusyAction } from "@/hooks/use-busy-action"
import { reasonText } from "@/lib/guards"
import {
  importStory,
  readJira,
  readMarkdown,
  type ImportRead,
  type JiraRead,
} from "@/lib/shape-flows"
import { errorText, mutedLabelText } from "@/components/shaping/styles"

type Source = "jira" | "file"

const sources = [
  ["jira", "A Jira story", "Read a story and start an interview from it."],
  [
    "file",
    "A markdown file",
    "Read headings and bullets as rules and examples.",
  ],
] as const

/** The two ways in: a Jira story read by key, or a markdown file. Both show what was understood before they start. */
export function ImportPanel({
  workspace,
  repos,
}: {
  workspace: string
  repos: string[]
}) {
  const navigate = useNavigate()
  const [source, setSource] = useState<Source>("jira")
  const [key, setKey] = useState("")
  const [repo, setRepo] = useState("")
  const [file, setFile] = useState<{ name: string; text: string } | null>(null)
  const [jira, setJira] = useState<JiraRead | null>(null)
  const [read, setRead] = useState<ImportRead | null>(null)
  const [note, setNote] = useState("")
  const reading = useBusyAction({
    onStart: () => setNote(""),
    onError: (err, fallback) => setNote(reasonText(err, fallback)),
  })
  // A started session leaves this page, so busy stays on once it works.
  const starting = useBusyAction({
    keepBusyOnSuccess: true,
    onStart: () => setNote(""),
    onError: (err, fallback) => setNote(reasonText(err, fallback)),
  })
  const busy = reading.busy || starting.busy

  function look() {
    void reading.run(async () => {
      if (source === "jira") {
        setJira(await readJira(workspace, key.trim()))
      } else if (file !== null) {
        setRead(await readMarkdown(workspace, file.name, file.text))
      }
    }, "the story could not be read")
  }

  function start() {
    void starting.run(async () => {
      const id = await importStory({
        workspace,
        repo,
        ...(source === "jira"
          ? { jira: key.trim() }
          : { markdown: file?.text ?? "", name: file?.name ?? "" }),
      })
      void navigate(`/shaping/${id}?workspace=${encodeURIComponent(workspace)}`)
    }, "the session did not start")
  }

  function choose(list: FileList | null) {
    const chosen = list?.item(0)
    if (chosen == null) {
      return
    }
    void chosen.text().then((text) => {
      setFile({ name: chosen.name, text })
      setRead(null)
    })
  }

  const ready = source === "jira" ? key.trim() !== "" : file !== null
  const looked = source === "jira" ? jira !== null : read !== null
  return (
    <div className={wideColumns}>
      <div className="flex min-w-0 flex-col gap-4">
        <Panel className="flex flex-col gap-4 p-4">
          <RadioCards
            legend="Start from"
            name="source"
            value={source}
            options={sources}
            onChange={setSource}
          />
          <div className="grid gap-4 sm:grid-cols-2">
            {source === "jira" ? (
              <div className="flex flex-col gap-1.5">
                <label htmlFor="jira-key" className={mutedLabelText}>
                  Jira key
                </label>
                <Input
                  id="jira-key"
                  value={key}
                  placeholder="BILL-2031"
                  onChange={(event) => {
                    setKey(event.target.value)
                    setJira(null)
                  }}
                />
              </div>
            ) : (
              <div className="flex flex-col gap-1.5">
                <label htmlFor="story-file" className={mutedLabelText}>
                  Markdown file
                </label>
                <input
                  id="story-file"
                  type="file"
                  accept=".md,text/markdown,text/plain"
                  onChange={(event) => choose(event.target.files)}
                  className="text-[13px]"
                />
              </div>
            )}
            <div className="flex flex-col gap-1.5">
              <span className={mutedLabelText}>Repository</span>
              <SelectField
                mono
                label="Repository"
                value={repo}
                onChange={setRepo}
                options={[
                  { value: "", label: "The folder itself" },
                  ...repos.map((name) => ({ value: name, label: name })),
                ]}
              />
            </div>
          </div>
          <div>
            <Button
              type="button"
              variant="outline"
              disabled={!ready || busy}
              onClick={look}
            >
              See what Span reads
            </Button>
          </div>
          {note === "" ? null : (
            <p role="alert" className={errorText}>
              {note}
            </p>
          )}
        </Panel>
        {source === "jira" && jira !== null ? (
          <Panel
            aria-label="What Span read"
            className="flex flex-col gap-2 p-4"
          >
            <div className="flex items-center gap-2">
              <span className="text-[13px] font-semibold">
                What Span read from {key.trim()}
              </span>
              <Pill tone="pass" small>
                Read only
              </Pill>
            </div>
            <CodeBlock label="Code">
              {[
                ["summary:", jira.summary],
                ["status:", jira.status],
                ["description:", `${jira.paragraphs} paragraphs`],
                ["criteria:", `${jira.criteria} bullets`],
                ["comments:", jira.comments],
              ].map(([name, value]) => (
                <div key={name} className="px-3.5">
                  <span className="text-muted-foreground">{name}</span> {value}
                </div>
              ))}
            </CodeBlock>
            <p className="text-xs text-muted-foreground">
              Span does not write to Jira from here. The key stays linked to the
              new session.
            </p>
          </Panel>
        ) : null}
        {source === "file" && read !== null ? (
          <Panel
            aria-label="How Span read it"
            className="flex flex-col gap-2 p-4"
          >
            <div className="text-[13px] font-semibold">
              {file?.name} · How Span read it
            </div>
            {read.lines.map((line) => (
              <p key={line} className="text-[13px] text-pretty">
                <span aria-hidden="true" className="mr-2 font-mono text-pass">
                  +
                </span>
                {line}
              </p>
            ))}
          </Panel>
        ) : null}
      </div>
      <aside className="flex min-w-0 flex-col gap-4 lg:sticky lg:top-4 lg:self-start">
        <ActionPanel
          title="If you start it"
          icon={DownloadIcon}
          rows={[
            {
              label: "Changes",
              body: "Creates a new session. Its idea is the story text.",
            },
            {
              label: "Kept",
              body:
                source === "jira"
                  ? `${key.trim() || "The story"} is not changed.`
                  : "Your file is not changed.",
            },
            {
              label: "Sources",
              body: "Imported items show source “import” with the line they came from.",
            },
            {
              label: "Next",
              body: "The agent interviews you about what the story leaves open.",
            },
          ]}
        >
          <Button
            type="button"
            disabled={!ready || !looked || busy}
            onClick={start}
          >
            Start session
          </Button>
        </ActionPanel>
      </aside>
    </div>
  )
}
