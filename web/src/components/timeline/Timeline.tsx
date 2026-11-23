import { useEffect, useRef } from "react"

import type { TimelineLine } from "@/lib/timeline"

const toneClass = {
  primary: "text-primary",
  pass: "text-pass",
  hold: "text-hold",
  stop: "text-destructive",
  muted: "text-muted-foreground",
} as const

/** Fixed terminal band. It follows the end until the reader scrolls up. */
export function Timeline({ lines }: { lines: TimelineLine[] }) {
  const scroller = useRef<HTMLDivElement>(null)
  const stick = useRef(true)
  const current = [...lines].reverse().find((line) => line.stage !== ">")?.stage ?? ""

  useEffect(() => {
    const node = scroller.current
    if (node && stick.current) {
      node.scrollTop = node.scrollHeight
    }
  }, [lines])

  function onScroll() {
    const node = scroller.current
    if (!node) {
      return
    }
    stick.current = node.scrollHeight - node.scrollTop - node.clientHeight < 8
  }

  return (
    <div ref={scroller} onScroll={onScroll} className="h-32 overflow-y-auto border-t border-border px-4 py-2 font-mono text-xs leading-5">
      {lines.length === 0 ? (
        <p className="text-muted-foreground">
          <span className="text-primary">$</span> waiting
        </p>
      ) : (
        lines.map((line) => (
          <p key={line.key} className="whitespace-pre-wrap">
            <span className="text-muted-foreground tabular-nums">{line.clock}</span>
            {line.stage === ">" ? (
              <>
                {"  "}
                <span className="text-primary">&gt;</span>
                {"  "}
                <span>{line.mark}</span>
              </>
            ) : (
              <>
                {"  "}
                <span className={line.stage === current ? "text-primary" : undefined}>{line.stage}</span>
                {"  "}
                <span className={toneClass[line.tone]}>{line.mark}</span>
                {line.detail !== "" ? <span className="text-muted-foreground">{"  "}{line.detail}</span> : null}
              </>
            )}
          </p>
        ))
      )}
    </div>
  )
}
