"use client"

import { useEffect, useState } from "react"
import { useParams, useSearchParams } from "react-router"
import { LockIcon } from "lucide-react"

import { Shell } from "@/components/shell/Shell"
import { Panel } from "@/components/shaping/session-ui"
import { reasonText } from "@/lib/guards"
import { openShared, type SharedStory } from "@/lib/shape-flows"
import { errorText, monoCaption } from "@/components/shaping/styles"

/** What a read-only link shows: the sealed story, its rules and examples and its digest. Nothing can be changed here. */
export function ShapeShared() {
  const { id = "", token = "" } = useParams()
  const [params] = useSearchParams()
  const workspace = params.get("workspace") ?? "."
  const [story, setStory] = useState<SharedStory | null>(null)
  const [note, setNote] = useState("")

  useEffect(() => {
    let live = true
    void openShared(id, workspace, token)
      .then((loaded) => live && setStory(loaded))
      .catch(
        (err: unknown) =>
          live && setNote(reasonText(err, "this link does not open"))
      )
    return () => {
      live = false
    }
  }, [id, workspace, token])

  return (
    <Shell path={`~/span/shaping/${id}/shared`} status="shared">
      <p className="flex items-center gap-2 text-[13px] text-muted-foreground">
        <LockIcon aria-hidden="true" className="size-4" />
        Read-only link. You can read this story, not change it.
      </p>
      {note === "" ? null : (
        <p role="alert" className={errorText}>
          {note}
        </p>
      )}
      {story === null ? null : (
        <>
          <header className="flex flex-col gap-1">
            <span className={monoCaption}>
              {story.ref} · {story.repo}
            </span>
            <h1 className="text-2xl font-semibold tracking-tight text-balance">
              {story.title}
            </h1>
          </header>
          <Panel className="p-4">
            <pre className="font-sans text-[13px] leading-6 whitespace-pre-wrap">
              {story.story}
            </pre>
          </Panel>
          <Panel aria-label="Rules" className="flex flex-col gap-2 p-4">
            <h2 className="text-lg font-semibold">Rules and examples</h2>
            {story.rules.map((rule) => (
              <div key={rule.id} className="flex flex-col gap-1">
                <p className="text-[13px] font-medium">
                  <span className="font-mono">{rule.id}</span> {rule.text}
                </p>
                {story.examples
                  .filter((example) => example.rule === rule.id)
                  .map((example) => (
                    <p
                      key={example.id}
                      className="pl-6 text-[13px] text-muted-foreground"
                    >
                      <span className="font-mono">{example.id}</span>{" "}
                      {example.text}
                    </p>
                  ))}
              </div>
            ))}
          </Panel>
          <p className="font-mono text-xs break-all text-muted-foreground">
            sha256 {story.digest}
          </p>
        </>
      )}
    </Shell>
  )
}
