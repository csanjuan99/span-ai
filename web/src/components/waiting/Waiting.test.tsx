import { cleanup, fireEvent, render, screen } from "@testing-library/react"
import { afterEach, describe, expect, it, vi } from "vitest"

import { Waiting } from "@/components/waiting/Waiting"

describe("Waiting", () => {
  afterEach(cleanup)

  it("sends the answer for the question on screen", async () => {
    const onAnswer = vi.fn(() => Promise.resolve())
    const onReload = vi.fn(() => Promise.resolve())
    render(
      <Waiting
        questions={["Which permission atom is Pago.Ver?"]}
        note=""
        onAnswer={onAnswer}
        onReload={onReload}
      />
    )
    fireEvent.change(
      screen.getByLabelText("Which permission atom is Pago.Ver?"),
      { target: { value: "Pago.Ver is new." } }
    )
    fireEvent.click(screen.getByRole("button", { name: "Answer and continue" }))
    await screen.findByRole("button", { name: "Answer and continue" })
    expect(onAnswer).toHaveBeenCalledWith([
      {
        question: "Which permission atom is Pago.Ver?",
        body: "Pago.Ver is new.",
      },
    ])
    fireEvent.click(
      screen.getByRole("button", { name: "Read the story again" })
    )
    await screen.findByRole("button", { name: "Read the story again" })
    expect(onReload).toHaveBeenCalledOnce()
  })

  it("asks for every answer: nothing is sent until each question has one", () => {
    const onAnswer = vi.fn(() => Promise.resolve())
    render(
      <Waiting
        questions={["One?", "Two?"]}
        note=""
        onAnswer={onAnswer}
        onReload={() => Promise.resolve()}
      />
    )
    const send = screen.getByRole("button", {
      name: "Answer and continue",
    }) as HTMLButtonElement
    expect(screen.getAllByText("Required")).toHaveLength(2)
    expect(screen.getByText("0 of 2 answered")).toBeTruthy()
    expect(send.disabled).toBe(true)
    fireEvent.change(screen.getByLabelText("One?"), {
      target: { value: "yes" },
    })
    expect(screen.getByText("1 of 2 answered")).toBeTruthy()
    expect(send.disabled).toBe(true)
    fireEvent.change(screen.getByLabelText("Two?"), { target: { value: "  " } })
    expect(send.disabled).toBe(true)
    fireEvent.change(screen.getByLabelText("Two?"), { target: { value: "no" } })
    expect(screen.getByText("All answered")).toBeTruthy()
    expect(send.disabled).toBe(false)
  })
})
