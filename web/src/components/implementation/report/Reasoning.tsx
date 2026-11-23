import type { FeatureReport } from "@/lib/featuredoc"
import type { Group, Kind } from "@/lib/report"

const kindLabel: Record<Kind, string> = {
  source: "source",
  test: "tests",
  docs: "docs",
  config: "config",
  deps: "dependencies",
  ci: "build & deploy",
  generated: "generated",
  other: "other",
}

/** What changed, by purpose, as a bar per family so a reader sees the shape of the change. */
function Shape({ groups }: { groups: Group[] }) {
  const max = Math.max(1, ...groups.map((group) => group.added + group.removed))
  return (
    <ul className="flex flex-col gap-1.5" aria-label="Changes by purpose">
      {groups.map((group) => (
        <li key={group.kind} className="grid grid-cols-[7rem_1fr_auto] items-center gap-3 text-sm">
          <span>{kindLabel[group.kind]}</span>
          <span className="h-2 rounded-[2px] bg-border" aria-hidden="true">
            <span
              className="block h-full rounded-[2px] bg-primary"
              style={{ width: `${Math.max(4, Math.round(((group.added + group.removed) / max) * 100))}%` }}
            />
          </span>
          <span className="text-xs text-muted-foreground tabular-nums">
            {group.files.length} {group.files.length === 1 ? "file" : "files"} · <span className="text-pass">+{group.added}</span>{" "}
            <span className="text-destructive">−{group.removed}</span>
          </span>
        </li>
      ))}
    </ul>
  )
}

/** The story this change answers, the shape of the change, and the choices the agent says it made. */
export function Reasoning({ feature, groups }: { feature: FeatureReport | null; groups: Group[] }) {
  const done = feature?.tasks.filter((task) => task.done).length ?? 0
  return (
    <div className="flex flex-col gap-6">
      <section aria-label="What changed and why" className="flex flex-col gap-3">
        <h3 className="text-sm">
          <span className="text-primary">$ </span>what and why
        </h3>
        {feature !== null && feature.title !== "" ? <p className="text-pretty">{feature.title}</p> : null}
        {feature !== null && feature.specs.length > 0 ? (
          <ul className="flex flex-col gap-1 text-sm">
            {feature.specs.slice(0, 6).map((spec) => (
              <li key={spec} className="text-pretty">
                <span className="text-muted-foreground" aria-hidden="true">
                  ·{" "}
                </span>
                {spec}
              </li>
            ))}
          </ul>
        ) : null}
        {feature !== null && feature.tasks.length > 0 ? (
          <p className="text-xs text-muted-foreground tabular-nums">
            {done}/{feature.tasks.length} tasks checked off in the feature document
          </p>
        ) : null}
        {groups.length > 0 ? <Shape groups={groups} /> : null}
      </section>
      <section aria-label="Decisions" className="flex flex-col gap-2">
        <h3 className="text-sm">
          <span className="text-primary">$ </span>decisions
          <span className="ml-3 text-xs text-muted-foreground tabular-nums">{feature?.decisions.length ?? 0}</span>
        </h3>
        {feature === null || feature.decisions.length === 0 ? (
          <p className="text-pretty text-sm text-muted-foreground">
            The agent recorded no decision. Choices the brief did not settle are invisible here until it writes "Decision:" lines in its log.
          </p>
        ) : (
          <ol className="flex flex-col gap-2">
            {feature.decisions.map((decision) => (
              <li key={decision.what} className="rounded-md border border-border p-3">
                <p className="text-pretty">{decision.what}</p>
                {decision.why !== "" ? (
                  <p className="mt-1 text-pretty text-sm text-muted-foreground">
                    <span className="text-primary">because </span>
                    {decision.why}
                  </p>
                ) : (
                  <p className="mt-1 text-sm text-hold">No reason given.</p>
                )}
                <p className="mt-1.5 text-xs text-muted-foreground">assumed by the agent · check it before you approve</p>
              </li>
            ))}
          </ol>
        )}
      </section>
    </div>
  )
}
