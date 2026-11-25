/** Keys as caps, one per key. */
export function KeyCaps({ keys }: { keys: string[] }) {
  return (
    <span className="flex shrink-0 items-center gap-1">
      {keys.map((key) => (
        <kbd
          key={key}
          className="min-w-5.5 rounded-md border border-border bg-muted px-1.5 py-0.5 text-center font-mono text-xs text-muted-foreground"
        >
          {key}
        </kbd>
      ))}
    </span>
  )
}
