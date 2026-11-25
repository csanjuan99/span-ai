import { Card } from "@/components/onboarding/Frame"
import { dryLook } from "@/lib/onboarding"

/** What each stage would do with the current settings. It reads nothing and writes nothing. */
export function DryLook() {
  return (
    <Card title="Dry look" note="Nothing has run">
      <div className="overflow-x-auto">
        <table className="w-full min-w-[34rem] text-left text-sm">
          <thead className="text-xs text-muted-foreground">
            <tr>
              <th className="py-1.5 pr-3 font-normal">Stage</th>
              <th className="py-1.5 pr-3 font-normal">What the agent does</th>
              <th className="py-1.5 pr-3 font-normal">Touches</th>
              <th className="py-1.5 font-normal">Asks you</th>
            </tr>
          </thead>
          <tbody>
            {dryLook.map((row) => (
              <tr key={row.stage} className="border-t border-border align-top">
                <td className="py-2 pr-3 font-mono text-xs">{row.stage}</td>
                <td className="py-2 pr-3">{row.does}</td>
                <td className="py-2 pr-3 text-muted-foreground">
                  {row.touches}
                </td>
                <td className="py-2 text-muted-foreground">{row.asks}</td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
      <p className="text-xs text-muted-foreground">
        Dry look only shows what each stage is set to do. It reads nothing and
        writes nothing.
      </p>
    </Card>
  )
}
