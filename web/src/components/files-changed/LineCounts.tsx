/** The lines a file or change added and removed, in the colours the diff uses. */
export function LineCounts({
  added,
  removed,
}: {
  added: number
  removed: number
}) {
  return (
    <>
      <span className="text-pass">+{added}</span>{" "}
      <span className="text-fail">−{removed}</span>
    </>
  )
}
