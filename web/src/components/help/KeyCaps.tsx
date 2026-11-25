/** Keys as caps, one per key. */
export function KeyCaps({ keys }: { keys: string[] }) {
  return (
    <span className="flex shrink-0 items-center gap-1">
      {keys.map((key) => (
        <kbd
          key={key}
          className="inline-flex h-[22px] min-w-[22px] items-center justify-center rounded-[6px] border border-b-2 border-line-strong bg-card px-1.5 font-mono text-xs leading-[18px] text-foreground"
        >
          {key}
        </kbd>
      ))}
    </span>
  )
}
