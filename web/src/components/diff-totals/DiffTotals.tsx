/** "7 files +184 −32": how big a change is, with the counts colored and the signs kept. */
export function DiffTotals({
  files,
  added,
  removed,
}: {
  files: number
  added: number
  removed: number
}) {
  return (
    <>
      {files} {files === 1 ? "file" : "files"}{" "}
      <span className="text-pass">+{added}</span>{" "}
      <span className="text-fail">−{removed}</span>
    </>
  )
}
