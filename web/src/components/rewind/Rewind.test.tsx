import { cleanup, render, screen } from "@testing-library/react"
import { afterEach, describe, expect, it } from "vitest"

import { Rewind } from "@/components/rewind/Rewind"

afterEach(cleanup)

describe("Rewind", () => {
  it("restarts from a stage when nothing blocks it", () => {
    render(<Rewind id="span-1" current="prove" />)
    expect((screen.getByRole("button", { name: /Start again at prove/ }) as HTMLButtonElement).disabled).toBe(false)
  })

  it("says why it is unavailable and cannot be pressed while a merge conflict is open", () => {
    render(<Rewind id="span-1" current="merge" blocked="Unavailable while a merge conflict is open. Abort the merge first." />)
    expect((screen.getByRole("button", { name: /Start again at merge/ }) as HTMLButtonElement).disabled).toBe(true)
    expect(screen.getByText(/Abort the merge first/)).toBeTruthy()
  })
})
