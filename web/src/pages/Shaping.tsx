"use client"

import { useEffect, useState, type FormEvent } from "react"
import { Link, useNavigate } from "react-router"

import { createShape, getJSON, listShapes } from "@/lib/api"
import { useBusyAction } from "@/hooks/use-busy-action"
import { reasonText } from "@/lib/guards"
import { repoNames } from "@/lib/repos"
import { Shell } from "@/components/shell/Shell"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import { Textarea } from "@/components/ui/textarea"
import type { ShapeSession } from "@/lib/types"
import { PromptHeading } from "@/components/shell/PromptHeading"

/** The shaping sessions of a folder and the form that starts a new one. */
export function Shaping() {
  const navigate = useNavigate()
  const [workspace, setWorkspace] = useState(".")
  const [sessions, setSessions] = useState<ShapeSession[]>([])
  const [repos, setRepos] = useState<string[]>([])
  const [repo, setRepo] = useState("")
  const [idea, setIdea] = useState("")
  const [note, setNote] = useState("")
  // A started session leaves this page, so busy stays on once it works.
  const { busy, run: begin } = useBusyAction({
    keepBusyOnSuccess: true,
    onStart: () => setNote(""),
    onError: (err, fallback) => setNote(reasonText(err, fallback)),
  })

  useEffect(() => {
    let live = true
    void listShapes(workspace)
      .then((rows) => {
        if (live) {
          setSessions(rows)
        }
      })
      .catch((err: unknown) => {
        if (live) {
          setNote(reasonText(err, "the sessions could not be read"))
        }
      })
    return () => {
      live = false
    }
  }, [workspace])

  async function findRepos() {
    setNote("")
    try {
      setRepos(
        repoNames(await getJSON(`/repos?path=${encodeURIComponent(workspace)}`))
      )
    } catch (err: unknown) {
      setNote(reasonText(err, "the folder could not be read"))
    }
  }

  function start(event: FormEvent<HTMLFormElement>) {
    event.preventDefault()
    if (busy || idea.trim() === "") {
      return
    }
    void begin(async () => {
      const text = idea.trim()
      const created = await createShape(
        repo === ""
          ? { workspace, idea: text }
          : { workspace, repo, idea: text }
      )
      void navigate(
        `/shaping/${created.id}?workspace=${encodeURIComponent(workspace)}`
      )
    }, "the session did not start")
  }

  return (
    <Shell path="~/span/shaping" status="shaping">
      <header>
        <PromptHeading as="h1">shape a story</PromptHeading>
        <p className="text-pretty text-muted-foreground">
          Start from an idea. Span asks what it needs to know, then seals one
          story you can create in Jira and run.
        </p>
        <p className="mt-1 text-sm">
          <Link
            className="text-primary underline-offset-4 hover:underline"
            to="/"
          >
            Desk
          </Link>
        </p>
      </header>
      <form
        className="flex flex-col gap-4"
        onSubmit={(event) => void start(event)}
      >
        <div className="flex flex-col gap-2">
          <Label htmlFor="shape-workspace">Folder</Label>
          <Input
            id="shape-workspace"
            value={workspace}
            onChange={(event) => setWorkspace(event.target.value)}
          />
        </div>
        <div className="flex flex-wrap items-end gap-2">
          <Button
            type="button"
            variant="secondary"
            onClick={() => void findRepos()}
          >
            Find repos
          </Button>
          {repos.length > 0 ? (
            <div className="flex flex-col gap-2">
              <Label htmlFor="shape-repo">Repository</Label>
              <select
                id="shape-repo"
                className="h-8 border border-border bg-background px-2 text-sm"
                value={repo}
                onChange={(event) => setRepo(event.target.value)}
              >
                <option value="">Whole folder</option>
                {repos.map((name) => (
                  <option key={name} value={name}>
                    {name}
                  </option>
                ))}
              </select>
            </div>
          ) : null}
        </div>
        <div className="flex flex-col gap-2">
          <Label htmlFor="shape-idea">Idea</Label>
          <Textarea
            id="shape-idea"
            rows={5}
            value={idea}
            onChange={(event) => setIdea(event.target.value)}
          />
        </div>
        <Button
          type="submit"
          className="self-start"
          disabled={busy || idea.trim() === ""}
        >
          {busy ? "Thinking…" : "Start shaping"}
        </Button>
        {busy ? (
          <p className="text-sm text-pretty text-muted-foreground">
            The agent is reading your idea and the repository. This can take a
            minute.
          </p>
        ) : null}
        {note !== "" ? (
          <p role="alert" className="text-sm text-pretty text-destructive">
            {note}
          </p>
        ) : null}
      </form>
      <section>
        <PromptHeading>sessions</PromptHeading>
        {sessions.length === 0 ? (
          <p className="mt-2 text-pretty text-muted-foreground">
            No sessions in this folder yet.
          </p>
        ) : (
          <ul className="mt-2 flex flex-col gap-1">
            {sessions.map((item) => (
              <li key={item.id}>
                <Link
                  className="text-primary underline-offset-4 hover:underline"
                  to={`/shaping/${item.id}?workspace=${encodeURIComponent(workspace)}`}
                >
                  {item.idea}
                </Link>
                <span className="text-muted-foreground">
                  {"  "}
                  {item.status}
                </span>
              </li>
            ))}
          </ul>
        )}
      </section>
    </Shell>
  )
}
