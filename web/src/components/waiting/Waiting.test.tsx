import { fireEvent, render, screen } from "@testing-library/react"
import { describe, expect, it, vi } from "vitest"

import { Waiting } from "@/components/waiting/Waiting"

describe("Waiting", () => {
  it("sends the answer for the question on screen", async () => {
    const onAnswer = vi.fn(() => Promise.resolve())
    const onReload = vi.fn(() => Promise.resolve())
    render(
      <Waiting questions={["Which permission atom is Pago.Ver?"]} note="" onAnswer={onAnswer} onReload={onReload} />,
    )
    fireEvent.change(screen.getByLabelText("Which permission atom is Pago.Ver?"), { target: { value: "Pago.Ver is new." } })
    fireEvent.click(screen.getByRole("button", { name: "Answer and continue" }))
    await screen.findByRole("button", { name: "Answer and continue" })
    expect(onAnswer).toHaveBeenCalledWith([{ question: "Which permission atom is Pago.Ver?", body: "Pago.Ver is new." }])
    fireEvent.click(screen.getByRole("button", { name: "Read the story again" }))
    await screen.findByRole("button", { name: "Read the story again" })
    expect(onReload).toHaveBeenCalledOnce()
  })
})
