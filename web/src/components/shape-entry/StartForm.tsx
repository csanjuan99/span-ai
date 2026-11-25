"use client"

import { PlayIcon, ShieldCheckIcon } from "lucide-react"
import { useState, type FormEvent } from "react"

import {
  AgentSelect,
  ModelSelect,
} from "@/components/shape-entry/AgentModelSelects"
import { IdeaField } from "@/components/shape-entry/IdeaField"
import { useInterviewAgents } from "@/components/shape-entry/useInterviewAgents"
import { SelectField } from "@/components/settings-edit/SelectField"
import { Consequences } from "@/components/status-block/Consequences"
import { Button } from "@/components/ui/button"
import { captionText, errorText, labelText } from "@/components/shaping/styles"

/** What the form hands to the page when the person starts the interview. */
export interface StartInput {
  idea: string
  repo: string
  agent: string
  model: string
}

const issueKey = /^[A-Z][A-Z0-9]+-\d+$/

/** The Jira key an idea is, when the whole box is one; the server then starts from that issue. */
export function issueOf(idea: string): string {
  const key = idea.trim().toUpperCase()
  return issueKey.test(key) ? key : ""
}

/** The start panel: the idea, where it works, which agent interviews, and what happens next. */
export function StartForm({
  workspace,
  repos,
  busy,
  note,
  onStart,
}: {
  workspace: string
  repos: string[]
  busy: boolean
  note: string
  onStart: (input: StartInput) => void
}) {
  const [idea, setIdea] = useState("")
  const [repo, setRepo] = useState("")
  const [agent, setAgent] = useState("")
  const [model, setModel] = useState("")
  const agents = useInterviewAgents(workspace, agent)
  const issue = issueOf(idea)

  function submit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault()
    if (!busy && idea.trim() !== "") {
      onStart({ idea: idea.trim(), repo, agent, model })
    }
  }

  return (
    <form
      onSubmit={submit}
      className="flex flex-col gap-4 rounded-[14px] border border-border bg-card p-4 shadow-sm md:px-5"
    >
      <IdeaField
        id="shape-idea"
        rows={5}
        value={idea}
        hint={
          issue === ""
            ? "Rough is fine. The agent asks only what it needs."
            : `Starts from the Jira story ${issue}.`
        }
        onChange={setIdea}
      />
      <div className="grid gap-4 md:grid-cols-3">
        <div className="flex flex-col gap-1.5">
          <span className={labelText}>Repository</span>
          <SelectField
            mono
            label="Repository"
            value={repo}
            className="h-10 w-full"
            wrapperClassName="w-full"
            options={[
              { value: "", label: "Whole folder" },
              ...repos.map((name) => ({ value: name, label: name })),
            ]}
            onChange={setRepo}
          />
        </div>
        <div className="flex flex-col gap-1.5">
          <span className={labelText}>Interview agent</span>
          <AgentSelect
            agents={agents}
            value={agent}
            className="h-10 w-full"
            wrapperClassName="w-full"
            onChange={(next) => {
              setAgent(next)
              setModel("")
            }}
          />
          <span className={captionText}>
            Project default:{" "}
            {agents.projectAgent === "" ? "none ready" : agents.projectAgent}
          </span>
        </div>
        <div className="flex flex-col gap-1.5">
          <span className={labelText}>Model</span>
          <ModelSelect
            agents={agents}
            value={model}
            className="h-10 w-full"
            wrapperClassName="w-full"
            onChange={setModel}
          />
          <span className={captionText}>
            Project default:{" "}
            {agents.projectModel === "" ? "the CLI's own" : agents.projectModel}
          </span>
        </div>
      </div>
      <Consequences
        title="When you start"
        icon={ShieldCheckIcon}
        rows={[
          {
            label: "Does",
            body: "The agent reads your idea, the code and the product context, then asks only what blocks.",
          },
          { label: "Changes", body: "Nothing in your repo or Jira." },
          { label: "Kept", body: "Everything it finds is saved as it goes." },
          {
            label: "Next",
            body: "You answer questions, then check an example map.",
          },
        ]}
      />
      {note !== "" ? (
        <p role="alert" className={errorText}>
          {note}
        </p>
      ) : null}
      <Button
        type="submit"
        className="h-11 rounded-[10px] px-4 md:h-10"
        disabled={busy || idea.trim() === ""}
      >
        <PlayIcon aria-hidden="true" />
        {busy ? "Thinking…" : "Start interview"}
      </Button>
      {busy ? (
        <p className="text-sm text-pretty text-muted-foreground">
          The agent is reading your idea and the repository. This can take a
          minute.
        </p>
      ) : null}
    </form>
  )
}
