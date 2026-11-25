import { Panel, toneText, type Tone } from "@/components/shaping/session-ui"

const marks: { mark: string; tone: Tone; text: string }[] = [
  { mark: "+", tone: "pass", text: "Added in After" },
  { mark: "~", tone: "hold", text: "Changed between the two" },
  { mark: "−", tone: "fail", text: "Removed from Before" },
]

/** What the marks of a comparison mean. Color is never the only sign: each has its mark. */
export function ReadGuide() {
  return (
    <Panel className="flex flex-col gap-2 p-4">
      <div className="text-sm font-semibold">How to read it</div>
      {marks.map((row) => (
        <div
          key={row.mark}
          className="grid grid-cols-[20px_minmax(0,1fr)] items-center gap-2 py-1"
        >
          <span
            aria-hidden="true"
            className={`font-mono font-semibold ${toneText[row.tone]}`}
          >
            {row.mark}
          </span>
          <span className="text-[13px] leading-5">{row.text}</span>
        </div>
      ))}
    </Panel>
  )
}
