import type { ReactNode } from "react"

/** The optional "Where" and "Kept" rows of the box that says what a choice does. */
export function whereKeptRows(
  where?: string,
  kept?: ReactNode
): { label: string; body: ReactNode }[] {
  return [
    ...(where
      ? [
          {
            label: "Where",
            body: <span className="font-mono text-xs break-all">{where}</span>,
          },
        ]
      : []),
    ...(kept ? [{ label: "Kept", body: kept }] : []),
  ]
}
