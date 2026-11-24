/** The id of a rule, example or question, muted, followed by a space. */
export function RefId({ id }: { id: string }) {
  return <span className="text-muted-foreground">{id} </span>
}
