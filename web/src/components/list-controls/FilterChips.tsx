"use client"

/** One chip of {@link FilterChips}. */
export interface FilterChip<Key extends string> {
  key: Key
  label: string
  count?: number
}

/** The filter chips of a list: one is pressed, each shows how many rows it would keep. */
export function FilterChips<Key extends string>({
  label,
  chips,
  value,
  onChange,
}: {
  label: string
  chips: FilterChip<Key>[]
  value: Key
  onChange: (key: Key) => void
}) {
  return (
    <div role="group" aria-label={label} className="flex flex-wrap gap-2">
      {chips.map((chip) => (
        <button
          key={chip.key}
          type="button"
          aria-pressed={value === chip.key}
          onClick={() => onChange(chip.key)}
          className="inline-flex h-11 items-center gap-1.5 rounded-full border border-input bg-card px-3 text-sm aria-pressed:border-primary aria-pressed:bg-accent aria-pressed:font-semibold aria-pressed:text-primary md:h-8"
        >
          {chip.label}
          {chip.count === undefined ? null : (
            <span className="font-mono text-xs">{chip.count}</span>
          )}
        </button>
      ))}
    </div>
  )
}
