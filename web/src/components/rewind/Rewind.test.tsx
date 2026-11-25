import { cleanup, fireEvent, render, screen } from "@testing-library/react"
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

describe("Rewind collapsed", () => {
  it("shows one row until it is opened, then the form at the stage that was asked for", () => {
    render(<Rewind id="span-1" current="prove" startOpen={false} />)
    expect(screen.queryByRole("button", { name: /^Start again at \w+$/ })).toBeNull()
    fireEvent.click(screen.getByRole("button", { name: /Restart from a stage/ }))
    expect(screen.getByRole("button", { name: /Start again at prove/ })).toBeTruthy()
  })

  it("opens by itself on the stage the page requests", () => {
    render(<Rewind id="span-1" current="prove" startOpen={false} requested="build" />)
    expect(screen.getByRole("button", { name: /Start again at build/ })).toBeTruthy()
  })
})
