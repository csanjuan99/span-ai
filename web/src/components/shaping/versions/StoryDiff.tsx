import { Panel } from "@/components/shaping/session-ui"
import { DiffRows } from "@/components/shaping/versions/ChangeCard"
import {
  sideRows,
  type RevisionDiff,
  type StoryLine,
} from "@/lib/shape-versions"
import { cardBody, countText } from "@/components/shaping/styles"

function rowOf(line: StoryLine, at: "old" | "new") {
  const sign =
    line.kind === "add"
      ? ("+" as const)
      : line.kind === "del"
        ? ("-" as const)
        : (" " as const)
  return {
    sign,
    label: String(at === "old" ? line.old : line.new),
    text: line.text,
  }
}

/** The story text between two revisions: two columns, or one list of removed and added lines. */
export function StoryDiff({
  diff,
  view,
}: {
  diff: RevisionDiff
  view: "side" | "inline"
}) {
  const rows = sideRows(diff.story)
  return (
    <Panel aria-label="Story text diff" className={cardBody}>
      <div className="flex flex-wrap items-baseline gap-x-3 gap-y-1">
        <h2 className="text-base font-semibold">Story text</h2>
        <span className={countText}>
          +{diff.storyAdded} −{diff.storyRemoved} lines · {diff.from.id} to{" "}
          {diff.to.id}
        </span>
      </div>
      {diff.story.length === 0 ? (
        <p className="text-sm text-muted-foreground">
          The story text is the same in both revisions.
        </p>
      ) : view === "side" ? (
        <div className="grid gap-3 lg:grid-cols-2">
          <div className="flex min-w-0 flex-col gap-1.5">
            <div className="text-xs font-medium text-muted-foreground">
              {diff.from.id} · before
            </div>
            <DiffRows
              rows={rows.map((row) =>
                row.left === null
                  ? { sign: " " as const, label: "", text: "" }
                  : rowOf(row.left, "old")
              )}
            />
          </div>
          <div className="flex min-w-0 flex-col gap-1.5">
            <div className="text-xs font-medium text-muted-foreground">
              {diff.to.id} · after
            </div>
            <DiffRows
              rows={rows.map((row) =>
                row.right === null
                  ? { sign: " " as const, label: "", text: "" }
                  : rowOf(row.right, "new")
              )}
            />
          </div>
        </div>
      ) : (
        <DiffRows
          rows={diff.story.map((line) =>
            rowOf(line, line.kind === "del" ? "old" : "new")
          )}
        />
      )}
    </Panel>
  )
}
