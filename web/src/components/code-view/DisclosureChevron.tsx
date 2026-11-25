import { ChevronRightIcon } from "lucide-react"

/** The arrow of a `<summary>` inside a `group` that turns down when the `<details>` opens. */
export function DisclosureChevron() {
  return (
    <ChevronRightIcon
      aria-hidden="true"
      className="size-3.5 text-muted-foreground transition-transform group-open:rotate-90"
    />
  )
}
