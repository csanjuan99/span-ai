import { Panel } from "@/components/shaping/session-ui"
import type { SameGroup } from "@/lib/shape-compare"
import { cardBody, proseText } from "@/components/shaping/styles"

/** What the sessions already agree on, once, with the sessions that say it. */
export function SameList({ groups }: { groups: SameGroup[] }) {
  return (
    <Panel aria-label="In common" className={cardBody}>
      <h2 className="text-base font-semibold">In common</h2>
      {groups.length === 0 ? (
        <p className="text-sm text-muted-foreground">
          These sessions share no rule or example.
        </p>
      ) : (
        <ul className="flex flex-col gap-2">
          {groups.map((group) => (
            <li
              key={`${group.kind}${group.text}`}
              className="flex flex-col gap-1 border-t border-border pt-2 first:border-t-0 first:pt-0"
            >
              <span className={proseText}>
                <span
                  aria-hidden="true"
                  className="mr-2 font-mono font-semibold text-pass"
                >
                  +
                </span>
                {group.text}
              </span>
              <span className="text-xs text-muted-foreground">
                {group.members.map((m) => `${m.ref} · ${m.item}`).join("  ·  ")}
              </span>
            </li>
          ))}
        </ul>
      )}
    </Panel>
  )
}
