/** The Span AI mark and name the top bar and the setup bar share. */
export function Brand() {
  return (
    <>
      <span
        aria-hidden="true"
        className="grid size-6 place-items-center rounded-md bg-primary font-mono text-[13px] font-semibold text-primary-foreground"
      >
        S
      </span>
      <span className="text-sm font-semibold text-foreground">Span AI</span>
    </>
  )
}
