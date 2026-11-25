"use client"

import { useEffect, useState } from "react"
import { useParams, useSearchParams } from "react-router"

import { ApiCli, type ApiAction } from "@/components/implementation/ApiCli"
import { IdChip } from "@/components/list-controls/IdChip"
import { Shell } from "@/components/shell/Shell"
import { Pill } from "@/components/shaping/session-ui"
import { ExportPanel } from "@/components/shaping/transfer/ExportPanel"
import { ImportPanel } from "@/components/shaping/transfer/ImportPanel"
import { SharePanel } from "@/components/shaping/transfer/SharePanel"
import { getJSON, getShape } from "@/lib/api"
import { reasonText } from "@/lib/guards"
import { repoNames } from "@/lib/repos"
import type { ShapeSession } from "@/lib/types"
import { errorText, wrapRow } from "@/components/shaping/styles"

const exits =
  "0 ok · 1 failed · 2 usage · 3 not found · 4 refused, such as a story that is not sealed · 5 agent failed"

const actions: ApiAction[] = [
  {
    name: "Read a story before starting",
    method: "POST",
    route: "/shapes/import  {jira, dryRun: true}",
    command: "span shape import --jira BILL-2031 --dry-run",
  },
  {
    name: "Start from Jira or a file",
    method: "POST",
    route: "/shapes/import  {jira | markdown, name}",
    command: "span shape import --file story.md",
  },
  {
    name: "Export a sealed story",
    method: "GET",
    route: "/shapes/{id}/export?format=md&digest=1",
    command: "span shape export SES-0137 --format md --digest --out story.md",
  },
  {
    name: "Make a read-only link",
    method: "POST",
    route: "/shapes/{id}/shares  {days}",
    command: "span shape share SES-0137 --days 7",
  },
  {
    name: "Revoke a link",
    method: "GET",
    route: "/shapes/{id}/shares/{token}  (DELETE)",
    command: "span shape share SES-0137 --revoke TOKEN",
  },
]

/** Import a Jira story or a markdown file, and export or share a sealed story. */
export function ShapeTransfer() {
  const { id = "" } = useParams()
  const [params] = useSearchParams()
  const workspace = params.get("workspace") ?? "."
  const [session, setSession] = useState<ShapeSession | null>(null)
  const [repos, setRepos] = useState<string[]>([])
  const [note, setNote] = useState("")

  useEffect(() => {
    let live = true
    void getJSON(`/repos?path=${encodeURIComponent(workspace)}`)
      .then((body) => live && setRepos(repoNames(body)))
      .catch(() => live && setRepos([]))
    if (id !== "") {
      void getShape(id, workspace)
        .then((loaded) => live && setSession(loaded))
        .catch(
          (err: unknown) =>
            live && setNote(reasonText(err, "the session could not be read"))
        )
    }
    return () => {
      live = false
    }
  }, [id, workspace])

  const sealed = session?.status === "sealed"
  return (
    <Shell
      path={`~/span/shaping/${id === "" ? "import" : `${id}/transfer`}`}
      status="transfer"
    >
      <header className="flex flex-col gap-2">
        {session === null ? null : (
          <div className={wrapRow}>
            <IdChip>{session.ref ?? session.id}</IdChip>
            <Pill tone={sealed ? "pass" : "hold"}>
              {sealed ? "Sealed" : session.status}
            </Pill>
            {sealed ? <Pill tone="mut">Read-only</Pill> : null}
          </div>
        )}
        <h1 className="text-2xl font-semibold tracking-tight">
          {id === "" ? "Import a story" : "Import, export and share"}
        </h1>
      </header>
      {note === "" ? null : (
        <p role="alert" className={errorText}>
          {note}
        </p>
      )}
      <section aria-label="Import" className="flex flex-col gap-3">
        <h2 className="text-lg font-semibold">Import</h2>
        <ImportPanel workspace={workspace} repos={repos} />
      </section>
      {id === "" ? null : (
        <>
          <section aria-label="Export" className="flex flex-col gap-3">
            <h2 className="text-lg font-semibold">Export</h2>
            <ExportPanel id={id} workspace={workspace} sealed={sealed} />
          </section>
          <section aria-label="Share" className="flex flex-col gap-3">
            <h2 className="text-lg font-semibold">Share</h2>
            <SharePanel id={id} workspace={workspace} sealed={sealed} />
          </section>
        </>
      )}
      <ApiCli actions={actions} exits={exits} />
    </Shell>
  )
}
