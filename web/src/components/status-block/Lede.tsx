/** What the block says in two lines: the heading that names the situation, then why. */
export function Lede({ title, body }: { title: string; body: string }) {
  return (
    <div className="flex flex-col gap-2">
      <h2 className="text-lg leading-[26px] font-semibold text-pretty">
        {title}
      </h2>
      <p className="text-sm leading-[22px] text-pretty text-muted-foreground">
        {body}
      </p>
    </div>
  )
}
